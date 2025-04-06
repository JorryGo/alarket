package binance

import (
	"alarket/internal/binance/events"
	"alarket/internal/trader"
	"encoding/json"

	"github.com/bitly/go-simplejson"

	"github.com/rs/zerolog/log"
)

// Handle processes WebSocket messages and routes them to appropriate event handlers
func Handle(message []byte, trader *trader.Trader) {
	j, err := simplejson.NewJson(message)
	if err != nil {
		log.Warn().Err(err)
		return
	}

	if _, ok := j.CheckGet(`e`); !ok {
		log.Info().Msgf(`Unhandled message %s`, message)
		return
	}

	if j.Get(`e`).MustString() == `error` {
		log.Warn().Msgf(`%s`, message)
	}

	if j.Get(`e`).MustString() == `trade` {
		event := new(events.TradeEvent)
		err := json.Unmarshal(message, event)
		if err != nil {
			log.Warn().Err(err)
			return
		}
		event.Handle(trader)
		return
	}

	log.Warn().Msgf(`Unhandled message %s`, message)
}
