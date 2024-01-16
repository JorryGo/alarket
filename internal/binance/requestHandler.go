package binance

import (
	"alarket/internal/binance/events"
	"encoding/json"
	"github.com/bitly/go-simplejson"

	"github.com/rs/zerolog/log"
)

type RequestHandler struct {
}

func NewRequestHandler() *RequestHandler {
	return &RequestHandler{}
}

func (r *RequestHandler) Handle(message []byte) {
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
		event.Handle()
		return
	}

	log.Warn().Msgf(`Unhandled message %s`, message)
}
