package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/deltron-fr/filmbox/server/internal/data"
	"github.com/deltron-fr/filmbox/server/internal/database"
	"github.com/google/uuid"
)

func TestUpsertReaction_ValidInput(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)
	ratingUserID := uuid.New()
	mediaID := uuid.New()

	body := fmt.Sprintf(
		`{"rating_user_id":"%s","media_id":"%s","reaction":"great_pick"}`,
		ratingUserID,
		mediaID,
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
}

func TestUpsertReaction_InvalidReaction(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)

	body := fmt.Sprintf(
		`{"rating_user_id":"%s","media_id":"%s","reaction":"love_it"}`,
		uuid.New(),
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

func TestDeleteReaction_Existing(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)
	ratingUserID := uuid.New()
	mediaID := uuid.New()

	mock := app.models.Reactions.(*mockReactionModel)
	mock.reactions[reactionKey(user.ID, ratingUserID, mediaID)] = data.Reaction{
		ID:            uuid.New(),
		ReactorUserID: user.ID,
		RatingUserID:  ratingUserID,
		MediaID:       mediaID,
		Reaction:      database.ReactionTypeGreatPick,
	}

	req := httptest.NewRequest("DELETE", "/api/v1/reactions/"+ratingUserID.String()+"/"+mediaID.String(), nil)
	req.SetPathValue("rating_user_id", ratingUserID.String())
	req.SetPathValue("media_id", mediaID.String())
	req = withUser(req, user)
	rr := httptest.NewRecorder()

	app.deleteReactionHandler(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestListUserReactions_ReturnsReactions(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)
	targetUserID := uuid.New()
	mediaID := uuid.New()

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
}

func TestGetReactionCountForRating_ReturnsCount(t *testing.T) {
	app := newTestApp("")
	user := newTestUser("Alice", "alice@example.com", "password123", true)
	ratingUserID := uuid.New()
	mediaID := uuid.New()

	mock := app.models.Reactions.(*mockReactionModel)
	for i := 0; i < 2; i++ {
		reactorID := uuid.New()
		mock.reactions[reactionKey(reactorID, ratingUserID, mediaID)] = data.Reaction{
			ID:            uuid.New(),
			ReactorUserID: reactorID,
			RatingUserID:  ratingUserID,
			MediaID:       mediaID,
			Reaction:      database.ReactionTypeHotTake,
		}
	}

	req := httptest.NewRequest(
		"GET",
		"/api/v1/ratings/"+ratingUserID.String()+"/"+mediaID.String()+"/reactions/count",
		nil,
	)
	req.SetPathValue("rating_user_id", ratingUserID.String())
	req.SetPathValue("media_id", mediaID.String())
	req = withUser(req, user)
	rr := httptest.NewRecorder()

	app.getReactionCountForRatingHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if int(resp["reaction_count"].(float64)) != 2 {
		t.Fatalf("expected reaction_count 2, got %v", resp["reaction_count"])
	}
}
