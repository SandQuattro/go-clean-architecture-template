package v1

import (
	"clean-arch-template/internal/entity"
)

func toUserDTO(user entity.User) UserDTO {
	return UserDTO{ID: user.ID, Name: user.Name}
}

func ToUserListOutputFromEntity(users []entity.User) *ListUserResponse {
	resp := &ListUserResponse{}
	resp.Body.Users = make([]UserDTO, 0, len(users))

	for _, user := range users {
		resp.Body.Users = append(resp.Body.Users, toUserDTO(user))
	}

	return resp
}

func ToUserOutputFromEntity(user *entity.User) *UserResponse {
	return &UserResponse{Body: toUserDTO(*user)}
}

func ToTransferEntity(dto TransferDTO) entity.Transfer {
	return entity.Transfer{
		FromAccountID: dto.FromAccountID,
		ToAccountID:   dto.ToAccountID,
		Amount:        dto.Amount,
	}
}
