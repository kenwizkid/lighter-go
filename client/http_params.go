package client

import "fmt"

// HTTPParams is the interface for HTTP request parameters
type HTTPParams interface {
	ToMap() map[string]string
}

// NextNonceParams represents parameters for GetNextNonce request
type NextNonceParams struct {
	AccountIndex int64 `json:"account_index"`
	APIKeyIndex  uint8 `json:"api_key_index"`
}

func (p NextNonceParams) ToMap() map[string]string {
	return map[string]string{
		"account_index": fmt.Sprintf("%d", p.AccountIndex),
		"api_key_index": fmt.Sprintf("%d", p.APIKeyIndex),
	}
}

// GetAccountParams represents parameters for GetAccount request
type GetAccountParams struct {
	By    string `json:"by"`
	Value string `json:"value"`
	Auth  string `json:"auth,omitempty"`
}

func (p GetAccountParams) ToMap() map[string]string {
	m := map[string]string{
		"by":    p.By,
		"value": p.Value,
	}
	if p.Auth != "" {
		m["auth"] = p.Auth
	}
	return m
}

// GetOrdersParams represents parameters for GetOrders request
type GetOrdersParams struct {
	AccountIndex int64  `json:"account_index"`
	MarketID     *uint8 `json:"market_id,omitempty"`
	Status       string `json:"status,omitempty"`
	Limit        int    `json:"limit,omitempty"`
	Offset       int    `json:"offset,omitempty"`
	Auth         string `json:"auth,omitempty"`
}

func (p GetOrdersParams) ToMap() map[string]string {
	m := map[string]string{
		"account_index": fmt.Sprintf("%d", p.AccountIndex),
	}
	if p.MarketID != nil {
		m["market_id"] = fmt.Sprintf("%d", *p.MarketID)
	}
	if p.Status != "" {
		m["status"] = p.Status
	}
	if p.Limit > 0 {
		m["limit"] = fmt.Sprintf("%d", p.Limit)
	}
	if p.Offset > 0 {
		m["offset"] = fmt.Sprintf("%d", p.Offset)
	}
	if p.Auth != "" {
		m["auth"] = p.Auth
	}
	return m
}

// GetTradesParams represents parameters for GetTrades request
type GetTradesParams struct {
	AccountIndex int64  `json:"account_index"`
	MarketID     *uint8 `json:"market_id,omitempty"`
	StartTime    *int64 `json:"start_time,omitempty"`
	EndTime      *int64 `json:"end_time,omitempty"`
	Limit        int    `json:"limit,omitempty"`
	Offset       int    `json:"offset,omitempty"`
	Auth         string `json:"auth,omitempty"`
}

func (p GetTradesParams) ToMap() map[string]string {
	m := map[string]string{
		"account_index": fmt.Sprintf("%d", p.AccountIndex),
	}
	if p.MarketID != nil {
		m["market_id"] = fmt.Sprintf("%d", *p.MarketID)
	}
	if p.StartTime != nil {
		m["start_time"] = fmt.Sprintf("%d", *p.StartTime)
	}
	if p.EndTime != nil {
		m["end_time"] = fmt.Sprintf("%d", *p.EndTime)
	}
	if p.Limit > 0 {
		m["limit"] = fmt.Sprintf("%d", p.Limit)
	}
	if p.Offset > 0 {
		m["offset"] = fmt.Sprintf("%d", p.Offset)
	}
	if p.Auth != "" {
		m["auth"] = p.Auth
	}
	return m
}
