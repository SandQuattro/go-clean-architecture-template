package usecase

import "clean-arch-template/internal/entity"

type (
	FindAllUsersCommand struct {
		Page int
		Size int
	}

	FindUserByIDCommand struct {
		ID int
	}

	CreateUpdateUserCommand struct {
		User entity.User
	}

	DeleteUserByIDCommand struct {
		ID int
	}

	TransferMoneyCommand struct {
		entity.Transfer
	}
)
