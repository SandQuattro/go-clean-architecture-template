package v1

import (
	"clean-arch-template/internal/entity"
	"clean-arch-template/internal/usecase"
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
)

var mockUsers = []entity.User{
	{
		ID:   1,
		Name: "Test User 1",
	},
	{
		ID:   2,
		Name: "Test User 2",
	},
}

func newTestAPI(t *testing.T) humatest.TestAPI {
	t.Helper()

	_, api := humatest.New(t, SetupHumaConfig())

	users := make([]entity.User, len(mockUsers))
	copy(users, mockUsers)

	mockRepo := &mockUserRepository{users: users}
	userUC := usecase.NewUserUseCase(mockRepo)
	SetupRoutes(api, NewUserHandler(userUC))

	return api
}

func TestListUsersSuccess(t *testing.T) {
	api := newTestAPI(t)

	resp := api.Get("/users/1/10")
	if resp.Code != http.StatusOK {
		t.Fatalf("Expected status code %d, got %d", http.StatusOK, resp.Code)
	}

	var response struct {
		Users []UserDTO `json:"users"`
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
	api := newTestAPI(t)

	// page минимум 1 — валидируется схемой Huma до хендлера, поэтому 422.
	resp := api.Get("/users/0/10")
	if resp.Code != http.StatusUnprocessableEntity {
		t.Fatalf("Expected status code %d for invalid parameters, got %d", http.StatusUnprocessableEntity, resp.Code)
	}
}

func TestListUsersSizeError(t *testing.T) {
	api := newTestAPI(t)

	resp := api.Get("/users/1/-1")
	if resp.Code != http.StatusUnprocessableEntity {
		t.Fatalf("Expected status code %d for invalid parameters, got %d", http.StatusUnprocessableEntity, resp.Code)
	}
}

func TestFindUserByIDSuccess(t *testing.T) {
	api := newTestAPI(t)

	resp := api.Get("/user/1")
	if resp.Code != http.StatusOK {
		t.Fatalf("Expected status code %d, got %d", http.StatusOK, resp.Code)
	}

	var user UserDTO
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	expectedUser := mockUsers[0]
	if user.ID != expectedUser.ID || user.Name != expectedUser.Name {
		t.Errorf("User does not match expected data. Got ID:%d Name:%s, want ID:%d Name:%s",
			user.ID, user.Name, expectedUser.ID, expectedUser.Name)
	}
}

func TestFindUserByIDNotFound(t *testing.T) {
	api := newTestAPI(t)

	resp := api.Get("/user/999")
	if resp.Code != http.StatusNotFound {
		t.Fatalf("Expected status code %d for non-existent user, got %d", http.StatusNotFound, resp.Code)
	}
}

func TestCreateUserSuccess(t *testing.T) {
	api := newTestAPI(t)

	resp := api.Post("/user", map[string]any{
		"name": "New Test User",
	})

	if resp.Code != http.StatusCreated {
		t.Fatalf("Expected status code %d, got %d", http.StatusCreated, resp.Code)
	}

	var response UserDTO
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Name != "New Test User" {
		t.Errorf("User name does not match. Got %s, want %s", response.Name, "New Test User")
	}
	if response.ID == 0 {
		t.Errorf("Expected created user to have a non-zero ID")
	}
}

func TestCreateUserInvalidData(t *testing.T) {
	api := newTestAPI(t)

	// Пустое имя отклоняется схемой (minLength=1) со статусом 422.
	resp := api.Post("/user", map[string]any{
		"name": "",
	})

	if resp.Code != http.StatusUnprocessableEntity {
		t.Fatalf("Expected status code %d for empty name, got %d", http.StatusUnprocessableEntity, resp.Code)
	}
}

func TestUpdateUserSuccess(t *testing.T) {
	api := newTestAPI(t)

	resp := api.Put("/user/1", map[string]any{
		"name": "Updated Test User",
	})

	if resp.Code != http.StatusOK {
		t.Fatalf("Expected status code %d, got %d", http.StatusOK, resp.Code)
	}

	var response UserDTO
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.ID != 1 || response.Name != "Updated Test User" {
		t.Errorf("User data does not match. Got ID:%d Name:%s, want ID:%d Name:%s",
			response.ID, response.Name, 1, "Updated Test User")
	}
}

func TestUpdateUserBodyIDRejected(t *testing.T) {
	api := newTestAPI(t)

	// ID принимается только из пути; лишние поля в теле отклоняются схемой.
	resp := api.Put("/user/1", map[string]any{
		"id":   7,
		"name": "Attacker",
	})

	if resp.Code != http.StatusUnprocessableEntity {
		t.Fatalf("Expected status code %d for body with id field, got %d", http.StatusUnprocessableEntity, resp.Code)
	}
}

func TestUpdateUserNotFound(t *testing.T) {
	api := newTestAPI(t)

	resp := api.Put("/user/999", map[string]any{
		"name": "Non-existent User",
	})

	if resp.Code != http.StatusNotFound {
		t.Fatalf("Expected status code %d for non-existent user, got %d", http.StatusNotFound, resp.Code)
	}
}

func TestDeleteUserSuccess(t *testing.T) {
	api := newTestAPI(t)

	resp := api.Delete("/user/1")
	if resp.Code != http.StatusNoContent {
		t.Fatalf("Expected status code %d, got %d", http.StatusNoContent, resp.Code)
	}
}

func TestDeleteUserNotFound(t *testing.T) {
	api := newTestAPI(t)

	resp := api.Delete("/user/999")
	if resp.Code != http.StatusNotFound {
		t.Fatalf("Expected status code %d for non-existent user, got %d", http.StatusNotFound, resp.Code)
	}
}

func TestTransferMoneySuccess(t *testing.T) {
	api := newTestAPI(t)

	resp := api.Post("/transfer", map[string]any{
		"from_account_id": 1,
		"to_account_id":   2,
		"amount":          100,
	})

	if resp.Code != http.StatusNoContent {
		t.Fatalf("Expected status code %d, got %d", http.StatusNoContent, resp.Code)
	}
}

func TestTransferMoneySameAccount(t *testing.T) {
	api := newTestAPI(t)

	resp := api.Post("/transfer", map[string]any{
		"from_account_id": 1,
		"to_account_id":   1,
		"amount":          100,
	})

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("Expected status code %d for same account transfer, got %d", http.StatusBadRequest, resp.Code)
	}
}

func TestTransferMoneyDestinationNotFound(t *testing.T) {
	api := newTestAPI(t)

	resp := api.Post("/transfer", map[string]any{
		"from_account_id": 1,
		"to_account_id":   999,
		"amount":          100,
	})

	if resp.Code != http.StatusNotFound {
		t.Fatalf("Expected status code %d for missing destination, got %d", http.StatusNotFound, resp.Code)
	}
}

func TestTransferMoneyInsufficientFunds(t *testing.T) {
	api := newTestAPI(t)

	resp := api.Post("/transfer", map[string]any{
		"from_account_id": 1,
		"to_account_id":   2,
		"amount":          1000000,
	})

	if resp.Code != http.StatusConflict {
		t.Fatalf("Expected status code %d for insufficient funds, got %d", http.StatusConflict, resp.Code)
	}
}

// mockUserRepository implements usecase.UserRepository interface for testing.
// Каждый существующий аккаунт считается имеющим баланс mockBalance:
// большие суммы дают entity.ErrInsufficientFunds.
type mockUserRepository struct {
	users []entity.User
}

const mockBalance = 1000

func (m *mockUserRepository) GetAllUsersWithOrders(_ context.Context, _, _ int) ([]entity.UserOrders, error) {
	return nil, nil
}

func (m *mockUserRepository) GetAllUsers(_ context.Context, _, _ int) ([]entity.User, error) {
	return m.users, nil
}

func (m *mockUserRepository) GetUserByID(_ context.Context, id int) (*entity.User, error) {
	for _, user := range m.users {
		if user.ID == id {
			return &user, nil
		}
	}
	return nil, entity.ErrUserNotFound
}

func (m *mockUserRepository) InsertUser(_ context.Context, user *entity.User) (*entity.User, error) {
	user.ID = len(m.users) + 1
	m.users = append(m.users, *user)
	return user, nil
}

func (m *mockUserRepository) UpdateUser(_ context.Context, user *entity.User) (*entity.User, error) {
	for i, existingUser := range m.users {
		if existingUser.ID == user.ID {
			m.users[i] = *user
			return user, nil
		}
	}
	return nil, entity.ErrUserNotFound
}

func (m *mockUserRepository) DeleteUser(_ context.Context, id int) error {
	for i, existingUser := range m.users {
		if existingUser.ID == id {
			m.users = append(m.users[:i], m.users[i+1:]...)
			return nil
		}
	}
	return entity.ErrUserNotFound
}

func (m *mockUserRepository) TransferMoney(_ context.Context, transfer entity.Transfer) error {
	if !m.userExists(transfer.FromAccountID) {
		return entity.ErrSourceAccountNotFound
	}
	if !m.userExists(transfer.ToAccountID) {
		return entity.ErrDestAccountNotFound
	}
	if transfer.Amount > mockBalance {
		return entity.ErrInsufficientFunds
	}
	return nil
}

func (m *mockUserRepository) userExists(id int64) bool {
	for _, user := range m.users {
		if int64(user.ID) == id {
			return true
		}
	}
	return false
}
