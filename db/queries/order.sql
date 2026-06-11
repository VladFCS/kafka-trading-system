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

-- name: ClaimUnpublishedOutboxEvents :many
WITH candidates AS (
  SELECT order_outbox.id, order_outbox.created_at
  FROM order_outbox
  WHERE order_outbox.published_at IS NULL
    AND (
      order_outbox.locked_at IS NULL
      OR order_outbox.locked_at < sqlc.arg(reclaim_before)
    )
  ORDER BY order_outbox.created_at ASC
  LIMIT sqlc.arg(limit_count)
  FOR UPDATE SKIP LOCKED
),
claimed AS (
  UPDATE order_outbox
  SET locked_at = NOW(),
      locked_by = sqlc.arg(locked_by)
  WHERE order_outbox.id IN (SELECT candidates.id FROM candidates)
  RETURNING *
)
SELECT claimed.*
FROM claimed
JOIN candidates ON candidates.id = claimed.id
ORDER BY candidates.created_at ASC;

-- name: MarkOutboxEventPublished :execrows
UPDATE order_outbox
SET published_at = NOW(),
    last_error = NULL,
    locked_at = NULL,
    locked_by = NULL
WHERE id = sqlc.arg(id)
  AND locked_by = sqlc.arg(locked_by)
  AND published_at IS NULL;

-- name: MarkOutboxEventFailed :execrows
UPDATE order_outbox
SET retry_count = retry_count + 1,
    last_error = sqlc.arg(last_error),
    locked_at = NULL,
    locked_by = NULL
WHERE id = sqlc.arg(id)
  AND locked_by = sqlc.arg(locked_by)
  AND published_at IS NULL;
