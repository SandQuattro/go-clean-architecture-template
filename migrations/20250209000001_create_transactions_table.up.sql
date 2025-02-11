CREATE TABLE IF NOT EXISTS transactions
(
    id           BIGSERIAL PRIMARY KEY,
    from_user_id BIGINT         NOT NULL,
    to_user_id   BIGINT         NOT NULL,
    amount       DECIMAL(15, 2) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
