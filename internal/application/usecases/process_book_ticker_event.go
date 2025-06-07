package usecases

import (
	"context"
	"log/slog"

	"alarket/internal/domain/entities"
	"alarket/internal/infrastructure/clickhouse"
)

type ProcessBookTickerEventUseCase struct {
	batchProcessor *clickhouse.BookTickerBatchProcessor
	logger         *slog.Logger
}

func NewProcessBookTickerEventUseCase(
	batchProcessor *clickhouse.BookTickerBatchProcessor,
	logger *slog.Logger,
) *ProcessBookTickerEventUseCase {
	return &ProcessBookTickerEventUseCase{
		batchProcessor: batchProcessor,
		logger:         logger,
	}
}

func (uc *ProcessBookTickerEventUseCase) Execute(ctx context.Context, ticker *entities.BookTicker) error {
	return uc.batchProcessor.AddBookTicker(ticker)
}