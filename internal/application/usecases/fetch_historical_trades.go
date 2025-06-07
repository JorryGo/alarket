package usecases

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"alarket/internal/domain/repositories"
	"alarket/internal/domain/services"
)

type FetchHistoricalTradesUseCase struct {
	tradeRepository       repositories.TradeRepository
	historicalDataService services.HistoricalDataService
	logger                *slog.Logger
	batchSize             int
	rateLimitDelay        time.Duration
}

func NewFetchHistoricalTradesUseCase(
	tradeRepository repositories.TradeRepository,
	historicalDataService services.HistoricalDataService,
	logger *slog.Logger,
) *FetchHistoricalTradesUseCase {
	return &FetchHistoricalTradesUseCase{
		tradeRepository:       tradeRepository,
		historicalDataService: historicalDataService,
		logger:                logger,
		batchSize:             1000,
		rateLimitDelay:        100 * time.Millisecond, // Binance allows 1200 requests per minute
	}
}

func (uc *FetchHistoricalTradesUseCase) Execute(ctx context.Context, symbol string, days int) error {
	targetTime := time.Now().AddDate(0, 0, -days)

	uc.logger.Info("Starting historical trades collection",
		"symbol", symbol,
		"days", days,
		"target_time", targetTime.Format(time.RFC3339))

	// Check if we already have trades for this symbol
	oldestTime, err := uc.tradeRepository.GetOldestTradeTime(ctx, symbol)
	if err != nil {
		return fmt.Errorf("failed to check existing trades: %w", err)
	}

	if oldestTime != nil {
		uc.logger.Info("Found existing trades",
			"oldest_trade_time", oldestTime.Format(time.RFC3339))

		// If we already have enough history, nothing to do
		if oldestTime.Before(targetTime) || oldestTime.Equal(targetTime) {
			uc.logger.Info("Already have sufficient trade history",
				"oldest_time", oldestTime.Format(time.RFC3339),
				"target_time", targetTime.Format(time.RFC3339))
			return nil
		}
	}

	// Determine starting point for fetching
	var fromID int64 = 0
	if oldestTime != nil {
		// We have existing data, get the oldest ID and go backwards
		oldestID, err := uc.tradeRepository.GetOldestTradeID(ctx, symbol)
		if err != nil {
			return fmt.Errorf("failed to get oldest trade ID: %w", err)
		}
		if oldestID != nil {
			fromID = *oldestID - int64(uc.batchSize)
			uc.logger.Info("Starting from existing oldest ID",
				"oldest_id", *oldestID,
				"starting_from_id", fromID)
		}
	} else {
		uc.logger.Info("No existing trades found, starting from latest available")
	}

	totalFetched := 0
	batchCount := 0
	oldestFetched := time.Now()

	for {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Fetch trades from Binance
		trades, err := uc.historicalDataService.FetchHistoricalTrades(ctx, symbol, fromID, uc.batchSize)
		if err != nil {
			return fmt.Errorf("failed to fetch historical trades: %w", err)
		}

		if len(trades) == 0 {
			uc.logger.Info("No more trades available")
			break
		}

		// Process all trades in this batch and find oldest
		for _, trade := range trades {
			if trade.Time.Before(oldestFetched) {
				oldestFetched = trade.Time
			}
		}

		// Save all trades to database
		if err := uc.tradeRepository.SaveBatch(ctx, trades); err != nil {
			return fmt.Errorf("failed to save trades batch: %w", err)
		}

		totalFetched += len(trades)
		batchCount++

		uc.logger.Info("Saved trades batch",
			"batch", batchCount,
			"trades_in_batch", len(trades),
			"total_fetched", totalFetched,
			"oldest_in_batch", oldestFetched.Format(time.RFC3339),
			"from_id", fromID)

		// Check if we've collected enough historical data
		if oldestFetched.Before(targetTime) || oldestFetched.Equal(targetTime) {
			uc.logger.Info("Reached target time",
				"target_time", targetTime.Format(time.RFC3339),
				"oldest_fetched", oldestFetched.Format(time.RFC3339))
			break
		}

		// If we got less than limit, we've reached the end of available data
		if len(trades) < uc.batchSize {
			uc.logger.Info("Reached end of available trades")
			break
		}

		// Move backwards in ID for next batch
		if len(trades) > 0 {
			firstTradeID, err := strconv.ParseInt(trades[0].ID, 10, 64)
			if err == nil {
				fromID = firstTradeID - int64(uc.batchSize)
				uc.logger.Debug("Moving backwards in history",
					"first_id_in_batch", firstTradeID,
					"next_from_id", fromID)
			}
		}

		// Prevent going into negative IDs
		if fromID < 0 {
			uc.logger.Info("Reached beginning of trade history (ID < 0)")
			break
		}

		// Rate limiting - Binance allows 1200 requests per minute
		time.Sleep(uc.rateLimitDelay)
	}

	uc.logger.Info("Historical trades collection completed",
		"symbol", symbol,
		"total_fetched", totalFetched,
		"batches", batchCount,
		"oldest_collected", oldestFetched.Format(time.RFC3339))

	return nil
}
