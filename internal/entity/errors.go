package entity

import "errors"

// Доменные ошибки, общие для всех слоёв: репозитории и use case возвращают
// эти сентинелы, транспортный слой маппит их в коды протокола (errors.go в handler).
var (
	ErrUserNotFound          = errors.New("user not found")
	ErrInvalidUserName       = errors.New("user name must be a non-empty valid UTF-8 string")
	ErrInvalidPagination     = errors.New("page and size must be greater than zero")
	ErrNegativeAmount        = errors.New("transfer amount must be positive")
	ErrSameAccount           = errors.New("cannot transfer to the same account")
	ErrInsufficientFunds     = errors.New("insufficient funds")
	ErrSourceAccountNotFound = errors.New("source account not found")
	ErrDestAccountNotFound   = errors.New("destination account not found")
)
