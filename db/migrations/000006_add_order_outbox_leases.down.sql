DROP INDEX IF EXISTS order_outbox_unpublished_locked_at_created_at_idx;

ALTER TABLE order_outbox
    DROP COLUMN IF EXISTS locked_by,
    DROP COLUMN IF EXISTS locked_at;
