package shared

const (
	OrderTopic = "orders"

	OrderCreatedEventType  = "orders.created"
	OrderCanceledEventType = "orders.cancelled"
	OrderUpdatedEventType  = "orders.updated"
)

type OrderCreatedEvent struct {
	EventID                string `json:"event_id"`
	EventType              string `json:"event_type"`
	OrderID                string `json:"order_id"`
	CustomerID             string `json:"customer_id"`
	Symbol                 string `json:"symbol"`
	Side                   string `json:"side"`
	PriceCents             int64  `json:"price_cents"`
	QuantityUnits          int64  `json:"quantity_units"`
	RemainingQuantityUnits int64  `json:"remaining_quantity_units"`
	Status                 string `json:"status"`
	OccurredAt             string `json:"occurred_at"`
}

type OrderCanceledEvent struct {
	EventID    string `json:"event_id"`
	EventType  string `json:"event_type"`
	OrderID    string `json:"order_id"`
	OccurredAt string `json:"occurred_at"`
}

type OrderUpdatedEvent struct {
	EventID                string `json:"event_id"`
	EventType              string `json:"event_type"`
	OrderID                string `json:"order_id"`
	Symbol                 string `json:"symbol"`
	PreviousStatus         string `json:"previous_status"`
	NewStatus              string `json:"new_status"`
	QuantityUnits          int64  `json:"quantity_units"`
	RemainingQuantityUnits int64  `json:"remaining_quantity_units"`
	OccurredAt             string `json:"occurred_at"`
}
