package v1

// DTO транспортного слоя: entity сюда не протекает, поэтому новые поля домена
// не попадают в публичный контракт автоматически. Маппинг — в converter.go.
type (
	UserDTO struct {
		ID   int    `json:"id"   doc:"User ID"   example:"1"`
		Name string `json:"name" doc:"User name" example:"Mike"`
	}

	CreateUpdateUserBody struct {
		Name string `json:"name" doc:"User name" example:"Mike" minLength:"1" maxLength:"255"`
	}

	TransferDTO struct {
		FromAccountID int64 `json:"from_account_id" doc:"Source account ID"      example:"1"   minimum:"1"`
		ToAccountID   int64 `json:"to_account_id"   doc:"Destination account ID" example:"2"   minimum:"1"`
		Amount        int64 `json:"amount"          doc:"Amount in minimal currency units, 100 cents = 1$" example:"100" minimum:"1"`
	}

	ListUserRequest struct {
		Page int `path:"page" minimum:"1" example:"1"  doc:"1-based page number"`
		Size int `path:"size" minimum:"1" maximum:"1000" example:"10" doc:"page size"`
	}

	FindUserRequest struct {
		ID int `path:"id" minimum:"1" example:"1" doc:"user id"`
	}

	CreateUserRequest struct {
		Body CreateUpdateUserBody
	}

	UpdateUserRequest struct {
		ID   int `path:"id" minimum:"1" example:"1" doc:"user id"`
		Body CreateUpdateUserBody
	}

	ListUserResponse struct {
		// Body обязательно (если есть тело запроса / ответа, json...), иначе поля уедут в headers
		Body struct {
			Users []UserDTO `json:"users"`
		}
	}

	UserResponse struct {
		// Body обязательно (если есть тело запроса / ответа, json...), иначе поля уедут в headers
		Body UserDTO
	}

	TransferMoneyRequest struct {
		Body TransferDTO
	}
)
