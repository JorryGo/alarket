package mocks

import (
	"context"
	"time"

	"alarket/internal/domain/entities"
	"github.com/stretchr/testify/mock"
)

// MockTradeRepository is a mock implementation of TradeRepository
type MockTradeRepository struct {
	mock.Mock
}

func (m *MockTradeRepository) Save(ctx context.Context, trade *entities.Trade) error {
	args := m.Called(ctx, trade)
	return args.Error(0)
}

func (m *MockTradeRepository) SaveBatch(ctx context.Context, trades []*entities.Trade) error {
	args := m.Called(ctx, trades)
	return args.Error(0)
}

func (m *MockTradeRepository) GetBySymbol(ctx context.Context, symbol string, from, to time.Time) ([]*entities.Trade, error) {
	args := m.Called(ctx, symbol, from, to)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.Trade), args.Error(1)
}

func (m *MockTradeRepository) GetByID(ctx context.Context, id string) (*entities.Trade, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Trade), args.Error(1)
}

func (m *MockTradeRepository) GetOldestTradeTime(ctx context.Context, symbol string) (*time.Time, error) {
	args := m.Called(ctx, symbol)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*time.Time), args.Error(1)
}

func (m *MockTradeRepository) GetOldestTradeID(ctx context.Context, symbol string) (*int64, error) {
	args := m.Called(ctx, symbol)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*int64), args.Error(1)
}

func (m *MockTradeRepository) GetNewestTradeID(ctx context.Context, symbol string) (*int64, error) {
	args := m.Called(ctx, symbol)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*int64), args.Error(1)
}

// MockSymbolRepository is a mock implementation of SymbolRepository
type MockSymbolRepository struct {
	mock.Mock
}

func (m *MockSymbolRepository) GetAll(ctx context.Context) ([]*entities.Symbol, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.Symbol), args.Error(1)
}

func (m *MockSymbolRepository) GetActiveUsdt(ctx context.Context) ([]*entities.Symbol, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.Symbol), args.Error(1)
}

func (m *MockSymbolRepository) GetByName(ctx context.Context, name string) (*entities.Symbol, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Symbol), args.Error(1)
}

func (m *MockSymbolRepository) UpdateStatus(ctx context.Context, name string, status entities.SymbolStatus) error {
	args := m.Called(ctx, name, status)
	return args.Error(0)
}

// MockBookTickerRepository is a mock implementation of BookTickerRepository
type MockBookTickerRepository struct {
	mock.Mock
}

func (m *MockBookTickerRepository) Save(ctx context.Context, ticker *entities.BookTicker) error {
	args := m.Called(ctx, ticker)
	return args.Error(0)
}

func (m *MockBookTickerRepository) SaveBatch(ctx context.Context, tickers []*entities.BookTicker) error {
	args := m.Called(ctx, tickers)
	return args.Error(0)
}

func (m *MockBookTickerRepository) GetLatestBySymbol(ctx context.Context, symbol string) (*entities.BookTicker, error) {
	args := m.Called(ctx, symbol)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.BookTicker), args.Error(1)
}

func (m *MockBookTickerRepository) GetBySymbol(ctx context.Context, symbol string, from, to time.Time) ([]*entities.BookTicker, error) {
	args := m.Called(ctx, symbol, from, to)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.BookTicker), args.Error(1)
}