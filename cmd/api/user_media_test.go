package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/parakliite/simila/internal/data"
)

func TestListUserHighestRatedMediaUsesCursorPaginationForAnyAuthenticatedUser(t *testing.T) {
	app := newTestApp("")
	authenticatedUser := newTestUser("Alice", "alice@example.com", "password123", true)
	targetUserID := uuid.New()
	mockRatings := app.models.Ratings.(*mockRatingModel)
	now := time.Now().UTC()
	for i := 0; i < 6; i++ {
		mockRatings.highestRatedMedia = append(mockRatings.highestRatedMedia, data.RatedMedia{
			Media:       data.Media{ID: uuid.New(), TmdbID: int32(i + 1), Title: "Title", MediaType: "movie"},
			RatingValue: int32(10 - i),
			RatedAt:     now.Add(-time.Duration(i) * time.Minute),
		})
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/"+targetUserID.String()+"/ratings/highest-rated", nil)
	req.SetPathValue("user_id", targetUserID.String())
	req = withUser(req, authenticatedUser)
	rr := httptest.NewRecorder()

	app.listUserHighestRatedMediaHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if mockRatings.lastRatedUserID != targetUserID {
		t.Fatalf("expected target user ID %s, got %s", targetUserID, mockRatings.lastRatedUserID)
	}
	if mockRatings.lastRatedLimit != userMediaDefaultLimit+1 {
		t.Fatalf("expected repository limit %d, got %d", userMediaDefaultLimit+1, mockRatings.lastRatedLimit)
	}
	if mockRatings.lastCursorRating != 11 {
		t.Fatalf("expected initial rating cursor 11, got %d", mockRatings.lastCursorRating)
	}

	var response struct {
		Media      []json.RawMessage `json:"media"`
		NextCursor *string           `json:"next_cursor"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Media) != int(userMediaDefaultLimit) {
		t.Fatalf("expected %d media items, got %d", userMediaDefaultLimit, len(response.Media))
	}
	if response.NextCursor == nil {
		t.Fatal("expected next_cursor")
	}
}

func TestListUserRecentRatedMediaUsesCursorPaginationForOtherUsers(t *testing.T) {
	app := newTestApp("")
	authenticatedUser := newTestUser("Alice", "alice@example.com", "password123", true)
	targetUserID := uuid.New()
	mockRatings := app.models.Ratings.(*mockRatingModel)
	now := time.Now().UTC()
	for i := 0; i < 6; i++ {
		mockRatings.recentRatedMedia = append(mockRatings.recentRatedMedia, data.RatedMedia{
			Media:   data.Media{ID: uuid.New(), TmdbID: int32(i + 1), Title: "Title", MediaType: "tv"},
			RatedAt: now.Add(-time.Duration(i) * time.Minute),
		})
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/"+targetUserID.String()+"/ratings/recent", nil)
	req.SetPathValue("user_id", targetUserID.String())
	req = withUser(req, authenticatedUser)
	rr := httptest.NewRecorder()

	app.listUserRecentRatedMediaHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if mockRatings.lastRatedUserID != targetUserID {
		t.Fatalf("expected target user ID %s, got %s", targetUserID, mockRatings.lastRatedUserID)
	}
	if mockRatings.lastRatedLimit != userMediaDefaultLimit+1 {
		t.Fatalf("expected repository limit %d, got %d", userMediaDefaultLimit+1, mockRatings.lastRatedLimit)
	}
	if mockRatings.lastRatedUserID == authenticatedUser.ID {
		t.Fatal("expected endpoint to query the target user without requiring self-access")
	}

	var response struct {
		Media      []json.RawMessage `json:"media"`
		NextCursor *string           `json:"next_cursor"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Media) != int(userMediaDefaultLimit) {
		t.Fatalf("expected %d media items, got %d", userMediaDefaultLimit, len(response.Media))
	}
	if response.NextCursor == nil {
		t.Fatal("expected next_cursor")
	}
}
