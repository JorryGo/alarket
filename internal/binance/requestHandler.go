package binance

import (
	"alarket/internal/binance/events"
	"encoding/json"
	"log/slog"

	"github.com/bitly/go-simplejson"
)

type Handler struct {
	logger       *slog.Logger
	eventService *events.EventService
}

func NewHandler(logger *slog.Logger) *Handler {
	return &Handler{
		logger:       logger,
		eventService: events.NewEventService(logger),
	}
}

// Handle processes WebSocket messages and routes them to appropriate event handlers
func (h *Handler) Handle(message []byte) {
	j, err := simplejson.NewJson(message)
	if err != nil {
		h.logger.Warn("Failed to parse JSON", "error", err)
		return
	}

	if _, ok := j.CheckGet(`e`); !ok {
		h.handleBookTicker(message)
		return
	}

	if j.Get(`e`).MustString() == `error` {
		h.logger.Warn("Error message received", "message", string(message))
	}

	if j.Get(`e`).MustString() == `trade` {
		event := new(events.TradeEvent)
		err := json.Unmarshal(message, event)
		if err != nil {
			h.logger.Warn("Failed to unmarshal trade event", "error", err)
			return
		}
		h.eventService.HandleTradeEvent(event)
		return
	}

	h.logger.Warn("Unhandled message", "message", string(message))
}

func (h *Handler) handleBookTicker(message []byte) {
	event := new(events.BookTickerEvent)
	err := json.Unmarshal(message, event)
	if err != nil {
		h.logger.Warn("Failed to unmarshal book ticker event", "error", err)
		return
	}

	h.eventService.HandleBookTickerEvent(event)
}
