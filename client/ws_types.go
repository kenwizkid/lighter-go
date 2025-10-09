package client

import "time"

// Handler function types for WebSocket clients

// MessageHandler is a callback function for handling raw WebSocket messages
type MessageHandler func(messageType string, rawData []byte)

// OrderBookUpdateHandler handles order book updates
type OrderBookUpdateHandler func(marketID uint8, update *OrderBookUpdate)

// OrderBookSnapshotHandler handles order book snapshots
type OrderBookSnapshotHandler func(marketID uint8, snapshot *OrderBookSnapshot)

// AccountMarketUpdateHandler handles account market updates
type AccountMarketUpdateHandler func(update *AccountMarketUpdate)

// AccountMarketSnapshotHandler handles account market snapshots
type AccountMarketSnapshotHandler func(snapshot *AccountMarketSnapshot)

// AccountMarketOrderUpdateHandler handles order updates specifically
type AccountMarketOrderUpdateHandler func(account int64, orders []AccountMarketOrder)

// AccountMarketPositionUpdateHandler handles position updates specifically
type AccountMarketPositionUpdateHandler func(account int64, position *AccountMarketPosition)

// AccountMarketTradeUpdateHandler handles trade updates specifically
type AccountMarketTradeUpdateHandler func(account int64, trades []Trade)

// AccountMarketFundingUpdateHandler handles funding updates specifically
type AccountMarketFundingUpdateHandler func(account int64, funding *FundingHistory)

// ErrorHandler handles error messages
type ErrorHandler func(error *ErrorMessage)

// AccountMarketSub represents an account-market pair to subscribe to
type AccountMarketSub struct {
	AccountIndex int64
	MarketID     uint8
	AuthToken    string    // Optional auth token for authentication (deprecated - use TxClient instead)
	TxClient     *TxClient // Optional TxClient for generating auth tokens
}

// OrderBookLevel represents a single price level in the order book
type OrderBookLevel struct {
	Price string `json:"price"`
	Size  string `json:"size"`
}

// OrderBookData contains the order book data
type OrderBookData struct {
	Code   int              `json:"code"`
	Asks   []OrderBookLevel `json:"asks"`
	Bids   []OrderBookLevel `json:"bids"`
	Offset int              `json:"offset"`
}

// OrderBookSnapshot represents the initial order book snapshot
type OrderBookSnapshot struct {
	Channel   string        `json:"channel"`
	Type      string        `json:"type"`
	OrderBook OrderBookData `json:"order_book"`
}

// OrderBookUpdate represents an order book update message
type OrderBookUpdate struct {
	Channel   string        `json:"channel"`
	Type      string        `json:"type"`
	Offset    int           `json:"offset"`
	OrderBook OrderBookData `json:"order_book"`
}

// Position represents a user's position
type Position struct {
	Symbol        string `json:"symbol"`
	MarketID      uint8  `json:"market_id"`
	Side          string `json:"side"`
	Quantity      string `json:"quantity"`
	EntryPrice    string `json:"entry_price"`
	MarkPrice     string `json:"mark_price"`
	UnrealizedPnL string `json:"unrealized_pnl"`
	RealizedPnL   string `json:"realized_pnl"`
}

// Order represents an order
type Order struct {
	OrderID        string    `json:"order_id"`
	ClientOrderID  string    `json:"client_order_id"`
	Symbol         string    `json:"symbol"`
	MarketID       uint8     `json:"market_id"`
	Side           string    `json:"side"`
	OrderType      string    `json:"order_type"`
	Price          string    `json:"price"`
	Quantity       string    `json:"quantity"`
	FilledQuantity string    `json:"filled_quantity"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// AccountData contains account information
type AccountData struct {
	AccountIndex     int64      `json:"account_id"`
	Balance          string     `json:"balance"`
	AvailableBalance string     `json:"available_balance"`
	MarginBalance    string     `json:"margin_balance"`
	UnrealizedPnL    string     `json:"unrealized_pnl"`
	RealizedPnL      string     `json:"realized_pnl"`
	MarginRatio      string     `json:"margin_ratio"`
	Positions        []Position `json:"positions"`
	Orders           []Order    `json:"orders"`
}

// AccountMarketPosition represents position data for a specific market
type AccountMarketPosition struct {
	MarketID               uint8  `json:"market_id"`
	Symbol                 string `json:"symbol"`
	InitialMarginFraction  string `json:"initial_margin_fraction"`
	OpenOrderCount         int    `json:"open_order_count"`
	PendingOrderCount      int    `json:"pending_order_count"`
	PositionTiedOrderCount int    `json:"position_tied_order_count"`
	Sign                   int    `json:"sign"`     // 1 for long, -1 for short
	Position               string `json:"position"` // position size
	AvgEntryPrice          string `json:"avg_entry_price"`
	PositionValue          string `json:"position_value"`
	UnrealizedPnL          string `json:"unrealized_pnl"`
	RealizedPnL            string `json:"realized_pnl"`
	LiquidationPrice       string `json:"liquidation_price"`
	TotalFundingPaidOut    string `json:"total_funding_paid_out"`
	MarginMode             int    `json:"margin_mode"` // 1 for cross, 2 for isolated
	AllocatedMargin        string `json:"allocated_margin"`
}

// AccountMarketOrder represents an order for a specific market
type AccountMarketOrder struct {
	OrderID          string `json:"order_id"`
	ClientOrderID    string `json:"client_order_id"`
	ClientOrderIndex int64  `json:"client_order_index"`
	MarketID         uint8  `json:"market_id"`
	OrderType        string `json:"type"`
	Price            string `json:"price"`
	Quantity         string `json:"initial_base_amount"`
	FilledQuantity   string `json:"filled_base_amount"`
	Status           string `json:"status"`
	CreatedAt        int64  `json:"created_at"`
	UpdatedAt        int64  `json:"updated_at"`
	IsAsk            bool   `json:"is_ask"`
}

// FundingHistory represents funding fee history data
type FundingHistory struct {
	Timestamp    int64  `json:"timestamp"`
	MarketID     int64  `json:"market_id"`
	FundingID    int64  `json:"funding_id"`
	Change       string `json:"change"`
	Rate         string `json:"rate"`
	PositionSize string `json:"position_size"`
	PositionSide string `json:"position_side"`
}

// Trade represents a trade record
type Trade struct {
	TradeID    string `json:"trade_id"`
	OrderID    string `json:"order_id"`
	MarketID   uint8  `json:"market_id"`
	Symbol     string `json:"symbol"`
	Side       string `json:"side"`
	Price      string `json:"price"`
	Quantity   string `json:"quantity"`
	Fee        string `json:"fee"`
	FeeAsset   string `json:"fee_asset"`
	IsMaker    bool   `json:"is_maker"`
	ExecutedAt int64  `json:"executed_at"`
}

// AccountMarketUpdate represents an account market update message (matches API docs)
type AccountMarketUpdate struct {
	Account        int64                  `json:"account"`
	Channel        string                 `json:"channel"`
	Type           string                 `json:"type"`
	FundingHistory *FundingHistory        `json:"funding_history,omitempty"`
	Orders         []AccountMarketOrder   `json:"orders"`
	Position       *AccountMarketPosition `json:"position,omitempty"`
	Trades         []Trade                `json:"trades"`
}

// AccountMarketSnapshot represents the initial account market subscription response
// Note: API docs don't mention snapshot, but keeping for backward compatibility
type AccountMarketSnapshot struct {
	Account        int64                  `json:"account"`
	Channel        string                 `json:"channel"`
	Type           string                 `json:"type"`
	FundingHistory *FundingHistory        `json:"funding_history,omitempty"`
	Orders         []AccountMarketOrder   `json:"orders"`
	Position       *AccountMarketPosition `json:"position,omitempty"`
	Trades         []Trade                `json:"trades"`
}

// Legacy AccountMarketData - kept for backward compatibility
type AccountMarketData struct {
	AccountIndex     int64                  `json:"account_id"`
	MarketID         uint8                  `json:"market_id"`
	Balance          string                 `json:"balance"`
	AvailableBalance string                 `json:"available_balance"`
	MarginBalance    string                 `json:"margin_balance"`
	Position         *AccountMarketPosition `json:"position,omitempty"`
	Orders           []AccountMarketOrder   `json:"orders"`
	MarginRatio      string                 `json:"margin_ratio"`
}

// ErrorMessage represents a WebSocket error message
type ErrorMessage struct {
	Type    string `json:"type"`
	Code    int    `json:"code"`
	Message string `json:"message"`
	Channel string `json:"channel,omitempty"`
}

// ConnectedMessage represents the initial connection message
type ConnectedMessage struct {
	Type      string `json:"type"`
	Timestamp int64  `json:"timestamp"`
}

// SubscriptionMessage represents a subscription confirmation
type SubscriptionMessage struct {
	Type    string `json:"type"`
	Channel string `json:"channel"`
	Success bool   `json:"success"`
}
