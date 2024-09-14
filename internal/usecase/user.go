package usecase

import (
	// !!! NO UPSTREAM DEPENDENCIES HERE, ONLY ENTITY/DOMAIN !!!
	"clean-arch-template/internal/entity"
	"context"
)

type UserUseCase struct {
	userRepo UserRepository
}

func NewUserUseCase(ur UserRepository) *UserUseCase {
	return &UserUseCase{userRepo: ur}
}

func (uc *UserUseCase) FindAllUsers(ctx context.Context, cmd FindAllUsersCommand) ([]entity.User, error) {
	// page 0, size 10 ->
	var offset int
	if cmd.Page == 1 {
		cmd.Page = 0
	}
	offset = cmd.Page * cmd.Size
	limit := cmd.Size

	users, err := uc.userRepo.GetAllUsers(ctx, offset, limit)
	if err != nil {
		return nil, err
	}

	return users, nil
}

func (uc *UserUseCase) FindUserByID(ctx context.Context, cmd FindUserByIDCommand) (*entity.User, error) {
	user, err := uc.userRepo.GetUserByID(ctx, cmd.ID)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (uc *UserUseCase) CreateUser(ctx context.Context, cmd CreateUpdateDeleteUserCommand) (*entity.User, error) {
	user, err := uc.userRepo.InsertUser(ctx, &cmd.User)
	if err != nil {
		return nil, err
	}
	return user, nil
}
func (uc *UserUseCase) UpdateUser(ctx context.Context, cmd CreateUpdateDeleteUserCommand) (*entity.User, error) {
	user, err := uc.userRepo.UpdateUser(ctx, &cmd.User)
	if err != nil {
		return nil, err
	}
	return user, nil
}
func (uc *UserUseCase) DeleteUser(ctx context.Context, cmd CreateUpdateDeleteUserCommand) error {
	return uc.userRepo.DeleteUser(ctx, &cmd.User)
}
