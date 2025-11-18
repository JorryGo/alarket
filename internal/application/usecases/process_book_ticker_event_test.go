package usecases

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"alarket/internal/domain/entities"
	"alarket/internal/domain/mocks"
	"alarket/internal/infrastructure/clickhouse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNewProcessBookTickerEventUseCase(t *testing.T) {
	logger := slog.Default()
	mockBookTickerRepo := new(mocks.MockBookTickerRepository)

	batchProcessor := clickhouse.NewBookTickerBatchProcessor(
		mockBookTickerRepo,
		logger,
		10,
		100*time.Millisecond,
	)
	defer func() { _ = batchProcessor.Close() }()

	uc := NewProcessBookTickerEventUseCase(batchProcessor, logger)

	assert.NotNil(t, uc)
	assert.NotNil(t, uc.batchProcessor)
	assert.NotNil(t, uc.logger)
}

func TestProcessBookTickerEventUseCase_Execute(t *testing.T) {
	logger := slog.Default()
	ctx := context.Background()

	t.Run("valid book ticker", func(t *testing.T) {
		mockBookTickerRepo := new(mocks.MockBookTickerRepository)

		// Set up expectation for batch save
		mockBookTickerRepo.On("SaveBatch", mock.Anything, mock.AnythingOfType("[]*entities.BookTicker")).Return(nil).Once()

		batchProcessor := clickhouse.NewBookTickerBatchProcessor(
			mockBookTickerRepo,
			logger,
			1, // batch size of 1 for immediate flush
			100*time.Millisecond,
		)
		defer func() { _ = batchProcessor.Close() }()

		uc := NewProcessBookTickerEventUseCase(batchProcessor, logger)

		bookTicker := &entities.BookTicker{
			UpdateID:        123456,
			Symbol:          "BTCUSDT",
			BestBidPrice:    49999.0,
			BestBidQuantity: 1.5,
			BestAskPrice:    50000.0,
			BestAskQuantity: 2.0,
			TransactionTime: time.Now(),
			EventTime:       time.Now(),
		}

		err := uc.Execute(ctx, bookTicker)
		assert.NoError(t, err)

		// Wait for batch processor to flush
		time.Sleep(200 * time.Millisecond)

		mockBookTickerRepo.AssertExpectations(t)
	})

	t.Run("invalid book ticker - empty symbol", func(t *testing.T) {
		mockBookTickerRepo := new(mocks.MockBookTickerRepository)

		batchProcessor := clickhouse.NewBookTickerBatchProcessor(
			mockBookTickerRepo,
			logger,
			10,
			100*time.Millisecond,
		)
		defer func() { _ = batchProcessor.Close() }()

		uc := NewProcessBookTickerEventUseCase(batchProcessor, logger)

		bookTicker := &entities.BookTicker{
			UpdateID:        123456,
			Symbol:          "", // Invalid: empty symbol
			BestBidPrice:    49999.0,
			BestBidQuantity: 1.5,
			BestAskPrice:    50000.0,
			BestAskQuantity: 2.0,
			TransactionTime: time.Now(),
			EventTime:       time.Now(),
		}

		err := uc.Execute(ctx, bookTicker)
		assert.Error(t, err)
		assert.Equal(t, entities.ErrInvalidSymbol, err)

		// Ensure no save was attempted
		mockBookTickerRepo.AssertNotCalled(t, "SaveBatch", mock.Anything, mock.Anything)
	})

	t.Run("invalid book ticker - negative price", func(t *testing.T) {
		mockBookTickerRepo := new(mocks.MockBookTickerRepository)

		batchProcessor := clickhouse.NewBookTickerBatchProcessor(
			mockBookTickerRepo,
			logger,
			10,
			100*time.Millisecond,
		)
		defer func() { _ = batchProcessor.Close() }()

		uc := NewProcessBookTickerEventUseCase(batchProcessor, logger)

		bookTicker := &entities.BookTicker{
			UpdateID:        123456,
			Symbol:          "BTCUSDT",
			BestBidPrice:    -100.0, // Invalid: negative price
			BestBidQuantity: 1.5,
			BestAskPrice:    50000.0,
			BestAskQuantity: 2.0,
			TransactionTime: time.Now(),
			EventTime:       time.Now(),
		}

		err := uc.Execute(ctx, bookTicker)
		assert.Error(t, err)
		assert.Equal(t, entities.ErrInvalidPrice, err)

		// Ensure no save was attempted
		mockBookTickerRepo.AssertNotCalled(t, "SaveBatch", mock.Anything, mock.Anything)
	})

	t.Run("invalid book ticker - inverted spread", func(t *testing.T) {
		mockBookTickerRepo := new(mocks.MockBookTickerRepository)

		batchProcessor := clickhouse.NewBookTickerBatchProcessor(
			mockBookTickerRepo,
			logger,
			10,
			100*time.Millisecond,
		)
		defer func() { _ = batchProcessor.Close() }()

		uc := NewProcessBookTickerEventUseCase(batchProcessor, logger)

		bookTicker := &entities.BookTicker{
			UpdateID:        123456,
			Symbol:          "BTCUSDT",
			BestBidPrice:    50001.0, // Invalid: bid > ask
			BestBidQuantity: 1.5,
			BestAskPrice:    50000.0,
			BestAskQuantity: 2.0,
			TransactionTime: time.Now(),
			EventTime:       time.Now(),
		}

		err := uc.Execute(ctx, bookTicker)
		assert.Error(t, err)
		assert.Equal(t, entities.ErrInvalidSpread, err)

		// Ensure no save was attempted
		mockBookTickerRepo.AssertNotCalled(t, "SaveBatch", mock.Anything, mock.Anything)
	})

	t.Run("batch processing with multiple book tickers", func(t *testing.T) {
		mockBookTickerRepo := new(mocks.MockBookTickerRepository)

		// Set up expectation for batch save with 5 book tickers
		mockBookTickerRepo.On("SaveBatch", mock.Anything, mock.MatchedBy(func(tickers []*entities.BookTicker) bool {
			return len(tickers) == 5
		})).Return(nil).Once()

		batchProcessor := clickhouse.NewBookTickerBatchProcessor(
			mockBookTickerRepo,
			logger,
			5, // batch size of 5
			100*time.Millisecond,
		)
		defer func() { _ = batchProcessor.Close() }()

		uc := NewProcessBookTickerEventUseCase(batchProcessor, logger)

		// Add 5 book tickers to trigger batch
		for i := 0; i < 5; i++ {
			bookTicker := &entities.BookTicker{
				UpdateID:        int64(100000 + i),
				Symbol:          "BTCUSDT",
				BestBidPrice:    49999.0 + float64(i),
				BestBidQuantity: 1.5,
				BestAskPrice:    50000.0 + float64(i),
				BestAskQuantity: 2.0,
				TransactionTime: time.Now(),
				EventTime:       time.Now(),
			}

			err := uc.Execute(ctx, bookTicker)
			assert.NoError(t, err)
		}

		// Wait for batch processor to flush
		time.Sleep(200 * time.Millisecond)

		mockBookTickerRepo.AssertExpectations(t)
	})
}
