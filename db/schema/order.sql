CREATE TABLE IF NOT EXISTS orders (
    id TEXT PRIMARY KEY,
    customer_id TEXT NOT NULL,
    symbol TEXT NOT NULL,
    side TEXT NOT NULL CHECK (side IN ('BUY', 'SELL')),
    price NUMERIC(20,8) NOT NULL CHECK (price > 0),
    quantity NUMERIC(20,8) NOT NULL CHECK (quantity > 0),
    remaining_quantity NUMERIC(20,8) NOT NULL CHECK (remaining_quantity >= 0 AND remaining_quantity <= quantity),
    status TEXT NOT NULL CHECK (status IN ('PENDING', 'FILLED', 'CANCELED')),
    idempotency_key TEXT,
    canceled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS orders_idempotency_key_uidx
    ON orders (idempotency_key)
    WHERE idempotency_key IS NOT NULL;

CREATE INDEX IF NOT EXISTS orders_customer_id_created_at_idx
    ON orders (customer_id, created_at DESC);

CREATE INDEX IF NOT EXISTS orders_symbol_status_created_at_idx
    ON orders (symbol, status, created_at ASC);

CREATE TABLE IF NOT EXISTS order_outbox (
    id TEXT PRIMARY KEY,
    aggregate_type TEXT NOT NULL,
    aggregate_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    topic TEXT NOT NULL,
    partition_key TEXT NOT NULL,
    payload JSONB NOT NULL,
    retry_count INTEGER NOT NULL DEFAULT 0 CHECK (retry_count >= 0),
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS order_outbox_unpublished_created_at_idx
    ON order_outbox (created_at ASC)
    WHERE published_at IS NULL;

CREATE INDEX IF NOT EXISTS order_outbox_aggregate_id_idx
    ON order_outbox (aggregate_id);
