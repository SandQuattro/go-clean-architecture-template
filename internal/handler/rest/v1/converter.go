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

func ToUserOrdersListOutputFromEntity(users []entity.UserOrders) *ListUserOrdersResponse {
	resp := &ListUserOrdersResponse{}
	resp.Body.Users = make([]UserOrdersDTO, 0, len(users))

	for _, user := range users {
		// Непустой срез: в JSON уходит [], а не null.
		orders := make([]OrderDTO, 0, len(user.Orders))
		for _, order := range user.Orders {
			orders = append(orders, OrderDTO{ID: order.ID, Amount: order.Amount})
		}

		resp.Body.Users = append(resp.Body.Users, UserOrdersDTO{ID: user.ID, Name: user.Name, Orders: orders})
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
