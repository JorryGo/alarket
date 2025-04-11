package events

import (
	"alarket/internal/trader"
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

// Handle processes trade events and uses the trader to update pricing data
func (e *TradeEvent) Handle(trader *trader.Trader) {
	//log.Info().Msgf(`Received trade event for %s`, e.Symbol)

	price, err := strconv.ParseFloat(e.Price, 64)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to parse price for %s", e.Symbol)
		return
	}

	trader.SetPrice(e.Symbol, price)
	trader.CheckLoopDiffs(e.Symbol)
	return
}

func (e *BookTickerEvent) Handle(trader *trader.Trader) {
	price, err := strconv.ParseFloat(e.BestBidPrice, 64)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to parse price for %s, %v", e.Symbol, e)
		return
	}

	oldPrice, ok := trader.Price(e.Symbol)

	if ok && oldPrice == price {
		return
	}

	trader.SetPrice(e.Symbol, price)

	if e.Symbol != "BTCUSDT" {
		return
	}

	//go trader.CheckLoopDiffs(e.Symbol)
}
