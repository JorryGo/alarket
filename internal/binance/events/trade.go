package events

import (
	"fmt"
	"strconv"

	"github.com/adshao/go-binance/v2"
	"github.com/rs/zerolog/log"
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

	price, err := strconv.ParseFloat(e.Price, 64)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to parse price for %s, %f", e.Symbol, price)
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
