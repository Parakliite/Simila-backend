package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/parakliite/simila/internal/data"
)

func TestListMatches_ReturnsMatchesAndNextCursor(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)
	mockMatches := app.models.Matches.(*mockMatchModel)
	now := time.Now().UTC()

	mockMatches.matches = []data.Match{
		{
			User: data.User{
				ID:   uuid.New(),
				Name: "Bob",
			},
			Score:              0.91,
			SharedMedia:        7,
			LastRecalculatedAt: now,
		},
		{
			User: data.User{
				ID:   uuid.New(),
				Name: "Cara",
			},
			Score:              0.82,
			SharedMedia:        6,
			LastRecalculatedAt: now.Add(-time.Minute),
		},
	}

	req := httptest.NewRequest("GET", "/api/v1/matches?limit=1", nil)
	req = withUser(req, user)
	rr := httptest.NewRecorder()

	app.listMatchesHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if mockMatches.lastTargetUserID != user.ID {
		t.Fatalf("expected target user ID %s, got %s", user.ID, mockMatches.lastTargetUserID)
	}
	if mockMatches.lastLimit != 2 {
		t.Fatalf("expected model limit 2, got %d", mockMatches.lastLimit)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}

	matches := resp["matches"].([]any)
	if len(matches) != 1 {
		t.Fatalf("expected 1 returned match, got %d", len(matches))
	}
	if resp["next_cursor"] == nil {
		t.Fatal("expected next_cursor to be present")
	}
}

func TestListMatches_InvalidCursor(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)

	req := httptest.NewRequest("GET", "/api/v1/matches?cursor=not-base64", nil)
	req = withUser(req, user)
	rr := httptest.NewRecorder()

	app.listMatchesHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestListMatches_ModelErrorMapsToServerError(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)
	app.models.Matches.(*mockMatchModel).getMatchesErr = fmt.Errorf("matches unavailable")

	req := httptest.NewRequest("GET", "/api/v1/matches", nil)
	req = withUser(req, user)
	rr := httptest.NewRecorder()

	app.listMatchesHandler(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", rr.Code, rr.Body.String())
	}
}
