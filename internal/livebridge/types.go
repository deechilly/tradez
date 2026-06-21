package livebridge

import "encoding/json"

type Operation string

const (
	OpGameState  Operation = "get_game_state"
	OpListStocks Operation = "list_stocks"
	OpGetStock   Operation = "get_stock"
	OpGetNews    Operation = "get_news"
	OpPortfolio  Operation = "get_portfolio"
	OpOrders     Operation = "get_orders"
	OpInfluences Operation = "get_influences"
	OpPlaceOrder Operation = "place_order"
	OpCancel     Operation = "cancel_order"
	OpSave       Operation = "save_game"
)

type Descriptor struct {
	URL        string `json:"url"`
	Token      string `json:"token"`
	PID        int    `json:"pid"`
	ActiveSlot int    `json:"active_slot"`
}

type RequestMsg struct {
	Operation Operation
	Payload   json.RawMessage
	Reply     chan Response
}

type Response struct {
	OK    bool
	Data  any
	Error string
}

type StockRequest struct {
	Symbol string `json:"symbol" jsonschema:"ticker symbol to inspect"`
}

type PlaceOrderRequest struct {
	Symbol     string  `json:"symbol" jsonschema:"ticker symbol to trade"`
	Type       string  `json:"type" jsonschema:"order type: market_buy, market_sell, limit_buy, limit_sell, short_sell, cover_short"`
	Shares     int     `json:"shares" jsonschema:"positive integer number of shares"`
	LimitPrice float64 `json:"limit_price,omitempty" jsonschema:"required for limit_buy and limit_sell"`
}

type CancelOrderRequest struct {
	ID int `json:"id" jsonschema:"pending order id to cancel"`
}
