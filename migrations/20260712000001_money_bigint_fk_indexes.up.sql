-- Деньги переводятся в BIGINT (минимальные единицы валюты, 100 центов = 1$):
-- целочисленная арифметика без ошибок округления двоичной запятой.
ALTER TABLE users
    ALTER COLUMN balance TYPE BIGINT USING ROUND(balance * 100)::BIGINT,
    ALTER COLUMN balance SET DEFAULT 0;

ALTER TABLE transactions
    ALTER COLUMN amount TYPE BIGINT USING ROUND(amount * 100)::BIGINT;

-- Ссылочная целостность и индексы под FK-поиски.
ALTER TABLE orders
    ADD CONSTRAINT fk_orders_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_orders_user_id ON orders (user_id);

ALTER TABLE transactions
    ADD CONSTRAINT fk_transactions_from_user FOREIGN KEY (from_user_id) REFERENCES users (id) ON DELETE CASCADE,
    ADD CONSTRAINT fk_transactions_to_user FOREIGN KEY (to_user_id) REFERENCES users (id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_transactions_from_user_id ON transactions (from_user_id);
CREATE INDEX IF NOT EXISTS idx_transactions_to_user_id ON transactions (to_user_id);
