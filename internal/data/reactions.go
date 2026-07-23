package data

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/parakliite/simila/internal/database"
	"github.com/parakliite/simila/internal/validator"
)

type ReactionModel struct {
	DB *sql.DB
	q  *database.Queries
}

type ReactionAggType string

const (
	GroupedReaction ReactionAggType = "grouped"
	SingleReaction  ReactionAggType = "single"
)

type Reaction struct {
	ID            uuid.UUID             `json:"id"`
	ReactorUserID uuid.UUID             `json:"reactor_user_id"`
	RatingUserID  uuid.UUID             `json:"rating_user_id"`
	MediaID       uuid.UUID             `json:"media_id"`
	Reaction      database.ReactionType `json:"reaction"`
	CreatedAt     time.Time             `json:"created_at"`
	UpdatedAt     time.Time             `json:"updated_at"`
}

type ReactionUser struct {
	ID                uuid.UUID `json:"id"`
	Name              string    `json:"name"`
	ProfilePictureURL string    `json:"profile_picture_url,omitempty"`
}

type ReactionMedia struct {
	ID            uuid.UUID `json:"id"`
	TmdbID        int32     `json:"tmdb_id"`
	Title         string    `json:"title"`
	OriginalTitle string    `json:"original_title"`
	PosterPath    string    `json:"poster_path"`
	BackdropPath  string    `json:"backdrop_path"`
	MediaType     string    `json:"media_type"`
}

type ReactionRating struct {
	RatingValue float64    `json:"rating_value"`
	WatchedDate *time.Time `json:"watched_date"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type ReactionWithDetails struct {
	ID        uuid.UUID             `json:"id"`
	Reaction  database.ReactionType `json:"reaction"`
	CreatedAt time.Time             `json:"created_at"`
	UpdatedAt time.Time             `json:"updated_at"`

	Reactor    ReactionUser   `json:"reactor"`
	RatingUser ReactionUser   `json:"rating_user"`
	Media      ReactionMedia  `json:"media"`
	Rating     ReactionRating `json:"rating"`
}

type ReactionAggregated struct {
	Type      ReactionAggType       `json:"type"`
	ID        uuid.UUID             `json:"id"`
	Reaction  database.ReactionType `json:"reaction"`
	CreatedAt time.Time             `json:"created_at"`
	UpdatedAt time.Time             `json:"updated_at"`

	Reactors   []ReactionUser `json:"reactors,omitempty"`
	RatingUser ReactionUser   `json:"rating_user"`
	Media      ReactionMedia  `json:"media"`
	Rating     ReactionRating `json:"rating"`
	Count      int            `json:"count"`
}

func (r ReactionModel) UpsertReaction(ctx context.Context, reaction Reaction) (ReactionWithDetails, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	row, err := r.q.UpsertUserReaction(ctx, database.UpsertUserReactionParams{
		ReactorUserID: reaction.ReactorUserID,
		RatingUserID:  reaction.RatingUserID,
		MediaID:       reaction.MediaID,
		Reaction:      reaction.Reaction,
	})
	if err != nil {
		return ReactionWithDetails{}, fmt.Errorf("upsert user reaction: %w", err)
	}

	return toReactionWithDetails(row), nil
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
) ([]ReactionWithDetails, error) {
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

	var reactions []ReactionWithDetails

	for _, row := range rows {
		reactions = append(reactions, toReactionWithDetailsFromList(row))
	}

	return reactions, nil
}

func toReactionWithDetails(row database.UpsertUserReactionRow) ReactionWithDetails {
	return ReactionWithDetails{
		ID:        row.ID,
		Reaction:  row.Reaction,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
		Reactor: ReactionUser{
			ID:                row.ReactorUserID,
			Name:              row.ReactorName,
			ProfilePictureURL: row.ReactorProfilePictureUrl,
		},
		RatingUser: ReactionUser{
			ID:                row.RatingUserID,
			Name:              row.RatingUserName,
			ProfilePictureURL: row.RatingUserProfilePictureUrl,
		},
		Media: ReactionMedia{
			ID:            row.MediaID,
			TmdbID:        row.TmdbID,
			Title:         row.Title,
			OriginalTitle: row.OriginalTitle,
			PosterPath:    row.PosterPath,
			BackdropPath:  row.BackdropPath,
			MediaType:     row.MediaType,
		},
		Rating: ReactionRating{
			RatingValue: translateRatingValueOutbound(row.RatingValue),
			WatchedDate: nullTimeToPtr(row.WatchedDate),
			CreatedAt:   row.RatingCreatedAt,
			UpdatedAt:   row.RatingUpdatedAt,
		},
	}
}

func toReactionWithDetailsFromList(row database.GetUserReactionsForTargetUserRow) ReactionWithDetails {
	return ReactionWithDetails{
		ID:        row.ID,
		Reaction:  row.Reaction,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
		Reactor: ReactionUser{
			ID:                row.ReactorUserID,
			Name:              row.ReactorName,
			ProfilePictureURL: row.ReactorProfilePictureUrl,
		},
		RatingUser: ReactionUser{
			ID:                row.RatingUserID,
			Name:              row.RatingUserName,
			ProfilePictureURL: row.RatingUserProfilePictureUrl,
		},
		Media: ReactionMedia{
			ID:            row.MediaID,
			TmdbID:        row.TmdbID,
			Title:         row.Title,
			OriginalTitle: row.OriginalTitle,
			PosterPath:    row.PosterPath,
			BackdropPath:  row.BackdropPath,
			MediaType:     row.MediaType,
		},
		Rating: ReactionRating{
			RatingValue: translateRatingValueOutbound(row.RatingValue),
			WatchedDate: nullTimeToPtr(row.WatchedDate),
			CreatedAt:   row.RatingCreatedAt,
			UpdatedAt:   row.RatingUpdatedAt,
		},
	}
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
