package usecase

import (
	"context"
	"unicode/utf8"

	// !!! NO UPSTREAM DEPENDENCIES HERE, ONLY ENTITY/DOMAIN !!!
	"clean-arch-template/internal/entity"
)

type UserUseCase struct {
	userRepo UserRepository
}

func NewUserUseCase(ur UserRepository) *UserUseCase {
	return &UserUseCase{userRepo: ur}
}

func (uc *UserUseCase) FindAllUsers(ctx context.Context, cmd FindAllUsersCommand) ([]entity.User, error) {
	if cmd.Page < 1 || cmd.Size < 1 {
		return nil, entity.ErrInvalidPagination
	}

	// Страницы нумеруются с 1: page=1 → строки [0, size).
	offset := (cmd.Page - 1) * cmd.Size

	return uc.userRepo.GetAllUsers(ctx, offset, cmd.Size)
}

func (uc *UserUseCase) FindUserByID(ctx context.Context, cmd FindUserByIDCommand) (*entity.User, error) {
	return uc.userRepo.GetUserByID(ctx, cmd.ID)
}

func (uc *UserUseCase) CreateUser(ctx context.Context, cmd CreateUpdateUserCommand) (*entity.User, error) {
	if err := validateUserName(cmd.User.Name); err != nil {
		return nil, err
	}

	return uc.userRepo.InsertUser(ctx, &cmd.User)
}

func (uc *UserUseCase) UpdateUser(ctx context.Context, cmd CreateUpdateUserCommand) (*entity.User, error) {
	if err := validateUserName(cmd.User.Name); err != nil {
		return nil, err
	}

	return uc.userRepo.UpdateUser(ctx, &cmd.User)
}

func (uc *UserUseCase) DeleteUser(ctx context.Context, cmd DeleteUserByIDCommand) error {
	return uc.userRepo.DeleteUser(ctx, cmd.ID)
}

func (uc *UserUseCase) TransferMoney(ctx context.Context, cmd TransferMoneyCommand) error {
	if cmd.Amount <= 0 {
		return entity.ErrNegativeAmount
	}
	if cmd.FromAccountID == cmd.ToAccountID {
		return entity.ErrSameAccount
	}

	return uc.userRepo.TransferMoney(ctx, cmd.Transfer)
}

func validateUserName(name string) error {
	if name == "" || !utf8.ValidString(name) {
		return entity.ErrInvalidUserName
	}

	return nil
}
