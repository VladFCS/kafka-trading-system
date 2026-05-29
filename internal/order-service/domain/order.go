package domain

import (
	"errors"
	"time"
)

var (
	ErrOrderNotFound = errors.New("order not found")
	ErrInvalidOrder  = errors.New("invalid order")
)

type OrderSide string

const (
	OrderSideBuy  OrderSide = "BUY"
	OrderSideSell OrderSide = "SELL"
)

type OrderStatus string

const (
	OrderStatusPending  OrderStatus = "PENDING"
	OrderStatusFilled   OrderStatus = "FILLED"
	OrderStatusCanceled OrderStatus = "CANCELED"
)

type Order struct {
	OrderID                string      `json:"order_id"`
	CustomerID             string      `json:"customer_id"`
	Symbol                 string      `json:"symbol"`
	Side                   OrderSide   `json:"side"`
	PriceCents             int64       `json:"price_cents"`
	QuantityUnits          int64       `json:"quantity_units"`
	RemainingQuantityUnits int64       `json:"remaining_quantity_units"`
	Status                 OrderStatus `json:"status"`
	IdempotencyKey         string      `json:"idempotency_key,omitempty"`
	CanceledAt             *time.Time  `json:"canceled_at,omitempty"`
	CreatedAt              time.Time   `json:"created_at"`
	UpdatedAt              time.Time   `json:"updated_at"`
}
