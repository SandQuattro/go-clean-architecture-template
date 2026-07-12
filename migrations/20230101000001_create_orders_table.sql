-- +goose Up
CREATE TABLE IF NOT EXISTS orders
(
    id   BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    amount BIGINT NOT NULL
);

-- +goose Down
DROP TABLE orders;
