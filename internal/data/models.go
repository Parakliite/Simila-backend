package data

import (
	"context"
	"database/sql"
	"time"

	"github.com/deltron-fr/filmbox/server/internal/database"
	"github.com/google/uuid"
)

type MediaQuerier interface {
	GetMedia(ctx context.Context, id int32, mediaType string) (*Media, error)
	InsertMedia(ctx context.Context, media *Media) error
}

type UserQuerier interface {
	Insert(ctx context.Context, user *User) error
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	UpdateUser(ctx context.Context, user *User) error
	GetForToken(ctx context.Context, tokenScope, tokenPlaintext string) (*User, error)
}

type TokenQuerier interface {
	New(ctx context.Context, userID uuid.UUID, ttl time.Duration, scope string) (*Token, error)
	Insert(ctx context.Context, token *Token) error
	DeleteAllForUser(ctx context.Context, scope string, userID uuid.UUID) error
}

type RatingQuerier interface {
	UpsertUserRating(ctx context.Context, rating Rating) (Rating, error)
	GetUsersRatings(ctx context.Context, createdAt time.Time, limit int32, userID, mediaID uuid.UUID,
	) ([]UserRating, error)
	GetUsersRating(ctx context.Context, userID, mediaID uuid.UUID,
	) (UserRating, error)
	GetAllRatingsForSingleMedia(ctx context.Context, createdAt time.Time, mediaID, userID uuid.UUID, limit int32,
	) ([]UserRating, error)
	GetRandomMedia(ctx context.Context, userID uuid.UUID, limit int32) ([]Media, error)
	DeleteUserRating(ctx context.Context, userID, mediaID uuid.UUID) error
}

type MatchQuerier interface {
	GetSimilarities(targetUserID uuid.UUID, ratings []Rating, indexMap map[uuid.UUID]int) (map[uuid.UUID]float64, error)
}

type ReactionQuerier interface {
	UpsertReaction(ctx context.Context, reaction Reaction) (Reaction, error)
	DeleteReaction(ctx context.Context, reactorUserID, ratingUserID, mediaID uuid.UUID) error
	GetReactionCountForRating(ctx context.Context, ratingUserID, mediaID uuid.UUID) (int64, error)
	GetUserReactionsForTargetUser(
		ctx context.Context,
		ratingUserID uuid.UUID,
		cursorCreatedAt time.Time,
		cursorReactorUserID uuid.UUID,
		limit int32,
	) ([]Reaction, error)
}

type WatchlistQuerier interface {
	InsertMediaToWatchlist(ctx context.Context, watchlist Watchlist) (Watchlist, error)
	UpdateWatchlistItemStatus(
		ctx context.Context,
		userID, mediaID uuid.UUID,
		status string,
	) (Watchlist, error)
	DeleteMediaFromWatchlist(ctx context.Context, userID, mediaID uuid.UUID) error
	GetAllItemsInWatchlist(
		ctx context.Context,
		createdAt time.Time,
		limit int32, status string,
		userID, mediaID uuid.UUID,
	) ([]UserWatchlistItem, error)
}

type DiscoveryQuerier interface {
	GetDiscoveryHistoryExpiry(ctx context.Context, userID, mediaID uuid.UUID) (time.Time, error)
	SetDiscoveryHistoryExpiry(ctx context.Context, userID, mediaID uuid.UUID, eligibleAt time.Time) error
}

type Models struct {
	Helper    *database.Queries
	Movies    MediaQuerier
	Users     UserQuerier
	Tokens    TokenQuerier
	Ratings   RatingQuerier
	Matches   MatchQuerier
	Reactions ReactionQuerier
	Watchlist WatchlistQuerier
	Discovery DiscoveryQuerier
}

func NewModels(db *sql.DB) Models {
	dbQueries := database.New(db)

	return Models{
		Helper: dbQueries,
		Movies: MediaModel{
			DB: db,
			q:  dbQueries,
		},
		Users: UserModel{
			DB: db,
			q:  dbQueries,
		},
		Tokens: TokenModel{
			DB: db,
			q:  dbQueries,
		},
		Ratings: RatingModel{
			DB: db,
			q:  dbQueries,
		},
		Matches: MatchModel{
			DB: db,
			q:  dbQueries,
		},
		Reactions: ReactionModel{
			DB: db,
			q:  dbQueries,
		},
		Watchlist: WatchlistModel{
			DB: db,
			q:  dbQueries,
		},
		Discovery: DiscoveryModel{
			DB: db,
			q:  dbQueries,
		},
	}
}
