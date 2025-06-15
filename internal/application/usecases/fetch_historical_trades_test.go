package usecases

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"alarket/internal/domain/entities"
	"alarket/internal/domain/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNewFetchHistoricalTradesUseCase(t *testing.T) {
	mockTradeRepo := new(mocks.MockTradeRepository)
	mockHistoricalService := new(mocks.MockHistoricalDataService)
	logger := slog.Default()
	
	uc := NewFetchHistoricalTradesUseCase(mockTradeRepo, mockHistoricalService, logger)
	
	assert.NotNil(t, uc)
	assert.NotNil(t, uc.tradeRepository)
	assert.NotNil(t, uc.historicalDataService)
	assert.NotNil(t, uc.logger)
	assert.Equal(t, 1000, uc.batchSize)
	assert.Equal(t, 100*time.Millisecond, uc.rateLimitDelay)
}

func TestFetchHistoricalTradesUseCase_ExecuteForward(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()
	
	t.Run("forward fetch with existing trades", func(t *testing.T) {
		mockTradeRepo := new(mocks.MockTradeRepository)
		mockHistoricalService := new(mocks.MockHistoricalDataService)
		
		// Setup existing newest trade ID
		newestID := int64(1000)
		mockTradeRepo.On("GetNewestTradeID", ctx, "BTCUSDT").Return(&newestID, nil)
		
		// Create test trades
		now := time.Now()
		trades := []*entities.Trade{
			{
				ID:       "1001",
				Symbol:   "BTCUSDT",
				Price:    50000.0,
				Quantity: 0.01,
				Time:     now.Add(-1 * time.Hour),
			},
			{
				ID:       "1002",
				Symbol:   "BTCUSDT",
				Price:    50100.0,
				Quantity: 0.02,
				Time:     now.Add(-30 * time.Minute),
			},
		}
		
		// Mock historical service to return trades
		mockHistoricalService.On("FetchHistoricalTrades", ctx, "BTCUSDT", int64(1001), 1000).Return(trades, nil)
		
		// Mock save batch
		mockTradeRepo.On("SaveBatch", ctx, trades).Return(nil)
		
		uc := NewFetchHistoricalTradesUseCase(mockTradeRepo, mockHistoricalService, logger)
		uc.batchSize = 1000
		uc.rateLimitDelay = 0 // No delay for testing
		
		err := uc.Execute(ctx, "BTCUSDT", 0, true) // Forward fetch to current time
		assert.NoError(t, err)
		
		mockTradeRepo.AssertExpectations(t)
		mockHistoricalService.AssertExpectations(t)
	})
	
	t.Run("forward fetch with no existing trades", func(t *testing.T) {
		mockTradeRepo := new(mocks.MockTradeRepository)
		mockHistoricalService := new(mocks.MockHistoricalDataService)
		
		// No existing trades
		mockTradeRepo.On("GetNewestTradeID", ctx, "BTCUSDT").Return(nil, nil)
		
		// Create test trades
		now := time.Now()
		trades := []*entities.Trade{
			{
				ID:       "1",
				Symbol:   "BTCUSDT",
				Price:    50000.0,
				Quantity: 0.01,
				Time:     now.Add(-2 * time.Hour),
			},
		}
		
		// Mock historical service
		mockHistoricalService.On("FetchHistoricalTrades", ctx, "BTCUSDT", int64(0), 1000).Return(trades, nil)
		
		// Mock save batch
		mockTradeRepo.On("SaveBatch", ctx, trades).Return(nil)
		
		uc := NewFetchHistoricalTradesUseCase(mockTradeRepo, mockHistoricalService, logger)
		uc.batchSize = 1000
		uc.rateLimitDelay = 0
		
		err := uc.Execute(ctx, "BTCUSDT", 0, true)
		assert.NoError(t, err)
		
		mockTradeRepo.AssertExpectations(t)
		mockHistoricalService.AssertExpectations(t)
	})
}

func TestFetchHistoricalTradesUseCase_ExecuteBackward(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()
	
	t.Run("backward fetch with sufficient existing history", func(t *testing.T) {
		mockTradeRepo := new(mocks.MockTradeRepository)
		mockHistoricalService := new(mocks.MockHistoricalDataService)
		
		// Setup existing oldest trade time (10 days ago)
		oldestTime := time.Now().AddDate(0, 0, -10)
		mockTradeRepo.On("GetOldestTradeTime", ctx, "BTCUSDT").Return(&oldestTime, nil)
		
		uc := NewFetchHistoricalTradesUseCase(mockTradeRepo, mockHistoricalService, logger)
		
		// Try to fetch 7 days of history (we already have 10 days)
		err := uc.Execute(ctx, "BTCUSDT", 7, false)
		assert.NoError(t, err)
		
		// Should not fetch any new data
		mockTradeRepo.AssertExpectations(t)
		mockHistoricalService.AssertNotCalled(t, "FetchHistoricalTrades", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})
	
	t.Run("backward fetch with insufficient history", func(t *testing.T) {
		mockTradeRepo := new(mocks.MockTradeRepository)
		mockHistoricalService := new(mocks.MockHistoricalDataService)
		
		// Setup existing oldest trade time (2 days ago)
		oldestTime := time.Now().AddDate(0, 0, -2)
		oldestID := int64(1000)
		mockTradeRepo.On("GetOldestTradeTime", ctx, "BTCUSDT").Return(&oldestTime, nil)
		mockTradeRepo.On("GetOldestTradeID", ctx, "BTCUSDT").Return(&oldestID, nil)
		
		// Create test trades (older than existing)
		trades := []*entities.Trade{
			{
				ID:       "900",
				Symbol:   "BTCUSDT",
				Price:    49000.0,
				Quantity: 0.01,
				Time:     time.Now().AddDate(0, 0, -5), // 5 days ago
			},
		}
		
		// Mock historical service
		mockHistoricalService.On("FetchHistoricalTrades", ctx, "BTCUSDT", int64(0), 1000).Return(trades, nil)
		
		// Mock save batch
		mockTradeRepo.On("SaveBatch", ctx, trades).Return(nil)
		
		uc := NewFetchHistoricalTradesUseCase(mockTradeRepo, mockHistoricalService, logger)
		uc.batchSize = 1000
		uc.rateLimitDelay = 0
		
		// Try to fetch 7 days of history
		err := uc.Execute(ctx, "BTCUSDT", 7, false)
		assert.NoError(t, err)
		
		mockTradeRepo.AssertExpectations(t)
		mockHistoricalService.AssertExpectations(t)
	})
	
	t.Run("backward fetch with no existing trades", func(t *testing.T) {
		mockTradeRepo := new(mocks.MockTradeRepository)
		mockHistoricalService := new(mocks.MockHistoricalDataService)
		
		// No existing trades
		mockTradeRepo.On("GetOldestTradeTime", ctx, "BTCUSDT").Return(nil, nil)
		
		// Create test trades
		trades := []*entities.Trade{
			{
				ID:       "1",
				Symbol:   "BTCUSDT",
				Price:    50000.0,
				Quantity: 0.01,
				Time:     time.Now().AddDate(0, 0, -3),
			},
		}
		
		// Mock historical service
		mockHistoricalService.On("FetchHistoricalTrades", ctx, "BTCUSDT", int64(0), 1000).Return(trades, nil)
		
		// Mock save batch
		mockTradeRepo.On("SaveBatch", ctx, trades).Return(nil)
		
		uc := NewFetchHistoricalTradesUseCase(mockTradeRepo, mockHistoricalService, logger)
		uc.batchSize = 1000
		uc.rateLimitDelay = 0
		
		err := uc.Execute(ctx, "BTCUSDT", 7, false)
		assert.NoError(t, err)
		
		mockTradeRepo.AssertExpectations(t)
		mockHistoricalService.AssertExpectations(t)
	})
}

func TestFetchHistoricalTradesUseCase_ErrorHandling(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()
	
	t.Run("error getting newest trade ID", func(t *testing.T) {
		mockTradeRepo := new(mocks.MockTradeRepository)
		mockHistoricalService := new(mocks.MockHistoricalDataService)
		
		expectedErr := errors.New("database error")
		mockTradeRepo.On("GetNewestTradeID", ctx, "BTCUSDT").Return(nil, expectedErr)
		
		uc := NewFetchHistoricalTradesUseCase(mockTradeRepo, mockHistoricalService, logger)
		
		err := uc.Execute(ctx, "BTCUSDT", 0, true)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get newest trade ID")
		
		mockTradeRepo.AssertExpectations(t)
	})
	
	t.Run("error fetching historical trades", func(t *testing.T) {
		mockTradeRepo := new(mocks.MockTradeRepository)
		mockHistoricalService := new(mocks.MockHistoricalDataService)
		
		mockTradeRepo.On("GetOldestTradeTime", ctx, "BTCUSDT").Return(nil, nil)
		
		expectedErr := errors.New("API error")
		mockHistoricalService.On("FetchHistoricalTrades", ctx, "BTCUSDT", int64(0), 1000).Return(nil, expectedErr)
		
		uc := NewFetchHistoricalTradesUseCase(mockTradeRepo, mockHistoricalService, logger)
		
		err := uc.Execute(ctx, "BTCUSDT", 7, false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to fetch historical trades")
		
		mockTradeRepo.AssertExpectations(t)
		mockHistoricalService.AssertExpectations(t)
	})
	
	t.Run("error saving trades batch", func(t *testing.T) {
		mockTradeRepo := new(mocks.MockTradeRepository)
		mockHistoricalService := new(mocks.MockHistoricalDataService)
		
		mockTradeRepo.On("GetOldestTradeTime", ctx, "BTCUSDT").Return(nil, nil)
		
		trades := []*entities.Trade{
			{
				ID:       "1",
				Symbol:   "BTCUSDT",
				Price:    50000.0,
				Quantity: 0.01,
				Time:     time.Now(),
			},
		}
		
		mockHistoricalService.On("FetchHistoricalTrades", ctx, "BTCUSDT", int64(0), 1000).Return(trades, nil)
		
		expectedErr := errors.New("save error")
		mockTradeRepo.On("SaveBatch", ctx, trades).Return(expectedErr)
		
		uc := NewFetchHistoricalTradesUseCase(mockTradeRepo, mockHistoricalService, logger)
		uc.rateLimitDelay = 0
		
		err := uc.Execute(ctx, "BTCUSDT", 7, false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to save trades batch")
		
		mockTradeRepo.AssertExpectations(t)
		mockHistoricalService.AssertExpectations(t)
	})
	
	t.Run("context cancellation", func(t *testing.T) {
		mockTradeRepo := new(mocks.MockTradeRepository)
		mockHistoricalService := new(mocks.MockHistoricalDataService)
		
		// Create a context that can be cancelled
		ctx, cancel := context.WithCancel(context.Background())
		
		mockTradeRepo.On("GetOldestTradeTime", ctx, "BTCUSDT").Return(nil, nil)
		
		// Cancel context immediately
		cancel()
		
		uc := NewFetchHistoricalTradesUseCase(mockTradeRepo, mockHistoricalService, logger)
		
		err := uc.Execute(ctx, "BTCUSDT", 7, false)
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
		
		mockTradeRepo.AssertExpectations(t)
	})
}