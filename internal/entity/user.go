package entity

type User struct {
	ID     int     `doc:"User ID" example:"1" json:"id"`
	Name   string  `doc:"User name" example:"Mike" json:"name"`
	Orders []Order `doc:"User orders" json:"orders,omitempty"`
}
