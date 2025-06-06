package connector

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type connection struct {
	id             int64
	conn           *websocket.Conn
	messageHandler func()
	closeChan      chan struct{}
	subs           []string
	mux            sync.Mutex
	ctx            context.Context
	logger         *slog.Logger
}

func (c *connection) close() {
	err := c.conn.Close()
	if err != nil {
		c.logger.Error("Failed to close connection", "error", err)
		return
	}

	c.logger.Info("Connection was successfully closed")
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
	c.logger.Info("Successfully subscribed to tickers", "tickers", tickers)

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
			// Check if this is due to graceful shutdown
			select {
			case <-c.ctx.Done():
				// Graceful shutdown, don't log as warning
				c.logger.Debug("Connection closed during shutdown")
			default:
				// Unexpected error
				c.logger.Warn("Connection read error", "error", err)
			}

			err = c.conn.Close()
			if err != nil {
				c.logger.Warn("Failed to close connection after read error", "error", err)
			}

			c.closeChan <- struct{}{}
			break
		}

		go handler(message)
	}
}
