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

	CreateUpdateDeleteUserCommand struct {
		User entity.User
	}
)
