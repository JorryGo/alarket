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

// Handle processes trade events and uses the trader to update pricing data
func (e *TradeEvent) Handle(trader *trader.Trader) {
	log.Info().Msgf(`Received trade event for %s`, e.Symbol)

	price, err := strconv.ParseFloat(e.Price, 64)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to parse price for %s", e.Symbol)
		return
	}

	trader.SetPrice(e.Symbol, price)
	trader.CheckLoopDiffs(e.Symbol)
	return
}
