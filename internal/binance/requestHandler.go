package binance

import (
	"alarket/internal/binance/events"
	"encoding/json"
	"log/slog"

	"github.com/bitly/go-simplejson"
)

// Handle processes WebSocket messages and routes them to appropriate event handlers
func Handle(message []byte) {
	j, err := simplejson.NewJson(message)
	if err != nil {
		slog.Warn("Failed to parse JSON", "error", err)
		return
	}

	if _, ok := j.CheckGet(`e`); !ok {
		handleBookTicker(message)
		return
	}

	if j.Get(`e`).MustString() == `error` {
		slog.Warn("Error message received", "message", string(message))
	}

	if j.Get(`e`).MustString() == `trade` {
		event := new(events.TradeEvent)
		err := json.Unmarshal(message, event)
		if err != nil {
			slog.Warn("Failed to unmarshal trade event", "error", err)
			return
		}
		event.Handle()
		return
	}

	slog.Warn("Unhandled message", "message", string(message))
}

func handleBookTicker(message []byte) {
	event := new(events.BookTickerEvent)
	err := json.Unmarshal(message, event)
	if err != nil {
		slog.Warn("Failed to unmarshal book ticker event", "error", err)
		return
	}

	event.Handle()
}
