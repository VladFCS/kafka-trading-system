package domain

import "time"

type OrderBookSide string

const (
	OrderBookSideBid OrderBookSide = "BID"
	OrderBookSideAsk OrderBookSide = "ASK"
)

type LatestPrice struct {
	Symbol     string    `json:"symbol"`
	PriceCents int64     `json:"price_cents"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type Trade struct {
	TradeID       string    `json:"trade_id"`
	SymbolID      string    `json:"symbol_id"`
	Symbol        string    `json:"symbol"`
	PriceCents    int64     `json:"price_cents"`
	QuantityUnits int64     `json:"quantity_units"`
	BuyOrderID    string    `json:"buy_order_id"`
	SellOrderID   string    `json:"sell_order_id"`
	ExecutedAt    time.Time `json:"executed_at"`
}

type OrderBookLevel struct {
	PriceCents    int64 `json:"price_cents"`
	QuantityUnits int64 `json:"quantity_units"`
	OrderCount    int32 `json:"order_count"`
}

type OrderBookSnapshot struct {
	SymbolID  string           `json:"symbol_id"`
	Symbol    string           `json:"symbol"`
	Bids      []OrderBookLevel `json:"bids"`
	Asks      []OrderBookLevel `json:"asks"`
	UpdatedAt time.Time        `json:"updated_at"`
}
