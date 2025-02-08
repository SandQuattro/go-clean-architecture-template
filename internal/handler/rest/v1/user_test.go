package v1

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"

	"clean-arch-template/internal/entity"
	"clean-arch-template/internal/usecase"
)

var (
	mockUsers = []entity.User{
		{
			ID:   1,
			Name: "Test User 1",
		},
		{
			ID:   2,
			Name: "Test User 2",
		},
	}
)

func setup(api huma.API) {
	mockRepo := &mockUserRepository{users: mockUsers}
	userUC := usecase.NewUserUseCase(mockRepo)

	setupUserRoutes(api, userUC)
}

func TestListUsersSuccess(t *testing.T) {
	_, api := humatest.New(t)
	setup(api)

	resp := api.Get("/users/0/10")
	if resp.Code != http.StatusOK {
		t.Fatalf("Expected status code %d, got %d", http.StatusOK, resp.Code)
	}

	var response struct {
		Users []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"Users"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(response.Users) != len(mockUsers) {
		t.Fatalf("Expected %d users, got %d", len(mockUsers), len(response.Users))
	}

	for i, user := range response.Users {
		if user.ID != mockUsers[i].ID || user.Name != mockUsers[i].Name {
			t.Errorf("User %d does not match expected data. Got ID:%d Name:%s, want ID:%d Name:%s",
				i, user.ID, user.Name, mockUsers[i].ID, mockUsers[i].Name)
		}
	}
}

func TestListUsersPageError(t *testing.T) {
	_, api := humatest.New(t)
	setup(api)

	resp := api.Get("/users/-1/10")
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("Expected status code %d for invalid parameters, got %d", http.StatusBadRequest, resp.Code)
	}
}

func TestListUsersSizeError(t *testing.T) {
	_, api := humatest.New(t)
	setup(api)

	resp := api.Get("/users/0/-1")
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("Expected status code %d for invalid parameters, got %d", http.StatusBadRequest, resp.Code)
	}
}

func TestFindUserByIDSuccess(t *testing.T) {
	_, api := humatest.New(t)
	setup(api)

	resp := api.Get("/user/1")
	if resp.Code != http.StatusOK {
		t.Fatalf("Expected status code %d, got %d", http.StatusOK, resp.Code)
	}

	var user struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	expectedUser := mockUsers[0]
	if user.ID != expectedUser.ID || user.Name != expectedUser.Name {
		t.Errorf("User does not match expected data. Got ID:%d Name:%s, want ID:%d Name:%s",
			user.ID, user.Name, expectedUser.ID, expectedUser.Name)
	}
}

// mockUserRepository implements usecase.UserRepository interface for testing
type mockUserRepository struct {
	users []entity.User
}

func (m *mockUserRepository) GetAllUsers(ctx context.Context, offset, limit int) ([]entity.User, error) {
	if offset < 0 || limit <= 0 {
		return nil, errors.New("invalid pagination parameters")
	}

	end := offset + limit
	if end > len(m.users) {
		end = len(m.users)
	}
	if offset >= len(m.users) {
		return []entity.User{}, nil
	}

	return m.users[offset:end], nil
}

func (m *mockUserRepository) GetUserByID(ctx context.Context, id int) (*entity.User, error) {
	for _, user := range m.users {
		if user.ID == id {
			return &user, nil
		}
	}
	return nil, usecase.ErrUserNotFound
}

func (m *mockUserRepository) InsertUser(ctx context.Context, user *entity.User) (*entity.User, error) {
	m.users = append(m.users, *user)
	return user, nil
}

func (m *mockUserRepository) UpdateUser(ctx context.Context, user *entity.User) (*entity.User, error) {
	for i, u := range m.users {
		if u.ID == user.ID {
			m.users[i] = *user
			return &m.users[i], nil
		}
	}
	return nil, usecase.ErrUserNotFound
}

func (m *mockUserRepository) DeleteUser(ctx context.Context, user *entity.User) error {
	for i, u := range m.users {
		if u.ID == user.ID {
			m.users = append(m.users[:i], m.users[i+1:]...)
			return nil
		}
	}
	return usecase.ErrUserNotFound
}

func setupUserRoutes(api huma.API, userUC *usecase.UserUseCase) {
	// Initialize handlers
	userHandler := NewUserHandler(userUC)

	registry := huma.NewMapRegistry("#/components/schemas/", huma.DefaultSchemaNamer)
	userListSchema := huma.SchemaFromType(registry, reflect.TypeOf(&ListUserResponse{}))

	huma.Register(api, huma.Operation{
		OperationID: "List users",
		Method:      http.MethodGet,
		Path:        "/users/{page}/{size}",
		Summary:     "list all users",
		Description: "Get a list of all users with pagination.",
		Tags:        []string{"Users"},
		Parameters: []*huma.Param{
			{
				Name:     "page",
				In:       "path",
				Required: true,
				Schema: &huma.Schema{
					Type: "integer",
				},
			},
			{
				Name:     "size",
				In:       "path",
				Required: true,
				Schema: &huma.Schema{
					Type: "integer",
				},
			},
		},
		Responses: map[string]*huma.Response{
			"200": {
				Description: "Successful response",
				Content: map[string]*huma.MediaType{
					"application/json": {
						Schema: userListSchema,
					},
				},
			},
		},
	}, userHandler.ListUsers)

	userbyIDSchema := huma.SchemaFromType(registry, reflect.TypeOf(&entity.User{}))

	huma.Register(api, huma.Operation{
		OperationID: "Get user by id",
		Method:      http.MethodGet,
		Path:        "/user/{id}",
		Summary:     "user by id",
		Description: "Get a user by id.",
		Tags:        []string{"Users"},
		Responses: map[string]*huma.Response{
			"200": {
				Description: "IUserUC response",
				Content: map[string]*huma.MediaType{
					"application/json": {
						Schema: userbyIDSchema,
					},
				},
			},
			"400": {
				Description: "Invalid request",
				Content: map[string]*huma.MediaType{
					"application/json": {
						Schema: &huma.Schema{
							Type: "object",
							Properties: map[string]*huma.Schema{
								"message": {Type: "string"},
								"field":   {Type: "string"},
							},
						},
					},
				},
			},
			"404": {
				Description: "IUserUC not found",
				Content: map[string]*huma.MediaType{
					"application/json": {
						Schema: &huma.Schema{
							Type: "object",
							Properties: map[string]*huma.Schema{
								"error": {Type: "string"},
							},
						},
					},
				},
			},
			"500": {
				Description: "Internal server error",
				Content: map[string]*huma.MediaType{
					"application/json": {
						Schema: &huma.Schema{
							Type: "object",
							Properties: map[string]*huma.Schema{
								"error": {Type: "string"},
							},
						},
					},
				},
			},
		},
	}, userHandler.FindUserByID)

	huma.Register(api, huma.Operation{
		OperationID:   "Create user",
		Method:        http.MethodPost,
		Path:          "/user",
		Summary:       "create new user",
		Description:   "Create a new user record.",
		Tags:          []string{"Users"},
		DefaultStatus: http.StatusCreated,
		Responses: map[string]*huma.Response{
			"201": {
				Description: "IUserUC created",
				Content: map[string]*huma.MediaType{
					"application/json": {
						Schema: &huma.Schema{
							Type: "object",
							Properties: map[string]*huma.Schema{
								"body": {
									Type: "object",
									Properties: map[string]*huma.Schema{
										"name": {Type: "string"},
									},
									Required: []string{"name"},
								},
							},
							Required: []string{"body"},
						},
					},
				},
				Headers: map[string]*huma.Param{
					"Location": {
						Description: "URL of the newly created user",
						Schema:      &huma.Schema{Type: "string"},
						Required:    true,
					},
				},
			},
			"400": {
				Description: "Invalid request",
				Content: map[string]*huma.MediaType{
					"application/json": {
						Schema: &huma.Schema{
							Type: "object",
							Properties: map[string]*huma.Schema{
								"message": {Type: "string"},
								"field":   {Type: "string"},
							},
						},
					},
				},
			},
		},
	}, userHandler.CreateUser)

	huma.Register(api, huma.Operation{
		OperationID: "Update user",
		Method:      http.MethodPut,
		Path:        "/user/{id}",
		Summary:     "update user",
		Description: "Update an existing user by ID.",
		Tags:        []string{"Users"},
		Responses: map[string]*huma.Response{
			"200": {
				Description: "IUserUC updated",
				Content: map[string]*huma.MediaType{
					"application/json": {
						Schema: &huma.Schema{
							Type: "object",
							Properties: map[string]*huma.Schema{
								"body": {
									Type: "object",
									Properties: map[string]*huma.Schema{
										"name": {Type: "string"},
									},
									Required: []string{"name"},
								},
							},
							Required: []string{"body"},
						},
					},
				},
			},
			"400": {
				Description: "Invalid request",
				Content: map[string]*huma.MediaType{
					"application/json": {
						Schema: &huma.Schema{
							Type: "object",
							Properties: map[string]*huma.Schema{
								"message": {Type: "string"},
								"field":   {Type: "string"},
							},
						},
					},
				},
			},
			"404": {
				Description: "IUserUC not found",
				Content: map[string]*huma.MediaType{
					"application/json": {
						Schema: &huma.Schema{
							Type: "object",
							Properties: map[string]*huma.Schema{
								"error": {Type: "string"},
							},
						},
					},
				},
			},
		},
	}, userHandler.UpdateUser)

	huma.Register(api, huma.Operation{
		OperationID:   "Delete user",
		Method:        http.MethodDelete,
		Path:          "/user/{id}",
		Summary:       "delete user",
		Description:   "Delete a user by ID.",
		Tags:          []string{"Users"},
		DefaultStatus: http.StatusNoContent,
		Responses: map[string]*huma.Response{
			"204": {
				Description: "IUserUC deleted",
				Content:     map[string]*huma.MediaType{},
			},
			"404": {
				Description: "IUserUC not found",
				Content: map[string]*huma.MediaType{
					"application/json": {
						Schema: &huma.Schema{
							Type: "object",
							Properties: map[string]*huma.Schema{
								"error": {Type: "string"},
							},
						},
					},
				},
			},
		},
	}, userHandler.DeleteUser)
}
