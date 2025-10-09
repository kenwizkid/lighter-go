package client

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WsOrderBookClient is a WebSocket client dedicated to order book subscriptions
type WsOrderBookClient struct {
	baseURL               string
	conn                  *websocket.Conn
	marketIDs             []uint8
	onOrderBookUpdate     OrderBookUpdateHandler
	onOrderBookSnapshot   OrderBookSnapshotHandler
	onMessage             MessageHandler
	onError               ErrorHandler
	mu                    sync.RWMutex
	ctx                   context.Context
	cancel                context.CancelFunc
	reconnectInterval     time.Duration
	maxReconnectAttempts  int
	isConnected           bool
	done                  chan struct{}
}

// NewWsOrderBookClient creates a new dedicated order book WebSocket client
func NewWsOrderBookClient(marketIDs []uint8, options ...WsOrderBookClientOption) (*WsOrderBookClient, error) {
	if len(marketIDs) == 0 {
		return nil, fmt.Errorf("no market IDs provided for order book subscription")
	}

	client := &WsOrderBookClient{
		baseURL:              "wss://mainnet.zklighter.elliot.ai/stream",
		marketIDs:            marketIDs,
		reconnectInterval:    5 * time.Second,
		maxReconnectAttempts: 10,
		done:                 make(chan struct{}),
	}

	// Apply options
	for _, opt := range options {
		opt(client)
	}

	// Create context
	client.ctx, client.cancel = context.WithCancel(context.Background())

	return client, nil
}

// WsOrderBookClientOption configures the WsOrderBookClient
type WsOrderBookClientOption func(*WsOrderBookClient)

// WithOrderBookHost sets the WebSocket host
func WithOrderBookHost(host string) WsOrderBookClientOption {
	return func(c *WsOrderBookClient) {
		host = strings.TrimPrefix(host, "https://")
		host = strings.TrimPrefix(host, "http://")
		c.baseURL = fmt.Sprintf("wss://%s/stream", host)
	}
}

// WithOrderBookUpdateHandler sets the order book update handler
func WithOrderBookUpdateHandler(handler OrderBookUpdateHandler) WsOrderBookClientOption {
	return func(c *WsOrderBookClient) {
		c.onOrderBookUpdate = handler
	}
}

// WithOrderBookSnapshotHandler sets the order book snapshot handler
func WithOrderBookSnapshotHandler(handler OrderBookSnapshotHandler) WsOrderBookClientOption {
	return func(c *WsOrderBookClient) {
		c.onOrderBookSnapshot = handler
	}
}

// WithOrderBookMessageHandler sets the raw message handler
func WithOrderBookMessageHandler(handler MessageHandler) WsOrderBookClientOption {
	return func(c *WsOrderBookClient) {
		c.onMessage = handler
	}
}

// WithOrderBookErrorHandler sets the error handler
func WithOrderBookErrorHandler(handler ErrorHandler) WsOrderBookClientOption {
	return func(c *WsOrderBookClient) {
		c.onError = handler
	}
}

// WithOrderBookReconnectInterval sets the reconnect interval
func WithOrderBookReconnectInterval(interval time.Duration) WsOrderBookClientOption {
	return func(c *WsOrderBookClient) {
		c.reconnectInterval = interval
	}
}

// Connect establishes a WebSocket connection
func (c *WsOrderBookClient) connect() error {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return fmt.Errorf("invalid WebSocket URL: %v", err)
	}

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to connect to WebSocket: %v", err)
	}

	c.conn = conn
	c.isConnected = true
	log.Printf("OrderBook client connected to: %s", c.baseURL)
	return nil
}

// Run starts the order book WebSocket client
func (c *WsOrderBookClient) Run() error {
	defer close(c.done)

	for attempt := 0; attempt < c.maxReconnectAttempts; attempt++ {
		if err := c.connect(); err != nil {
			if attempt == c.maxReconnectAttempts-1 {
				return fmt.Errorf("failed to connect after %d attempts: %v", c.maxReconnectAttempts, err)
			}
			log.Printf("OrderBook connection attempt %d failed: %v. Retrying in %v", attempt+1, err, c.reconnectInterval)
			time.Sleep(c.reconnectInterval)
			continue
		}

		// Reset attempt counter on successful connection
		attempt = 0

		// Handle messages
		if err := c.handleMessages(); err != nil {
			log.Printf("OrderBook message handling error: %v", err)
			c.isConnected = false
			if c.conn != nil {
				c.conn.Close()
			}

			select {
			case <-c.ctx.Done():
				return nil
			default:
				time.Sleep(c.reconnectInterval)
			}
		}
	}

	return fmt.Errorf("exceeded maximum reconnection attempts")
}

// Stop stops the order book client
func (c *WsOrderBookClient) Stop() {
	c.cancel()
	if c.conn != nil {
		c.conn.Close()
	}
	<-c.done
}

// IsConnected returns whether the client is connected
func (c *WsOrderBookClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isConnected
}

// handleMessages processes incoming WebSocket messages for order book
func (c *WsOrderBookClient) handleMessages() error {
	// Set up ping/pong handlers
	c.conn.SetPingHandler(func(message string) error {
		return c.conn.WriteControl(websocket.PongMessage, []byte(message), time.Now().Add(10*time.Second))
	})

	// Send periodic pings
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	go func() {
		for {
			select {
			case <-ticker.C:
				if err := c.conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(10*time.Second)); err != nil {
					log.Printf("OrderBook failed to send ping: %v", err)
					return
				}
			case <-c.ctx.Done():
				return
			}
		}
	}()

	// Read messages
	for {
		select {
		case <-c.ctx.Done():
			return nil
		default:
			_, message, err := c.conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					return fmt.Errorf("OrderBook WebSocket read error: %v", err)
				}
				return err
			}

			go c.processMessage(message)
		}
	}
}

// processMessage handles incoming messages specifically for order book
func (c *WsOrderBookClient) processMessage(message []byte) {
	var baseMsg struct {
		Type    string `json:"type"`
		Channel string `json:"channel,omitempty"`
		Error   *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}

	if err := json.Unmarshal(message, &baseMsg); err != nil {
		log.Printf("OrderBook failed to unmarshal base message: %v", err)
		return
	}

	// Handle errors
	if baseMsg.Error != nil {
		c.handleError(message)
		return
	}

	// Call raw message handler if set
	if c.onMessage != nil {
		c.onMessage(baseMsg.Type, message)
	}

	// Handle specific message types - ONLY ORDER BOOK RELATED
	switch baseMsg.Type {
	case "connected":
		c.handleConnected()
	case "subscribed":
		log.Printf("OrderBook subscription confirmed for channel: %s", baseMsg.Channel)
	case "subscribed/order_book":
		c.handleSubscribedOrderBook(message)
	case "update/order_book":
		c.handleUpdateOrderBook(message)
	case "error":
		c.handleError(message)
	case "ping":
		c.handlePing(message)
	default:
		if baseMsg.Type != "" {
			log.Printf("OrderBook unhandled message type: %s", baseMsg.Type)
		}
	}
}

// handleConnected subscribes ONLY to order books
func (c *WsOrderBookClient) handleConnected() {
	log.Println("OrderBook client connected, subscribing to order books only")
	
	time.Sleep(100 * time.Millisecond)

	// Subscribe ONLY to order books
	for _, marketID := range c.marketIDs {
		msg := NewOrderBookSubscription(marketID)
		log.Printf("OrderBook subscribing to market %d", marketID)
		if err := c.sendMessage(msg); err != nil {
			log.Printf("OrderBook failed to subscribe to market %d: %v", marketID, err)
		}
	}
}

// handleSubscribedOrderBook handles the initial order book subscription response
func (c *WsOrderBookClient) handleSubscribedOrderBook(message []byte) {
	var snapshot OrderBookSnapshot
	if err := json.Unmarshal(message, &snapshot); err != nil {
		log.Printf("OrderBook failed to unmarshal snapshot: %v", err)
		return
	}

	marketID := extractMarketID(snapshot.Channel)
	if marketID == 255 {
		log.Printf("OrderBook failed to extract market ID from channel: %s", snapshot.Channel)
		return
	}

	if c.onOrderBookSnapshot != nil {
		c.onOrderBookSnapshot(marketID, &snapshot)
	}
	log.Printf("OrderBook received snapshot for market %d", marketID)
}

// handleUpdateOrderBook handles order book update messages
func (c *WsOrderBookClient) handleUpdateOrderBook(message []byte) {
	var update OrderBookUpdate
	if err := json.Unmarshal(message, &update); err != nil {
		log.Printf("OrderBook failed to unmarshal update: %v", err)
		return
	}

	marketID := extractMarketID(update.Channel)
	if marketID == 255 {
		log.Printf("OrderBook failed to extract market ID from channel: %s", update.Channel)
		return
	}

	if c.onOrderBookUpdate != nil {
		c.onOrderBookUpdate(marketID, &update)
	}
}

// handleError handles error messages
func (c *WsOrderBookClient) handleError(message []byte) {
	var errorMsg ErrorMessage
	if err := json.Unmarshal(message, &errorMsg); err != nil {
		log.Printf("OrderBook failed to unmarshal error: %v", err)
		return
	}

	log.Printf("OrderBook error: Code=%d, Message=%s", errorMsg.Code, errorMsg.Message)
	
	if c.onError != nil {
		c.onError(&errorMsg)
	}
}

// handlePing responds to ping messages
func (c *WsOrderBookClient) handlePing(message []byte) {
	var ping PingMessage
	if err := json.Unmarshal(message, &ping); err != nil {
		log.Printf("OrderBook failed to unmarshal ping: %v", err)
		return
	}

	pong := PongMessage{
		Type:      "pong",
		Timestamp: ping.Timestamp,
	}

	if err := c.sendMessage(pong); err != nil {
		log.Printf("OrderBook failed to send pong: %v", err)
	}
}

// sendMessage sends a message through the WebSocket connection
func (c *WsOrderBookClient) sendMessage(message interface{}) error {
	if !c.isConnected || c.conn == nil {
		return fmt.Errorf("OrderBook not connected to WebSocket")
	}

	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("OrderBook failed to marshal message: %v", err)
	}
	
	log.Printf("OrderBook sending message: %s", string(data))
	return c.conn.WriteMessage(websocket.TextMessage, data)
}