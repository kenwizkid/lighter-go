package client

import (
	"fmt"
	"strconv"
	"strings"
)

// WSMessage is the interface for all WebSocket messages
type WSMessage interface {
	GetType() string
}

// SubscribeMessage represents a subscription request
type SubscribeMessage struct {
	Type    string `json:"type"`
	Channel string `json:"channel"`
}

func (m SubscribeMessage) GetType() string {
	return m.Type
}

// UnsubscribeMessage represents an unsubscription request
type UnsubscribeMessage struct {
	Type    string `json:"type"`
	Channel string `json:"channel"`
}

func (m UnsubscribeMessage) GetType() string {
	return m.Type
}

// PingMessage represents a ping message
type PingMessage struct {
	Type      string `json:"type"`
	Timestamp int64  `json:"timestamp"`
}

func (m PingMessage) GetType() string {
	return m.Type
}

// PongMessage represents a pong message response
type PongMessage struct {
	Type      string `json:"type"`
	Timestamp int64  `json:"timestamp"`
}

func (m PongMessage) GetType() string {
	return m.Type
}

// Helper functions to create messages

// NewOrderBookSubscription creates a subscription message for an order book
func NewOrderBookSubscription(marketID uint8) SubscribeMessage {
	return SubscribeMessage{
		Type:    "subscribe",
		Channel: fmt.Sprintf("order_book/%d", marketID),
	}
}

// AccountMarketSubscription represents a market-specific account subscription
type AccountMarketSubscription struct {
	Type    string `json:"type"`
	Channel string `json:"channel"`
	Auth    string `json:"auth,omitempty"` // Optional auth token for private subscriptions
}

func (m AccountMarketSubscription) GetType() string {
	return m.Type
}

// NewAccountMarketSubscription creates a subscription message for an account on a specific market
func NewAccountMarketSubscription(accountIndex int64, marketID uint8, authToken string) AccountMarketSubscription {
	return AccountMarketSubscription{
		Type:    "subscribe",
		Channel: fmt.Sprintf("account_market/%d/%d", marketID, accountIndex),
		Auth:    authToken,
	}
}

// NewOrderBookUnsubscription creates an unsubscription message for an order book
func NewOrderBookUnsubscription(marketID uint8) UnsubscribeMessage {
	return UnsubscribeMessage{
		Type:    "unsubscribe",
		Channel: fmt.Sprintf("order_book/%d", marketID),
	}
}

// NewAccountMarketUnsubscription creates an unsubscription message for an account market
func NewAccountMarketUnsubscription(accountID int, marketID uint8) UnsubscribeMessage {
	return UnsubscribeMessage{
		Type:    "unsubscribe",
		Channel: fmt.Sprintf("account_market/%d/%d", marketID, accountID),
	}
}

// Helper functions

// extractMarketID extracts the market ID from a channel string
func extractMarketID(channel string) uint8 {
	parts := strings.Split(channel, ":")
	if len(parts) >= 2 {
		id, err := strconv.ParseUint(parts[1], 10, 8)
		if err == nil {
			return uint8(id)
		}
	}
	
	// Try with forward slash
	parts = strings.Split(channel, "/")
	if len(parts) >= 2 {
		id, err := strconv.ParseUint(parts[1], 10, 8)
		if err == nil {
			return uint8(id)
		}
	}
	
	return 255 // Invalid market ID (max uint8 + 1)
}

// extractAccountMarketIDs extracts both account ID and market ID from a channel string
// Format: "account_market/<market_id>/<account_id>"
func extractAccountMarketIDs(channel string) (accountID int, marketID uint8) {
	parts := strings.Split(channel, "/")
	if len(parts) >= 3 {
		mid, err1 := strconv.ParseUint(parts[1], 10, 8)
		aid, err2 := strconv.Atoi(parts[2])
		if err1 == nil && err2 == nil {
			return aid, uint8(mid)
		}
	}
	return -1, 255
}