package services

import (
	"context"
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