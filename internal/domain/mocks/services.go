package mocks

import (
	"context"

	"alarket/internal/domain/entities"
	"github.com/stretchr/testify/mock"
)

// MockExchangeClient is a mock implementation of ExchangeClient
type MockExchangeClient struct {
	mock.Mock
}

func (m *MockExchangeClient) SubscribeToTrades(ctx context.Context, symbols []string) error {
	args := m.Called(ctx, symbols)
	return args.Error(0)
}

func (m *MockExchangeClient) SubscribeToBookTickers(ctx context.Context, symbols []string) error {
	args := m.Called(ctx, symbols)
	return args.Error(0)
}

func (m *MockExchangeClient) UnsubscribeFromTrades(ctx context.Context, symbols []string) error {
	args := m.Called(ctx, symbols)
	return args.Error(0)
}

func (m *MockExchangeClient) UnsubscribeFromBookTickers(ctx context.Context, symbols []string) error {
	args := m.Called(ctx, symbols)
	return args.Error(0)
}

func (m *MockExchangeClient) Close() error {
	args := m.Called()
	return args.Error(0)
}

// MockEventPublisher is a mock implementation of EventPublisher
type MockEventPublisher struct {
	mock.Mock
}

func (m *MockEventPublisher) Publish(ctx context.Context, event interface{}) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

// MockEventSubscriber is a mock implementation of EventSubscriber
type MockEventSubscriber struct {
	mock.Mock
}

func (m *MockEventSubscriber) Subscribe(ctx context.Context, handler func(event interface{}) error) error {
	args := m.Called(ctx, handler)
	return args.Error(0)
}

// MockHistoricalDataService is a mock implementation of HistoricalDataService
type MockHistoricalDataService struct {
	mock.Mock
}

func (m *MockHistoricalDataService) FetchHistoricalTrades(ctx context.Context, symbol string, fromID int64, limit int) ([]*entities.Trade, error) {
	args := m.Called(ctx, symbol, fromID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.Trade), args.Error(1)
}