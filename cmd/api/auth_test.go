package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type authTokenPayload struct {
	Token  string `json:"token"`
	Expiry string `json:"expiry"`
}

type authResponse struct {
	AccessToken  authTokenPayload `json:"access_token"`
	RefreshToken authTokenPayload `json:"refresh_token"`
}

func decodeAuthResponse(t *testing.T, rr *httptest.ResponseRecorder) authResponse {
	t.Helper()

	var resp authResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode auth response: %v", err)
	}

	return resp
}

func assertTokenPayload(t *testing.T, token authTokenPayload, name string) {
	t.Helper()

	if token.Token == "" {
		t.Fatalf("expected %s token in response", name)
	}
	if len(token.Token) != 26 {
		t.Fatalf("expected %s token length 26, got %d", name, len(token.Token))
	}
	if token.Expiry == "" {
		t.Fatalf("expected %s expiry in response", name)
	}
}

func loginTestUser(t *testing.T, app *application) authResponse {
	t.Helper()

	user := newTestUser("Alice", "alice@example.com", "password123", true)
	seedUser(app, user)

	body := `{"email":"alice@example.com","password":"password123"}`
	req := httptest.NewRequest("POST", "/api/v1/login", strings.NewReader(body))
	rr := httptest.NewRecorder()

	app.loginHandler(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	return decodeAuthResponse(t, rr)
}

// --- loginHandler tests ---

func TestCreateAuthToken_ValidCredentials(t *testing.T) {
	app := newTestApp("")

	user := newTestUser("Alice", "alice@example.com", "password123", true)
	seedUser(app, user)

	body := `{"email":"alice@example.com","password":"password123"}`
	req := httptest.NewRequest("POST", "/api/v1/login", strings.NewReader(body))
	rr := httptest.NewRecorder()

	app.loginHandler(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	resp := decodeAuthResponse(t, rr)

	assertTokenPayload(t, resp.AccessToken, "access")
	assertTokenPayload(t, resp.RefreshToken, "refresh")
	if resp.AccessToken.Token == resp.RefreshToken.Token {
		t.Fatal("expected access and refresh tokens to be different")
	}
}

func TestCreateAuthToken_WrongPassword(t *testing.T) {
	app := newTestApp("")

	user := newTestUser("Alice", "alice@example.com", "password123", true)
	seedUser(app, user)

	body := `{"email":"alice@example.com","password":"wrongpassword"}`
	req := httptest.NewRequest("POST", "/api/v1/login", strings.NewReader(body))
	rr := httptest.NewRecorder()

	app.loginHandler(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestCreateAuthToken_UnknownEmail(t *testing.T) {
	app := newTestApp("")

	body := `{"email":"nobody@example.com","password":"password123"}`
	req := httptest.NewRequest("POST", "/api/v1/login", strings.NewReader(body))
	rr := httptest.NewRecorder()

	app.loginHandler(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestCreateAuthToken_EmptyBody(t *testing.T) {
	app := newTestApp("")

	req := httptest.NewRequest("POST", "/api/v1/login", nil)
	rr := httptest.NewRecorder()

	app.loginHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestCreateAuthToken_MissingPassword(t *testing.T) {
	app := newTestApp("")

	body := `{"email":"alice@example.com"}`
	req := httptest.NewRequest("POST", "/api/v1/login", strings.NewReader(body))
	rr := httptest.NewRecorder()

	app.loginHandler(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestCreateAuthToken_MissingEmail(t *testing.T) {
	app := newTestApp("")

	body := `{"password":"password123"}`
	req := httptest.NewRequest("POST", "/api/v1/login", strings.NewReader(body))
	rr := httptest.NewRecorder()

	app.loginHandler(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", rr.Code, rr.Body.String())
	}
}

// --- refreshHandler tests ---

func TestRefreshToken_ValidRefreshTokenRotatesTokens(t *testing.T) {
	app := newTestApp("")
	loginResp := loginTestUser(t, app)

	req := httptest.NewRequest("POST", "/api/v1/refresh", nil)
	req.Header.Set("Authorization", "Bearer "+loginResp.RefreshToken.Token)
	rr := httptest.NewRecorder()

	app.refreshHandler(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	refreshResp := decodeAuthResponse(t, rr)
	assertTokenPayload(t, refreshResp.AccessToken, "access")
	assertTokenPayload(t, refreshResp.RefreshToken, "refresh")
	if refreshResp.AccessToken.Token == loginResp.AccessToken.Token {
		t.Fatal("expected a new access token")
	}
	if refreshResp.RefreshToken.Token == loginResp.RefreshToken.Token {
		t.Fatal("expected a rotated refresh token")
	}
}

func TestRefreshToken_ReusedRefreshTokenIsRejected(t *testing.T) {
	app := newTestApp("")
	loginResp := loginTestUser(t, app)

	req := httptest.NewRequest("POST", "/api/v1/refresh", nil)
	req.Header.Set("Authorization", "Bearer "+loginResp.RefreshToken.Token)
	rr := httptest.NewRecorder()
	app.refreshHandler(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected first refresh to succeed, got %d: %s", rr.Code, rr.Body.String())
	}

	req = httptest.NewRequest("POST", "/api/v1/refresh", nil)
	req.Header.Set("Authorization", "Bearer "+loginResp.RefreshToken.Token)
	rr = httptest.NewRecorder()
	app.refreshHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected reused refresh token to return 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestRefreshToken_MissingHeader(t *testing.T) {
	app := newTestApp("")

	req := httptest.NewRequest("POST", "/api/v1/refresh", nil)
	rr := httptest.NewRecorder()

	app.refreshHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestRefreshToken_InvalidBearerFormat(t *testing.T) {
	app := newTestApp("")

	req := httptest.NewRequest("POST", "/api/v1/refresh", nil)
	req.Header.Set("Authorization", "Token ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	rr := httptest.NewRecorder()

	app.refreshHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}
