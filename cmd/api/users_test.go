package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestGetUserProfile_UserNotFoundMapsTo404(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)
	user.ID = uuid.New()

	req := httptest.NewRequest("GET", "/api/v1/users/profile", nil)
	req = withUser(req, user)
	rr := httptest.NewRecorder()

	app.getUserProfileHandler(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestUpdateUserProfile_NoFieldsProvided(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)
	seedUser(app, user)

	req := httptest.NewRequest("PATCH", "/api/v1/users/profile", strings.NewReader(`{}`))
	req = withUser(req, user)
	rr := httptest.NewRecorder()

	app.updateUserProfileHandler(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestRegisterUser_ValidInputReturnsUserContract(t *testing.T) {
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

func TestRegisterUser_DuplicateEmailMapsToValidationError(t *testing.T) {
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

func TestActivateUser_UnknownTokenMapsToValidationError(t *testing.T) {
	app := newTestApp("")

	body := `{"token":"AAAAAAAAAAAAAAAAAAAAAAAAAA"}`
	req := httptest.NewRequest("PUT", "/api/v1/users/activated", strings.NewReader(body))
	rr := httptest.NewRecorder()

	app.activateUserHandler(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", rr.Code, rr.Body.String())
	}
}
