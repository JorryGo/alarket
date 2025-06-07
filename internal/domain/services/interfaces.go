package services

import (
	"alarket/internal/domain/entities"
)

type ExchangeClient interface {
	SubscribeToTrades(ctx context.Context, symbols []string) error
	SubscribeToBookTickers(ctx context.Context, symbols []string) error
	UnsubscribeFromTrades(ctx context.Context, symbols []string) error
	UnsubscribeFromBookTickers(ctx context.Context, symbols []string) error
	Close() error
}

type EventPublisher interface {
	Publish(ctx context.Context, event interface{}) error
}

type EventSubscriber interface {
	Subscribe(ctx context.Context, handler func(event interface{}) error) error
}

type HistoricalDataService interface {
	FetchHistoricalTrades(ctx context.Context, symbol string, fromID int64, limit int) ([]*entities.Trade, error)
	GetLastTradeID(ctx context.Context, symbol string) (int64, error)
}
