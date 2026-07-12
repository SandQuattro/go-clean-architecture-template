package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"
)

var httpClient = &http.Client{Timeout: 10 * time.Second}

type userResponse struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func doJSON(t *testing.T, method, url string, body any) (int, []byte) {
	t.Helper()

	var reqBody io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
		reqBody = bytes.NewReader(raw)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}

	return resp.StatusCode, respBody
}

func createUser(t *testing.T, name string) userResponse {
	t.Helper()

	status, body := doJSON(t, http.MethodPost, baseURL+"/user", map[string]any{"name": name})
	if status != http.StatusCreated {
		t.Fatalf("create user: expected status %d, got %d (%s)", http.StatusCreated, status, body)
	}

	var user userResponse
	if err := json.Unmarshal(body, &user); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if user.ID == 0 || user.Name != name {
		t.Fatalf("create user: unexpected response %+v", user)
	}

	return user
}

func TestUserCRUD(t *testing.T) {
	created := createUser(t, "integration-user")

	userURL := fmt.Sprintf("%s/user/%d", baseURL, created.ID)

	status, body := doJSON(t, http.MethodGet, userURL, nil)
	if status != http.StatusOK {
		t.Fatalf("get user: expected status %d, got %d (%s)", http.StatusOK, status, body)
	}

	status, body = doJSON(t, http.MethodPut, userURL, map[string]any{"name": "integration-user-renamed"})
	if status != http.StatusOK {
		t.Fatalf("update user: expected status %d, got %d (%s)", http.StatusOK, status, body)
	}

	var updated userResponse
	if err := json.Unmarshal(body, &updated); err != nil {
		t.Fatalf("decode update response: %v", err)
	}
	if updated.ID != created.ID || updated.Name != "integration-user-renamed" {
		t.Fatalf("update user: unexpected response %+v", updated)
	}

	status, body = doJSON(t, http.MethodGet, baseURL+"/users/1/100", nil)
	if status != http.StatusOK {
		t.Fatalf("list users: expected status %d, got %d (%s)", http.StatusOK, status, body)
	}

	var list struct {
		Users []userResponse `json:"users"`
	}
	if err := json.Unmarshal(body, &list); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if len(list.Users) == 0 {
		t.Fatalf("list users: expected at least one user")
	}

	status, body = doJSON(t, http.MethodDelete, userURL, nil)
	if status != http.StatusNoContent {
		t.Fatalf("delete user: expected status %d, got %d (%s)", http.StatusNoContent, status, body)
	}

	status, _ = doJSON(t, http.MethodGet, userURL, nil)
	if status != http.StatusNotFound {
		t.Fatalf("get deleted user: expected status %d, got %d", http.StatusNotFound, status)
	}
}

func TestUserNotFound(t *testing.T) {
	status, _ := doJSON(t, http.MethodGet, baseURL+"/user/999999", nil)
	if status != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, status)
	}
}

func TestCreateUserEmptyNameRejected(t *testing.T) {
	status, _ := doJSON(t, http.MethodPost, baseURL+"/user", map[string]any{"name": ""})
	if status != http.StatusUnprocessableEntity {
		t.Fatalf("expected status %d, got %d", http.StatusUnprocessableEntity, status)
	}
}

func TestListUsersInvalidPageRejected(t *testing.T) {
	status, _ := doJSON(t, http.MethodGet, baseURL+"/users/0/10", nil)
	if status != http.StatusUnprocessableEntity {
		t.Fatalf("expected status %d, got %d", http.StatusUnprocessableEntity, status)
	}
}

// Новые пользователи создаются с нулевым балансом, поэтому позитивный сценарий
// перевода недостижим через публичное API — проверяем доменные ошибки.
func TestTransferMoneyInsufficientFunds(t *testing.T) {
	from := createUser(t, "transfer-source")
	to := createUser(t, "transfer-destination")

	status, body := doJSON(t, http.MethodPost, baseURL+"/transfer", map[string]any{
		"from_account_id": from.ID,
		"to_account_id":   to.ID,
		"amount":          100,
	})
	if status != http.StatusConflict {
		t.Fatalf("expected status %d for insufficient funds, got %d (%s)", http.StatusConflict, status, body)
	}
}

func TestTransferMoneySameAccountRejected(t *testing.T) {
	user := createUser(t, "transfer-same-account")

	status, body := doJSON(t, http.MethodPost, baseURL+"/transfer", map[string]any{
		"from_account_id": user.ID,
		"to_account_id":   user.ID,
		"amount":          100,
	})
	if status != http.StatusBadRequest {
		t.Fatalf("expected status %d for same-account transfer, got %d (%s)", http.StatusBadRequest, status, body)
	}
}

func TestTransferMoneySourceNotFound(t *testing.T) {
	to := createUser(t, "transfer-orphan-destination")

	status, body := doJSON(t, http.MethodPost, baseURL+"/transfer", map[string]any{
		"from_account_id": 999999,
		"to_account_id":   to.ID,
		"amount":          100,
	})
	if status != http.StatusNotFound {
		t.Fatalf("expected status %d for missing source account, got %d (%s)", http.StatusNotFound, status, body)
	}
}
