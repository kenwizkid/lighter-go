package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/drinkthere/lighter-go/types/txtypes"
)

// parseResultStatus parses the result status from API response
func (c *HTTPClient) parseResultStatus(respBody []byte) error {
	resultStatus := &ResultCode{}
	if err := json.Unmarshal(respBody, resultStatus); err != nil {
		return err
	}
	if resultStatus.Code != CodeOK {
		return errors.New(resultStatus.Message)
	}
	return nil
}

// getAndParseL2HTTPResponse handles legacy API calls with map[string]any parameters
func (c *HTTPClient) getAndParseL2HTTPResponse(path string, params map[string]any, result interface{}) error {
	u, err := url.Parse(c.endpoint)
	if err != nil {
		return err
	}
	u.Path = path

	q := u.Query()
	for k, v := range params {
		q.Set(k, fmt.Sprintf("%v", v))
	}
	u.RawQuery = q.Encode()
	resp, err := c.client.Get(u.String())
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return errors.New(string(body))
	}
	if err = c.parseResultStatus(body); err != nil {
		return err
	}
	if err := json.Unmarshal(body, result); err != nil {
		return err
	}
	return nil
}

// getAndParseResponse handles new API calls with typed HTTPParams
func (c *HTTPClient) getAndParseResponse(path string, params HTTPParams, result interface{}) error {
	u, err := url.Parse(c.endpoint)
	if err != nil {
		return err
	}
	u.Path = path

	if params != nil {
		q := u.Query()
		for k, v := range params.ToMap() {
			q.Set(k, v)
		}
		u.RawQuery = q.Encode()
	}

	resp, err := c.client.Get(u.String())
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return errors.New(string(body))
	}

	if err = c.parseResultStatus(body); err != nil {
		return err
	}
	
	if err := json.Unmarshal(body, result); err != nil {
		return err
	}

	return nil
}

// =============================================================================
// Legacy API Methods (for backward compatibility)
// =============================================================================

// GetNextNonce fetches the next nonce using legacy parameters
func (c *HTTPClient) GetNextNonce(accountIndex int64, apiKeyIndex uint8) (int64, error) {
	result := &NextNonce{}
	err := c.getAndParseL2HTTPResponse("api/v1/nextNonce", map[string]any{"account_index": accountIndex, "api_key_index": apiKeyIndex}, result)
	if err != nil {
		return -1, err
	}
	return result.Nonce, nil
}

// GetApiKey fetches API key information using legacy parameters
func (c *HTTPClient) GetApiKey(accountIndex int64, apiKeyIndex uint8) (*AccountApiKeys, error) {
	result := &AccountApiKeys{}
	err := c.getAndParseL2HTTPResponse("api/v1/apikeys", map[string]any{"account_index": accountIndex, "api_key_index": apiKeyIndex}, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// GetTransferFeeInfo fetches transfer fee information
func (c *HTTPClient) GetTransferFeeInfo(accountIndex, toAccountIndex int64, auth string) (*TransferFeeInfo, error) {
	result := &TransferFeeInfo{}
	err := c.getAndParseL2HTTPResponse("api/v1/transferFeeInfo", map[string]any{
		"account_index":    accountIndex,
		"to_account_index": toAccountIndex,
		"auth":             auth,
	}, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// =============================================================================
// New Typed API Methods (recommended)
// =============================================================================

// GetNextNonceV2 fetches the next nonce with typed parameters
func (c *HTTPClient) GetNextNonceV2(params NextNonceParams) (*NextNonce, error) {
	result := &NextNonce{}
	err := c.getAndParseResponse("api/v1/nextNonce", params, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// GetApiKeyV2 fetches API key information with typed parameters
func (c *HTTPClient) GetApiKeyV2(params GetAccountParams) (*AccountApiKeys, error) {
	result := &AccountApiKeys{}
	err := c.getAndParseResponse("api/v1/apikeys", params, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// GetAccountInfo fetches account information
func (c *HTTPClient) GetAccountInfo(accountIndex int64) (*AccountInfo, error) {
	params := GetAccountParams{
		By:    "index",
		Value: fmt.Sprintf("%d", accountIndex),
	}
	result := &AccountResponse{}
	err := c.getAndParseResponse("api/v1/account", params, result)
	if err != nil {
		return nil, err
	}
	if len(result.Accounts) == 0 {
		return nil, fmt.Errorf("no account found with index %d", accountIndex)
	}
	return &result.Accounts[0], nil
}

// GetAccountInfoWithAuth fetches account information with authentication
func (c *HTTPClient) GetAccountInfoWithAuth(accountIndex int64, authToken string) (*AccountInfo, error) {
	params := GetAccountParams{
		By:    "index",
		Value: fmt.Sprintf("%d", accountIndex),
		Auth:  authToken,
	}
	result := &AccountResponse{}
	err := c.getAndParseResponse("api/v1/account", params, result)
	if err != nil {
		return nil, err
	}
	if len(result.Accounts) == 0 {
		return nil, fmt.Errorf("no account found with index %d", accountIndex)
	}
	return &result.Accounts[0], nil
}

// GetOrders fetches orders with typed parameters
func (c *HTTPClient) GetOrders(params GetOrdersParams) (*OrdersResponse, error) {
	result := &OrdersResponse{}
	err := c.getAndParseResponse("api/v1/accountActiveOrders", params, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// GetTrades fetches trades with typed parameters
func (c *HTTPClient) GetTrades(params GetTradesParams) (*TradesResponse, error) {
	result := &TradesResponse{}
	err := c.getAndParseResponse("api/v1/trades", params, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// =============================================================================
// Transaction Methods
// =============================================================================

// SendRawTx sends a raw transaction to the exchange
func (c *HTTPClient) SendRawTx(tx txtypes.TxInfo) (string, error) {
	txType := tx.GetTxType()
	txInfo, err := tx.GetTxInfo()
	if err != nil {
		return "", err
	}

	data := url.Values{"tx_type": {strconv.Itoa(int(txType))}, "tx_info": {txInfo}}

	if c.fatFingerProtection == false {
		data.Add("price_protection", "false")
	}

	req, _ := http.NewRequest("POST", c.endpoint+"/api/v1/sendTx", strings.NewReader(data.Encode()))
	req.Header.Set("Channel-Name", c.channelName)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", errors.New(string(body))
	}
	if err = c.parseResultStatus(body); err != nil {
		return "", err
	}
	res := &TxHash{}
	if err := json.Unmarshal(body, res); err != nil {
		return "", err
	}

	return res.TxHash, nil
}

// =============================================================================
// Response Types
// =============================================================================

// AccountInfo represents account information
// AccountResponse represents the full API response for account queries
type AccountResponse struct {
	Code     int           `json:"code"`
	Total    int           `json:"total"`
	Accounts []AccountInfo `json:"accounts"`
}

type AccountInfo struct {
	Code                    int            `json:"code"`
	AccountType             int            `json:"account_type"`
	AccountIndex            int64          `json:"index"`
	L1Address               string         `json:"l1_address"`
	CancelAllTime           int64          `json:"cancel_all_time"`
	TotalOrderCount         int            `json:"total_order_count"`
	TotalIsolatedOrderCount int            `json:"total_isolated_order_count"`
	Balance                 string         `json:"balance"`
	AvailableBalance        string         `json:"available_balance"`
	MarginBalance           string         `json:"margin_balance"`
	UnrealizedPnL           string         `json:"unrealized_pnl"`
	RealizedPnL             string         `json:"realized_pnl"`
	Positions               []PositionInfo `json:"positions"`
	Orders                  []OrderInfo    `json:"orders"`
}

// PositionInfo represents position information
type PositionInfo struct {
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

// OrderInfo represents order information from REST API
type OrderInfo struct {
	OrderIndex          int64  `json:"order_index"`
	ClientOrderIndex    int64  `json:"client_order_index"`
	OrderID             string `json:"order_id"`
	ClientOrderID       string `json:"client_order_id"`
	MarketIndex         uint8  `json:"market_index"`
	OwnerAccountIndex   int64  `json:"owner_account_index"`
	InitialBaseAmount   string `json:"initial_base_amount"`
	Price               string `json:"price"`
	Nonce               int64  `json:"nonce"`
	RemainingBaseAmount string `json:"remaining_base_amount"`
	IsAsk               bool   `json:"is_ask"`
	BaseSize            int64  `json:"base_size"`
	BasePrice           int64  `json:"base_price"`
	FilledBaseAmount    string `json:"filled_base_amount"`
	FilledQuoteAmount   string `json:"filled_quote_amount"`
	Side                string `json:"side"`
	Type                string `json:"type"`
	TimeInForce         string `json:"time_in_force"`
	ReduceOnly          bool   `json:"reduce_only"`
	TriggerPrice        string `json:"trigger_price"`
	OrderExpiry         int64  `json:"order_expiry"`
	Status              string `json:"status"`
	TriggerStatus       string `json:"trigger_status"`
	TriggerTime         int64  `json:"trigger_time"`
	ParentOrderIndex    int64  `json:"parent_order_index"`
	ParentOrderID       string `json:"parent_order_id"`
	ToTriggerOrderID0   string `json:"to_trigger_order_id_0"`
	ToTriggerOrderID1   string `json:"to_trigger_order_id_1"`
	ToCancelOrderID0    string `json:"to_cancel_order_id_0"`
	BlockHeight         int64  `json:"block_height"`
	Timestamp           int64  `json:"timestamp"`
	CreatedAt           int64  `json:"created_at"`
	UpdatedAt           int64  `json:"updated_at"`
}

// TradeInfo represents trade information
type TradeInfo struct {
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

// OrdersResponse represents the response for orders query
type OrdersResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Orders  []OrderInfo `json:"orders"`
	Total   int         `json:"total"`
}

// TradesResponse represents the response for trades query
type TradesResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Trades  []TradeInfo `json:"trades"`
	Total   int         `json:"total"`
}
