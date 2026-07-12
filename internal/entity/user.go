package entity

type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type UserOrders struct {
	ID     int     `json:"id"`
	Name   string  `json:"name"`
	Orders []Order `json:"orders,omitempty"`
}
