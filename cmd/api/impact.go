package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/deltron-fr/filmbox/server/internal/data"
	"github.com/google/uuid"
)

func (app *application) getImpact(w http.ResponseWriter, req *http.Request) {
	user := app.contextGetUser(req)

	ratingUserID, err := uuid.Parse(req.PathValue("user_id"))
	if err != nil {
		app.badRequestResponse(w, req, fmt.Errorf("invalid user_id"))
		return
	}

	if user.ID != ratingUserID {
		app.unauthorizedRequestResponse(w, req)
		return
	}

	qs := req.URL.Query()
	limit := app.readInt(qs, "limit", 20)
	if limit < 1 || limit > 70 {
		limit = 20
	}

	cursorStr := app.readString(qs, "cursor", "")
	cursorCreatedAt := time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)
	cursorReactorUserID := maxCursorUUID

	if cursorStr != "" {
		cursorReactorUserID, cursorCreatedAt, err = app.decodeCursor(cursorStr)
		if err != nil {
			app.badRequestResponse(w, req, err)
			return
		}
	}

	reactions, err := app.models.Reactions.GetUserReactionsForTargetUser(
		req.Context(),
		ratingUserID,
		cursorCreatedAt,
		cursorReactorUserID, int32(limit))
	if err != nil {
		app.serverErrorResponse(w, req, err)
		return
	}

	var nextCursor *string
	if len(reactions) > limit {
		reactions = reactions[:limit]
		last := reactions[len(reactions)-1]
		c := app.encodeCursor(last.Reactor.ID, last.CreatedAt)
		nextCursor = &c
	}

	type ReactionGroupkey struct {
		RatingUserID uuid.UUID
		MediaID      uuid.UUID
		ReactionType string
	}

	groups := make(map[ReactionGroupkey]int)
	groupedReactions := []data.ReactionAggregated{}

	for _, reaction := range reactions {
		key := ReactionGroupkey{
			RatingUserID: reaction.RatingUser.ID,
			MediaID:      reaction.Media.ID,
			ReactionType: string(reaction.Reaction),
		}

		idx, exists := groups[key]
		if !exists {
			newReaction := data.ReactionAggregated{
				Type:       data.SingleReaction,
				ID:         reaction.ID,
				Reaction:   reaction.Reaction,
				CreatedAt:  reaction.CreatedAt,
				UpdatedAt:  reaction.UpdatedAt,
				RatingUser: reaction.RatingUser,
				Media:      reaction.Media,
				Rating:     reaction.Rating,
				Count:      1,
				Reactors:   []data.ReactionUser{reaction.Reactor},
			}

			groupedReactions = append(groupedReactions, newReaction)
			groups[key] = len(groupedReactions) - 1
		} else {
			groupedReactions[idx].Reactors = append(groupedReactions[idx].Reactors, reaction.Reactor)
			groupedReactions[idx].Type = data.GroupedReaction
			groupedReactions[idx].Count++
		}
	}

	app.writeJSON(w, http.StatusOK, envelope{"impact": groupedReactions, "next_cursor": nextCursor}, nil)
}
