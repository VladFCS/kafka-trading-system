package events

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/vladfc/kafka-trading-system/internal/order-service/domain"
	shared "github.com/vladfc/kafka-trading-system/internal/shared/events"
)

type OutboxMessage struct {
	ID            string
	AggregateType string
	AggregateID   string
	EventType     string
	Topic         string
	Key           string
	Payload       []byte
	CreatedAt     time.Time
}

func NewOrderCreatedOutboxMessage(order domain.Order) (*OutboxMessage, error) {
	if strings.TrimSpace(order.OrderID) == "" {
		return nil, fmt.Errorf("order id is required")
	}
	if strings.TrimSpace(order.Symbol) == "" {
		return nil, fmt.Errorf("order symbol is required")
	}

	eventID := newEventID()
	occurredAt := time.Now().UTC()

	event := shared.OrderCreatedEvent{
		EventID:                eventID,
		EventType:              shared.OrderCreatedEventType,
		OrderID:                order.OrderID,
		CustomerID:             order.CustomerID,
		Symbol:                 order.Symbol,
		Side:                   string(order.Side),
		PriceCents:             order.PriceCents,
		QuantityUnits:          order.QuantityUnits,
		RemainingQuantityUnits: order.RemainingQuantityUnits,
		Status:                 string(order.Status),
		OccurredAt:             occurredAt.Format(time.RFC3339Nano),
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("marshal order created event: %w", err)
	}

	return &OutboxMessage{
		ID:            eventID,
		AggregateType: "order",
		AggregateID:   order.OrderID,
		EventType:     shared.OrderCreatedEventType,
		Topic:         shared.OrderTopic,
		Key:           order.Symbol,
		Payload:       payload,
		CreatedAt:     occurredAt,
	}, nil
}

func NewOrderCanceledOutboxMessage(order domain.Order) (*OutboxMessage, error) {
	if strings.TrimSpace(order.OrderID) == "" {
		return nil, fmt.Errorf("order id is required")
	}
	if strings.TrimSpace(order.Symbol) == "" {
		return nil, fmt.Errorf("order symbol is required")
	}

	eventID := newEventID()
	occurredAt := time.Now().UTC()

	event := shared.OrderCanceledEvent{
		EventID:    eventID,
		EventType:  shared.OrderCanceledEventType,
		OrderID:    order.OrderID,
		OccurredAt: occurredAt.Format(time.RFC3339Nano),
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("marshal order canceled event: %w", err)
	}

	return &OutboxMessage{
		ID:            eventID,
		AggregateType: "order",
		AggregateID:   order.OrderID,
		EventType:     shared.OrderCanceledEventType,
		Topic:         shared.OrderTopic,
		Key:           order.Symbol,
		Payload:       payload,
		CreatedAt:     occurredAt,
	}, nil
}

func newEventID() string {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return time.Now().UTC().Format("20060102150405.000000000")
	}
	return hex.EncodeToString(raw[:])
}
