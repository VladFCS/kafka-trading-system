package domain

import (
	"errors"
	"time"
)

var (
	ErrOrderNotFound                    = errors.New("order not found")
	ErrInvalidOrder                     = errors.New("invalid order")
	ErrMissingCustomerID                = errors.New("missing customer id")
	ErrMissingSymbol                    = errors.New("missing symbol")
	ErrInvalidOrderSide                 = errors.New("invalid order side")
	ErrInvalidPriceCents                = errors.New("invalid price cents")
	ErrInvalidQuantityUnits             = errors.New("invalid quantity units")
	ErrInvalidRemainingQuantityUnits    = errors.New("invalid remaining quantity units")
	ErrRemainingQuantityExceedsQuantity = errors.New("remaining quantity exceeds quantity")
	ErrMissingOrderID                   = errors.New("missing order id")
	ErrMissingOrderStatus               = errors.New("missing order status")
	ErrInvalidOrderStatus               = errors.New("invalid order status")
	ErrCanceledOrderOnCreate            = errors.New("canceled order cannot be created")
	ErrOrderTerminal                    = errors.New("order is already terminal")
	ErrOrderUpdateConflict              = errors.New("order update conflict")
	ErrInvalidFillQuantity              = errors.New("invalid fill quantity")
	ErrFillQuantityExceedsRemaining     = errors.New("fill quantity exceeds remaining quantity")
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

func (o *Order) ApplyFill(fillQtyUnits int64) error {
	if o.Status != OrderStatusPending {
		return ErrOrderTerminal
	}
	if fillQtyUnits <= 0 {
		return ErrInvalidFillQuantity
	}
	if fillQtyUnits > o.RemainingQuantityUnits {
		return ErrFillQuantityExceedsRemaining
	}

	o.RemainingQuantityUnits -= fillQtyUnits
	if o.RemainingQuantityUnits == 0 {
		o.Status = OrderStatusFilled
	}

	return nil
}

func (o *Order) Cancel() error {
	if o.Status != OrderStatusPending {
		return ErrOrderTerminal
	}

	o.Status = OrderStatusCanceled
	return nil
}
