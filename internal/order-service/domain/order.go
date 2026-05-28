package domain

import (
	"time"

	orderv1 "github.com/vladfc/kafka-trading-system/gen/order/v1"
)

var (
	ErrOrderNotFound = "order not found"
	ErrInvalidOrder = "invalid order"
)

type Order struct {
	ID        string    `json:"id"`
	CustomerID string    `json:"customer_id"`
	Symbol 	string    `json:"symbol"`
	Quantity   int32     `json:"quantity"`
	Price      float64   `json:"price"`
	Status     orderv1.OrderStatus `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}