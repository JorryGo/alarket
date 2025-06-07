package repositories

import (
	"context"
	"time"

	"alarket/internal/domain/entities"
)

type TradeRepository interface {
	Save(ctx context.Context, trade *entities.Trade) error
	SaveBatch(ctx context.Context, trades []*entities.Trade) error
	GetBySymbol(ctx context.Context, symbol string, from, to time.Time) ([]*entities.Trade, error)
	GetByID(ctx context.Context, id string) (*entities.Trade, error)
	GetOldestTradeTime(ctx context.Context, symbol string) (*time.Time, error)
	GetOldestTradeID(ctx context.Context, symbol string) (*int64, error)
}
