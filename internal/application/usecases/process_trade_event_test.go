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

func TestNewProcessTradeEventUseCase(t *testing.T) {
	logger := slog.Default()
	mockTradeRepo := new(mocks.MockTradeRepository)

	batchProcessor := clickhouse.NewTradeBatchProcessor(
		mockTradeRepo,
		logger,
		10,
		100*time.Millisecond,
	)
	defer func() { _ = batchProcessor.Close() }()

	uc := NewProcessTradeEventUseCase(batchProcessor, logger)

	assert.NotNil(t, uc)
	assert.NotNil(t, uc.batchProcessor)
	assert.NotNil(t, uc.logger)
}

func TestProcessTradeEventUseCase_Execute(t *testing.T) {
	logger := slog.Default()
	ctx := context.Background()

	t.Run("valid trade", func(t *testing.T) {
		mockTradeRepo := new(mocks.MockTradeRepository)

		// Set up expectation for batch save
		mockTradeRepo.On("SaveBatch", mock.Anything, mock.AnythingOfType("[]*entities.Trade")).Return(nil).Once()

		batchProcessor := clickhouse.NewTradeBatchProcessor(
			mockTradeRepo,
			logger,
			1, // batch size of 1 for immediate flush
			100*time.Millisecond,
		)
		defer func() { _ = batchProcessor.Close() }()

		uc := NewProcessTradeEventUseCase(batchProcessor, logger)

		trade := &entities.Trade{
			ID:           "123456",
			Symbol:       "BTCUSDT",
			Price:        50000.0,
			Quantity:     0.01,
			Time:         time.Now(),
			IsBuyerMaker: true,
			EventTime:    time.Now(),
		}

		err := uc.Execute(ctx, trade)
		assert.NoError(t, err)

		// Wait for batch processor to flush
		time.Sleep(200 * time.Millisecond)

		mockTradeRepo.AssertExpectations(t)
	})

	t.Run("invalid trade - empty symbol", func(t *testing.T) {
		mockTradeRepo := new(mocks.MockTradeRepository)

		batchProcessor := clickhouse.NewTradeBatchProcessor(
			mockTradeRepo,
			logger,
			10,
			100*time.Millisecond,
		)
		defer func() { _ = batchProcessor.Close() }()

		uc := NewProcessTradeEventUseCase(batchProcessor, logger)

		trade := &entities.Trade{
			ID:           "123456",
			Symbol:       "", // Invalid: empty symbol
			Price:        50000.0,
			Quantity:     0.01,
			Time:         time.Now(),
			IsBuyerMaker: true,
			EventTime:    time.Now(),
		}

		err := uc.Execute(ctx, trade)
		assert.Error(t, err)
		assert.Equal(t, entities.ErrInvalidSymbol, err)

		// Ensure no save was attempted
		mockTradeRepo.AssertNotCalled(t, "SaveBatch", mock.Anything, mock.Anything)
	})

	t.Run("invalid trade - negative price", func(t *testing.T) {
		mockTradeRepo := new(mocks.MockTradeRepository)

		batchProcessor := clickhouse.NewTradeBatchProcessor(
			mockTradeRepo,
			logger,
			10,
			100*time.Millisecond,
		)
		defer func() { _ = batchProcessor.Close() }()

		uc := NewProcessTradeEventUseCase(batchProcessor, logger)

		trade := &entities.Trade{
			ID:           "123456",
			Symbol:       "BTCUSDT",
			Price:        -100.0, // Invalid: negative price
			Quantity:     0.01,
			Time:         time.Now(),
			IsBuyerMaker: true,
			EventTime:    time.Now(),
		}

		err := uc.Execute(ctx, trade)
		assert.Error(t, err)
		assert.Equal(t, entities.ErrInvalidPrice, err)

		// Ensure no save was attempted
		mockTradeRepo.AssertNotCalled(t, "SaveBatch", mock.Anything, mock.Anything)
	})

	t.Run("batch processing with multiple trades", func(t *testing.T) {
		mockTradeRepo := new(mocks.MockTradeRepository)

		// Set up expectation for batch save with 3 trades
		mockTradeRepo.On("SaveBatch", mock.Anything, mock.MatchedBy(func(trades []*entities.Trade) bool {
			return len(trades) == 3
		})).Return(nil).Once()

		batchProcessor := clickhouse.NewTradeBatchProcessor(
			mockTradeRepo,
			logger,
			3, // batch size of 3
			100*time.Millisecond,
		)
		defer func() { _ = batchProcessor.Close() }()

		uc := NewProcessTradeEventUseCase(batchProcessor, logger)

		// Add 3 trades to trigger batch
		for i := 0; i < 3; i++ {
			trade := &entities.Trade{
				ID:           string(rune('1' + i)),
				Symbol:       "BTCUSDT",
				Price:        50000.0 + float64(i),
				Quantity:     0.01,
				Time:         time.Now(),
				IsBuyerMaker: i%2 == 0,
				EventTime:    time.Now(),
			}

			err := uc.Execute(ctx, trade)
			assert.NoError(t, err)
		}

		// Wait for batch processor to flush
		time.Sleep(200 * time.Millisecond)

		mockTradeRepo.AssertExpectations(t)
	})
}
