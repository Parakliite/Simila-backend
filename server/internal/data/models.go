package data

import (
	"database/sql"
	"errors"
	"time"

	"github.com/deltron-fr/filmbox/server/internal/database"
	"github.com/google/uuid"
)

var ErrRecordNotFound = errors.New("record not found")

type MediaQuerier interface {
	GetMedia(id int32, mediaType string) (*Media, error)
	InsertMedia(media *Media) (database.Medium, error)
}

type UserQuerier interface {
	Insert(user *User) error
	GetByEmail(email string) (*User, error)
	GetByID(id uuid.UUID) (*User, error)
	UpdateUser(user *User) error
	GetForToken(tokenScope, tokenPlaintext string) (*User, error)
}

type TokenQuerier interface {
	New(userID uuid.UUID, ttl time.Duration, scope string) (*Token, error)
	Insert(token *Token) error
	DeleteAllForUser(scope string, userID uuid.UUID) error
}

type Models struct {
	Movies MediaQuerier
	Users  UserQuerier
	Tokens TokenQuerier
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
	}
}
