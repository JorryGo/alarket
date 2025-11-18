package services

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"alarket/internal/application/dto"
	"alarket/internal/application/usecases"
	"alarket/internal/domain/mocks"
	"alarket/internal/infrastructure/clickhouse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewEventHandler(t *testing.T) {
	logger := slog.Default()

	// Create mock repositories
	mockTradeRepo := new(mocks.MockTradeRepository)
	mockBookTickerRepo := new(mocks.MockBookTickerRepository)

	// Create batch processors with small timeout for testing
	tradeBatchProcessor := clickhouse.NewTradeBatchProcessor(
		mockTradeRepo,
		logger,
		10,                  // small batch size
		10*time.Millisecond, // short timeout
	)
	defer func() { _ = tradeBatchProcessor.Close() }()

	bookTickerBatchProcessor := clickhouse.NewBookTickerBatchProcessor(
		mockBookTickerRepo,
		logger,
		10,
		10*time.Millisecond,
	)
	defer func() { _ = bookTickerBatchProcessor.Close() }()

	// Create use cases with batch processors
	processTradeUC := usecases.NewProcessTradeEventUseCase(tradeBatchProcessor, logger)
	processBookTickerUC := usecases.NewProcessBookTickerEventUseCase(bookTickerBatchProcessor, logger)

	handler := NewEventHandler(processTradeUC, processBookTickerUC, logger)

	assert.NotNil(t, handler)
	assert.NotNil(t, handler.processTradeUC)
	assert.NotNil(t, handler.processBookTickerUC)
	assert.NotNil(t, handler.logger)
}

func TestEventHandler_HandleMessage_TradeEvent(t *testing.T) {
	logger := slog.Default()
	ctx := context.Background()

	t.Run("valid trade event", func(t *testing.T) {
		// Create mock repository
		mockTradeRepo := new(mocks.MockTradeRepository)
		mockBookTickerRepo := new(mocks.MockBookTickerRepository)

		// Set up expectation for batch save
		mockTradeRepo.On("SaveBatch", mock.Anything, mock.AnythingOfType("[]*entities.Trade")).Return(nil).Once()

		// Create batch processors
		tradeBatchProcessor := clickhouse.NewTradeBatchProcessor(
			mockTradeRepo,
			logger,
			1, // batch size of 1 for immediate flush
			100*time.Millisecond,
		)
		defer func() { _ = tradeBatchProcessor.Close() }()

		bookTickerBatchProcessor := clickhouse.NewBookTickerBatchProcessor(
			mockBookTickerRepo,
			logger,
			10,
			100*time.Millisecond,
		)
		defer func() { _ = bookTickerBatchProcessor.Close() }()

		// Create use cases and handler
		processTradeUC := usecases.NewProcessTradeEventUseCase(tradeBatchProcessor, logger)
		processBookTickerUC := usecases.NewProcessBookTickerEventUseCase(bookTickerBatchProcessor, logger)
		handler := NewEventHandler(processTradeUC, processBookTickerUC, logger)

		// Create trade event
		tradeEvent := dto.TradeEventDTO{
			EventType:          "trade",
			EventTime:          time.Now().UnixMilli(),
			Symbol:             "BTCUSDT",
			TradeID:            123456,
			Price:              "50000.00",
			Quantity:           "0.01",
			TradeTime:          time.Now().UnixMilli(),
			IsBuyerMarketMaker: true,
			Ignore:             false,
		}

		message, err := json.Marshal(tradeEvent)
		require.NoError(t, err)

		err = handler.HandleMessage(ctx, message)
		assert.NoError(t, err)

		// Wait for batch processor to flush
		time.Sleep(200 * time.Millisecond)

		mockTradeRepo.AssertExpectations(t)
	})

	t.Run("invalid price in trade event", func(t *testing.T) {
		handler := createTestHandler()

		tradeEvent := dto.TradeEventDTO{
			EventType: "trade",
			EventTime: time.Now().UnixMilli(),
			Symbol:    "BTCUSDT",
			TradeID:   123456,
			Price:     "invalid", // Invalid price
			Quantity:  "0.01",
			TradeTime: time.Now().UnixMilli(),
		}

		message, err := json.Marshal(tradeEvent)
		require.NoError(t, err)

		err = handler.HandleMessage(ctx, message)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid price")
	})

	t.Run("invalid quantity in trade event", func(t *testing.T) {
		handler := createTestHandler()

		tradeEvent := dto.TradeEventDTO{
			EventType: "trade",
			EventTime: time.Now().UnixMilli(),
			Symbol:    "BTCUSDT",
			TradeID:   123456,
			Price:     "50000.00",
			Quantity:  "invalid", // Invalid quantity
			TradeTime: time.Now().UnixMilli(),
		}

		message, err := json.Marshal(tradeEvent)
		require.NoError(t, err)

		err = handler.HandleMessage(ctx, message)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid quantity")
	})
}

func TestEventHandler_HandleMessage_BookTickerEvent(t *testing.T) {
	logger := slog.Default()
	ctx := context.Background()

	t.Run("valid book ticker event", func(t *testing.T) {
		// Create mock repository
		mockTradeRepo := new(mocks.MockTradeRepository)
		mockBookTickerRepo := new(mocks.MockBookTickerRepository)

		// Set up expectation for batch save
		mockBookTickerRepo.On("SaveBatch", mock.Anything, mock.AnythingOfType("[]*entities.BookTicker")).Return(nil).Once()

		// Create batch processors
		tradeBatchProcessor := clickhouse.NewTradeBatchProcessor(
			mockTradeRepo,
			logger,
			10,
			100*time.Millisecond,
		)
		defer func() { _ = tradeBatchProcessor.Close() }()

		bookTickerBatchProcessor := clickhouse.NewBookTickerBatchProcessor(
			mockBookTickerRepo,
			logger,
			1, // batch size of 1 for immediate flush
			100*time.Millisecond,
		)
		defer func() { _ = bookTickerBatchProcessor.Close() }()

		// Create use cases and handler
		processTradeUC := usecases.NewProcessTradeEventUseCase(tradeBatchProcessor, logger)
		processBookTickerUC := usecases.NewProcessBookTickerEventUseCase(bookTickerBatchProcessor, logger)
		handler := NewEventHandler(processTradeUC, processBookTickerUC, logger)

		// Create book ticker event
		bookTickerEvent := dto.BookTickerEventDTO{
			UpdateID:        789012,
			Symbol:          "BTCUSDT",
			BestBidPrice:    "49999.00",
			BestBidQuantity: "1.5",
			BestAskPrice:    "50000.00",
			BestAskQuantity: "2.0",
		}

		message, err := json.Marshal(bookTickerEvent)
		require.NoError(t, err)

		err = handler.HandleMessage(ctx, message)
		assert.NoError(t, err)

		// Wait for batch processor to flush
		time.Sleep(200 * time.Millisecond)

		mockBookTickerRepo.AssertExpectations(t)
	})

	t.Run("invalid bid price in book ticker event", func(t *testing.T) {
		handler := createTestHandler()

		bookTickerEvent := dto.BookTickerEventDTO{
			UpdateID:        789012,
			Symbol:          "BTCUSDT",
			BestBidPrice:    "invalid", // Invalid price
			BestBidQuantity: "1.5",
			BestAskPrice:    "50000.00",
			BestAskQuantity: "2.0",
		}

		message, err := json.Marshal(bookTickerEvent)
		require.NoError(t, err)

		err = handler.HandleMessage(ctx, message)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid bid price")
	})

	t.Run("invalid ask quantity in book ticker event", func(t *testing.T) {
		handler := createTestHandler()

		bookTickerEvent := dto.BookTickerEventDTO{
			UpdateID:        789012,
			Symbol:          "BTCUSDT",
			BestBidPrice:    "49999.00",
			BestBidQuantity: "1.5",
			BestAskPrice:    "50000.00",
			BestAskQuantity: "invalid", // Invalid quantity
		}

		message, err := json.Marshal(bookTickerEvent)
		require.NoError(t, err)

		err = handler.HandleMessage(ctx, message)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid ask quantity")
	})
}

func TestEventHandler_HandleMessage_InvalidMessages(t *testing.T) {
	handler := createTestHandler()
	ctx := context.Background()

	t.Run("invalid JSON", func(t *testing.T) {
		message := []byte("invalid json")
		err := handler.HandleMessage(ctx, message)
		assert.Error(t, err)
	})

	t.Run("unknown event type", func(t *testing.T) {
		unknownEvent := map[string]interface{}{
			"e": "unknown",
		}
		message, err := json.Marshal(unknownEvent)
		require.NoError(t, err)

		err = handler.HandleMessage(ctx, message)
		assert.NoError(t, err) // Should not error, just log and return
	})

	t.Run("non-event message", func(t *testing.T) {
		nonEvent := map[string]interface{}{
			"some": "data",
		}
		message, err := json.Marshal(nonEvent)
		require.NoError(t, err)

		err = handler.HandleMessage(ctx, message)
		assert.NoError(t, err) // Should not error, just log and return
	})

	t.Run("empty message", func(t *testing.T) {
		message := []byte("{}")
		err := handler.HandleMessage(ctx, message)
		assert.NoError(t, err) // Should not error, just log and return
	})
}

// Helper function to create a test handler with mocked dependencies
func createTestHandler() *EventHandler {
	logger := slog.Default()

	// Create mock repositories
	mockTradeRepo := new(mocks.MockTradeRepository)
	mockBookTickerRepo := new(mocks.MockBookTickerRepository)

	// Create batch processors
	tradeBatchProcessor := clickhouse.NewTradeBatchProcessor(
		mockTradeRepo,
		logger,
		100,
		1*time.Second,
	)

	bookTickerBatchProcessor := clickhouse.NewBookTickerBatchProcessor(
		mockBookTickerRepo,
		logger,
		100,
		1*time.Second,
	)

	// Create use cases
	processTradeUC := usecases.NewProcessTradeEventUseCase(tradeBatchProcessor, logger)
	processBookTickerUC := usecases.NewProcessBookTickerEventUseCase(bookTickerBatchProcessor, logger)

	// Create handler
	handler := NewEventHandler(processTradeUC, processBookTickerUC, logger)

	// Close processors after a delay to ensure cleanup
	go func() {
		time.Sleep(5 * time.Second)
		_ = tradeBatchProcessor.Close()
		_ = bookTickerBatchProcessor.Close()
	}()

	return handler
}
