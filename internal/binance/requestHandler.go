package binance

import (
	"alarket/internal/binance/events"
	"encoding/json"

	"github.com/bitly/go-simplejson"

	"github.com/rs/zerolog/log"
)

// Handle processes WebSocket messages and routes them to appropriate event handlers
func Handle(message []byte) {
	j, err := simplejson.NewJson(message)
	if err != nil {
		log.Warn().Err(err)
		return
	}

	if _, ok := j.CheckGet(`e`); !ok {
		handleBookTicker(message)
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
		event.Handle()
		return
	}

	log.Warn().Msgf(`Unhandled message %s`, message)
}

func handleBookTicker(message []byte) {
	event := new(events.BookTickerEvent)
	err := json.Unmarshal(message, event)
	if err != nil {
		log.Warn().Err(err)
		return
	}

	event.Handle()
}
