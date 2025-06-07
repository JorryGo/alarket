package binance

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"alarket/internal/domain/entities"
)

type HistoricalTradesService struct {
	client     *binance.Client
	logger     *slog.Logger
	useTestnet bool
}

func NewHistoricalTradesService(apiKey, secretKey string, useTestnet bool, logger *slog.Logger) *HistoricalTradesService {
	client := binance.NewClient(apiKey, secretKey)

	if useTestnet {
		binance.UseTestnet = true
	}

	return &HistoricalTradesService{
		client:     client,
		logger:     logger,
		useTestnet: useTestnet,
	}
}

func (s *HistoricalTradesService) FetchHistoricalTrades(ctx context.Context, symbol string, fromID int64, limit int) ([]*entities.Trade, error) {
	service := s.client.NewHistoricalTradesService().
		Symbol(symbol).
		Limit(limit)

	if fromID > 0 {
		service = service.FromID(fromID)
	}

	binanceTrades, err := service.Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch historical trades from Binance: %w", err)
	}

	trades := make([]*entities.Trade, 0, len(binanceTrades))
	for _, bt := range binanceTrades {
		price, err := strconv.ParseFloat(bt.Price, 64)
		if err != nil {
			s.logger.Warn("Failed to parse price", "price", bt.Price, "error", err)
			continue
		}

		quantity, err := strconv.ParseFloat(bt.Quantity, 64)
		if err != nil {
			s.logger.Warn("Failed to parse quantity", "quantity", bt.Quantity, "error", err)
			continue
		}

		tradeTime := time.Unix(0, bt.Time*int64(time.Millisecond))

		trade := entities.NewTrade(
			fmt.Sprintf("%d", bt.ID),
			symbol,
			price,
			quantity,
			tradeTime,
			bt.IsBuyerMaker,
			tradeTime, // Using trade time as event time for historical data
		)

		trades = append(trades, trade)
	}

	return trades, nil
}

func (s *HistoricalTradesService) GetLastTradeID(ctx context.Context, symbol string) (int64, error) {
	// Get the most recent trades
	trades, err := s.FetchHistoricalTrades(ctx, symbol, 0, 1)
	if err != nil {
		return 0, fmt.Errorf("failed to get last trade: %w", err)
	}

	if len(trades) == 0 {
		return 0, fmt.Errorf("no trades found for symbol %s", symbol)
	}

	lastID, err := strconv.ParseInt(trades[0].ID, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse trade ID: %w", err)
	}

	return lastID, nil
}
