ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS customer_id TEXT NOT NULL DEFAULT 'anonymous';

ALTER TABLE orders
    ALTER COLUMN customer_id DROP DEFAULT;

CREATE INDEX IF NOT EXISTS orders_customer_id_created_at_idx
    ON orders (customer_id, created_at DESC);
