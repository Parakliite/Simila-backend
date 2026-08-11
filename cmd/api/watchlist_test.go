package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInsertToWatchlist_ValidInputReturnsDefaultStatus(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)

	body := `{"tmdb_id":550,"media_type":"movie","source":"self"}`
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
	if watchlist["status"] != "not_watched" {
		t.Fatalf("expected default status not_watched, got %v", watchlist["status"])
	}
}

func TestInsertToWatchlist_FromMatchRequiresSourceMatchID(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)

	body := `{"tmdb_id":550,"media_type":"movie","source":"from_match"}`
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

	body := `{"tmdb_id":550,"media_type":"movie","source":"friend"}`
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

	req := httptest.NewRequest("PATCH", "/api/v1/watchlist/tmdb/550?type=movie", strings.NewReader(`{"status":"queued"}`))
	req.SetPathValue("tmdb_id", "550")
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

	req := httptest.NewRequest("DELETE", "/api/v1/watchlist/tmdb/550?type=movie", nil)
	req.SetPathValue("tmdb_id", "550")
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
