package data

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/parakliite/simila/internal/database"
	"github.com/parakliite/simila/internal/validator"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrDuplicateEmail = errors.New("duplicate email")
	ErrEditConflict   = errors.New(
		"unable to update the record due to an edit conflict, please try again",
	)
)

// UserModel is a wrapper around the db connection pool
type UserModel struct {
	DB *sql.DB
	q  *database.Queries
}

type User struct {
	ID                uuid.UUID `json:"id"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	Name              string    `json:"name"`
	ProfilePictureURL string    `json:"profile_picture_url,omitempty"`
	About             string    `json:"about,omitempty"`
	Email             string    `json:"email"`
	Password          password  `json:"-"`
	Activated         bool      `json:"activated"`
	Version           int32     `json:"-"`
}

type password struct {
	plaintext *string
	hash      []byte
}

func (p *password) Set(plaintextPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(plaintextPassword), 12)
	if err != nil {
		return err
	}

	p.plaintext = &plaintextPassword
	p.hash = hash

	return nil
}

func (p *password) Matches(plaintextPassword string) (bool, error) {
	err := bcrypt.CompareHashAndPassword(p.hash, []byte(plaintextPassword))
	if err != nil {
		switch {
		case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
			return false, nil
		default:
			return false, err
		}
	}

	return true, nil
}

func (m UserModel) Insert(ctx context.Context, user *User) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	row, err := m.q.CreateUser(ctx, database.CreateUserParams{
		Name:         user.Name,
		Email:        user.Email,
		PasswordHash: user.Password.hash,
		About:        sql.NullString{String: user.About, Valid: user.About != ""},
		ProfilePictureUrl: sql.NullString{
			String: user.ProfilePictureURL,
			Valid:  user.ProfilePictureURL != "",
		},
	})
	if err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"`:
			return ErrDuplicateEmail
		default:
			return err
		}
	}

	user.ID = row.ID
	user.CreatedAt = row.CreatedAt
	user.UpdatedAt = row.UpdatedAt
	user.Activated = row.Activated
	user.Version = row.Version

	return nil
}

// this function is internally used by the login handler, that is why it returns the password hash
func (m UserModel) GetByEmail(ctx context.Context, email string) (*User, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	row, err := m.q.GetUserByEmail(ctx, email)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	user := &User{
		ID:                row.ID,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
		Name:              row.Name,
		Email:             row.Email,
		Password:          password{hash: row.PasswordHash},
		ProfilePictureURL: row.ProfilePictureUrl.String,
		About:             row.About.String,
		Activated:         row.Activated,
		Version:           row.Version,
	}

	return user, nil
}

func (m UserModel) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	row, err := m.q.GetUserByID(ctx, id)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	user := &User{
		ID:                row.ID,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
		Name:              row.Name,
		Email:             row.Email,
		Password:          password{hash: row.PasswordHash},
		ProfilePictureURL: row.ProfilePictureUrl.String,
		About:             row.About.String,
		Activated:         row.Activated,
		Version:           row.Version,
	}

	return user, nil
}

func (m UserModel) UpdateUser(ctx context.Context, user *User) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	version, err := m.q.UpdateUser(ctx, database.UpdateUserParams{
		Name:  user.Name,
		Email: user.Email,
		ProfilePictureUrl: sql.NullString{
			String: user.ProfilePictureURL,
			Valid:  user.ProfilePictureURL != "",
		},
		About: sql.NullString{
			String: user.About,
			Valid:  user.About != "",
		},
		Activated:    user.Activated,
		PasswordHash: user.Password.hash,
		ID:           user.ID,
		Version:      user.Version,
	})
	if err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"`:
			return ErrDuplicateEmail
		case errors.Is(err, sql.ErrNoRows):
			return ErrEditConflict
		default:
			return err
		}
	}
	user.Version = version
	return nil
}

func (m UserModel) GetForToken(ctx context.Context, tokenScope, tokenPlaintext string) (*User, error) {
	tokenHash := sha256.Sum256([]byte(tokenPlaintext))

	row, err := m.q.GetTokenForUser(ctx, database.GetTokenForUserParams{
		TokenHash: tokenHash[:],
		Scope:     tokenScope,
		Expiry:    time.Now().UTC(),
	})
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	user := &User{
		ID:                row.ID,
		CreatedAt:         row.CreatedAt,
		Name:              row.Name,
		Email:             row.Email,
		Password:          password{hash: row.PasswordHash},
		ProfilePictureURL: row.ProfilePictureUrl.String,
		About:             row.About.String,
		Activated:         row.Activated,
		Version:           row.Version,
	}

	return user, nil
}

func ValidateEmail(v *validator.Validator, email string) {
	v.Check(email != "", "email", "must be provided")
	v.Check(validator.Matches(email, validator.EmailRX), "email", "must be a valid email address")
}

func ValidatePasswordPlaintext(v *validator.Validator, password string) {
	v.Check(password != "", "password", "must be provided")
	v.Check(len(password) >= 8, "password", "must be at least 8 bytes long")
	v.Check(len(password) <= 72, "password", "must not be more than 72 bytes long")
}

func ValidateUser(v *validator.Validator, user *User) {
	v.Check(user.Name != "", "name", "must be provided")
	v.Check(len(user.Name) <= 500, "name", "must not be more than 500 bytes long")

	ValidateEmail(v, user.Email)

	if user.Password.plaintext != nil {
		ValidatePasswordPlaintext(v, *user.Password.plaintext)
	}

	if user.Password.hash == nil {
		panic("missing password hash for user")
	}
}
