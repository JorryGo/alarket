package connector

import (
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
	"time"
)

type Connector struct {
	url               string
	connectionPool    []*connection
	messageHandler    func([]byte)
	maxStreamsPerConn int
	maxSubsPerRequest int
}

func New(uri string, handler func([]byte)) *Connector {
	return &Connector{
		url:               uri,
		maxStreamsPerConn: 1022,
		maxSubsPerRequest: 100,
		messageHandler:    handler,
	}
}

func (c *Connector) Run() {
	c.makeNewConnection()
}

func (c *Connector) SubscribeStreams(tickers []string) error {
	for i := 0; i < len(c.connectionPool); i++ {
		if len(tickers) == 0 {
			return nil
		}

		conn := c.connectionPool[i]
		if len(conn.getSubs()) >= c.maxStreamsPerConn {
			continue
		}

		avaiableNewSubsAmount := c.maxStreamsPerConn - len(conn.getSubs())
		if avaiableNewSubsAmount > c.maxSubsPerRequest {
			avaiableNewSubsAmount = c.maxSubsPerRequest
		}

		if avaiableNewSubsAmount > len(tickers) {
			avaiableNewSubsAmount = len(tickers)
		}

		if err := conn.addSubs(tickers[:avaiableNewSubsAmount]); err != nil {
			return err
		}

		time.Sleep(time.Second / 4)

		tickers = tickers[avaiableNewSubsAmount:]

		if len(conn.getSubs()) < c.maxStreamsPerConn {
			i--
		}
	}

	c.makeNewConnection()
	return c.SubscribeStreams(tickers)
}

func (c *Connector) ClosePool() {
	for key := range c.connectionPool {
		c.connectionPool[key].close()
	}
}

func (c *Connector) makeNewConnection() {
	log.Info().Msgf("connecting to %s", c.url)

	conn, httpInfo, err := websocket.DefaultDialer.Dial(c.url, nil)
	if err != nil {
		log.Fatal().Msgf("Connection error: %s", err)
	}

	log.Info().Msgf(`Succesfully conected to %s (%s)`, conn.RemoteAddr().String(), httpInfo.Status)

	newConn := &connection{
		conn:      conn,
		closeChan: make(chan struct{}),
	}

	go newConn.runHandler(c.messageHandler)

	c.connectionPool = append(c.connectionPool, newConn)
}
