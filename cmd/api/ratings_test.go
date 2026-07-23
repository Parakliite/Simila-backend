package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/google/uuid"
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

// --- upsertRatingHandler tests ---

func TestUpsertRating_ValidHalfStepRatingIsReturnedOnClientScale(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)
	mediaID := uuid.New()

	body := fmt.Sprintf(`{"media_id":"%s","rating_value":4.5}`, mediaID)
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
	if rating["media_id"] != mediaID.String() {
		t.Errorf("expected media_id %s, got %v", mediaID, rating["media_id"])
	}
}

func TestUpsertRating_UpdatesExistingRatingAndReturnsClientScale(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)
	mediaID := uuid.New()

	body := fmt.Sprintf(`{"media_id":"%s","rating_value":2.5}`, mediaID)
	req := httptest.NewRequest("PUT", "/api/v1/ratings", strings.NewReader(body))
	req = withUser(req, user)
	rr := httptest.NewRecorder()
	app.upsertRatingHandler(rr, req)

	body = fmt.Sprintf(`{"media_id":"%s","rating_value":3.5}`, mediaID)
	req = httptest.NewRequest("PUT", "/api/v1/ratings", strings.NewReader(body))
	req = withUser(req, user)
	rr = httptest.NewRecorder()
	app.upsertRatingHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(rr.Body.Bytes(), &resp)

	rating := resp["rating"].(map[string]any)
	assertFloatRatingValue(t, rating["rating_value"], 3.5)
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

	body := fmt.Sprintf(`{"media_id":"%s","rating_value":5.5}`, uuid.New())
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

	body := fmt.Sprintf(`{"media_id":"%s","rating_value":0.5}`, uuid.New())
	req := httptest.NewRequest("PUT", "/api/v1/ratings", strings.NewReader(body))
	req = withUser(req, user)
	rr := httptest.NewRecorder()

	app.upsertRatingHandler(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", rr.Code, rr.Body.String())
	}
}

// --- deleteRatingHandler tests ---

func TestDeleteRating_Existing(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)
	mediaID := uuid.New()

	mock := app.models.Ratings.(*mockRatingModel)
	mock.ratings[ratingKey(user.ID, mediaID)] = data.UserRating{
		Rating: data.Rating{UserID: user.ID, MediaID: mediaID, RatingValue: 8},
	}

	req := httptest.NewRequest("DELETE", "/api/v1/ratings/"+mediaID.String(), nil)
	req.SetPathValue("media_id", mediaID.String())
	req = withUser(req, user)
	rr := httptest.NewRecorder()

	app.deleteRatingHandler(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestDeleteRating_NonExistent(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)
	mediaID := uuid.New()

	req := httptest.NewRequest("DELETE", "/api/v1/ratings/"+mediaID.String(), nil)
	req.SetPathValue("media_id", mediaID.String())
	req = withUser(req, user)
	rr := httptest.NewRecorder()

	app.deleteRatingHandler(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestDeleteRating_InvalidMediaID(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)

	req := httptest.NewRequest("DELETE", "/api/v1/ratings/not-a-uuid", nil)
	req.SetPathValue("media_id", "not-a-uuid")
	req = withUser(req, user)
	rr := httptest.NewRecorder()

	app.deleteRatingHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

// --- getRatingHandler tests ---

func TestGetRating_ExistingReturnsTranslatedRatingValue(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)
	mediaID := uuid.New()

	mock := app.models.Ratings.(*mockRatingModel)
	mock.ratings[ratingKey(user.ID, mediaID)] = data.UserRating{
		Rating: data.Rating{UserID: user.ID, MediaID: mediaID, RatingValue: 7},
	}

	req := httptest.NewRequest("GET", "/api/v1/ratings/"+mediaID.String(), nil)
	req.SetPathValue("media_id", mediaID.String())
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

func TestGetRating_NonExistent(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)
	mediaID := uuid.New()

	req := httptest.NewRequest("GET", "/api/v1/ratings/"+mediaID.String(), nil)
	req.SetPathValue("media_id", mediaID.String())
	req = withUser(req, user)
	rr := httptest.NewRecorder()

	app.getRatingHandler(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestGetRating_InvalidMediaID(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)

	req := httptest.NewRequest("GET", "/api/v1/ratings/not-a-uuid", nil)
	req.SetPathValue("media_id", "not-a-uuid")
	req = withUser(req, user)
	rr := httptest.NewRecorder()

	app.getRatingHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

// --- listRatingsHandler tests ---

func TestListRatings_ReturnsTranslatedUserRatings(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)

	mock := app.models.Ratings.(*mockRatingModel)
	for _, tc := range []struct {
		mediaID      uuid.UUID
		storedRating float64
	}{
		{mediaID: uuid.New(), storedRating: 10},
		{mediaID: uuid.New(), storedRating: 9},
		{mediaID: uuid.New(), storedRating: 7},
	} {
		mock.ratings[ratingKey(user.ID, tc.mediaID)] = data.UserRating{
			Rating: data.Rating{UserID: user.ID, MediaID: tc.mediaID, RatingValue: tc.storedRating},
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
	if len(ratings) != 3 {
		t.Errorf("expected 3 ratings, got %d", len(ratings))
	}

	values := sortedResponseRatingValues(t, ratings)
	want := []float64{3.5, 4.5, 5.0}
	for i := range want {
		if values[i] != want[i] {
			t.Fatalf("expected translated ratings %v, got %v", want, values)
		}
	}
}

func TestListRatings_EmptyForNewUser(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)

	req := httptest.NewRequest("GET", "/api/v1/ratings", nil)
	req = withUser(req, user)
	rr := httptest.NewRecorder()

	app.listRatingsHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

// --- listRatingsForMedia tests ---

func TestListRatingsForMedia_ReturnsTranslatedRatings(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)
	mediaID := uuid.New()

	mock := app.models.Ratings.(*mockRatingModel)
	for _, tc := range []struct {
		userID       uuid.UUID
		storedRating float64
	}{
		{userID: uuid.New(), storedRating: 8},
		{userID: uuid.New(), storedRating: 3},
	} {
		mock.ratings[ratingKey(tc.userID, mediaID)] = data.UserRating{
			Rating: data.Rating{UserID: tc.userID, MediaID: mediaID, RatingValue: tc.storedRating},
		}
	}

	req := httptest.NewRequest("GET", "/api/v1/media/"+mediaID.String()+"/ratings", nil)
	req.SetPathValue("media_id", mediaID.String())
	req = withUser(req, user)
	rr := httptest.NewRecorder()

	app.listRatingsForMedia(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(rr.Body.Bytes(), &resp)

	ratings := resp["ratings"].([]any)
	if len(ratings) != 2 {
		t.Errorf("expected 2 ratings, got %d", len(ratings))
	}

	values := sortedResponseRatingValues(t, ratings)
	want := []float64{1.5, 4.0}
	for i := range want {
		if values[i] != want[i] {
			t.Fatalf("expected translated ratings %v, got %v", want, values)
		}
	}
}

func TestListRatingsForMedia_InvalidMediaID(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)

	req := httptest.NewRequest("GET", "/api/v1/media/not-a-uuid/ratings", nil)
	req.SetPathValue("media_id", "not-a-uuid")
	req = withUser(req, user)
	rr := httptest.NewRecorder()

	app.listRatingsForMedia(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}
