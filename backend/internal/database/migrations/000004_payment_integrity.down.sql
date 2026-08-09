DROP TABLE IF EXISTS payment_webhook_events;
DROP INDEX IF EXISTS idx_payments_transaction_id;
DROP INDEX IF EXISTS idx_payments_user_id;
DROP INDEX IF EXISTS idx_payments_order_id;

ALTER TABLE payments
    DROP COLUMN IF EXISTS transaction_id,
    DROP COLUMN IF EXISTS redirect_url,
    DROP COLUMN IF EXISTS order_id,
    DROP COLUMN IF EXISTS provider,
    DROP COLUMN IF EXISTS user_id;
