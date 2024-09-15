package entity

type Order struct {
	ID     int64 `doc:"Order ID" example:"1" json:"id"`
	UserID int64 `doc:"User ID" example:"1" json:"user_id"`
	Amount int64 `doc:"Amount, in minimal currency value, 100 cents = 1$" example:"100" json:"amount"`
}
