-- name: CreateOrder :one
INSERT INTO orders (
  order_id,
  customer_id,
  symbol,
  side,
  price,
  quantity,
  remaining_quantity,
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
  sqlc.arg(price),
  sqlc.arg(quantity),
  sqlc.arg(quantity),
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

-- name: UpdateOrderStatus :exec
UPDATE orders
SET status = $2,
    updated_at = NOW()
WHERE order_id = $1;
