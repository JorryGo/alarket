package events

import (
	"fmt"
	"log/slog"
	"strconv"

	"github.com/adshao/go-binance/v2"
)

type TradeEvent struct {
	binance.WsTradeEvent
}

type BookTickerEvent struct {
	binance.WsBookTickerEvent
}

type EventService struct {
	logger *slog.Logger
}

func NewEventService(logger *slog.Logger) *EventService {
	return &EventService{
		logger: logger,
	}
}

// HandleTradeEvent processes trade events
func (s *EventService) HandleTradeEvent(e *TradeEvent) {
	//s.logger.Info("Received trade event", "symbol", e.Symbol)

	_, err := strconv.ParseFloat(e.Price, 64)
	if err != nil {
		s.logger.Error("Failed to parse price", "symbol", e.Symbol, "price_string", e.Price, "error", err)
		return
	}

	fmt.Println(e)
	return
}

func (s *EventService) HandleBookTickerEvent(e *BookTickerEvent) {
	bidPrice, err := strconv.ParseFloat(e.BestBidPrice, 64)
	if err != nil {
		s.logger.Warn("Failed to parse best bid price", "symbol", e.Symbol, "price", e.BestBidPrice, "error", err)
		return
	}

	askPrice, err := strconv.ParseFloat(e.BestAskPrice, 64)
	if err != nil {
		s.logger.Warn("Failed to parse best ask price", "symbol", e.Symbol, "price", e.BestAskPrice, "error", err)
		return
	}

	// TODO: Process bid and ask prices
	_ = bidPrice
	_ = askPrice
}
