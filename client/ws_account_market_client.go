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

// WsAccountMarketClient is a WebSocket client dedicated to account market subscriptions
type WsAccountMarketClient struct {
	baseURL                 string
	conn                    *websocket.Conn
	accountMarketSubs       []AccountMarketSub
	onAccountMarketUpdate   AccountMarketUpdateHandler
	onAccountMarketSnapshot AccountMarketSnapshotHandler
	onOrderUpdate           AccountMarketOrderUpdateHandler
	onPositionUpdate        AccountMarketPositionUpdateHandler
	onTradeUpdate           AccountMarketTradeUpdateHandler
	onFundingUpdate         AccountMarketFundingUpdateHandler
	onMessage               MessageHandler
	onError                 ErrorHandler
	txClient                *TxClient // For generating auth tokens
	mu                      sync.RWMutex
	ctx                     context.Context
	cancel                  context.CancelFunc
	reconnectInterval       time.Duration
	maxReconnectAttempts    int
	isConnected             bool
	done                    chan struct{}
}

// NewWsAccountMarketClient creates a new dedicated account market WebSocket client
func NewWsAccountMarketClient(accountMarketSubs []AccountMarketSub, options ...WsAccountMarketClientOption) (*WsAccountMarketClient, error) {
	if len(accountMarketSubs) == 0 {
		return nil, fmt.Errorf("no account market subscriptions provided")
	}

	client := &WsAccountMarketClient{
		baseURL:              "wss://mainnet.zklighter.elliot.ai/stream",
		accountMarketSubs:    accountMarketSubs,
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

// WsAccountMarketClientOption configures the WsAccountMarketClient
type WsAccountMarketClientOption func(*WsAccountMarketClient)

// WithAccountMarketHost sets the WebSocket host
func WithAccountMarketHost(host string) WsAccountMarketClientOption {
	return func(c *WsAccountMarketClient) {
		host = strings.TrimPrefix(host, "https://")
		host = strings.TrimPrefix(host, "http://")
		c.baseURL = fmt.Sprintf("wss://%s/stream", host)
	}
}

// WithAccountMarketUpdateHandler sets the account market update handler
func WithAccountMarketUpdateHandler(handler AccountMarketUpdateHandler) WsAccountMarketClientOption {
	return func(c *WsAccountMarketClient) {
		c.onAccountMarketUpdate = handler
	}
}

// WithAccountMarketSnapshotHandler sets the account market snapshot handler
func WithAccountMarketSnapshotHandler(handler AccountMarketSnapshotHandler) WsAccountMarketClientOption {
	return func(c *WsAccountMarketClient) {
		c.onAccountMarketSnapshot = handler
	}
}

// WithAccountMarketMessageHandler sets the raw message handler
func WithAccountMarketMessageHandler(handler MessageHandler) WsAccountMarketClientOption {
	return func(c *WsAccountMarketClient) {
		c.onMessage = handler
	}
}

// WithAccountMarketErrorHandler sets the error handler
func WithAccountMarketErrorHandler(handler ErrorHandler) WsAccountMarketClientOption {
	return func(c *WsAccountMarketClient) {
		c.onError = handler
	}
}

// WithAccountMarketTxClient sets the TxClient for generating auth tokens
func WithAccountMarketTxClient(txClient *TxClient) WsAccountMarketClientOption {
	return func(c *WsAccountMarketClient) {
		c.txClient = txClient
	}
}

// WithAccountMarketReconnectInterval sets the reconnect interval
func WithAccountMarketReconnectInterval(interval time.Duration) WsAccountMarketClientOption {
	return func(c *WsAccountMarketClient) {
		c.reconnectInterval = interval
	}
}

// WithAccountMarketOrderUpdateHandler sets the order update handler
func WithAccountMarketOrderUpdateHandler(handler AccountMarketOrderUpdateHandler) WsAccountMarketClientOption {
	return func(c *WsAccountMarketClient) {
		c.onOrderUpdate = handler
	}
}

// WithAccountMarketPositionUpdateHandler sets the position update handler
func WithAccountMarketPositionUpdateHandler(handler AccountMarketPositionUpdateHandler) WsAccountMarketClientOption {
	return func(c *WsAccountMarketClient) {
		c.onPositionUpdate = handler
	}
}

// WithAccountMarketTradeUpdateHandler sets the trade update handler
func WithAccountMarketTradeUpdateHandler(handler AccountMarketTradeUpdateHandler) WsAccountMarketClientOption {
	return func(c *WsAccountMarketClient) {
		c.onTradeUpdate = handler
	}
}

// WithAccountMarketFundingUpdateHandler sets the funding update handler
func WithAccountMarketFundingUpdateHandler(handler AccountMarketFundingUpdateHandler) WsAccountMarketClientOption {
	return func(c *WsAccountMarketClient) {
		c.onFundingUpdate = handler
	}
}

// Connect establishes a WebSocket connection
func (c *WsAccountMarketClient) connect() error {
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
	// log.Printf("AccountMarket client connected to: %s", c.baseURL)
	return nil
}

// Run starts the account market WebSocket client
func (c *WsAccountMarketClient) Run() error {
	defer close(c.done)

	for attempt := 0; attempt < c.maxReconnectAttempts; attempt++ {
		if err := c.connect(); err != nil {
			if attempt == c.maxReconnectAttempts-1 {
				return fmt.Errorf("failed to connect after %d attempts: %v", c.maxReconnectAttempts, err)
			}
			log.Printf("AccountMarket connection attempt %d failed: %v. Retrying in %v", attempt+1, err, c.reconnectInterval)
			time.Sleep(c.reconnectInterval)
			continue
		}

		// Reset attempt counter on successful connection
		attempt = 0

		// Handle messages
		if err := c.handleMessages(); err != nil {
			log.Printf("AccountMarket message handling error: %v", err)
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

// Stop stops the account market client
func (c *WsAccountMarketClient) Stop() {
	c.cancel()
	if c.conn != nil {
		c.conn.Close()
	}
	<-c.done
}

// IsConnected returns whether the client is connected
func (c *WsAccountMarketClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isConnected
}

// handleMessages processes incoming WebSocket messages for account market
func (c *WsAccountMarketClient) handleMessages() error {
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
					log.Printf("AccountMarket failed to send ping: %v", err)
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
					return fmt.Errorf("AccountMarket WebSocket read error: %v", err)
				}
				return err
			}

			go c.processMessage(message)
		}
	}
}

// processMessage handles incoming messages specifically for account market
func (c *WsAccountMarketClient) processMessage(message []byte) {
	var baseMsg struct {
		Type    string `json:"type"`
		Channel string `json:"channel,omitempty"`
		Error   *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}

	if err := json.Unmarshal(message, &baseMsg); err != nil {
		log.Printf("AccountMarket failed to unmarshal base message: %v", err)
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

	// Handle specific message types - ONLY ACCOUNT MARKET RELATED
	switch baseMsg.Type {
	case "connected":
		c.handleConnected()
	case "subscribed":
		log.Printf("AccountMarket subscription confirmed for channel: %s", baseMsg.Channel)
	case "subscribed/account_market":
		c.handleSubscribedAccountMarket(message)
	case "update/account_market":
		c.handleUpdateAccountMarket(message)
	case "error":
		c.handleError(message)
	case "ping":
		c.handlePing(message)
	default:
		if baseMsg.Type != "" {
			log.Printf("AccountMarket unhandled message type: %s", baseMsg.Type)
		}
	}
}

// handleConnected subscribes ONLY to account markets
func (c *WsAccountMarketClient) handleConnected() {
	time.Sleep(100 * time.Millisecond)

	// Subscribe ONLY to account markets
	for _, sub := range c.accountMarketSubs {
		authToken := sub.AuthToken

		// Generate auth token if TxClient is provided
		if sub.TxClient != nil {
			deadline := time.Now().Add(6 * time.Hour)
			token, err := sub.TxClient.GetAuthToken(deadline)
			if err != nil {
				log.Printf("AccountMarket failed to generate auth token for account %d market %d: %v", sub.AccountIndex, sub.MarketID, err)
				continue
			}
			authToken = token
		} else if c.txClient != nil && authToken == "" {
			// Use global TxClient if available
			deadline := time.Now().Add(6 * time.Hour)
			token, err := c.txClient.GetAuthToken(deadline)
			if err != nil {
				log.Printf("AccountMarket failed to generate auth token for account %d market %d: %v", sub.AccountIndex, sub.MarketID, err)
				continue
			}
			authToken = token
		}

		msg := NewAccountMarketSubscription(sub.AccountIndex, sub.MarketID, authToken)
		//log.Printf("AccountMarket subscribing to account %d market %d", sub.AccountIndex, sub.MarketID)
		if err := c.sendMessage(msg); err != nil {
			log.Printf("AccountMarket failed to subscribe to account %d market %d: %v", sub.AccountIndex, sub.MarketID, err)
		}
	}
}

// handleSubscribedAccountMarket handles initial account market subscription response
func (c *WsAccountMarketClient) handleSubscribedAccountMarket(message []byte) {
	// Log raw message for debugging
	// log.Printf("AccountMarket raw snapshot message: %s", string(message))

	var snapshot AccountMarketSnapshot
	if err := json.Unmarshal(message, &snapshot); err != nil {
		log.Printf("AccountMarket failed to unmarshal snapshot: %v", err)

		// Try to unmarshal as generic map to see the structure
		var genericData map[string]interface{}
		if err2 := json.Unmarshal(message, &genericData); err2 == nil {
			log.Printf("AccountMarket snapshot structure: %+v", genericData)
		}
		return
	}

	if c.onAccountMarketSnapshot != nil {
		c.onAccountMarketSnapshot(&snapshot)
	}
}

// handleUpdateAccountMarket handles account market update messages
func (c *WsAccountMarketClient) handleUpdateAccountMarket(message []byte) {
	// Log raw message for debugging
	// log.Printf("AccountMarket raw update message: %s", string(message))

	var update AccountMarketUpdate
	if err := json.Unmarshal(message, &update); err != nil {
		log.Printf("AccountMarket failed to unmarshal update: %v", err)

		// Try to unmarshal as generic map to see the structure
		var genericData map[string]interface{}
		if err2 := json.Unmarshal(message, &genericData); err2 == nil {
			log.Printf("AccountMarket update structure: %+v", genericData)
		}
		return
	}

	// log.Printf("AccountMarket received update for account %d", update.Account)

	// Call the general update handler if set
	if c.onAccountMarketUpdate != nil {
		c.onAccountMarketUpdate(&update)
	}

	// Call specific handlers based on what data is present

	// Handle order updates - check if orders array has content
	if len(update.Orders) > 0 && c.onOrderUpdate != nil {
		// log.Printf("AccountMarket calling order update handler with %d orders", len(update.Orders))
		c.onOrderUpdate(update.Account, update.Orders)
	}

	// Handle position updates - check if position is not nil
	if update.Position != nil && c.onPositionUpdate != nil {
		// log.Printf("AccountMarket calling position update handler")
		c.onPositionUpdate(update.Account, update.Position)
	}

	// Handle trade updates - check if trades array has content
	if len(update.Trades) > 0 && c.onTradeUpdate != nil {
		// log.Printf("AccountMarket calling trade update handler with %d trades", len(update.Trades))
		c.onTradeUpdate(update.Account, update.Trades)
	}

	// Handle funding updates - check if funding history is not nil
	if update.FundingHistory != nil && c.onFundingUpdate != nil {
		// log.Printf("AccountMarket calling funding update handler")
		c.onFundingUpdate(update.Account, update.FundingHistory)
	}
}

// handleError handles error messages
func (c *WsAccountMarketClient) handleError(message []byte) {
	// Try nested error structure first
	var nestedError struct {
		Error struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
		Channel string `json:"channel,omitempty"`
	}

	if err := json.Unmarshal(message, &nestedError); err == nil && nestedError.Error.Code != 0 {
		errorMsg := &ErrorMessage{
			Type:    "error",
			Code:    nestedError.Error.Code,
			Message: nestedError.Error.Message,
			Channel: nestedError.Channel,
		}
		log.Printf("AccountMarket error: Code=%d, Message=%s, Channel=%s", errorMsg.Code, errorMsg.Message, errorMsg.Channel)

		if c.onError != nil {
			c.onError(errorMsg)
		}
		return
	}

	// Fallback to original format
	var errorMsg ErrorMessage
	if err := json.Unmarshal(message, &errorMsg); err != nil {
		log.Printf("AccountMarket failed to unmarshal error: %v", err)
		return
	}

	log.Printf("AccountMarket error: Code=%d, Message=%s", errorMsg.Code, errorMsg.Message)

	if c.onError != nil {
		c.onError(&errorMsg)
	}
}

// handlePing responds to ping messages
func (c *WsAccountMarketClient) handlePing(message []byte) {
	var ping PingMessage
	if err := json.Unmarshal(message, &ping); err != nil {
		log.Printf("AccountMarket failed to unmarshal ping: %v", err)
		return
	}

	pong := PongMessage{
		Type:      "pong",
		Timestamp: ping.Timestamp,
	}

	if err := c.sendMessage(pong); err != nil {
		log.Printf("AccountMarket failed to send pong: %v", err)
	}
}

// sendMessage sends a message through the WebSocket connection
func (c *WsAccountMarketClient) sendMessage(message interface{}) error {
	if !c.isConnected || c.conn == nil {
		return fmt.Errorf("AccountMarket not connected to WebSocket")
	}

	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("AccountMarket failed to marshal message: %v", err)
	}

	// log.Printf("AccountMarket sending message: %s", string(data))
	return c.conn.WriteMessage(websocket.TextMessage, data)
}
