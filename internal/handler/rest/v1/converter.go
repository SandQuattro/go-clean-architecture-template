package v1

import (
	"clean-arch-template/internal/entity"
)

func ToUserListOutputFromEntity(users []entity.User) ListUserResponse {
	return ListUserResponse{
		Body: struct{ Users []entity.User }{Users: users},
	}
}

func ToUserOutputFromEntity(user *entity.User) UserResponse {
	return UserResponse{
		Body: struct{ *entity.User }{user},
	}
}
