package v1

import "clean-arch-template/internal/entity"

type (
	ListUserRequest struct {
		Page int `path:"page" maxLength:"3" example:"1" doc:"page"`
		Size int `path:"size" maxLength:"3" example:"1" doc:"size"`
	}

	FindUserRequest struct {
		ID int `path:"id" maxLength:"30" example:"1" doc:"user id"`
	}

	UserRequest struct {
		Body struct {
			entity.User
		}
	}

	UpdateUserRequest struct {
		ID   int `path:"id" maxLength:"30" example:"1" doc:"user id"`
		Body struct {
			entity.User
		}
	}

	ListUserResponse struct {
		// Body обязательно (если есть тело запроса / ответа, json...), иначе поля уедут в headers
		Body struct {
			Users []entity.User
		}
	}

	UserResponse struct {
		// Body обязательно (если есть тело запроса / ответа, json...), иначе поля уедут в headers
		Body struct {
			*entity.User
		}
	}
)
