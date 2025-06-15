package events

import (
	"testing"
	"time"

	"alarket/internal/domain/entities"
	"github.com/stretchr/testify/assert"
)

func TestTradeEvent_Type(t *testing.T) {
	trade := &entities.Trade{
		ID:           "12345",
		Symbol:       "BTCUSDT",
		Price:        50000.0,
		Quantity:     0.01,
		Time:         time.Now(),
		IsBuyerMaker: true,
		EventTime:    time.Now(),
	}

	event := TradeEvent{Trade: trade}
	assert.Equal(t, TradeEventType, event.Type())
}

func TestBookTickerEvent_Type(t *testing.T) {
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

	event := BookTickerEvent{BookTicker: bookTicker}
	assert.Equal(t, BookTickerEventType, event.Type())
}

func TestDomainEvent_Interface(t *testing.T) {
	t.Run("TradeEvent implements DomainEvent", func(t *testing.T) {
		var event DomainEvent = TradeEvent{
			Trade: &entities.Trade{Symbol: "BTCUSDT"},
		}
		assert.Equal(t, TradeEventType, event.Type())
	})

	t.Run("BookTickerEvent implements DomainEvent", func(t *testing.T) {
		var event DomainEvent = BookTickerEvent{
			BookTicker: &entities.BookTicker{Symbol: "BTCUSDT"},
		}
		assert.Equal(t, BookTickerEventType, event.Type())
	})
}

func TestEventType_Constants(t *testing.T) {
	// Ensure constants remain consistent
	assert.Equal(t, EventType("trade"), TradeEventType)
	assert.Equal(t, EventType("bookTicker"), BookTickerEventType)
}

func TestEvents_NilData(t *testing.T) {
	t.Run("TradeEvent with nil trade", func(t *testing.T) {
		event := TradeEvent{Trade: nil}
		// Should not panic
		assert.Equal(t, TradeEventType, event.Type())
	})

	t.Run("BookTickerEvent with nil book ticker", func(t *testing.T) {
		event := BookTickerEvent{BookTicker: nil}
		// Should not panic
		assert.Equal(t, BookTickerEventType, event.Type())
	})
}