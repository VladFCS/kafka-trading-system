ALTER TABLE order_outbox
    ADD COLUMN IF NOT EXISTS locked_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS locked_by TEXT;

CREATE INDEX IF NOT EXISTS order_outbox_unpublished_locked_at_created_at_idx
    ON order_outbox (locked_at, created_at ASC)
    WHERE published_at IS NULL;
