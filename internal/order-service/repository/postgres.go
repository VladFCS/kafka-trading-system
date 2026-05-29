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

func (r *PostgresRepository) CreateOrder(ctx context.Context, order domain.Order) (domain.Order, error) {
	if err := validateOrder(order); err != nil {
		return domain.Order{}, err
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Order{}, err
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	qtx := r.queries.WithTx(tx)

	row, err := qtx.CreateOrder(ctx, toCreateOrderParams(order))
	if err != nil {
		return domain.Order{}, err
	}

	mappedOrder, err := mapDBOrder(row)
	if err != nil {
		return domain.Order{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Order{}, err
	}

	return mappedOrder, nil
}

func validateOrder(order domain.Order) error {
	if order.CustomerID == "" {
		return domain.ErrInvalidOrder
	}
	if order.Symbol == "" {
		return domain.ErrInvalidOrder
	}
	if order.Side != domain.OrderSideBuy && order.Side != domain.OrderSideSell {
		return domain.ErrInvalidOrder
	}
	if order.PriceCents <= 0 || order.QuantityUnits <= 0 {
		return domain.ErrInvalidOrder
	}
	if order.RemainingQuantityUnits < 0 || order.RemainingQuantityUnits > order.QuantityUnits {
		return domain.ErrInvalidOrder
	}
	if order.OrderID == "" {
		return domain.ErrInvalidOrder
	}
	if order.Status == "" {
		return domain.ErrInvalidOrder
	}
	return nil
}

func toCreateOrderParams(order domain.Order) orderdb.CreateOrderParams {
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
		OrderID:        order.OrderID,
		CustomerID:     order.CustomerID,
		Symbol:         order.Symbol,
		Side:           string(order.Side),
		PriceCents:     order.PriceCents,
		QuantityUnits:  order.QuantityUnits,
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
		OrderID:                order.OrderID,
		CustomerID:             order.CustomerID,
		Symbol:                 order.Symbol,
		Side:                   domain.OrderSide(order.Side),
		PriceCents:             order.PriceCents,
		QuantityUnits:          order.QuantityUnits,
		RemainingQuantityUnits: order.RemainingQuantityUnits,
		Status:                 domain.OrderStatus(order.Status),
		IdempotencyKey:         order.IdempotencyKey.String,
		CanceledAt:             canceledAt,
		CreatedAt:              createdAt,
		UpdatedAt:              updatedAt,
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
