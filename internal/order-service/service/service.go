package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/vladfc/kafka-trading-system/internal/order-service/domain"
	"github.com/vladfc/kafka-trading-system/internal/order-service/repository"
)

type OrderService struct {
	repository repository.OrderRepository
}

func NewOrderService(repository repository.OrderRepository) *OrderService {
	return &OrderService{
		repository: repository,
	}
}

func (s *OrderService) CreateOrder(ctx context.Context, order domain.Order) (domain.Order, error) {
	normalizedOrder, err := normalizeCreateOrder(order)
	if err != nil {
		return domain.Order{}, err
	}

	createdOrder, err := s.repository.CreateOrder(ctx, normalizedOrder)
	if err != nil {
		return domain.Order{}, err
	}
	return createdOrder, nil
}

func normalizeCreateOrder(order domain.Order) (domain.Order, error) {
	if order.CustomerID == "" {
		return domain.Order{}, domain.ErrInvalidOrder
	}
	if order.Symbol == "" {
		return domain.Order{}, domain.ErrInvalidOrder
	}
	if order.Side != domain.OrderSideBuy && order.Side != domain.OrderSideSell {
		return domain.Order{}, domain.ErrInvalidOrder
	}
	if order.PriceCents <= 0 || order.QuantityUnits <= 0 {
		return domain.Order{}, domain.ErrInvalidOrder
	}

	if order.OrderID == "" {
		orderID, err := newOrderID()
		if err != nil {
			return domain.Order{}, err
		}
		order.OrderID = orderID
	}

	if order.Status == "" {
		order.Status = domain.OrderStatusPending
	}
	if order.Status != domain.OrderStatusPending {
		return domain.Order{}, domain.ErrInvalidOrder
	}

	if order.CanceledAt != nil {
		return domain.Order{}, domain.ErrInvalidOrder
	}

	if order.RemainingQuantityUnits == 0 {
		order.RemainingQuantityUnits = order.QuantityUnits
	}

	if order.RemainingQuantityUnits <= 0 {
		return domain.Order{}, domain.ErrInvalidOrder
	}

	if order.RemainingQuantityUnits > order.QuantityUnits {
		return domain.Order{}, domain.ErrInvalidOrder
	}

	now := time.Now().UTC()
	if order.CreatedAt.IsZero() {
		order.CreatedAt = now
	}
	order.UpdatedAt = now

	return order, nil
}

func newOrderID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw[:]), nil
}
