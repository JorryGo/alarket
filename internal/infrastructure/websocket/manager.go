package websocket

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type MessageHandler func(message []byte) error

type Connection struct {
	conn           *websocket.Conn
	url            string
	id             string
	messageHandler MessageHandler
	logger         *slog.Logger
	mu             sync.Mutex
	closed         bool
	pingTicker     *time.Ticker
}

type Manager struct {
	connections    map[string]*Connection
	mu             sync.RWMutex
	logger         *slog.Logger
	messageHandler MessageHandler
}

func NewManager(logger *slog.Logger, messageHandler MessageHandler) *Manager {
	return &Manager{
		connections:    make(map[string]*Connection),
		logger:         logger,
		messageHandler: messageHandler,
	}
}

func (m *Manager) Connect(ctx context.Context, url, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.connections[id]; exists {
		return fmt.Errorf("connection with id %s already exists", id)
	}

	dialer := websocket.Dialer{
		HandshakeTimeout: 45 * time.Second,
	}

	conn, _, err := dialer.DialContext(ctx, url, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", url, err)
	}

	connection := &Connection{
		conn:           conn,
		url:            url,
		id:             id,
		messageHandler: m.messageHandler,
		logger:         m.logger,
		pingTicker:     time.NewTicker(30 * time.Second),
	}

	m.connections[id] = connection

	go connection.readLoop(ctx)
	go connection.pingLoop(ctx)

	m.logger.Info("WebSocket connection established", "id", id, "url", url)
	return nil
}

func (m *Manager) Send(id string, message []byte) error {
	m.mu.RLock()
	conn, exists := m.connections[id]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("connection %s not found", id)
	}

	return conn.send(message)
}

func (m *Manager) Close(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	conn, exists := m.connections[id]
	if !exists {
		return fmt.Errorf("connection %s not found", id)
	}

	if err := conn.close(); err != nil {
		return err
	}

	delete(m.connections, id)
	return nil
}

func (m *Manager) CloseAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, conn := range m.connections {
		if err := conn.close(); err != nil {
			m.logger.Error("Failed to close connection", "id", id, "error", err)
		}
	}

	m.connections = make(map[string]*Connection)
	return nil
}

func (c *Connection) readLoop(ctx context.Context) {
	defer func() {
		_ = c.close()
		c.logger.Info("Read loop terminated", "id", c.id)
	}()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			_, message, err := c.conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					c.logger.Error("WebSocket read error", "id", c.id, "error", err)
				}
				return
			}

			if err := c.messageHandler(message); err != nil {
				c.logger.Error("Message handler error", "id", c.id, "error", err)
			}
		}
	}
}

func (c *Connection) pingLoop(ctx context.Context) {
	defer c.pingTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.pingTicker.C:
			c.mu.Lock()
			if c.closed {
				c.mu.Unlock()
				return
			}
			err := c.conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(10*time.Second))
			c.mu.Unlock()

			if err != nil {
				c.logger.Error("Ping failed", "id", c.id, "error", err)
				_ = c.close()
				return
			}
		}
	}
}

func (c *Connection) send(message []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return fmt.Errorf("connection %s is closed", c.id)
	}

	if err := c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return fmt.Errorf("failed to set write deadline: %w", err)
	}
	return c.conn.WriteMessage(websocket.TextMessage, message)
}

func (c *Connection) close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true
	c.pingTicker.Stop()

	if err := c.conn.WriteControl(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		time.Now().Add(5*time.Second),
	); err != nil {
		c.logger.Debug("Failed to write close message", "error", err)
	}

	return c.conn.Close()
}
