CREATE TABLE IF NOT EXISTS transactions
(
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT         NOT NULL,
    balance    DECIMAL(15, 2) NOT NULL CHECK (balance >= 0),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_transactions_user_id ON transactions (user_id);
