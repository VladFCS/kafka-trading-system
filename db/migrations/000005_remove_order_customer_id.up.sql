DROP INDEX IF EXISTS orders_customer_id_created_at_idx;

ALTER TABLE orders
    DROP COLUMN IF EXISTS customer_id;
