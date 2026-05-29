package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vladfc/kafka-trading-system/internal/order-service/domain"
	orderdb "github.com/vladfc/kafka-trading-system/internal/order-service/repository/sqlc"
)

type PostgresRepository struct {
	pool    *pgxpool.Pool
	queries *orderdb.Queries
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{
		pool:    pool,
		queries: orderdb.New(pool),
	}
}

func (r *PostgresRepository) CreateOrder(ctx context.Context, order *domain.Order) (domain.Order, error) {
	if err := validateOrder(order); err != nil {
		return domain.Order{}, err
	}
	row, err := r.queries.CreateOrder(ctx, toCreateOrderParams(order))
	if err != nil {
		return domain.Order{}, err
	}

	mappedOrder, err := mapDBOrder(row)
	if err != nil {
		return domain.Order{}, err
	}

	return mappedOrder, nil
}

func validateOrder(order *domain.Order) error {
	if order == nil {
		return domain.ErrInvalidOrder
	}
	if order.CustomerID == "" {
		return domain.ErrInvalidOrder
	}
	if order.Symbol == "" {
		return domain.ErrInvalidOrder
	}
	if order.Side != domain.OrderSideBuy && order.Side != domain.OrderSideSell {
		return domain.ErrInvalidOrder
	}
	if !isPositiveNumeric(order.Price) || !isPositiveNumeric(order.Quantity) {
		return domain.ErrInvalidOrder
	}
	if order.ID == "" {
		return domain.ErrInvalidOrder
	}
	if order.Status == "" {
		return domain.ErrInvalidOrder
	}
	return nil
}

func toCreateOrderParams(order *domain.Order) orderdb.CreateOrderParams {
	idempotencyKey := pgtype.Text{}
	if order.IdempotencyKey != "" {
		idempotencyKey = pgtype.Text{String: order.IdempotencyKey, Valid: true}
	}

	canceledAt := pgtype.Timestamptz{}
	if order.CanceledAt != nil {
		canceledAt = pgtype.Timestamptz{Time: *order.CanceledAt, Valid: true}
	}

	createdAt := pgtype.Timestamptz{}
	if !order.CreatedAt.IsZero() {
		createdAt = pgtype.Timestamptz{Time: order.CreatedAt, Valid: true}
	}

	return orderdb.CreateOrderParams{
		ID:             order.ID,
		CustomerID:     order.CustomerID,
		Symbol:         order.Symbol,
		Side:           string(order.Side),
		Price:          order.Price,
		Quantity:       order.Quantity,
		Status:         string(order.Status),
		IdempotencyKey: idempotencyKey,
		CanceledAt:     canceledAt,
		CreatedAt:      createdAt,
	}
}

func mapDBOrder(order orderdb.Order) (domain.Order, error) {
	createdAt, err := timestamptzToTime(order.CreatedAt)
	if err != nil {
		return domain.Order{}, err
	}

	updatedAt, err := timestamptzToTime(order.UpdatedAt)
	if err != nil {
		return domain.Order{}, err
	}

	var canceledAt *time.Time
	if order.CanceledAt.Valid {
		if order.CanceledAt.InfinityModifier != 0 {
			return domain.Order{}, errors.New("invalid canceled_at infinity value")
		}
		t := order.CanceledAt.Time
		canceledAt = &t
	}

	return domain.Order{
		ID:                order.ID,
		CustomerID:        order.CustomerID,
		Symbol:            order.Symbol,
		Side:              domain.OrderSide(order.Side),
		Price:             order.Price,
		Quantity:          order.Quantity,
		RemainingQuantity: order.RemainingQuantity,
		Status:            domain.OrderStatus(order.Status),
		IdempotencyKey:    order.IdempotencyKey.String,
		CanceledAt:        canceledAt,
		CreatedAt:         createdAt,
		UpdatedAt:         updatedAt,
	}, nil
}

func timestamptzToTime(value pgtype.Timestamptz) (time.Time, error) {
	if !value.Valid {
		return time.Time{}, errors.New("invalid timestamptz value")
	}
	if value.InfinityModifier != 0 {
		return time.Time{}, errors.New("invalid timestamptz infinity value")
	}
	return value.Time, nil
}

func isPositiveNumeric(value pgtype.Numeric) bool {
	floatValue, err := value.Float64Value()
	if err != nil || !floatValue.Valid {
		return false
	}
	return floatValue.Float64 > 0
}
