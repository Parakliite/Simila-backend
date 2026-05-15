package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// --- createAuthenticationTokenHandler tests ---

func TestCreateAuthToken_ValidCredentials(t *testing.T) {
	app := newTestApp("")

	user := newTestUser("Alice", "alice@example.com", "password123", true)
	seedUser(app, user)

	body := `{"email":"alice@example.com","password":"password123"}`
	req := httptest.NewRequest("POST", "/api/v1/login", strings.NewReader(body))
	rr := httptest.NewRecorder()

	app.createAuthenticationTokenHandler(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(rr.Body.Bytes(), &resp)

	token := resp["token"].(map[string]any)
	if token["token"] == nil || token["token"] == "" {
		t.Error("expected token in response")
	}
	if token["expiry"] == nil {
		t.Error("expected expiry in response")
	}
}

func TestCreateAuthToken_WrongPassword(t *testing.T) {
	app := newTestApp("")

	user := newTestUser("Alice", "alice@example.com", "password123", true)
	seedUser(app, user)

	body := `{"email":"alice@example.com","password":"wrongpassword"}`
	req := httptest.NewRequest("POST", "/api/v1/login", strings.NewReader(body))
	rr := httptest.NewRecorder()

	app.createAuthenticationTokenHandler(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestCreateAuthToken_UnknownEmail(t *testing.T) {
	app := newTestApp("")

	body := `{"email":"nobody@example.com","password":"password123"}`
	req := httptest.NewRequest("POST", "/api/v1/login", strings.NewReader(body))
	rr := httptest.NewRecorder()

	app.createAuthenticationTokenHandler(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestCreateAuthToken_EmptyBody(t *testing.T) {
	app := newTestApp("")

	req := httptest.NewRequest("POST", "/api/v1/login", nil)
	rr := httptest.NewRecorder()

	app.createAuthenticationTokenHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestCreateAuthToken_MissingPassword(t *testing.T) {
	app := newTestApp("")

	body := `{"email":"alice@example.com"}`
	req := httptest.NewRequest("POST", "/api/v1/login", strings.NewReader(body))
	rr := httptest.NewRecorder()

	app.createAuthenticationTokenHandler(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestCreateAuthToken_MissingEmail(t *testing.T) {
	app := newTestApp("")

	body := `{"password":"password123"}`
	req := httptest.NewRequest("POST", "/api/v1/login", strings.NewReader(body))
	rr := httptest.NewRecorder()

	app.createAuthenticationTokenHandler(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", rr.Code, rr.Body.String())
	}
}
