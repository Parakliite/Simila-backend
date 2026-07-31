package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/parakliite/simila/internal/data"
)

func TestGetUserRecommendations_ReturnsRecommendationsForMatch(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)
	targetUserID := uuid.New()

	mockMatches := app.models.Matches.(*mockMatchModel)
	mockMatches.matchExists = true

	mockRecommendations := app.models.Recommendations.(*mockRecommendationModel)
	mockRecommendations.recommendations = data.Recommendations{
		Media: []data.Media{
			{
				ID:        uuid.New(),
				TmdbID:    680,
				Title:     "Pulp Fiction",
				MediaType: "movie",
			},
		},
	}

	req := httptest.NewRequest("GET", "/api/v1/recommendations/"+targetUserID.String()+"?limit=1", nil)
	req.SetPathValue("user_id", targetUserID.String())
	req = withUser(req, user)
	rr := httptest.NewRecorder()

	app.getUserRecommendations(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if mockRecommendations.lastUserID != user.ID {
		t.Fatalf("expected user ID %s, got %s", user.ID, mockRecommendations.lastUserID)
	}
	if mockRecommendations.lastTargetUserID != targetUserID {
		t.Fatalf("expected target user ID %s, got %s", targetUserID, mockRecommendations.lastTargetUserID)
	}
	if mockRecommendations.lastThreshold != 7 {
		t.Fatalf("expected threshold 7, got %d", mockRecommendations.lastThreshold)
	}
	if mockRecommendations.lastLimit != 1 {
		t.Fatalf("expected limit 1, got %d", mockRecommendations.lastLimit)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}

	recommendations := resp["recommendations"].(map[string]any)
	media := recommendations["media"].([]any)
	if len(media) != 1 {
		t.Fatalf("expected 1 recommendation, got %d", len(media))
	}
}

func TestGetUserRecommendations_InvalidTargetUserID(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)

	req := httptest.NewRequest("GET", "/api/v1/recommendations/not-a-uuid", nil)
	req.SetPathValue("user_id", "not-a-uuid")
	req = withUser(req, user)
	rr := httptest.NewRecorder()

	app.getUserRecommendations(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestGetUserRecommendations_SelfRequest(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)

	req := httptest.NewRequest("GET", "/api/v1/recommendations/"+user.ID.String(), nil)
	req.SetPathValue("user_id", user.ID.String())
	req = withUser(req, user)
	rr := httptest.NewRecorder()

	app.getUserRecommendations(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestGetUserRecommendations_NotMatchedMapsToNotFound(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)
	targetUserID := uuid.New()

	req := httptest.NewRequest("GET", "/api/v1/recommendations/"+targetUserID.String(), nil)
	req.SetPathValue("user_id", targetUserID.String())
	req = withUser(req, user)
	rr := httptest.NewRecorder()

	app.getUserRecommendations(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestGetUserRecommendations_MatchCheckErrorMapsToServerError(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)
	targetUserID := uuid.New()
	app.models.Matches.(*mockMatchModel).checkMatchErr = fmt.Errorf("match check failed")

	req := httptest.NewRequest("GET", "/api/v1/recommendations/"+targetUserID.String(), nil)
	req.SetPathValue("user_id", targetUserID.String())
	req = withUser(req, user)
	rr := httptest.NewRecorder()

	app.getUserRecommendations(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestGetUserRecommendations_ModelErrorMapsToServerError(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)
	targetUserID := uuid.New()
	app.models.Matches.(*mockMatchModel).matchExists = true
	app.models.Recommendations.(*mockRecommendationModel).err = fmt.Errorf("recommendations unavailable")

	req := httptest.NewRequest("GET", "/api/v1/recommendations/"+targetUserID.String(), nil)
	req.SetPathValue("user_id", targetUserID.String())
	req = withUser(req, user)
	rr := httptest.NewRecorder()

	app.getUserRecommendations(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", rr.Code, rr.Body.String())
	}
}
