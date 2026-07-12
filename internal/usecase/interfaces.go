package usecase

import (
	"clean-arch-template/internal/entity"
	"context"
)

//go:generate mockgen -source=interfaces.go -destination=./mocks.go -package=usecase

type UserRepository interface {
	GetAllUsers(ctx context.Context, offset, limit int) ([]entity.User, error)
	GetAllUsersWithOrders(ctx context.Context, offset, limit int) ([]entity.UserOrders, error)
	GetUserByID(ctx context.Context, id int) (*entity.User, error)
	InsertUser(ctx context.Context, input *entity.User) (*entity.User, error)
	UpdateUser(ctx context.Context, input *entity.User) (*entity.User, error)
	DeleteUser(ctx context.Context, id int) error

	TransferMoney(ctx context.Context, transfer entity.Transfer) error
}
