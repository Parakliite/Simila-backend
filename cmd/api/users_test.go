package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/parakliite/simila/internal/data"
)

// --- getUserProfileHandler tests ---

func TestGetUserProfile_ReturnsAuthenticatedUser(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)
	user.About = "Loves slow burns"
	user.ProfilePictureURL = "https://example.com/alice.jpg"
	seedUser(app, user)

	req := httptest.NewRequest("GET", "/api/v1/users/profile", nil)
	req = withUser(req, user)
	rr := httptest.NewRecorder()

	app.getUserProfileHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(rr.Body.Bytes(), &resp)

	gotUser := resp["user"].(map[string]any)
	if gotUser["id"] != user.ID.String() {
		t.Fatalf("expected id %s, got %v", user.ID, gotUser["id"])
	}
	if gotUser["name"] != user.Name {
		t.Fatalf("expected name %q, got %v", user.Name, gotUser["name"])
	}
	if gotUser["email"] != user.Email {
		t.Fatalf("expected email %q, got %v", user.Email, gotUser["email"])
	}
	if gotUser["about"] != user.About {
		t.Fatalf("expected about %q, got %v", user.About, gotUser["about"])
	}
	if gotUser["profile_picture_url"] != user.ProfilePictureURL {
		t.Fatalf("expected profile_picture_url %q, got %v", user.ProfilePictureURL, gotUser["profile_picture_url"])
	}
}

func TestGetUserProfile_UserNotFound(t *testing.T) {
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

// --- updateUserProfileHandler tests ---

func TestUpdateUserProfile_ValidInput(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)
	seedUser(app, user)

	body := `{"name":"Alice Updated","about":"Updated bio","profile_picture_url":"https://example.com/new.jpg"}`
	req := httptest.NewRequest("PATCH", "/api/v1/users/profile", strings.NewReader(body))
	req = withUser(req, user)
	rr := httptest.NewRecorder()

	app.updateUserProfileHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(rr.Body.Bytes(), &resp)

	gotUser := resp["user"].(map[string]any)
	if gotUser["name"] != "Alice Updated" {
		t.Fatalf("expected updated name, got %v", gotUser["name"])
	}
	if gotUser["about"] != "Updated bio" {
		t.Fatalf("expected updated about, got %v", gotUser["about"])
	}
	if gotUser["profile_picture_url"] != "https://example.com/new.jpg" {
		t.Fatalf("expected updated profile_picture_url, got %v", gotUser["profile_picture_url"])
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
