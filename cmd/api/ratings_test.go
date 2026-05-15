package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/deltron-fr/filmbox/server/internal/data"
	"github.com/google/uuid"
)

// --- upsertRatingHandler tests ---

func TestUpsertRating_ValidRating(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)
	mediaID := uuid.New()

	body := fmt.Sprintf(`{"media_id":"%s","rating_value":8}`, mediaID)
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
	if int(rating["rating_value"].(float64)) != 8 {
		t.Errorf("expected rating_value 8, got %v", rating["rating_value"])
	}
	if rating["media_id"] != mediaID.String() {
		t.Errorf("expected media_id %s, got %v", mediaID, rating["media_id"])
	}
}

func TestUpsertRating_UpdatesExisting(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)
	mediaID := uuid.New()

	body := fmt.Sprintf(`{"media_id":"%s","rating_value":5}`, mediaID)
	req := httptest.NewRequest("PUT", "/api/v1/ratings", strings.NewReader(body))
	req = withUser(req, user)
	rr := httptest.NewRecorder()
	app.upsertRatingHandler(rr, req)

	body = fmt.Sprintf(`{"media_id":"%s","rating_value":9}`, mediaID)
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
	if int(rating["rating_value"].(float64)) != 9 {
		t.Errorf("expected updated rating_value 9, got %v", rating["rating_value"])
	}
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

	body := fmt.Sprintf(`{"media_id":"%s","rating_value":11}`, uuid.New())
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

	body := fmt.Sprintf(`{"media_id":"%s","rating_value":0}`, uuid.New())
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
		Rating: data.Rating{UserID: user.ID, MediaID: mediaID, RatingValue: 7},
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

func TestGetRating_Existing(t *testing.T) {
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
	if int(r["rating_value"].(float64)) != 7 {
		t.Errorf("expected rating_value 7, got %v", r["rating_value"])
	}
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

func TestListRatings_ReturnsUserRatings(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)

	mock := app.models.Ratings.(*mockRatingModel)
	for i := 0; i < 3; i++ {
		mediaID := uuid.New()
		mock.ratings[ratingKey(user.ID, mediaID)] = data.UserRating{
			Rating: data.Rating{UserID: user.ID, MediaID: mediaID, RatingValue: int32(i + 5)},
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

func TestListRatingsForMedia_ReturnsRatings(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)
	mediaID := uuid.New()

	mock := app.models.Ratings.(*mockRatingModel)
	for i := 0; i < 2; i++ {
		uid := uuid.New()
		mock.ratings[ratingKey(uid, mediaID)] = data.UserRating{
			Rating: data.Rating{UserID: uid, MediaID: mediaID, RatingValue: int32(i + 6)},
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
