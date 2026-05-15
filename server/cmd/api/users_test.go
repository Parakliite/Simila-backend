package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/deltron-fr/filmbox/server/internal/data"
)

// --- registerUserHandler tests ---

func TestRegisterUser_ValidInput(t *testing.T) {
	app := newTestApp("")

	body := `{"name":"Alice","email":"alice@example.com","password":"password123"}`
	req := httptest.NewRequest("POST", "/api/v1/users", strings.NewReader(body))
	rr := httptest.NewRecorder()

	app.registerUserHandler(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(rr.Body.Bytes(), &resp)

	user := resp["user"].(map[string]any)
	if user["name"] != "Alice" {
		t.Errorf("expected name %q, got %q", "Alice", user["name"])
	}
	if user["email"] != "alice@example.com" {
		t.Errorf("expected email %q, got %q", "alice@example.com", user["email"])
	}
	if user["activated"] != false {
		t.Errorf("expected activated false, got %v", user["activated"])
	}
}

func TestRegisterUser_EmptyBody(t *testing.T) {
	app := newTestApp("")

	req := httptest.NewRequest("POST", "/api/v1/users", nil)
	rr := httptest.NewRecorder()

	app.registerUserHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestRegisterUser_MissingName(t *testing.T) {
	app := newTestApp("")

	body := `{"email":"alice@example.com","password":"password123"}`
	req := httptest.NewRequest("POST", "/api/v1/users", strings.NewReader(body))
	rr := httptest.NewRecorder()

	app.registerUserHandler(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestRegisterUser_InvalidEmail(t *testing.T) {
	app := newTestApp("")

	body := `{"name":"Alice","email":"not-an-email","password":"password123"}`
	req := httptest.NewRequest("POST", "/api/v1/users", strings.NewReader(body))
	rr := httptest.NewRecorder()

	app.registerUserHandler(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestRegisterUser_ShortPassword(t *testing.T) {
	app := newTestApp("")

	body := `{"name":"Alice","email":"alice@example.com","password":"short"}`
	req := httptest.NewRequest("POST", "/api/v1/users", strings.NewReader(body))
	rr := httptest.NewRecorder()

	app.registerUserHandler(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestRegisterUser_DuplicateEmail(t *testing.T) {
	app := newTestApp("")

	existing := newTestUser("Bob", "alice@example.com", "password123", true)
	seedUser(app, existing)

	body := `{"name":"Alice","email":"alice@example.com","password":"password123"}`
	req := httptest.NewRequest("POST", "/api/v1/users", strings.NewReader(body))
	rr := httptest.NewRecorder()

	app.registerUserHandler(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", rr.Code, rr.Body.String())
	}
}

// --- activateUserHandler tests ---

func TestActivateUser_ValidToken(t *testing.T) {
	app := newTestApp("")

	user := newTestUser("Alice", "alice@example.com", "password123", false)
	seedUser(app, user)

	token := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	app.models.Users.(*mockUserModel).forToken[data.ScopeActivation+":"+token] = user

	body := `{"token":"ABCDEFGHIJKLMNOPQRSTUVWXYZ"}`
	req := httptest.NewRequest("PUT", "/api/v1/users/activated", strings.NewReader(body))
	rr := httptest.NewRecorder()

	app.activateUserHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	if !user.Activated {
		t.Error("expected user to be activated")
	}
}

func TestActivateUser_EmptyBody(t *testing.T) {
	app := newTestApp("")

	req := httptest.NewRequest("PUT", "/api/v1/users/activated", nil)
	rr := httptest.NewRecorder()

	app.activateUserHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestActivateUser_InvalidTokenFormat(t *testing.T) {
	app := newTestApp("")

	body := `{"token":"tooshort"}`
	req := httptest.NewRequest("PUT", "/api/v1/users/activated", strings.NewReader(body))
	rr := httptest.NewRecorder()

	app.activateUserHandler(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestActivateUser_UnknownToken(t *testing.T) {
	app := newTestApp("")

	body := `{"token":"AAAAAAAAAAAAAAAAAAAAAAAAAA"}`
	req := httptest.NewRequest("PUT", "/api/v1/users/activated", strings.NewReader(body))
	rr := httptest.NewRecorder()

	app.activateUserHandler(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", rr.Code, rr.Body.String())
	}
}
