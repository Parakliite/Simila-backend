package data

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/parakliite/simila/internal/database"
)

type MediaQuerier interface {
	GetMedia(ctx context.Context, id int32, mediaType string) (*Media, error)
	GetOrCreateMediaByTmdbID(ctx context.Context, tmdbID int32, mediaType string) (uuid.UUID, error)
	InsertMedia(ctx context.Context, media *Media) error
}

type UserQuerier interface {
	Insert(ctx context.Context, user *User) error
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	UpdateUser(ctx context.Context, user *User) error
	GetUserForToken(ctx context.Context, tokenScope, tokenPlaintext string) (*User, error)
	SearchUsers(ctx context.Context, query string, limit int32) ([]User, error)
	GetAllUsers(ctx context.Context) ([]uuid.UUID, error)
	RevokeAllTokensForSession(ctx context.Context, sessionID uuid.NullUUID, scope string) error
}

type TokenQuerier interface {
	New(
		ctx context.Context,
		userID uuid.UUID,
		sessionID uuid.NullUUID,
		ttl time.Duration,
		scope string,
	) (*Token, error)
	Insert(ctx context.Context, token *Token) error
	GetUserIDFromToken(
		ctx context.Context,
		scope string,
		tokenHash []byte,
	) (userID uuid.UUID, sessionID uuid.UUID, err error)
	RevokePreviousToken(ctx context.Context, scope string, tokenHash []byte) error
	RevokeAllPreviousTokens(ctx context.Context, scope string, userID uuid.UUID) error
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
	CalculateSimilarities(
		ctx context.Context,
		targetUserID uuid.UUID,
		ratings []Rating,
		indexMap map[uuid.UUID]int,
	) error
	MapMediaToIndex(ctx context.Context) (map[uuid.UUID]int, error)
	GetAllUserRatingsForMatches(ctx context.Context, userID uuid.UUID) ([]Rating, error)
	GetUserMatches(
		ctx context.Context,
		targetUserID uuid.UUID,
		limit int32,
		cursorOtherUserID uuid.NullUUID,
		cursorScore sql.NullFloat64,
		cursorSharedMediaCount sql.NullInt32,
		cursorLastRecalculatedAt sql.NullTime,
	) ([]Match, error)
	CheckMatchExists(
		ctx context.Context,
		userID, otherUserID uuid.UUID,
	) (bool, error)
}

type ReactionQuerier interface {
	UpsertReaction(ctx context.Context, reaction Reaction) (ReactionWithDetails, error)
	DeleteReaction(ctx context.Context, reactorUserID, ratingUserID, mediaID uuid.UUID) error
	GetReactionCountForRating(ctx context.Context, ratingUserID, mediaID uuid.UUID) (int64, error)
	GetUserReactionsForTargetUser(
		ctx context.Context,
		ratingUserID uuid.UUID,
		cursorCreatedAt time.Time,
		cursorReactorUserID uuid.UUID,
		limit int32,
	) ([]ReactionWithDetails, error)
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

type RecommendationsQuerier interface {
	GetUserRecommendations(
		ctx context.Context,
		userID, targetUserID uuid.UUID,
		threshold, limit int32,
	) (Recommendations, error)
}

type HomeQuerier interface {
	GetTopRatedByMatches(ctx context.Context, userID uuid.UUID, limit int32) ([]MatchRatedMedia, error)
}

type Models struct {
	Movies          MediaQuerier
	Users           UserQuerier
	Tokens          TokenQuerier
	Ratings         RatingQuerier
	Matches         MatchQuerier
	Reactions       ReactionQuerier
	Watchlist       WatchlistQuerier
	Discovery       DiscoveryQuerier
	Recommendations RecommendationsQuerier
	Home            HomeQuerier
}

func NewModels(db *sql.DB) Models {
	dbQueries := database.New(db)

	return Models{
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
		Recommendations: RecommendationModel{
			DB: db,
			q:  dbQueries,
		},
		Home: HomeModel{
			DB: db,
			q:  dbQueries,
		},
	}
}
