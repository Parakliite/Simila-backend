package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/parakliite/simila/internal/data"
	"github.com/parakliite/simila/internal/database"
)

func TestUpsertReaction_ValidInput(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)
	ratingUserID := uuid.New()

	body := fmt.Sprintf(
		`{"rating_user_id":"%s","tmdb_id":550,"media_type":"movie","reaction":"great_pick"}`,
		ratingUserID,
	)
	req := httptest.NewRequest("PUT", "/api/v1/reactions", strings.NewReader(body))
	req = withUser(req, user)
	rr := httptest.NewRecorder()

	app.upsertReactionHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(rr.Body.Bytes(), &resp)

	reaction := resp["reaction"].(map[string]any)
	if reaction["reaction"] != "great_pick" {
		t.Fatalf("expected reaction %q, got %v", "great_pick", reaction["reaction"])
	}
	if _, ok := reaction["reactor"].(map[string]any); !ok {
		t.Fatalf("expected nested reactor object, got %T", reaction["reactor"])
	}
	if _, ok := reaction["rating_user"].(map[string]any); !ok {
		t.Fatalf("expected nested rating_user object, got %T", reaction["rating_user"])
	}
	if _, ok := reaction["media"].(map[string]any); !ok {
		t.Fatalf("expected nested media object, got %T", reaction["media"])
	}
	if _, ok := reaction["rating"].(map[string]any); !ok {
		t.Fatalf("expected nested rating object, got %T", reaction["rating"])
	}
}

func TestUpsertReaction_InvalidReaction(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)

	body := fmt.Sprintf(
		`{"rating_user_id":"%s","tmdb_id":550,"media_type":"movie","reaction":"love_it"}`,
		uuid.New(),
	)
	req := httptest.NewRequest("PUT", "/api/v1/reactions", strings.NewReader(body))
	req = withUser(req, user)
	rr := httptest.NewRecorder()

	app.upsertReactionHandler(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestListUserReactions_ReturnsReactions(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)
	targetUserID := uuid.New()

	mediaID, err := app.models.Movies.GetOrCreateMediaByTmdbID(context.Background(), 550, "movie")
	if err != nil {
		t.Fatalf("failed to get or create media: %v", err)
	}

	mock := app.models.Reactions.(*mockReactionModel)
	for i := 0; i < 2; i++ {
		reactorID := uuid.New()
		mock.reactions[reactionKey(reactorID, targetUserID, mediaID)] = data.Reaction{
			ID:            uuid.New(),
			ReactorUserID: reactorID,
			RatingUserID:  targetUserID,
			MediaID:       mediaID,
			Reaction:      database.ReactionTypeCurious,
		}
	}

	req := httptest.NewRequest("GET", "/api/v1/users/"+targetUserID.String()+"/reactions", nil)
	req.SetPathValue("user_id", targetUserID.String())
	req = withUser(req, user)
	rr := httptest.NewRecorder()

	app.listUserReactionsHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(rr.Body.Bytes(), &resp)

	reactions := resp["reactions"].([]any)
	if len(reactions) != 2 {
		t.Fatalf("expected 2 reactions, got %d", len(reactions))
	}

	reaction := reactions[0].(map[string]any)
	if _, ok := reaction["reactor"].(map[string]any); !ok {
		t.Fatalf("expected nested reactor object, got %T", reaction["reactor"])
	}
	if _, ok := reaction["rating_user"].(map[string]any); !ok {
		t.Fatalf("expected nested rating_user object, got %T", reaction["rating_user"])
	}
	if _, ok := reaction["media"].(map[string]any); !ok {
		t.Fatalf("expected nested media object, got %T", reaction["media"])
	}
	if _, ok := reaction["rating"].(map[string]any); !ok {
		t.Fatalf("expected nested rating object, got %T", reaction["rating"])
	}
}
