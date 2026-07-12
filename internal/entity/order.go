package entity

type Order struct {
	ID     int64 `json:"id"`
	UserID int64 `json:"user_id"`
	// Amount in minimal currency units, 100 cents = 1$.
	Amount int64 `json:"amount"`
}
