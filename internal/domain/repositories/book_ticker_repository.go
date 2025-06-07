package repositories

import (
	"context"
	"time"

	"alarket/internal/domain/entities"
)

type BookTickerRepository interface {
	Save(ctx context.Context, ticker *entities.BookTicker) error
	SaveBatch(ctx context.Context, tickers []*entities.BookTicker) error
	GetLatestBySymbol(ctx context.Context, symbol string) (*entities.BookTicker, error)
	GetBySymbol(ctx context.Context, symbol string, from, to time.Time) ([]*entities.BookTicker, error)
}
