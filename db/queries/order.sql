-- name: CreateOrder :one
INSERT INTO orders (
  order_id,
  symbol,
  side,
  price_cents,
  quantity_units,
  remaining_quantity_units,
  status,
  idempotency_key,
  canceled_at,
  created_at,
  updated_at
) VALUES (
  sqlc.arg(order_id),
  sqlc.arg(symbol),
  sqlc.arg(side),
  sqlc.arg(price_cents),
  sqlc.arg(quantity_units),
  sqlc.arg(quantity_units),
  sqlc.arg(status),
  sqlc.narg(idempotency_key),
  sqlc.narg(canceled_at),
  COALESCE(sqlc.narg(created_at), NOW()),
  NOW()
)
RETURNING *;

-- name: CreateOutboxEvent :exec
INSERT INTO order_outbox (
  id,
  aggregate_type,
  aggregate_id,
  event_type,
  topic,
  partition_key,
  payload
) VALUES (
  sqlc.arg(id),
  sqlc.arg(aggregate_type),
  sqlc.arg(aggregate_id),
  sqlc.arg(event_type),
  sqlc.arg(topic),
  sqlc.arg(partition_key),
  sqlc.arg(payload)
);

-- name: GetOrderByID :one
SELECT *
FROM orders
WHERE order_id = $1;

-- name: GetOrderByIdempotencyKey :one
SELECT *
FROM orders
WHERE idempotency_key = $1;

-- name: UpdateOrderExecution :execrows
UPDATE orders
SET remaining_quantity_units = sqlc.arg(remaining_quantity_units),
    status = sqlc.arg(status),
    updated_at = NOW()
WHERE order_id = sqlc.arg(order_id)
  AND status = sqlc.arg(previous_status);

-- name: CancelOrder :execrows
UPDATE orders
SET status = 'CANCELED',
    canceled_at = NOW(),
    updated_at = NOW()
WHERE order_id = sqlc.arg(order_id)
  AND status = sqlc.arg(previous_status);

-- name: LockUnpublishedOutboxEvents :many
SELECT *
FROM order_outbox
WHERE published_at IS NULL
ORDER BY created_at ASC
LIMIT sqlc.arg(limit_count)
FOR UPDATE SKIP LOCKED;

-- name: MarkOutboxEventPublished :exec
UPDATE order_outbox
SET published_at = NOW(),
    last_error = NULL
WHERE id = sqlc.arg(id);

-- name: MarkOutboxEventFailed :exec
UPDATE order_outbox
SET retry_count = retry_count + 1,
    last_error = sqlc.arg(last_error)
WHERE id = sqlc.arg(id);
