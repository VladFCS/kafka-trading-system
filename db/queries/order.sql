-- name: CreateOrder :one
INSERT INTO orders (
  order_id,
  customer_id,
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
  sqlc.arg(customer_id),
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

-- name: GetOrderByID :one
SELECT *
FROM orders
WHERE order_id = $1;

-- name: GetListOrdersByCustomerID :many
SELECT *
FROM orders
WHERE customer_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: UpdateOrderExecution :execrows
UPDATE orders
SET remaining_quantity_units = sqlc.arg(remaining_quantity_units),
    status = sqlc.arg(status),
    updated_at = NOW()
WHERE order_id = sqlc.arg(order_id)
  AND status = 'PENDING';
