package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestInsertToWatchlist_ValidInputReturnsDefaultStatus(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)
	mediaID := uuid.New()

	body := fmt.Sprintf(`{"media_id":"%s","source":"self"}`, mediaID)
	req := httptest.NewRequest("POST", "/api/v1/watchlist", strings.NewReader(body))
	req = withUser(req, user)
	rr := httptest.NewRecorder()

	app.insertToWatchlistHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(rr.Body.Bytes(), &resp)

	watchlist := resp["watchlist"].(map[string]any)
	if watchlist["media_id"] != mediaID.String() {
		t.Fatalf("expected media_id %s, got %v", mediaID, watchlist["media_id"])
	}
	if watchlist["status"] != "not_watched" {
		t.Fatalf("expected default status not_watched, got %v", watchlist["status"])
	}
}

func TestInsertToWatchlist_FromMatchRequiresSourceMatchID(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)

	body := fmt.Sprintf(`{"media_id":"%s","source":"from_match"}`, uuid.New())
	req := httptest.NewRequest("POST", "/api/v1/watchlist", strings.NewReader(body))
	req = withUser(req, user)
	rr := httptest.NewRecorder()

	app.insertToWatchlistHandler(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestInsertToWatchlist_InvalidSource(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)

	body := fmt.Sprintf(`{"media_id":"%s","source":"friend"}`, uuid.New())
	req := httptest.NewRequest("POST", "/api/v1/watchlist", strings.NewReader(body))
	req = withUser(req, user)
	rr := httptest.NewRecorder()

	app.insertToWatchlistHandler(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestUpdateWatchlistStatus_InvalidStatus(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)
	mediaID := uuid.New()

	req := httptest.NewRequest("PATCH", "/api/v1/watchlist/"+mediaID.String(), strings.NewReader(`{"status":"queued"}`))
	req.SetPathValue("media_id", mediaID.String())
	req = withUser(req, user)
	rr := httptest.NewRecorder()

	app.updateWatchlistStatusHandler(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestDeleteWatchlistItem_NonExistentMapsToNotFound(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)
	mediaID := uuid.New()

	req := httptest.NewRequest("DELETE", "/api/v1/watchlist/"+mediaID.String(), nil)
	req.SetPathValue("media_id", mediaID.String())
	req = withUser(req, user)
	rr := httptest.NewRecorder()

	app.deleteWatchlistItemHandler(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestListWatchlistItems_InvalidCursor(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)

	req := httptest.NewRequest("GET", "/api/v1/watchlist?cursor=not-base64", nil)
	req = withUser(req, user)
	rr := httptest.NewRecorder()

	app.listWatchlistItemsHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}
