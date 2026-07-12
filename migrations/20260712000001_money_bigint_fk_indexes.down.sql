DROP INDEX IF EXISTS idx_transactions_to_user_id;
DROP INDEX IF EXISTS idx_transactions_from_user_id;

ALTER TABLE transactions
    DROP CONSTRAINT IF EXISTS fk_transactions_to_user,
    DROP CONSTRAINT IF EXISTS fk_transactions_from_user;

DROP INDEX IF EXISTS idx_orders_user_id;

ALTER TABLE orders
    DROP CONSTRAINT IF EXISTS fk_orders_user;

ALTER TABLE transactions
    ALTER COLUMN amount TYPE DECIMAL(15, 2) USING amount / 100.0;

ALTER TABLE users
    ALTER COLUMN balance DROP DEFAULT,
    ALTER COLUMN balance TYPE DECIMAL(15, 2) USING balance / 100.0;
