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

func newDiscoveryMedia(title string) data.Media {
	return data.Media{
		ID:            uuid.New(),
		TmdbID:        100,
		Title:         title,
		OriginalTitle: title,
		PosterPath:    "/" + title + ".jpg",
		BackdropPath:  "/" + title + "-backdrop.jpg",
		MediaType:     "movie",
		Genre:         []data.Genre{},
	}
}

type discoveryResponseMedia struct {
	ID    uuid.UUID `json:"_id"`
	Title string    `json:"title"`
}

func getDiscoveryResponse(t *testing.T, app *application, user *data.User) ([]discoveryResponseMedia, *httptest.ResponseRecorder) {
	t.Helper()

	req := httptest.NewRequest("GET", "/api/v1/discovery/rating-candidates", nil)
	req = withUser(req, user)
	rr := httptest.NewRecorder()

	app.ListRandomMedia(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Media []discoveryResponseMedia `json:"media"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	return resp.Media, rr
}

func TestDiscoveryRatingCandidates_BasicDiscoveryReturnsMediaAndRecordsHistory(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Jerry", "jerry@example.com", "password123", true)
	media := newDiscoveryMedia("The Matrix")

	app.models.Ratings.(*mockRatingModel).randomMedia = []data.Media{media}

	before := time.Now()
	got, _ := getDiscoveryResponse(t, app, user)
	after := time.Now()

	if len(got) != 1 {
		t.Fatalf("expected 1 media candidate, got %d", len(got))
	}
	if got[0].ID != media.ID {
		t.Fatalf("expected media ID %s, got %s", media.ID, got[0].ID)
	}

	expiry, err := app.models.Discovery.(*mockDiscoveryModel).GetDiscoveryHistoryExpiry(
		t.Context(),
		user.ID,
		media.ID,
	)
	if err != nil {
		t.Fatalf("expected discovery history to be recorded: %v", err)
	}
	if expiry.Before(before.Add(7*24*time.Hour)) || expiry.After(after.Add(7*24*time.Hour)) {
		t.Fatalf("expected expiry about 7 days from now, got %s", expiry)
	}
}

func TestDiscoveryRatingCandidates_DoesNotReturnRatedMedia(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Jerry", "jerry@example.com", "password123", true)
	rated := newDiscoveryMedia("The Lord of the Rings")
	unrated := newDiscoveryMedia("Transformers")

	mockRatings := app.models.Ratings.(*mockRatingModel)
	mockRatings.randomMedia = []data.Media{rated, unrated}
	mockRatings.ratings[ratingKey(user.ID, rated.ID)] = data.UserRating{
		Rating: data.Rating{UserID: user.ID, MediaID: rated.ID, RatingValue: 8},
	}

	got, _ := getDiscoveryResponse(t, app, user)

	if len(got) != 1 {
		t.Fatalf("expected 1 media candidate, got %d", len(got))
	}
	if got[0].ID == rated.ID {
		t.Fatal("rated media was returned as a discovery candidate")
	}
	if got[0].ID != unrated.ID {
		t.Fatalf("expected unrated media ID %s, got %s", unrated.ID, got[0].ID)
	}
}

func TestDiscoveryRatingCandidates_ExhaustedDiscoverySetReturnsEmptyList(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Jerry", "jerry@example.com", "password123", true)
	mockRatings := app.models.Ratings.(*mockRatingModel)
	mockDiscovery := app.models.Discovery.(*mockDiscoveryModel)
	futureExpiry := time.Now().Add(24 * time.Hour)

	for i := 0; i < 3; i++ {
		media := newDiscoveryMedia(fmt.Sprintf("Seen Candidate %d", i))
		mockRatings.randomMedia = append(mockRatings.randomMedia, media)
		mockDiscovery.history[discoveryKey(user.ID, media.ID)] = futureExpiry
	}

	got, _ := getDiscoveryResponse(t, app, user)

	if len(got) != 0 {
		t.Fatalf("expected no media candidates, got %d", len(got))
	}
}

func TestDiscoveryRatingCandidates_ExpiredHistoryReturnsMediaAndRefreshesHistory(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Jerry", "jerry@example.com", "password123", true)
	media := newDiscoveryMedia("Game of Thrones")
	mockRatings := app.models.Ratings.(*mockRatingModel)
	mockDiscovery := app.models.Discovery.(*mockDiscoveryModel)
	expiredAt := time.Now().Add(-time.Hour)

	mockRatings.randomMedia = []data.Media{media}
	mockDiscovery.history[discoveryKey(user.ID, media.ID)] = expiredAt

	got, _ := getDiscoveryResponse(t, app, user)

	if len(got) != 1 {
		t.Fatalf("expected 1 media candidate, got %d", len(got))
	}
	if got[0].ID != media.ID {
		t.Fatalf("expected media ID %s, got %s", media.ID, got[0].ID)
	}

	refreshedAt := mockDiscovery.history[discoveryKey(user.ID, media.ID)]
	if !refreshedAt.After(expiredAt) {
		t.Fatalf("expected discovery history to be refreshed, got %s", refreshedAt)
	}
}
