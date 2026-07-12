package entity

import "time"

type Transaction struct {
	ID         int64 `json:"id"`
	FromUserID int64 `json:"from_user_id"`
	ToUserID   int64 `json:"to_user_id"`
	// Amount in minimal currency units, 100 cents = 1$.
	Amount    int64     `json:"amount"`
	CreatedAt time.Time `json:"created_at"`
}

type Transfer struct {
	FromAccountID int64 `json:"from_account_id"`
	ToAccountID   int64 `json:"to_account_id"`
	// Amount in minimal currency units, 100 cents = 1$.
	Amount int64 `json:"amount"`
}
