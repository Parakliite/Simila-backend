package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/parakliite/simila/internal/data"
)

func TestInsertToWatchlist_ValidInput(t *testing.T) {
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

func TestUpdateWatchlistStatus_Existing(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)
	mediaID := uuid.New()

	mock := app.models.Watchlist.(*mockWatchlistModel)
	mock.items[watchlistKey(user.ID, mediaID)] = data.UserWatchlistItem{
		Media: data.Media{ID: mediaID},
		Watchlist: data.Watchlist{
			UserID:    user.ID,
			MediaID:   mediaID,
			CreatedAt: time.Now(),
			Status:    "not_watched",
			Source:    "self",
		},
	}

	req := httptest.NewRequest("PATCH", "/api/v1/watchlist/"+mediaID.String(), strings.NewReader(`{"status":"watched"}`))
	req.SetPathValue("media_id", mediaID.String())
	req = withUser(req, user)
	rr := httptest.NewRecorder()

	app.updateWatchlistStatusHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(rr.Body.Bytes(), &resp)

	watchlist := resp["watchlist"].(map[string]any)
	if watchlist["status"] != "watched" {
		t.Fatalf("expected status watched, got %v", watchlist["status"])
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

func TestDeleteWatchlistItem_Existing(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)
	mediaID := uuid.New()

	mock := app.models.Watchlist.(*mockWatchlistModel)
	mock.items[watchlistKey(user.ID, mediaID)] = data.UserWatchlistItem{
		Media: data.Media{ID: mediaID},
		Watchlist: data.Watchlist{
			UserID:  user.ID,
			MediaID: mediaID,
			Status:  "not_watched",
			Source:  "self",
		},
	}

	req := httptest.NewRequest("DELETE", "/api/v1/watchlist/"+mediaID.String(), nil)
	req.SetPathValue("media_id", mediaID.String())
	req = withUser(req, user)
	rr := httptest.NewRecorder()

	app.deleteWatchlistItemHandler(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestDeleteWatchlistItem_NonExistent(t *testing.T) {
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

func TestListWatchlistItems_ReturnsItems(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)

	mock := app.models.Watchlist.(*mockWatchlistModel)
	for i := 0; i < 3; i++ {
		mediaID := uuid.New()
		mock.items[watchlistKey(user.ID, mediaID)] = data.UserWatchlistItem{
			Media: data.Media{
				ID:    mediaID,
				Title: fmt.Sprintf("Movie %d", i+1),
			},
			Watchlist: data.Watchlist{
				UserID:    user.ID,
				MediaID:   mediaID,
				CreatedAt: time.Now().Add(-time.Duration(i) * time.Minute),
				Status:    "not_watched",
				Source:    "self",
			},
		}
	}

	req := httptest.NewRequest("GET", "/api/v1/watchlist", nil)
	req = withUser(req, user)
	rr := httptest.NewRecorder()

	app.listWatchlistItemsHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(rr.Body.Bytes(), &resp)

	items := resp["watchlist_items"].([]any)
	if len(items) != 3 {
		t.Fatalf("expected 3 watchlist items, got %d", len(items))
	}
}

func TestListWatchlistItems_FiltersByWatchedStatus(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)

	mock := app.models.Watchlist.(*mockWatchlistModel)
	statuses := []string{"watched", "not_watched", "watched"}
	for i, status := range statuses {
		mediaID := uuid.New()
		mock.items[watchlistKey(user.ID, mediaID)] = data.UserWatchlistItem{
			Media: data.Media{
				ID:    mediaID,
				Title: fmt.Sprintf("Movie %d", i+1),
			},
			Watchlist: data.Watchlist{
				UserID:    user.ID,
				MediaID:   mediaID,
				CreatedAt: time.Now().Add(-time.Duration(i) * time.Minute),
				Status:    status,
				Source:    "self",
			},
		}
	}

	req := httptest.NewRequest("GET", "/api/v1/watchlist?status=watched", nil)
	req = withUser(req, user)
	rr := httptest.NewRecorder()

	app.listWatchlistItemsHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(rr.Body.Bytes(), &resp)

	items := resp["watchlist_items"].([]any)
	if len(items) != 2 {
		t.Fatalf("expected 2 watchlist items, got %d", len(items))
	}

	for _, item := range items {
		watchlistItem := item.(map[string]any)
		watchlist := watchlistItem["watchlist"].(map[string]any)
		if watchlist["status"] != "watched" {
			t.Fatalf("expected status watched, got %v", watchlist["status"])
		}
	}
}

func TestListWatchlistItems_FiltersByNotWatchedStatus(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)

	mock := app.models.Watchlist.(*mockWatchlistModel)
	statuses := []string{"watched", "not_watched", "not_watched"}
	for i, status := range statuses {
		mediaID := uuid.New()
		mock.items[watchlistKey(user.ID, mediaID)] = data.UserWatchlistItem{
			Media: data.Media{
				ID:    mediaID,
				Title: fmt.Sprintf("Movie %d", i+1),
			},
			Watchlist: data.Watchlist{
				UserID:    user.ID,
				MediaID:   mediaID,
				CreatedAt: time.Now().Add(-time.Duration(i) * time.Minute),
				Status:    status,
				Source:    "self",
			},
		}
	}

	req := httptest.NewRequest("GET", "/api/v1/watchlist?status=not_watched", nil)
	req = withUser(req, user)
	rr := httptest.NewRecorder()

	app.listWatchlistItemsHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(rr.Body.Bytes(), &resp)

	items := resp["watchlist_items"].([]any)
	if len(items) != 2 {
		t.Fatalf("expected 2 watchlist items, got %d", len(items))
	}

	for _, item := range items {
		watchlistItem := item.(map[string]any)
		watchlist := watchlistItem["watchlist"].(map[string]any)
		if watchlist["status"] != "not_watched" {
			t.Fatalf("expected status not_watched, got %v", watchlist["status"])
		}
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
