package repository

import (
	"context"

	"github.com/vladfc/kafka-trading-system/internal/order-service/domain"
)

type OrderRepository interface {
	CreateOrder(ctx context.Context, order *domain.Order) (*domain.Order, error)
	GetOrderByID(ctx context.Context, id string) (*domain.Order, error)
	GetListOrdersByCustomerID(ctx context.Context, customerID string) ([]*domain.Order, error)
	UpdateOrderStatus(ctx context.Context, id string, status domain.OrderStatus) error
}
