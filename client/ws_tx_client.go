package client

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// TxResponseHandler handles transaction response callbacks
type TxResponseHandler func(requestID string, response *TxResponse)

// BatchTxResponseHandler handles batch transaction response callbacks
type BatchTxResponseHandler func(requestID string, response *BatchTxResponse)

// TxResponse represents a transaction response
type TxResponse struct {
	ID     string                 `json:"id"`
	Status string                 `json:"status"`
	TxHash string                 `json:"tx_hash,omitempty"`
	Error  string                 `json:"error,omitempty"`
	Data   map[string]interface{} `json:"data,omitempty"`
}

// BatchTxResponse represents a batch transaction response
type BatchTxResponse struct {
	ID       string                   `json:"id"`
	Status   string                   `json:"status"`
	TxHashes []string                 `json:"tx_hashes,omitempty"`
	Results  []map[string]interface{} `json:"results,omitempty"`
	Failures []map[string]interface{} `json:"failures,omitempty"`
	Error    string                   `json:"error,omitempty"`
}

// WsTxClient is a WebSocket client dedicated to sending transactions
type WsTxClient struct {
	baseURL               string
	conn                  *websocket.Conn
	onTxResponse          TxResponseHandler
	onBatchTxResponse     BatchTxResponseHandler
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

// NewWsTxClient creates a new dedicated transaction WebSocket client
func NewWsTxClient(options ...WsTxClientOption) (*WsTxClient, error) {
	client := &WsTxClient{
		baseURL:              "wss://mainnet.zklighter.elliot.ai/stream",
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

// WsTxClientOption configures the WsTxClient
type WsTxClientOption func(*WsTxClient)

// WithTxWsHost sets the WebSocket host
func WithTxWsHost(host string) WsTxClientOption {
	return func(c *WsTxClient) {
		host = strings.TrimPrefix(host, "https://")
		host = strings.TrimPrefix(host, "http://")
		c.baseURL = fmt.Sprintf("wss://%s/stream", host)
	}
}

// WithTxResponseHandler sets the transaction response handler
func WithTxResponseHandler(handler TxResponseHandler) WsTxClientOption {
	return func(c *WsTxClient) {
		c.onTxResponse = handler
	}
}

// WithBatchTxResponseHandler sets the batch transaction response handler
func WithBatchTxResponseHandler(handler BatchTxResponseHandler) WsTxClientOption {
	return func(c *WsTxClient) {
		c.onBatchTxResponse = handler
	}
}

// WithTxWsMessageHandler sets the raw message handler
func WithTxWsMessageHandler(handler MessageHandler) WsTxClientOption {
	return func(c *WsTxClient) {
		c.onMessage = handler
	}
}

// WithTxWsErrorHandler sets the error handler
func WithTxWsErrorHandler(handler ErrorHandler) WsTxClientOption {
	return func(c *WsTxClient) {
		c.onError = handler
	}
}

// WithTxWsReconnectInterval sets the reconnect interval
func WithTxWsReconnectInterval(interval time.Duration) WsTxClientOption {
	return func(c *WsTxClient) {
		c.reconnectInterval = interval
	}
}

// Connect establishes a WebSocket connection
func (c *WsTxClient) connect() error {
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
	log.Printf("TxWs client connected to: %s", c.baseURL)
	return nil
}

// Run starts the transaction WebSocket client
func (c *WsTxClient) Run() error {
	defer close(c.done)

	for attempt := 0; attempt < c.maxReconnectAttempts; attempt++ {
		if err := c.connect(); err != nil {
			if attempt == c.maxReconnectAttempts-1 {
				return fmt.Errorf("failed to connect after %d attempts: %v", c.maxReconnectAttempts, err)
			}
			log.Printf("TxWs connection attempt %d failed: %v. Retrying in %v", attempt+1, err, c.reconnectInterval)
			time.Sleep(c.reconnectInterval)
			continue
		}

		// Reset attempt counter on successful connection
		attempt = 0

		// Handle messages
		if err := c.handleMessages(); err != nil {
			log.Printf("TxWs message handling error: %v", err)
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

// Stop stops the transaction client
func (c *WsTxClient) Stop() {
	c.cancel()
	if c.conn != nil {
		c.conn.Close()
	}
	<-c.done
}

// IsConnected returns whether the client is connected
func (c *WsTxClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isConnected
}

// SendTransaction sends a single transaction via WebSocket
func (c *WsTxClient) SendTransaction(signedTx string) (string, error) {
	if !c.isConnected || c.conn == nil {
		return "", fmt.Errorf("TxWs not connected to WebSocket")
	}

	// Generate unique request ID
	requestID := c.generateRequestID()

	message := map[string]interface{}{
		"method": "jsonapi/sendtx",
		"params": map[string]interface{}{
			"tx": signedTx,
		},
		"id": requestID,
	}

	data, err := json.Marshal(message)
	if err != nil {
		return "", fmt.Errorf("TxWs failed to marshal transaction message: %v", err)
	}
	
	log.Printf("TxWs sending transaction: %s", string(data))
	
	if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
		return "", fmt.Errorf("TxWs failed to send transaction: %v", err)
	}

	return requestID, nil
}

// SendBatchTransactions sends multiple transactions via WebSocket
func (c *WsTxClient) SendBatchTransactions(signedTxs []string) (string, error) {
	if !c.isConnected || c.conn == nil {
		return "", fmt.Errorf("TxWs not connected to WebSocket")
	}

	// Generate unique request ID
	requestID := c.generateRequestID()

	message := map[string]interface{}{
		"method": "jsonapi/sendtxbatch",
		"params": map[string]interface{}{
			"txs": signedTxs,
		},
		"id": requestID,
	}

	data, err := json.Marshal(message)
	if err != nil {
		return "", fmt.Errorf("TxWs failed to marshal batch transaction message: %v", err)
	}
	
	log.Printf("TxWs sending batch transactions: %s", string(data))
	
	if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
		return "", fmt.Errorf("TxWs failed to send batch transactions: %v", err)
	}

	return requestID, nil
}

// generateRequestID generates a unique request ID
func (c *WsTxClient) generateRequestID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// handleMessages processes incoming WebSocket messages for transactions
func (c *WsTxClient) handleMessages() error {
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
					log.Printf("TxWs failed to send ping: %v", err)
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
					return fmt.Errorf("TxWs WebSocket read error: %v", err)
				}
				return err
			}

			go c.processMessage(message)
		}
	}
}

// processMessage handles incoming messages specifically for transactions
func (c *WsTxClient) processMessage(message []byte) {
	var baseMsg struct {
		Type    string `json:"type"`
		Channel string `json:"channel,omitempty"`
		Error   *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}

	if err := json.Unmarshal(message, &baseMsg); err != nil {
		log.Printf("TxWs failed to unmarshal base message: %v", err)
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

	// Handle specific message types - ONLY TRANSACTION RELATED
	switch baseMsg.Type {
	case "connected":
		c.handleConnected()
	case "error":
		c.handleError(message)
	case "ping":
		c.handlePing(message)
	case "jsonapi/sendtx":
		c.handleTxResponse(message)
	case "jsonapi/sendtxbatch":
		c.handleBatchTxResponse(message)
	case "response/jsonapi/sendtx":
		c.handleTxResponse(message)
	case "response/jsonapi/sendtxbatch":
		c.handleBatchTxResponse(message)
	default:
		if baseMsg.Type != "" {
			log.Printf("TxWs unhandled message type: %s", baseMsg.Type)
		}
	}
}

// handleConnected handles the initial connection - NO SUBSCRIPTIONS
func (c *WsTxClient) handleConnected() {
	log.Println("TxWs client connected - ready to send transactions (no subscriptions)")
	// NO automatic subscriptions - this client is ONLY for sending transactions
}

// handleTxResponse handles transaction response messages
func (c *WsTxClient) handleTxResponse(message []byte) {
	if c.onTxResponse == nil {
		return
	}

	var msg struct {
		Type string `json:"type"`
		Data struct {
			ID       string                 `json:"id"`
			Status   string                 `json:"status"`
			TxHash   string                 `json:"tx_hash,omitempty"`
			Error    string                 `json:"error,omitempty"`
			Data     map[string]interface{} `json:"data,omitempty"`
		} `json:"data"`
	}

	if err := json.Unmarshal(message, &msg); err != nil {
		log.Printf("TxWs failed to unmarshal transaction response: %v", err)
		return
	}

	response := &TxResponse{
		ID:     msg.Data.ID,
		Status: msg.Data.Status,
		TxHash: msg.Data.TxHash,
		Error:  msg.Data.Error,
		Data:   msg.Data.Data,
	}

	c.onTxResponse(msg.Data.ID, response)
}

// handleBatchTxResponse handles batch transaction response messages
func (c *WsTxClient) handleBatchTxResponse(message []byte) {
	if c.onBatchTxResponse == nil {
		return
	}

	var msg struct {
		Type string `json:"type"`
		Data struct {
			ID       string                   `json:"id"`
			Status   string                   `json:"status"`
			TxHashes []string                 `json:"tx_hashes,omitempty"`
			Results  []map[string]interface{} `json:"results,omitempty"`
			Failures []map[string]interface{} `json:"failures,omitempty"`
			Error    string                   `json:"error,omitempty"`
		} `json:"data"`
	}

	if err := json.Unmarshal(message, &msg); err != nil {
		log.Printf("TxWs failed to unmarshal batch transaction response: %v", err)
		return
	}

	response := &BatchTxResponse{
		ID:       msg.Data.ID,
		Status:   msg.Data.Status,
		TxHashes: msg.Data.TxHashes,
		Results:  msg.Data.Results,
		Failures: msg.Data.Failures,
		Error:    msg.Data.Error,
	}

	c.onBatchTxResponse(msg.Data.ID, response)
}

// handleError handles error messages
func (c *WsTxClient) handleError(message []byte) {
	var errorMsg ErrorMessage
	if err := json.Unmarshal(message, &errorMsg); err != nil {
		log.Printf("TxWs failed to unmarshal error: %v", err)
		return
	}

	log.Printf("TxWs error: Code=%d, Message=%s", errorMsg.Code, errorMsg.Message)
	
	if c.onError != nil {
		c.onError(&errorMsg)
	}
}

// handlePing responds to ping messages
func (c *WsTxClient) handlePing(message []byte) {
	var ping PingMessage
	if err := json.Unmarshal(message, &ping); err != nil {
		log.Printf("TxWs failed to unmarshal ping: %v", err)
		return
	}

	pong := PongMessage{
		Type:      "pong",
		Timestamp: ping.Timestamp,
	}

	if err := c.sendMessage(pong); err != nil {
		log.Printf("TxWs failed to send pong: %v", err)
	}
}

// sendMessage sends a message through the WebSocket connection
func (c *WsTxClient) sendMessage(message interface{}) error {
	if !c.isConnected || c.conn == nil {
		return fmt.Errorf("TxWs not connected to WebSocket")
	}

	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("TxWs failed to marshal message: %v", err)
	}
	
	return c.conn.WriteMessage(websocket.TextMessage, data)
}