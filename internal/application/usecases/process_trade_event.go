package usecases

import (
	"context"
	"log/slog"

	"alarket/internal/domain/entities"
	"alarket/internal/infrastructure/clickhouse"
)

type ProcessTradeEventUseCase struct {
	batchProcessor *clickhouse.TradeBatchProcessor
	logger         *slog.Logger
}

func NewProcessTradeEventUseCase(
	batchProcessor *clickhouse.TradeBatchProcessor,
	logger *slog.Logger,
) *ProcessTradeEventUseCase {
	return &ProcessTradeEventUseCase{
		batchProcessor: batchProcessor,
		logger:         logger,
	}
}

func (uc *ProcessTradeEventUseCase) Execute(ctx context.Context, trade *entities.Trade) error {
	return uc.batchProcessor.AddTrade(trade)
}