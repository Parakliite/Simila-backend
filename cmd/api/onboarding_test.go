package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/parakliite/simila/internal/data"
)

func TestListOnboardingMediaReturnsTwentyFivePopularCandidates(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)
	mock := app.models.Onboarding.(*mockOnboardingModel)
	for i := 0; i < int(onboardingMediaLimit); i++ {
		mock.media = append(mock.media, data.Media{
			ID:        uuid.New(),
			TmdbID:    int32(i + 1),
			Title:     "Title",
			MediaType: "movie",
		})
	}

	req := withUser(httptest.NewRequest(http.MethodGet, "/api/v1/onboarding/media", nil), user)
	rr := httptest.NewRecorder()

	app.listOnboardingMediaHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if mock.lastThreshold != onboardingPopularityThreshold {
		t.Fatalf("expected popularity threshold %v, got %v", onboardingPopularityThreshold, mock.lastThreshold)
	}
	if mock.lastLimit != onboardingMediaLimit {
		t.Fatalf("expected media limit %d, got %d", onboardingMediaLimit, mock.lastLimit)
	}

	var response struct {
		Media []json.RawMessage `json:"media"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(response.Media) != int(onboardingMediaLimit) {
		t.Fatalf("expected %d media items, got %d", onboardingMediaLimit, len(response.Media))
	}
}

func TestListOnboardingMediaReturnsServiceUnavailableWhenPoolIsTooSmall(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)
	mock := app.models.Onboarding.(*mockOnboardingModel)
	mock.media = make([]data.Media, int(onboardingMediaLimit)-1)

	req := withUser(httptest.NewRequest(http.MethodGet, "/api/v1/onboarding/media", nil), user)
	rr := httptest.NewRecorder()

	app.listOnboardingMediaHandler(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestSubmitOnboardingRatingsCompletesOnboarding(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)
	updatedUser := *user
	updatedUser.Onboarded = true
	mock := app.models.Onboarding.(*mockOnboardingModel)
	mock.completeUser = &updatedUser

	ratings := make([]map[string]any, 0, onboardingRatingsMin)
	for i := 0; i < onboardingRatingsMin; i++ {
		ratings = append(ratings, map[string]any{
			"tmdb_id":      i + 1,
			"media_type":   "movie",
			"rating_value": 4.0,
		})
	}
	body, err := json.Marshal(map[string]any{"ratings": ratings})
	if err != nil {
		t.Fatal(err)
	}

	req := withUser(httptest.NewRequest(
		http.MethodPost,
		"/api/v1/onboarding/ratings",
		strings.NewReader(string(body)),
	), user)
	rr := httptest.NewRecorder()

	app.submitOnboardingRatingsHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if mock.completeCalls != 1 {
		t.Fatalf("expected one completion call, got %d", mock.completeCalls)
	}
	if len(mock.lastInputRatings) != onboardingRatingsMin {
		t.Fatalf("expected %d ratings, got %d", onboardingRatingsMin, len(mock.lastInputRatings))
	}

	var response struct {
		User data.User `json:"user"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !response.User.Onboarded {
		t.Fatal("expected response user to be onboarded")
	}
}

func TestSubmitOnboardingRatingsRejectsTooFewRatings(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)
	body := `{"ratings":[{"tmdb_id":1,"media_type":"movie","rating_value":4}]}`

	req := withUser(httptest.NewRequest(
		http.MethodPost,
		"/api/v1/onboarding/ratings",
		strings.NewReader(body),
	), user)
	rr := httptest.NewRecorder()

	app.submitOnboardingRatingsHandler(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", rr.Code, rr.Body.String())
	}
	if calls := app.models.Onboarding.(*mockOnboardingModel).completeCalls; calls != 0 {
		t.Fatalf("expected no completion call, got %d", calls)
	}
}

func TestSubmitOnboardingRatingsRejectsDuplicateMedia(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)
	ratings := make([]map[string]any, 0, onboardingRatingsMin)
	for i := 0; i < onboardingRatingsMin; i++ {
		tmdbID := i + 1
		if i == onboardingRatingsMin-1 {
			tmdbID = 1
		}
		ratings = append(ratings, map[string]any{
			"tmdb_id":      tmdbID,
			"media_type":   "movie",
			"rating_value": 4.0,
		})
	}
	body, err := json.Marshal(map[string]any{"ratings": ratings})
	if err != nil {
		t.Fatal(err)
	}

	req := withUser(httptest.NewRequest(
		http.MethodPost,
		"/api/v1/onboarding/ratings",
		strings.NewReader(string(body)),
	), user)
	rr := httptest.NewRecorder()

	app.submitOnboardingRatingsHandler(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", rr.Code, rr.Body.String())
	}
	if calls := app.models.Onboarding.(*mockOnboardingModel).completeCalls; calls != 0 {
		t.Fatalf("expected no completion call, got %d", calls)
	}
}
