package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"

	"alarket/internal/application/dto"
	"alarket/internal/infrastructure/websocket"
)

const (
	maxStreamsPerConnection = 1022
	maxSubscriptionsPerRequest = 100
	baseWSURL = "wss://stream.binance.com:443/ws"
	testnetWSURL = "wss://testnet.binance.vision/ws"
)

type Client struct {
	wsManager      *websocket.Manager
	logger         *slog.Logger
	useTestnet     bool
	streamCount    atomic.Int32
	connectionID   atomic.Int32
	subscriptions  map[string]string // stream -> connectionID
	mu             sync.RWMutex
	messageHandler websocket.MessageHandler
}

func NewClient(logger *slog.Logger, useTestnet bool, messageHandler websocket.MessageHandler) *Client {
	wsManager := websocket.NewManager(logger, messageHandler)
	
	return &Client{
		wsManager:      wsManager,
		logger:         logger,
		useTestnet:     useTestnet,
		subscriptions:  make(map[string]string),
		messageHandler: messageHandler,
	}
}

func (c *Client) SubscribeToTrades(ctx context.Context, symbols []string) error {
	streams := make([]string, len(symbols))
	for i, symbol := range symbols {
		streams[i] = fmt.Sprintf("%s@trade", strings.ToLower(symbol))
	}
	return c.subscribe(ctx, streams)
}

func (c *Client) SubscribeToBookTickers(ctx context.Context, symbols []string) error {
	streams := make([]string, len(symbols))
	for i, symbol := range symbols {
		streams[i] = fmt.Sprintf("%s@bookTicker", strings.ToLower(symbol))
	}
	return c.subscribe(ctx, streams)
}

func (c *Client) UnsubscribeFromTrades(ctx context.Context, symbols []string) error {
	streams := make([]string, len(symbols))
	for i, symbol := range symbols {
		streams[i] = fmt.Sprintf("%s@trade", strings.ToLower(symbol))
	}
	return c.unsubscribe(ctx, streams)
}

func (c *Client) UnsubscribeFromBookTickers(ctx context.Context, symbols []string) error {
	streams := make([]string, len(symbols))
	for i, symbol := range symbols {
		streams[i] = fmt.Sprintf("%s@bookTicker", strings.ToLower(symbol))
	}
	return c.unsubscribe(ctx, streams)
}

func (c *Client) Close() error {
	return c.wsManager.CloseAll()
}

func (c *Client) subscribe(ctx context.Context, streams []string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Group streams by connection
	connectionStreams := make(map[string][]string)
	
	for _, stream := range streams {
		if _, exists := c.subscriptions[stream]; exists {
			c.logger.Debug("Stream already subscribed", "stream", stream)
			continue
		}

		// Find or create connection for this stream
		connID := c.findOrCreateConnection(ctx)
		connectionStreams[connID] = append(connectionStreams[connID], stream)
		c.subscriptions[stream] = connID
	}

	// Subscribe on each connection
	for connID, connStreams := range connectionStreams {
		if err := c.subscribeOnConnection(connID, connStreams); err != nil {
			// Clean up subscriptions on error
			for _, stream := range connStreams {
				delete(c.subscriptions, stream)
			}
			return fmt.Errorf("failed to subscribe on connection %s: %w", connID, err)
		}
		c.streamCount.Add(int32(len(connStreams)))
	}

	return nil
}

func (c *Client) unsubscribe(ctx context.Context, streams []string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Group streams by connection
	connectionStreams := make(map[string][]string)
	
	for _, stream := range streams {
		if connID, exists := c.subscriptions[stream]; exists {
			connectionStreams[connID] = append(connectionStreams[connID], stream)
		}
	}

	// Unsubscribe on each connection
	for connID, connStreams := range connectionStreams {
		if err := c.unsubscribeOnConnection(connID, connStreams); err != nil {
			c.logger.Error("Failed to unsubscribe", "connection", connID, "error", err)
			continue
		}
		
		// Clean up subscriptions
		for _, stream := range connStreams {
			delete(c.subscriptions, stream)
		}
		c.streamCount.Add(-int32(len(connStreams)))
	}

	return nil
}

func (c *Client) findOrCreateConnection(ctx context.Context) string {
	// Count streams per connection
	streamCounts := make(map[string]int)
	for _, connID := range c.subscriptions {
		streamCounts[connID]++
	}

	// Find connection with available capacity
	for connID, count := range streamCounts {
		if count < maxStreamsPerConnection {
			return connID
		}
	}

	// Create new connection
	connID := fmt.Sprintf("conn-%d", c.connectionID.Add(1))
	wsURL := c.getWebSocketURL()
	
	if err := c.wsManager.Connect(ctx, wsURL, connID); err != nil {
		c.logger.Error("Failed to create new connection", "error", err)
		// Return first available connection as fallback
		for connID := range streamCounts {
			return connID
		}
	}

	return connID
}

func (c *Client) subscribeOnConnection(connID string, streams []string) error {
	// Split into batches if needed
	for i := 0; i < len(streams); i += maxSubscriptionsPerRequest {
		end := i + maxSubscriptionsPerRequest
		if end > len(streams) {
			end = len(streams)
		}

		batch := streams[i:end]
		req := dto.SubscriptionRequest{
			Method: "SUBSCRIBE",
			Params: batch,
			ID:     1,
		}

		data, err := json.Marshal(req)
		if err != nil {
			return fmt.Errorf("failed to marshal subscribe request: %w", err)
		}

		if err := c.wsManager.Send(connID, data); err != nil {
			return fmt.Errorf("failed to send subscribe request: %w", err)
		}

		c.logger.Info("Subscribed to streams", "connection", connID, "count", len(batch))
	}

	return nil
}

func (c *Client) unsubscribeOnConnection(connID string, streams []string) error {
	req := dto.SubscriptionRequest{
		Method: "UNSUBSCRIBE",
		Params: streams,
		ID:     1,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal unsubscribe request: %w", err)
	}

	if err := c.wsManager.Send(connID, data); err != nil {
		return fmt.Errorf("failed to send unsubscribe request: %w", err)
	}

	c.logger.Info("Unsubscribed from streams", "connection", connID, "count", len(streams))
	return nil
}

func (c *Client) getWebSocketURL() string {
	if c.useTestnet {
		return testnetWSURL
	}
	return baseWSURL
}