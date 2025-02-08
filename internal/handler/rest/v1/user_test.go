package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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

	handler := NewUserHandler(userUC)

	SetupRoutes(api, handler)
}

func TestListUsersSuccess(t *testing.T) {
	humaConfig := SetupHumaConfig()
	_, api := humatest.New(t, humaConfig)
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
	humaConfig := SetupHumaConfig()
	_, api := humatest.New(t, humaConfig)
	setup(api)

	resp := api.Get("/users/-1/10")
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("Expected status code %d for invalid parameters, got %d", http.StatusBadRequest, resp.Code)
	}
}

func TestListUsersSizeError(t *testing.T) {
	humaConfig := SetupHumaConfig()
	_, api := humatest.New(t, humaConfig)
	setup(api)

	resp := api.Get("/users/0/-1")
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("Expected status code %d for invalid parameters, got %d", http.StatusBadRequest, resp.Code)
	}
}

func TestFindUserByIDSuccess(t *testing.T) {
	humaConfig := SetupHumaConfig()
	_, api := humatest.New(t, humaConfig)
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

func TestCreateUserSuccess(t *testing.T) {
	humaConfig := SetupHumaConfig()
	_, api := humatest.New(t, humaConfig)
	setup(api)

	resp := api.Post("/user", map[string]interface{}{
		"id":   0,
		"name": "New Test User",
	})

	if resp.Code != http.StatusCreated {
		t.Fatalf("Expected status code %d, got %d", http.StatusCreated, resp.Code)
	}

	var response struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Name != "New Test User" {
		t.Errorf("User name does not match. Got %s, want %s", response.Name, "New Test User")
	}
}

func TestCreateUserInvalidData(t *testing.T) {
	humaConfig := SetupHumaConfig()
	_, api := humatest.New(t, humaConfig)
	setup(api)

	// Test with empty name
	resp := api.Post("/user", map[string]interface{}{
		"id":   0,
		"name": "",
	})

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("Expected status code %d for empty name, got %d", http.StatusBadRequest, resp.Code)
	}
}

func TestUpdateUserSuccess(t *testing.T) {
	humaConfig := SetupHumaConfig()
	_, api := humatest.New(t, humaConfig)
	setup(api)

	resp := api.Put("/user/1", map[string]interface{}{
		"id":   1,
		"name": "Updated Test User",
	})

	if resp.Code != http.StatusOK {
		t.Fatalf("Expected status code %d, got %d", http.StatusOK, resp.Code)
	}

	var response struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.ID != 1 || response.Name != "Updated Test User" {
		t.Errorf("User data does not match. Got ID:%d Name:%s, want ID:%d Name:%s",
			response.ID, response.Name, 1, "Updated Test User")
	}
}

func TestUpdateUserNotFound(t *testing.T) {
	humaConfig := SetupHumaConfig()
	_, api := humatest.New(t, humaConfig)
	setup(api)

	resp := api.Put("/user/999", map[string]interface{}{
		"id":   999,
		"name": "Non-existent User",
	})

	if resp.Code != http.StatusNotFound {
		t.Fatalf("Expected status code %d for non-existent user, got %d", http.StatusNotFound, resp.Code)
	}
}

func TestDeleteUserSuccess(t *testing.T) {
	humaConfig := SetupHumaConfig()
	_, api := humatest.New(t, humaConfig)
	setup(api)

	resp := api.Delete("/user/1")
	if resp.Code != http.StatusNoContent {
		t.Fatalf("Expected status code %d, got %d", http.StatusNoContent, resp.Code)
	}
}

func TestDeleteUserNotFound(t *testing.T) {
	humaConfig := SetupHumaConfig()
	_, api := humatest.New(t, humaConfig)
	setup(api)

	resp := api.Delete("/user/999")
	if resp.Code != http.StatusNotFound {
		t.Fatalf("Expected status code %d for non-existent user, got %d", http.StatusNotFound, resp.Code)
	}
}

// mockUserRepository implements usecase.UserRepository interface for testing
type mockUserRepository struct {
	users []entity.User
}

func (m *mockUserRepository) GetAllUsers(ctx context.Context, offset, limit int) ([]entity.User, error) {
	if offset < 0 || limit <= 0 {
		return nil, fmt.Errorf("invalid pagination parameters")
	}
	return m.users, nil
}

func (m *mockUserRepository) GetUserByID(ctx context.Context, id int) (*entity.User, error) {
	for _, user := range m.users {
		if user.ID == id {
			return &user, nil
		}
	}
	return nil, nil
}

func (m *mockUserRepository) InsertUser(ctx context.Context, user *entity.User) (*entity.User, error) {
	if user.Name == "" {
		return nil, fmt.Errorf("name is required")
	}

	user.ID = len(m.users) + 1
	m.users = append(m.users, *user)
	return user, nil
}

func (m *mockUserRepository) UpdateUser(ctx context.Context, user *entity.User) (*entity.User, error) {
	for i, existingUser := range m.users {
		if existingUser.ID == user.ID {
			m.users[i] = *user
			return user, nil
		}
	}
	return nil, fmt.Errorf("user not found")
}

func (m *mockUserRepository) DeleteUser(ctx context.Context, user *entity.User) error {
	for i, existingUser := range m.users {
		if existingUser.ID == user.ID {
			m.users = append(m.users[:i], m.users[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("user not found")
}
