package usecases

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"alarket/internal/domain/entities"
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

	var fromID int64 = 0
	endTime := time.Now()

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

		// Set end time to before the oldest trade we have
		endTime = oldestTime.Add(-time.Millisecond)
	}

	totalFetched := 0
	batchCount := 0

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

		// Filter trades by end time and find oldest
		var tradesToSave []*entities.Trade
		oldestInBatch := time.Now()

		for _, trade := range trades {
			// Skip trades that are newer than our end time
			if trade.Time.After(endTime) {
				continue
			}

			// Update oldest time in this batch
			if trade.Time.Before(oldestInBatch) {
				oldestInBatch = trade.Time
			}

			tradesToSave = append(tradesToSave, trade)

			// Update fromID for next batch
			tradeID, err := strconv.ParseInt(trade.ID, 10, 64)
			if err == nil && tradeID > fromID {
				fromID = tradeID
			}
		}

		// Save trades to database
		if len(tradesToSave) > 0 {
			if err := uc.tradeRepository.SaveBatch(ctx, tradesToSave); err != nil {
				return fmt.Errorf("failed to save trades batch: %w", err)
			}

			totalFetched += len(tradesToSave)
			batchCount++

			uc.logger.Info("Saved trades batch",
				"batch", batchCount,
				"trades_in_batch", len(tradesToSave),
				"total_fetched", totalFetched,
				"oldest_in_batch", oldestInBatch.Format(time.RFC3339))
		}

		// Check if we've reached our target time
		if oldestInBatch.Before(targetTime) || oldestInBatch.Equal(targetTime) {
			uc.logger.Info("Reached target time",
				"target_time", targetTime.Format(time.RFC3339),
				"oldest_fetched", oldestInBatch.Format(time.RFC3339))
			break
		}

		// If we got less than limit, we've reached the end
		if len(trades) < uc.batchSize {
			uc.logger.Info("Reached end of available trades")
			break
		}

		// Rate limiting
		time.Sleep(uc.rateLimitDelay)
	}

	uc.logger.Info("Historical trades collection completed",
		"symbol", symbol,
		"total_fetched", totalFetched,
		"batches", batchCount)

	return nil
}
