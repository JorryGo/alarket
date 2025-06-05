package connector

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Connector struct {
	url               string
	connectionPool    []*connection
	messageHandler    func([]byte)
	maxStreamsPerConn int
	maxSubsPerRequest int
	mux               sync.Mutex
	ctx               context.Context
	cancel            context.CancelFunc
}

func New(uri string, handler func([]byte)) *Connector {
	ctx, cancel := context.WithCancel(context.Background())
	return &Connector{
		url:               uri,
		maxStreamsPerConn: 1022,
		maxSubsPerRequest: 100,
		messageHandler:    handler,
		ctx:               ctx,
		cancel:            cancel,
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
	c.cancel() // Signal graceful shutdown
	for key := range c.connectionPool {
		c.connectionPool[key].closeChan <- struct{}{}
	}
}

func (c *Connector) makeNewConnection() {
	// Check if shutdown was requested
	select {
	case <-c.ctx.Done():
		slog.Info("Shutdown requested, not creating new connection")
		return
	default:
	}

	slog.Info("Connecting to WebSocket", "url", c.url)

	conn, httpInfo, err := websocket.DefaultDialer.Dial(c.url, nil)
	if err != nil {
		slog.Warn("Connection error", "error", err)
		// Retry immediately in goroutine to avoid infinite recursion
		go c.makeNewConnection()
		return
	}

	slog.Info("Successfully connected to WebSocket", "remote_addr", conn.RemoteAddr().String(), "status", httpInfo.Status)

	newConn := &connection{
		conn:      conn,
		closeChan: make(chan struct{}),
		id:        time.Now().UnixNano(),
		ctx:       c.ctx,
	}

	go newConn.runHandler(c.messageHandler)
	go c.handleConnection(newConn)

	c.connectionPool = append(c.connectionPool, newConn)
}

func (c *Connector) handleConnection(conn *connection) {

	conn.conn.SetPongHandler(func(pongMessage string) error {
		err := conn.conn.SetReadDeadline(time.Now().Add(time.Second * 60))
		if err != nil {
			slog.Warn("Pong error", "error", err)
		}

		return nil
	})

	go func(conn *connection) {
		ticker := time.NewTicker(time.Second * 10)
		defer ticker.Stop()
		for {
			select {
			case <-c.ctx.Done():
				slog.Info("Connection graceful shutdown", "connection_id", conn.id)
				c.removeConnection(conn)
				conn.close()
				return
			case <-conn.closeChan:
				// Check if this is graceful shutdown
				select {
				case <-c.ctx.Done():
					slog.Info("Connection graceful shutdown", "connection_id", conn.id)
					c.removeConnection(conn)
					conn.close()
					return
				default:
					slog.Warn("Connection unexpected disconnect, reconnecting", "connection_id", conn.id)
					c.removeConnection(conn)
					err := c.SubscribeStreams(conn.getSubs())
					if err != nil {
						slog.Error("Failed to resubscribe streams", "error", err)
					}
					conn.close()
					return
				}
			case <-ticker.C:
				err := conn.conn.WriteMessage(websocket.PingMessage, []byte(``))
				if err != nil {
					slog.Warn("Error sending ping message", "error", err)
				}
				//log.Info().Msg(`Ping sent`)
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
