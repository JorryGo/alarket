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

// Handle processes trade events
func (e *TradeEvent) Handle() {
	//log.Info().Msgf(`Received trade event for %s`, e.Symbol)

	_, err := strconv.ParseFloat(e.Price, 64)
	if err != nil {
		slog.Error("Failed to parse price", "symbol", e.Symbol, "price_string", e.Price, "error", err)
		return
	}

	fmt.Println(e)
	return
}

func (e *BookTickerEvent) Handle() {
	bidPrice, err := strconv.ParseFloat(e.BestBidPrice, 64)
	if err != nil {
		return
	}

	askPrice, err := strconv.ParseFloat(e.BestAskPrice, 64)
	if err != nil {
		return
	}

	// TODO: Process bid and ask prices
	_ = bidPrice
	_ = askPrice
}
