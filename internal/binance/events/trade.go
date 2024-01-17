package events

import (
	"github.com/adshao/go-binance/v2"
	"github.com/rs/zerolog/log"
)

type TradeEvent struct {
	binance.WsTradeEvent
}

// Handle is a method that handles TradeEvent and writes it in a clickhouse database
func (e *TradeEvent) Handle() {
	log.Info().Msgf(`%s`, e)

	return
}
