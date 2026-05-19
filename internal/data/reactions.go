package data

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/deltron-fr/filmbox/server/internal/database"
	"github.com/deltron-fr/filmbox/server/internal/validator"
	"github.com/google/uuid"
)

type ReactionModel struct {
	DB *sql.DB
	q  *database.Queries
}

type Reaction struct {
	ID            uuid.UUID             `json:"id"`
	ReactorUserID uuid.UUID             `json:"reactor_user_id"`
	RatingUserID  uuid.UUID             `json:"rating_user_id"`
	MediaID       uuid.UUID             `json:"media_id"`
	Reaction      database.ReactionType `json:"reaction"`
	CreatedAt     time.Time             `json:"created_at"`
	UpdatedAt     time.Time             `json:"updated_at"`
}

func (r ReactionModel) UpsertReaction(ctx context.Context, reaction Reaction) (Reaction, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	row, err := r.q.UpsertUserReaction(ctx, database.UpsertUserReactionParams{
		ReactorUserID: reaction.ReactorUserID,
		RatingUserID:  reaction.RatingUserID,
		MediaID:       reaction.MediaID,
		Reaction:      reaction.Reaction,
	})
	if err != nil {
		return Reaction{}, fmt.Errorf("upsert user reaction: %w", err)
	}

	return Reaction{
		ID:            row.ID,
		ReactorUserID: row.ReactorUserID,
		RatingUserID:  row.RatingUserID,
		MediaID:       row.MediaID,
		Reaction:      row.Reaction,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}, nil
}

func (r ReactionModel) DeleteReaction(ctx context.Context, reactorUserID, ratingUserID, mediaID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	row, err := r.q.DeleteUserReaction(ctx, database.DeleteUserReactionParams{
		ReactorUserID: reactorUserID,
		RatingUserID:  ratingUserID,
		MediaID:       mediaID,
	})
	if err != nil {
		return fmt.Errorf("delete user reaction: %w", err)
	}

	if row == 0 {
		return ErrRecordNotFound
	}

	return nil
}

func (r ReactionModel) GetReactionCountForRating(ctx context.Context, ratingUserID, mediaID uuid.UUID) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	row, err := r.q.GetReactionCountForRating(ctx, database.GetReactionCountForRatingParams{
		RatingUserID: ratingUserID,
		MediaID:      mediaID,
	})
	if err != nil {
		return 0, fmt.Errorf("get reaction count for rating: %w", err)
	}

	return row, nil
}

func (r ReactionModel) GetUserReactionsForTargetUser(
	ctx context.Context,
	ratingUserID uuid.UUID,
	cursorCreatedAt time.Time,
	cursorReactorUserID uuid.UUID,
	limit int32,
) ([]Reaction, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	rows, err := r.q.GetUserReactionsForTargetUser(ctx, database.GetUserReactionsForTargetUserParams{
		RatingUserID:        ratingUserID,
		CursorCreatedAt:     cursorCreatedAt,
		CursorReactorUserID: cursorReactorUserID,
		Limit:               limit,
	})
	if err != nil {
		return nil, fmt.Errorf("get user reactions for target user: %w", err)
	}

	var reactions []Reaction

	for _, row := range rows {
		reactions = append(reactions, Reaction{
			ID:            row.ID,
			ReactorUserID: row.ReactorUserID,
			RatingUserID:  row.RatingUserID,
			MediaID:       row.MediaID,
			Reaction:      row.Reaction,
			CreatedAt:     row.CreatedAt,
			UpdatedAt:     row.UpdatedAt,
		})
	}

	return reactions, nil
}

func ValidateReactionType(v *validator.Validator, reaction string) {
	v.Check(reaction != "", "reaction", "must be provided")
	v.Check(
		validator.In(
			reaction,
			string(database.ReactionTypeWatchedBecauseOfYou),
			string(database.ReactionTypeGreatPick),
			string(database.ReactionTypeCurious),
			string(database.ReactionTypeHotTake),
		),
		"reaction",
		"must be a valid reaction type",
	)
}
