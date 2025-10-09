package client

import (
	"fmt"
)

// OrderBook represents a market on Lighter exchange
type OrderBook struct {
	MarketID                uint8   `json:"market_id"`
	Symbol                  string  `json:"symbol"`
	Status                  string  `json:"status"`
	TakerFee                string  `json:"taker_fee"`
	MakerFee                string  `json:"maker_fee"`
	LiquidationFee          string  `json:"liquidation_fee"`
	MinBaseAmount           string  `json:"min_base_amount"`
	MinQuoteAmount          string  `json:"min_quote_amount"`
	SupportedSizeDecimals   int     `json:"supported_size_decimals"`
	SupportedPriceDecimals  int     `json:"supported_price_decimals"`
	SupportedQuoteDecimals  int     `json:"supported_quote_decimals"`
}

// OrderBooksResponse represents the response from order_books API
type OrderBooksResponse struct {
	Code        int          `json:"code"`
	Message     string       `json:"message"`
	OrderBooks  []OrderBook  `json:"order_books"`
}

// GetOrderBooks fetches all available markets from Lighter exchange
func (c *HTTPClient) GetOrderBooks() (*OrderBooksResponse, error) {
	result := &OrderBooksResponse{}
	err := c.getAndParseL2HTTPResponse("api/v1/order_books", nil, result)
	if err != nil {
		return nil, fmt.Errorf("failed to get order books: %v", err)
	}
	return result, nil
}

// GetMarketByID fetches information for a specific market
func (c *HTTPClient) GetMarketByID(marketID uint8) (*OrderBook, error) {
	orderBooks, err := c.GetOrderBooks()
	if err != nil {
		return nil, err
	}
	
	for _, ob := range orderBooks.OrderBooks {
		if ob.MarketID == marketID {
			return &ob, nil
		}
	}
	
	return nil, fmt.Errorf("market ID %d not found", marketID)
}

// GetMarketBySymbol fetches market information by trading pair symbol
func (c *HTTPClient) GetMarketBySymbol(symbol string) (*OrderBook, error) {
	orderBooks, err := c.GetOrderBooks()
	if err != nil {
		return nil, err
	}
	
	for _, ob := range orderBooks.OrderBooks {
		if ob.Symbol == symbol {
			return &ob, nil
		}
	}
	
	return nil, fmt.Errorf("market symbol %s not found", symbol)
}

// GetActiveMarkets returns only active markets
func (c *HTTPClient) GetActiveMarkets() ([]OrderBook, error) {
	orderBooks, err := c.GetOrderBooks()
	if err != nil {
		return nil, err
	}
	
	activeMarkets := []OrderBook{}
	for _, ob := range orderBooks.OrderBooks {
		if ob.Status == "active" {
			activeMarkets = append(activeMarkets, ob)
		}
	}
	
	return activeMarkets, nil
}

// PrintMarkets prints all markets in a formatted way (for debugging)
func (c *HTTPClient) PrintMarkets() error {
	orderBooks, err := c.GetOrderBooks()
	if err != nil {
		return err
	}
	
	fmt.Println("Lighter Markets:")
	fmt.Println("================")
	
	for _, ob := range orderBooks.OrderBooks {
		fmt.Printf("Market ID: %d\n", ob.MarketID)
		fmt.Printf("Symbol: %s\n", ob.Symbol)
		fmt.Printf("Status: %s\n", ob.Status)
		fmt.Printf("Taker Fee: %s\n", ob.TakerFee)
		fmt.Printf("Maker Fee: %s\n", ob.MakerFee)
		fmt.Printf("Min Base Amount: %s\n", ob.MinBaseAmount)
		fmt.Printf("Min Quote Amount: %s\n", ob.MinQuoteAmount)
		fmt.Println("----------------")
	}
	
	return nil
}