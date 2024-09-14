package v1

import "clean-arch-template/internal/entity"

type (
	FindUserRequest struct {
		ID int `path:"id" maxLength:"30" example:"1" doc:"IUserUC ID"`
	}

	UserRequest struct {
		Body struct {
			entity.User
		}
	}

	ListUserRequest struct {
		// Body обязательно (если есть тело запроса / ответа, json...), иначе поля уедут в headers
		Body struct {
			Page, Size int
		}
	}

	ListUserResponse struct {
		// Body обязательно (если есть тело запроса / ответа, json...), иначе поля уедут в headers
		Users []entity.User
	}

	UserResponse struct {
		// Body обязательно (если есть тело запроса / ответа, json...), иначе поля уедут в headers
		Body struct {
			*entity.User
		}
	}
)
