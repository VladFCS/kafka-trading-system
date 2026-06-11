package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vladfc/kafka-trading-system/internal/order-service/domain"
	"github.com/vladfc/kafka-trading-system/internal/order-service/events"
	orderdb "github.com/vladfc/kafka-trading-system/internal/order-service/repository/sqlc"
)

type PostgresRepository struct {
	pool    *pgxpool.Pool
	queries *orderdb.Queries
}

var _ events.OutboxRepository = (*PostgresRepository)(nil)

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
		if order.IdempotencyKey != "" && isUniqueViolation(err) {
			_ = tx.Rollback(ctx)
			existingOrder, getErr := r.GetOrderByIdempotencyKey(ctx, order.IdempotencyKey)
			if getErr != nil {
				return domain.Order{}, fmt.Errorf("get existing order by idempotency key after duplicate create: %w", getErr)
			}
			return existingOrder, nil
		}
		return domain.Order{}, err
	}

	mappedOrder, err := mapDBOrder(row)
	if err != nil {
		return domain.Order{}, err
	}

	if err := insertOrderCreatedOutboxEvent(ctx, qtx, mappedOrder); err != nil {
		return domain.Order{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Order{}, err
	}

	return mappedOrder, nil
}

func (r *PostgresRepository) GetOrderByIdempotencyKey(ctx context.Context, idempotencyKey string) (domain.Order, error) {
	if idempotencyKey == "" {
		return domain.Order{}, domain.ErrInvalidOrder
	}

	row, err := r.queries.GetOrderByIdempotencyKey(ctx, pgtype.Text{
		String: idempotencyKey,
		Valid:  true,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Order{}, domain.ErrOrderNotFound
	}
	if err != nil {
		return domain.Order{}, fmt.Errorf("get order by idempotency key: %w", err)
	}

	mappedOrder, err := mapDBOrder(row)
	if err != nil {
		return domain.Order{}, err
	}

	return mappedOrder, nil
}

func (r *PostgresRepository) GetOrderByID(ctx context.Context, orderID string) (domain.Order, error) {
	if orderID == "" {
		return domain.Order{}, domain.ErrMissingOrderID
	}
	row, err := r.queries.GetOrderByID(ctx, orderID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Order{}, domain.ErrOrderNotFound
	}
	if err != nil {
		return domain.Order{}, err
	}

	mappedOrder, err := mapDBOrder(row)
	if err != nil {
		return domain.Order{}, err
	}

	return mappedOrder, nil
}

func (r *PostgresRepository) UpdateOrderExecution(ctx context.Context, order domain.Order) error {
	if err := validateOrder(order); err != nil {
		return err
	}
	if order.Status != domain.OrderStatusPending && order.Status != domain.OrderStatusFilled {
		return domain.ErrInvalidOrderStatus
	}
	if order.Status == domain.OrderStatusFilled && order.RemainingQuantityUnits != 0 {
		return domain.ErrInvalidRemainingQuantityUnits
	}
	if order.Status == domain.OrderStatusPending && order.RemainingQuantityUnits == 0 {
		return domain.ErrInvalidRemainingQuantityUnits
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("update order execution transaction begin: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	qtx := r.queries.WithTx(tx)

	rowsAffected, err := qtx.UpdateOrderExecution(ctx, toUpdateOrderExecutionParams(order))
	if err != nil {
		return fmt.Errorf("update order execution: %w", err)
	}
	if rowsAffected == 0 {
		currentOrderRow, err := qtx.GetOrderByID(ctx, order.OrderID)
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrOrderNotFound
		}
		if err != nil {
			return fmt.Errorf("get order by ID: %w", err)
		}
		currentOrder, err := mapDBOrder(currentOrderRow)
		if err != nil {
			return fmt.Errorf("map order: %w", err)
		}
		if currentOrder.Status != domain.OrderStatusPending {
			return domain.ErrOrderTerminal
		}

		return domain.ErrOrderUpdateConflict
	}

	if order.Status == domain.OrderStatusFilled {
		if err := insertOrderUpdatedOutboxEvent(ctx, qtx, order, domain.OrderStatusPending); err != nil {
			return fmt.Errorf("insert order filled outbox event: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("update order execution transaction commit: %w", err)
	}
	return nil
}

func (r *PostgresRepository) CancelOrder(ctx context.Context, order domain.Order) error {
	if err := validateOrder(order); err != nil {
		return err
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("cancel order transaction begin: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	qtx := r.queries.WithTx(tx)

	if order.Status != domain.OrderStatusCanceled {
		return domain.ErrInvalidOrderStatus
	}

	rowsAffected, err := qtx.CancelOrder(ctx, orderdb.CancelOrderParams{
		OrderID:        order.OrderID,
		PreviousStatus: string(domain.OrderStatusPending),
	})
	if err != nil {
		return fmt.Errorf("cancel order: %w", err)
	}
	if rowsAffected == 0 {
		currentOrderRow, err := qtx.GetOrderByID(ctx, order.OrderID)
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrOrderNotFound
		}
		if err != nil {
			return fmt.Errorf("get order by ID: %w", err)
		}
		currentOrder, err := mapDBOrder(currentOrderRow)
		if err != nil {
			return fmt.Errorf("map order: %w", err)
		}
		if currentOrder.Status != domain.OrderStatusPending {
			return domain.ErrOrderTerminal
		}

		return domain.ErrOrderUpdateConflict
	}

	if err := insertOrderUpdatedOutboxEvent(ctx, qtx, order, domain.OrderStatusPending); err != nil {
		return fmt.Errorf("insert order canceled outbox event: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("cancel order transaction commit: %w", err)
	}
	return nil
}

func (r *PostgresRepository) ClaimUnpublished(ctx context.Context, limit int32, lockedBy string, reclaimBefore time.Time) ([]events.OutboxMessage, error) {
	if limit <= 0 {
		return nil, domain.ErrInvalidOrder
	}
	if lockedBy == "" {
		return nil, domain.ErrInvalidOrder
	}
	if reclaimBefore.IsZero() {
		return nil, domain.ErrInvalidOrder
	}

	rows, err := r.queries.ClaimUnpublishedOutboxEvents(ctx, orderdb.ClaimUnpublishedOutboxEventsParams{
		LimitCount: limit,
		LockedBy: pgtype.Text{
			String: lockedBy,
			Valid:  true,
		},
		ReclaimBefore: pgtype.Timestamptz{
			Time:  reclaimBefore,
			Valid: true,
		},
	})
	if err != nil {
		return nil, err
	}

	messages := make([]events.OutboxMessage, 0, len(rows))
	for _, row := range rows {
		message, err := mapDBOutboxMessage(row)
		if err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}

	return messages, nil
}

func (r *PostgresRepository) MarkPublished(ctx context.Context, id string, lockedBy string) error {
	if id == "" {
		return domain.ErrInvalidOrder
	}
	if lockedBy == "" {
		return domain.ErrInvalidOrder
	}

	_, err := r.queries.MarkOutboxEventPublished(ctx, orderdb.MarkOutboxEventPublishedParams{
		ID: id,
		LockedBy: pgtype.Text{
			String: lockedBy,
			Valid:  true,
		},
	})
	return err
}

func (r *PostgresRepository) MarkFailed(ctx context.Context, id string, lockedBy string, publishErr error) error {
	if id == "" {
		return domain.ErrInvalidOrder
	}
	if lockedBy == "" {
		return domain.ErrInvalidOrder
	}

	lastError := "unknown publish error"
	if publishErr != nil {
		lastError = publishErr.Error()
	}

	_, err := r.queries.MarkOutboxEventFailed(ctx, orderdb.MarkOutboxEventFailedParams{
		ID: id,
		LockedBy: pgtype.Text{
			String: lockedBy,
			Valid:  true,
		},
		LastError: pgtype.Text{
			String: lastError,
			Valid:  true,
		},
	})
	return err
}

func validateOrder(order domain.Order) error {
	if order.Symbol == "" {
		return domain.ErrMissingSymbol
	}
	if order.Side != domain.OrderSideBuy && order.Side != domain.OrderSideSell {
		return domain.ErrInvalidOrderSide
	}
	if order.PriceCents <= 0 {
		return domain.ErrInvalidPriceCents
	}
	if order.QuantityUnits <= 0 {
		return domain.ErrInvalidQuantityUnits
	}
	if order.RemainingQuantityUnits < 0 {
		return domain.ErrInvalidRemainingQuantityUnits
	}
	if order.RemainingQuantityUnits > order.QuantityUnits {
		return domain.ErrRemainingQuantityExceedsQuantity
	}
	if order.OrderID == "" {
		return domain.ErrMissingOrderID
	}
	if order.Status == "" {
		return domain.ErrMissingOrderStatus
	}
	if order.Status != domain.OrderStatusPending &&
		order.Status != domain.OrderStatusFilled &&
		order.Status != domain.OrderStatusCanceled {
		return domain.ErrInvalidOrderStatus
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

func toUpdateOrderExecutionParams(order domain.Order) orderdb.UpdateOrderExecutionParams {
	return orderdb.UpdateOrderExecutionParams{
		OrderID:                order.OrderID,
		RemainingQuantityUnits: order.RemainingQuantityUnits,
		Status:                 string(order.Status),
		PreviousStatus:         string(domain.OrderStatusPending),
	}
}

func toCreateOutboxEventParams(message *events.OutboxMessage) orderdb.CreateOutboxEventParams {
	if message == nil {
		return orderdb.CreateOutboxEventParams{}
	}

	return orderdb.CreateOutboxEventParams{
		ID:            message.ID,
		AggregateType: message.AggregateType,
		AggregateID:   message.AggregateID,
		EventType:     message.EventType,
		Topic:         message.Topic,
		PartitionKey:  message.Key,
		Payload:       message.Payload,
	}
}

func insertOrderCreatedOutboxEvent(ctx context.Context, q *orderdb.Queries, order domain.Order) error {
	message, err := events.NewOrderCreatedOutboxMessage(order)
	if err != nil {
		return fmt.Errorf("create order outbox event in postgres: %w", err)
	}

	if err := q.CreateOutboxEvent(ctx, toCreateOutboxEventParams(message)); err != nil {
		return fmt.Errorf("create order outbox event in postgres: %w", err)
	}

	return nil
}

func insertOrderUpdatedOutboxEvent(ctx context.Context, q *orderdb.Queries, order domain.Order, previousStatus domain.OrderStatus) error {
	message, err := events.NewOrderUpdatedOutboxMessage(order, previousStatus)
	if err != nil {
		return fmt.Errorf("create order updated outbox event in postgres: %w", err)
	}

	if err := q.CreateOutboxEvent(ctx, toCreateOutboxEventParams(message)); err != nil {
		return fmt.Errorf("create order updated outbox event in postgres: %w", err)
	}
	return nil
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

func mapDBOutboxMessage(row orderdb.ClaimUnpublishedOutboxEventsRow) (events.OutboxMessage, error) {
	createdAt, err := timestamptzToTime(row.CreatedAt)
	if err != nil {
		return events.OutboxMessage{}, err
	}

	return events.OutboxMessage{
		ID:            row.ID,
		AggregateType: row.AggregateType,
		AggregateID:   row.AggregateID,
		EventType:     row.EventType,
		Topic:         row.Topic,
		Key:           row.PartitionKey,
		Payload:       row.Payload,
		CreatedAt:     createdAt,
	}, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
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
