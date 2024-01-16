package events

import (
	"github.com/adshao/go-binance/v2"
	"github.com/rs/zerolog/log"
)

type TradeEvent struct {
	binance.WsTradeEvent
}

func (e *TradeEvent) Handle() {
	log.Info().Msgf(`%s`, e)
}
