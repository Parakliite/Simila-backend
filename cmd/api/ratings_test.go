package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/parakliite/simila/internal/data"
)

func assertFloatRatingValue(t *testing.T, got any, want float64) {
	t.Helper()

	value, ok := got.(float64)
	if !ok {
		t.Fatalf("expected rating_value to be a float64, got %T", got)
	}

	if value != want {
		t.Fatalf("expected rating_value %.1f, got %.1f", want, value)
	}
}

func sortedResponseRatingValues(t *testing.T, ratings []any) []float64 {
	t.Helper()

	values := make([]float64, 0, len(ratings))
	for _, item := range ratings {
		userRating, ok := item.(map[string]any)
		if !ok {
			t.Fatalf("expected list item to be an object, got %T", item)
		}

		rating, ok := userRating["rating"].(map[string]any)
		if !ok {
			t.Fatalf("expected nested rating object, got %T", userRating["rating"])
		}

		value, ok := rating["rating_value"].(float64)
		if !ok {
			t.Fatalf("expected nested rating_value to be a float64, got %T", rating["rating_value"])
		}

		values = append(values, value)
	}

	sort.Float64s(values)
	return values
}

func TestUpsertRating_ValidHalfStepRatingIsReturnedOnClientScale(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)

	body := `{"tmdb_id":550,"media_type":"movie","rating_value":4.5}`
	req := httptest.NewRequest("PUT", "/api/v1/ratings", strings.NewReader(body))
	req = withUser(req, user)
	rr := httptest.NewRecorder()

	app.upsertRatingHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(rr.Body.Bytes(), &resp)

	rating := resp["rating"].(map[string]any)
	assertFloatRatingValue(t, rating["rating_value"], 4.5)
}

func TestUpsertRating_EmptyBody(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)

	req := httptest.NewRequest("PUT", "/api/v1/ratings", nil)
	req = withUser(req, user)
	rr := httptest.NewRecorder()

	app.upsertRatingHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestUpsertRating_RatingTooHigh(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)

	body := `{"tmdb_id":550,"media_type":"movie","rating_value":5.5}`
	req := httptest.NewRequest("PUT", "/api/v1/ratings", strings.NewReader(body))
	req = withUser(req, user)
	rr := httptest.NewRecorder()

	app.upsertRatingHandler(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestUpsertRating_RatingTooLow(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)

	body := `{"tmdb_id":550,"media_type":"movie","rating_value":0.5}`
	req := httptest.NewRequest("PUT", "/api/v1/ratings", strings.NewReader(body))
	req = withUser(req, user)
	rr := httptest.NewRecorder()

	app.upsertRatingHandler(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestUpsertRating_InvalidMediaType(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)

	body := `{"tmdb_id":550,"media_type":"podcast","rating_value":4.0}`
	req := httptest.NewRequest("PUT", "/api/v1/ratings", strings.NewReader(body))
	req = withUser(req, user)
	rr := httptest.NewRecorder()

	app.upsertRatingHandler(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestDeleteRating_NonExistentMapsToNotFound(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)

	req := httptest.NewRequest("DELETE", "/api/v1/ratings/tmdb/550?type=movie", nil)
	req.SetPathValue("tmdb_id", "550")
	req = withUser(req, user)
	rr := httptest.NewRecorder()

	app.deleteRatingHandler(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestDeleteRating_InvalidTmdbID(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)

	req := httptest.NewRequest("DELETE", "/api/v1/ratings/tmdb/not-a-number", nil)
	req.SetPathValue("tmdb_id", "not-a-number")
	req = withUser(req, user)
	rr := httptest.NewRecorder()

	app.deleteRatingHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestGetRating_ExistingReturnsTranslatedRatingValue(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)

	mock := app.models.Ratings.(*mockRatingModel)
	mediaUUID, err := app.models.Movies.GetOrCreateMediaByTmdbID(context.Background(), 550, "movie")
	if err != nil {
		t.Fatalf("failed to get or create media: %v", err)
	}
	mock.ratings[ratingKey(user.ID, mediaUUID)] = data.UserRating{
		Rating: data.Rating{UserID: user.ID, MediaID: mediaUUID, RatingValue: 7},
	}

	req := httptest.NewRequest("GET", "/api/v1/ratings/tmdb/550?type=movie", nil)
	req.SetPathValue("tmdb_id", "550")
	req = withUser(req, user)
	rr := httptest.NewRecorder()

	app.getRatingHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(rr.Body.Bytes(), &resp)

	rating := resp["rating"].(map[string]any)
	r := rating["rating"].(map[string]any)
	assertFloatRatingValue(t, r["rating_value"], 3.5)
}

func TestGetRating_NonExistentMapsToNotFound(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)

	req := httptest.NewRequest("GET", "/api/v1/ratings/tmdb/550?type=movie", nil)
	req.SetPathValue("tmdb_id", "550")
	req = withUser(req, user)
	rr := httptest.NewRecorder()

	app.getRatingHandler(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestGetRating_InvalidTmdbID(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)

	req := httptest.NewRequest("GET", "/api/v1/ratings/tmdb/not-a-number", nil)
	req.SetPathValue("tmdb_id", "not-a-number")
	req = withUser(req, user)
	rr := httptest.NewRecorder()

	app.getRatingHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestListRatings_ReturnsTranslatedUserRatings(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)

	mock := app.models.Ratings.(*mockRatingModel)
	for _, tc := range []struct {
		tmdbID       int32
		mediaType    string
		storedRating float64
	}{
		{tmdbID: 100, mediaType: "movie", storedRating: 10},
		{tmdbID: 200, mediaType: "movie", storedRating: 9},
		{tmdbID: 300, mediaType: "movie", storedRating: 7},
	} {
		mediaID, err := app.models.Movies.GetOrCreateMediaByTmdbID(context.Background(), tc.tmdbID, tc.mediaType)
		if err != nil {
			t.Fatalf("failed to get or create media: %v", err)
		}
		mock.ratings[ratingKey(user.ID, mediaID)] = data.UserRating{
			Rating: data.Rating{UserID: user.ID, MediaID: mediaID, RatingValue: tc.storedRating},
		}
	}

	req := httptest.NewRequest("GET", "/api/v1/ratings", nil)
	req = withUser(req, user)
	rr := httptest.NewRecorder()

	app.listRatingsHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(rr.Body.Bytes(), &resp)

	ratings := resp["ratings"].([]any)
	values := sortedResponseRatingValues(t, ratings)
	want := []float64{3.5, 4.5, 5.0}
	for i := range want {
		if values[i] != want[i] {
			t.Fatalf("expected translated ratings %v, got %v", want, values)
		}
	}
}

func TestListRatingsForMedia_ReturnsTranslatedRatings(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)

	mock := app.models.Ratings.(*mockRatingModel)
	mediaID, err := app.models.Movies.GetOrCreateMediaByTmdbID(context.Background(), 550, "movie")
	if err != nil {
		t.Fatalf("failed to get or create media: %v", err)
	}
	for _, tc := range []struct {
		userName     string
		storedRating float64
	}{
		{userName: "Bob", storedRating: 8},
		{userName: "Carol", storedRating: 3},
	} {
		otherUser := newTestUser(tc.userName, tc.userName+"@example.com", "password123", true)
		seedUser(app, otherUser)
		mock.ratings[ratingKey(otherUser.ID, mediaID)] = data.UserRating{
			Rating: data.Rating{UserID: otherUser.ID, MediaID: mediaID, RatingValue: tc.storedRating},
		}
	}

	req := httptest.NewRequest("GET", "/api/v1/media/tmdb/550/ratings?type=movie", nil)
	req.SetPathValue("tmdb_id", "550")
	req = withUser(req, user)
	rr := httptest.NewRecorder()

	app.listRatingsForMedia(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(rr.Body.Bytes(), &resp)

	ratings := resp["ratings"].([]any)
	values := sortedResponseRatingValues(t, ratings)
	want := []float64{1.5, 4.0}
	for i := range want {
		if values[i] != want[i] {
			t.Fatalf("expected translated ratings %v, got %v", want, values)
		}
	}
}

func TestListRatingsForMedia_InvalidTmdbID(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)

	req := httptest.NewRequest("GET", "/api/v1/media/tmdb/not-a-number/ratings", nil)
	req.SetPathValue("tmdb_id", "not-a-number")
	req = withUser(req, user)
	rr := httptest.NewRecorder()

	app.listRatingsForMedia(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}
