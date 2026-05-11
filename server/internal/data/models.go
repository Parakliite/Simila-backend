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
	DeleteUserRating(ctx context.Context, userID, mediaID uuid.UUID) error
}

type Models struct {
	Movies  MediaQuerier
	Users   UserQuerier
	Tokens  TokenQuerier
	Ratings RatingQuerier
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
	}
}
