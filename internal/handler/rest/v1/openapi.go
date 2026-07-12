package v1

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

type Handler interface {
	ListUsers(ctx context.Context, req *ListUserRequest) (*ListUserResponse, error)
	FindUserByID(ctx context.Context, req *FindUserRequest) (*UserResponse, error)
	CreateUser(ctx context.Context, req *CreateUserRequest) (*UserResponse, error)
	UpdateUser(ctx context.Context, req *UpdateUserRequest) (*UserResponse, error)
	DeleteUser(ctx context.Context, req *FindUserRequest) (*struct{}, error)
	TransferMoney(ctx context.Context, req *TransferMoneyRequest) (*struct{}, error)
}

func SetupHumaConfig() huma.Config {
	openapiConfig := huma.DefaultConfig("Clean Architecture Template", "1.0.0")
	openapiConfig.Components.SecuritySchemes = map[string]*huma.SecurityScheme{
		"auth": {
			Type:         "http",
			Scheme:       "bearer",
			BearerFormat: "JWT",
		},
	}
	openapiConfig.Security = []map[string][]string{
		{"auth": {""}},
	}

	return openapiConfig
}

// SetupRoutes регистрирует операции. Схемы запросов/ответов и параметры Huma
// генерирует из Go-типов (см. schemas.go) — руками их не описываем, чтобы
// контракт не расходился с кодом. Errors добавляет коды ошибок в OpenAPI.
func SetupRoutes(api huma.API, userHandler Handler) {
	huma.Register(api, huma.Operation{
		OperationID: "list-users",
		Method:      http.MethodGet,
		Path:        "/users/{page}/{size}",
		Summary:     "list all users",
		Description: "Get a page of users. Pages are 1-based.",
		Tags:        []string{"Users"},
		Errors:      []int{http.StatusBadRequest, http.StatusInternalServerError},
	}, userHandler.ListUsers)

	huma.Register(api, huma.Operation{
		OperationID: "get-user-by-id",
		Method:      http.MethodGet,
		Path:        "/user/{id}",
		Summary:     "user by id",
		Description: "Get a user by id.",
		Tags:        []string{"Users"},
		Errors:      []int{http.StatusBadRequest, http.StatusNotFound, http.StatusInternalServerError},
	}, userHandler.FindUserByID)

	huma.Register(api, huma.Operation{
		OperationID:   "create-user",
		Method:        http.MethodPost,
		Path:          "/user",
		Summary:       "create new user",
		Description:   "Create a new user record.",
		Tags:          []string{"Users"},
		DefaultStatus: http.StatusCreated,
		Errors:        []int{http.StatusBadRequest, http.StatusInternalServerError},
	}, userHandler.CreateUser)

	huma.Register(api, huma.Operation{
		OperationID: "update-user",
		Method:      http.MethodPut,
		Path:        "/user/{id}",
		Summary:     "update user",
		Description: "Update an existing user by ID. The ID from the path is authoritative; any ID in the body is ignored.",
		Tags:        []string{"Users"},
		Errors:      []int{http.StatusBadRequest, http.StatusNotFound, http.StatusInternalServerError},
	}, userHandler.UpdateUser)

	huma.Register(api, huma.Operation{
		OperationID:   "delete-user",
		Method:        http.MethodDelete,
		Path:          "/user/{id}",
		Summary:       "delete user",
		Description:   "Delete a user by ID.",
		Tags:          []string{"Users"},
		DefaultStatus: http.StatusNoContent,
		Errors:        []int{http.StatusNotFound, http.StatusInternalServerError},
	}, userHandler.DeleteUser)

	huma.Register(api, huma.Operation{
		OperationID:   "transfer-money",
		Method:        http.MethodPost,
		Path:          "/transfer",
		Summary:       "transfer money",
		Description:   "Transfer money between two accounts.",
		Tags:          []string{"Users"},
		DefaultStatus: http.StatusNoContent,
		Errors: []int{
			http.StatusBadRequest,
			http.StatusNotFound,
			http.StatusConflict,
			http.StatusInternalServerError,
		},
	}, userHandler.TransferMoney)
}
