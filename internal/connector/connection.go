package connector

import (
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
	"strings"
	"sync"
	"time"
)

type connection struct {
	conn           *websocket.Conn
	messageHandler func()
	closeChan      chan struct{}
	subs           []string
	mux            sync.Mutex
}

func (c *connection) close() {
	err := c.conn.Close()
	if err != nil {
		log.Err(err)
		return
	}

	log.Info().Msgf(`Connection was successfully closed`)
}

func (c *connection) addSubs(tickers []string) error {
	subRequest := Request{
		Method: `SUBSCRIBE`,
		Id:     time.Now().UnixMicro(),
		Params: make([]string, 0, len(tickers)),
	}

	for _, ticker := range tickers {
		subRequest.Params = append(subRequest.Params, strings.ToLower(ticker)+`@trade`)
	}

	err := c.conn.WriteJSON(subRequest)
	if err != nil {
		return err
	}

	c.mux.Lock()
	c.subs = append(c.subs, tickers...)
	c.mux.Unlock()
	log.Info().Msgf(`Succesfully subscribed %v`, tickers)

	return nil
}

func (c *connection) getSubs() []string {
	c.mux.Lock()
	defer c.mux.Unlock()

	return c.subs
}

func (c *connection) runHandler(handler func([]byte)) {
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			log.Warn().Msgf("conn read error: %s", err)
			continue
		}

		go handler(message)
	}
}
