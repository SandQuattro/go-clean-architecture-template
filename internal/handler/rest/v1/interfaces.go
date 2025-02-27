package v1

import (
	"context"

	"clean-arch-template/internal/entity"
	"clean-arch-template/internal/usecase"
)

//go:generate mockgen -source=interfaces.go -destination=./mocks.go -package=v1

type UserUseCase interface {
	FindAllUsers(ctx context.Context, cmd usecase.FindAllUsersCommand) ([]entity.User, error)
	FindUserByID(ctx context.Context, cmd usecase.FindUserByIDCommand) (*entity.User, error)
	CreateUser(ctx context.Context, cmd usecase.CreateUpdateUserCommand) (*entity.User, error)
	UpdateUser(ctx context.Context, cmd usecase.CreateUpdateUserCommand) (*entity.User, error)
	DeleteUser(ctx context.Context, cmd usecase.DeleteUserByIDCommand) error
	TransferMoney(ctx context.Context, cmd usecase.TransferMoneyCommand) error
}
