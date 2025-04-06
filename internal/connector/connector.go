package connector

import (
	"alarket/internal/trader"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

type Connector struct {
	url               string
	connectionPool    []*connection
	messageHandler    func([]byte, *trader.Trader)
	trader            *trader.Trader
	maxStreamsPerConn int
	maxSubsPerRequest int
	mux               sync.Mutex
}

func New(uri string, handler func([]byte, *trader.Trader), trader *trader.Trader) *Connector {
	return &Connector{
		url:               uri,
		maxStreamsPerConn: 1022,
		maxSubsPerRequest: 100,
		messageHandler:    handler,
		trader:            trader,
	}
}

func (c *Connector) Run() {
	c.mux.Lock()
	defer c.mux.Unlock()
	c.makeNewConnection()
}

func (c *Connector) SubscribeStreams(tickers []string) error {
	c.mux.Lock()
	defer c.mux.Unlock()

	return c.doSubscription(tickers)
}

func (c *Connector) doSubscription(tickers []string) error {
	for i := 0; i < len(c.connectionPool); i++ {
		if len(tickers) == 0 {
			return nil
		}

		conn := c.connectionPool[i]
		if len(conn.getSubs()) >= c.maxStreamsPerConn {
			continue
		}

		availableNewSubsAmount := c.maxStreamsPerConn - len(conn.getSubs())
		if availableNewSubsAmount > c.maxSubsPerRequest {
			availableNewSubsAmount = c.maxSubsPerRequest
		}

		if availableNewSubsAmount > len(tickers) {
			availableNewSubsAmount = len(tickers)
		}

		if err := conn.addSubs(tickers[:availableNewSubsAmount]); err != nil {
			return err
		}

		time.Sleep(time.Second / 4)

		tickers = tickers[availableNewSubsAmount:]

		if len(conn.getSubs()) < c.maxStreamsPerConn {
			i--
		}
	}

	c.makeNewConnection()
	return c.doSubscription(tickers)
}

func (c *Connector) ClosePool() {
	for key := range c.connectionPool {
		c.connectionPool[key].closeChan <- struct{}{}
	}
}

func (c *Connector) makeNewConnection() {
	log.Info().Msgf("connecting to %s", c.url)

	conn, httpInfo, err := websocket.DefaultDialer.Dial(c.url, nil)
	if err != nil {
		log.Warn().Msgf("Connection error: %s", err)
		c.makeNewConnection()
		return
	}

	log.Info().Msgf(`Succesfully conected to %s (%s)`, conn.RemoteAddr().String(), httpInfo.Status)

	newConn := &connection{
		conn:      conn,
		closeChan: make(chan struct{}),
		id:        time.Now().UnixNano(),
	}

	go newConn.runHandler(c.messageHandler, c.trader)
	go c.handleConnection(newConn)

	c.connectionPool = append(c.connectionPool, newConn)
}

func (c *Connector) handleConnection(conn *connection) {
	conn.conn.SetPongHandler(func(pongMessage string) error {
		err := conn.conn.SetReadDeadline(time.Now().Add(time.Second * 15))
		if err != nil {
			log.Warn().Msgf(`Pong err: %s`, err)
		}

		return nil
	})

	go func(conn *connection) {
		ticker := time.NewTicker(time.Second * 10)
		defer ticker.Stop()
		for {
			select {
			case <-conn.closeChan:
				log.Warn().Msgf(`Connection %d closed. Reconnect...`, conn.id)
				c.removeConnection(conn)
				err := c.SubscribeStreams(conn.getSubs())
				if err != nil {
					log.Err(err)
				}
				conn.close()
				return
			case <-ticker.C:
				err := conn.conn.WriteMessage(websocket.PingMessage, []byte(``))
				if err != nil {
					log.Warn().Msgf(`Error with sending a ping message: %s`, err)
				}
				log.Info().Msg(`Ping sent`)
			}
		}
	}(conn)

	return
}

func (c *Connector) removeConnection(conn *connection) {
	c.mux.Lock()
	defer c.mux.Unlock()

	for i, poolConn := range c.connectionPool {
		if poolConn == conn {
			c.connectionPool = append(c.connectionPool[:i], c.connectionPool[i+1:]...)
			break
		}
	}
}
