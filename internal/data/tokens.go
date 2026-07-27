package data

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base32"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/parakliite/simila/internal/database"
	"github.com/parakliite/simila/internal/validator"
)

const (
	ScopeActivation     = "activation"
	ScopeAuthentication = "authentication"
	ScopeRefresh        = "refresh"
)

type Token struct {
	Plaintext string        `json:"token"`
	Hash      []byte        `json:"-"`
	SessionID uuid.NullUUID `json:"session_id"`
	UserID    uuid.UUID     `json:"-"`
	Expiry    time.Time     `json:"-"`
	Scope     string        `json:"-"`
}

func generateToken(userID uuid.UUID, sessionID uuid.NullUUID, ttl time.Duration, scope string) (*Token, error) {
	token := &Token{
		UserID:    userID,
		SessionID: sessionID,
		Expiry:    time.Now().Add(ttl),
		Scope:     scope,
	}

	randomBytes := make([]byte, 16)

	_, err := rand.Read(randomBytes)
	if err != nil {
		return nil, err
	}

	token.Plaintext = base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randomBytes)

	hash := sha256.Sum256([]byte(token.Plaintext))
	token.Hash = hash[:]

	return token, nil
}

func ValidateTokenPlaintext(v *validator.Validator, tokenPlaintext string) {
	v.Check(tokenPlaintext != "", "token", "must be provided")
	v.Check(len(tokenPlaintext) == 26, "token", "must be 26 bytes long")
}

type TokenModel struct {
	DB *sql.DB
	q  *database.Queries
}

// The New() method is a shortcut which creates a new Token struct and then inserts the
// data in the tokens table.
func (m TokenModel) New(ctx context.Context, userID uuid.UUID, sessionID uuid.NullUUID, ttl time.Duration, scope string) (*Token, error) {
	token, err := generateToken(userID, sessionID, ttl, scope)
	if err != nil {
		return nil, err
	}

	err = m.Insert(ctx, token)
	return token, err
}

func (m TokenModel) Insert(ctx context.Context, token *Token) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	err := m.q.CreateToken(ctx, database.CreateTokenParams{
		TokenHash: token.Hash,
		SessionID: token.SessionID,
		UserID:    token.UserID,
		Expiry:    token.Expiry,
		Scope:     token.Scope,
	})

	return err
}

func (m TokenModel) GetUserIDFromToken(ctx context.Context, scope string, tokenHash []byte) (userID uuid.UUID, sessionID uuid.UUID, err error) {
	row, err := m.q.GetUserFromToken(ctx, database.GetUserFromTokenParams{
		Scope:     scope,
		TokenHash: tokenHash,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return uuid.Nil, uuid.Nil, ErrRecordNotFound
		}
		return uuid.Nil, uuid.Nil, err
	}

	if row.SessionID.Valid {
		sessionID = row.SessionID.UUID
	}

	return row.UserID, sessionID, nil
}

func (m TokenModel) RevokePreviousToken(ctx context.Context, scope string, tokenHash []byte) error {
	row, err := m.q.RevokePreviousToken(ctx, database.RevokePreviousTokenParams{
		Scope:     scope,
		TokenHash: tokenHash,
	})
	if err != nil {
		return err
	}

	if row == 0 {
		return ErrRecordNotFound
	}

	return nil
}

// Force the user to login again on all sessions.
func (m TokenModel) RevokeAllPreviousTokens(ctx context.Context, scope string, userID uuid.UUID) error {
	return m.q.RevokeAllPreviousTokens(ctx, database.RevokeAllPreviousTokensParams{
		UserID: userID,
		Scope:  scope,
	})
}

func (m TokenModel) DeleteAllForUser(ctx context.Context, scope string, userID uuid.UUID) error {
	err := m.q.DeleteAllTokensAllForUser(ctx, database.DeleteAllTokensAllForUserParams{
		Scope:  scope,
		UserID: userID,
	})

	return err
}
