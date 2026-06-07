-- name: UpsertLatestPrice :exec
INSERT INTO market_latest_prices (
  symbol,
  price_cents,
  updated_at
) VALUES (
  sqlc.arg(symbol),
  sqlc.arg(price_cents),
  COALESCE(sqlc.narg(updated_at), NOW())
)
ON CONFLICT (symbol) DO UPDATE
SET price_cents = EXCLUDED.price_cents,
    updated_at = EXCLUDED.updated_at;

-- name: GetLatestPrice :one
SELECT *
FROM market_latest_prices
WHERE symbol = $1;

-- name: InsertTrade :exec
INSERT INTO market_trades (
  trade_id,
  symbol,
  price_cents,
  quantity_units,
  buy_order_id,
  sell_order_id,
  executed_at
) VALUES (
  sqlc.arg(trade_id),
  sqlc.arg(symbol),
  sqlc.arg(price_cents),
  sqlc.arg(quantity_units),
  sqlc.arg(buy_order_id),
  sqlc.arg(sell_order_id),
  sqlc.arg(executed_at)
);

-- name: GetRecentTrades :many
SELECT *
FROM market_trades
WHERE symbol = sqlc.arg(symbol)
ORDER BY executed_at DESC
LIMIT sqlc.arg(limit_count);

-- name: UpsertOrderBookLevel :exec
INSERT INTO market_order_book_levels (
  symbol,
  side,
  price_cents,
  quantity_units,
  order_count,
  updated_at
) VALUES (
  sqlc.arg(symbol),
  sqlc.arg(side),
  sqlc.arg(price_cents),
  sqlc.arg(quantity_units),
  sqlc.arg(order_count),
  COALESCE(sqlc.narg(updated_at), NOW())
)
ON CONFLICT (symbol, side, price_cents) DO UPDATE
SET quantity_units = EXCLUDED.quantity_units,
    order_count = EXCLUDED.order_count,
    updated_at = EXCLUDED.updated_at;

-- name: DeleteOrderBookLevel :exec
DELETE FROM market_order_book_levels
WHERE symbol = sqlc.arg(symbol)
  AND side = sqlc.arg(side)
  AND price_cents = sqlc.arg(price_cents);

-- name: GetOrderBookBids :many
SELECT *
FROM market_order_book_levels
WHERE symbol = sqlc.arg(symbol)
  AND side = 'BID'
  AND quantity_units > 0
  AND order_count > 0
ORDER BY price_cents DESC
LIMIT sqlc.arg(depth);

-- name: GetOrderBookAsks :many
SELECT *
FROM market_order_book_levels
WHERE symbol = sqlc.arg(symbol)
  AND side = 'ASK'
  AND quantity_units > 0
  AND order_count > 0
ORDER BY price_cents ASC
LIMIT sqlc.arg(depth);
