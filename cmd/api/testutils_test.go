package main

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"

	"github.com/deltron-fr/filmbox/server/internal/data"
	"github.com/deltron-fr/filmbox/server/internal/database"
	"github.com/deltron-fr/filmbox/server/internal/jsonlog"
	"github.com/google/uuid"
)

// --- mock media model ---

type mockMediaModel struct {
	media   map[string]*data.Media
	lastCtx context.Context
}

func newMockMediaModel() *mockMediaModel {
	return &mockMediaModel{media: make(map[string]*data.Media)}
}

func (m *mockMediaModel) GetMedia(ctx context.Context, id int32, mediaType string) (*data.Media, error) {
	m.lastCtx = ctx
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	key := fmt.Sprintf("%d:%s", id, mediaType)
	media, ok := m.media[key]
	if !ok {
		return nil, sql.ErrNoRows
	}
	return media, nil
}

func (m *mockMediaModel) InsertMedia(ctx context.Context, media *data.Media) error {
	m.lastCtx = ctx
	if err := ctx.Err(); err != nil {
		return err
	}
	key := fmt.Sprintf("%d:%s", media.TmdbID, media.MediaType)
	media.ID = uuid.New()
	media.CreatedAt = time.Now()
	media.UpdatedAt = time.Now()
	m.media[key] = media
	return nil
}

// --- mock user model ---

type mockUserModel struct {
	users    map[uuid.UUID]*data.User
	byEmail  map[string]*data.User
	forToken map[string]*data.User
}

func newMockUserModel() *mockUserModel {
	return &mockUserModel{
		users:    make(map[uuid.UUID]*data.User),
		byEmail:  make(map[string]*data.User),
		forToken: make(map[string]*data.User),
	}
}

func (m *mockUserModel) Insert(ctx context.Context, user *data.User) error {
	if _, exists := m.byEmail[user.Email]; exists {
		return data.ErrDuplicateEmail
	}
	user.ID = uuid.New()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	user.Version = 1
	m.users[user.ID] = user
	m.byEmail[user.Email] = user
	return nil
}

func (m *mockUserModel) GetByEmail(ctx context.Context, email string) (*data.User, error) {
	user, ok := m.byEmail[email]
	if !ok {
		return nil, data.ErrRecordNotFound
	}
	return user, nil
}

func (m *mockUserModel) GetByID(ctx context.Context, id uuid.UUID) (*data.User, error) {
	user, ok := m.users[id]
	if !ok {
		return nil, data.ErrRecordNotFound
	}
	return user, nil
}

func (m *mockUserModel) UpdateUser(ctx context.Context, user *data.User) error {
	m.users[user.ID] = user
	m.byEmail[user.Email] = user
	user.Version++
	return nil
}

func (m *mockUserModel) GetForToken(ctx context.Context, tokenScope, tokenPlaintext string) (*data.User, error) {
	key := tokenScope + ":" + tokenPlaintext
	user, ok := m.forToken[key]
	if !ok {
		return nil, data.ErrRecordNotFound
	}
	return user, nil
}

// --- mock token model ---

type mockTokenModel struct {
	tokens map[uuid.UUID][]*data.Token
}

func newMockTokenModel() *mockTokenModel {
	return &mockTokenModel{
		tokens: make(map[uuid.UUID][]*data.Token),
	}
}

func (m *mockTokenModel) New(ctx context.Context, userID uuid.UUID, ttl time.Duration, scope string) (*data.Token, error) {
	token := &data.Token{
		Plaintext: "ABCDEFGHIJKLMNOPQRSTUVWXYZ",
		UserID:    userID,
		Expiry:    time.Now().Add(ttl),
		Scope:     scope,
	}
	m.tokens[userID] = append(m.tokens[userID], token)
	return token, nil
}

func (m *mockTokenModel) Insert(ctx context.Context, token *data.Token) error {
	m.tokens[token.UserID] = append(m.tokens[token.UserID], token)
	return nil
}

func (m *mockTokenModel) DeleteAllForUser(ctx context.Context, scope string, userID uuid.UUID) error {
	var remaining []*data.Token
	for _, t := range m.tokens[userID] {
		if t.Scope != scope {
			remaining = append(remaining, t)
		}
	}
	m.tokens[userID] = remaining
	return nil
}

// --- mock rating model ---

type mockRatingModel struct {
	ratings map[string]data.UserRating
}

func newMockRatingModel() *mockRatingModel {
	return &mockRatingModel{
		ratings: make(map[string]data.UserRating),
	}
}

func ratingKey(userID, mediaID uuid.UUID) string {
	return userID.String() + ":" + mediaID.String()
}

func (m *mockRatingModel) UpsertUserRating(ctx context.Context, rating data.Rating) (data.Rating, error) {
	now := time.Now()
	key := ratingKey(rating.UserID, rating.MediaID)
	if existing, ok := m.ratings[key]; ok {
		rating.CreatedAt = existing.Rating.CreatedAt
	} else {
		rating.CreatedAt = now
	}
	rating.UpdatedAt = now
	m.ratings[key] = data.UserRating{Rating: rating}
	return rating, nil
}

func (m *mockRatingModel) DeleteUserRating(ctx context.Context, userID, mediaID uuid.UUID) error {
	key := ratingKey(userID, mediaID)
	if _, ok := m.ratings[key]; !ok {
		return data.ErrRecordNotFound
	}
	delete(m.ratings, key)
	return nil
}

func (m *mockRatingModel) GetUsersRating(ctx context.Context, userID, mediaID uuid.UUID) (data.UserRating, error) {
	key := ratingKey(userID, mediaID)
	ur, ok := m.ratings[key]
	if !ok {
		return data.UserRating{}, data.ErrRecordNotFound
	}
	return ur, nil
}

func (m *mockRatingModel) GetUsersRatings(ctx context.Context, createdAt time.Time, limit int32, userID, mediaID uuid.UUID) ([]data.UserRating, error) {
	var result []data.UserRating
	for _, ur := range m.ratings {
		if ur.Rating.UserID == userID {
			result = append(result, ur)
		}
	}
	return result, nil
}

func (m *mockRatingModel) GetAllRatingsForSingleMedia(ctx context.Context, createdAt time.Time, mediaID, userID uuid.UUID, limit int32) ([]data.UserRating, error) {
	var result []data.UserRating
	for _, ur := range m.ratings {
		if ur.Rating.MediaID == mediaID {
			result = append(result, ur)
		}
	}
	return result, nil
}

// --- mock match model ---

type mockMatchModel struct{}

func (m *mockMatchModel) GetSimilarities(targetUserID uuid.UUID, ratings []data.Rating, indexMap map[uuid.UUID]int) (map[uuid.UUID]float64, error) {
	return make(map[uuid.UUID]float64), nil
}

// --- mock reaction model ---

type mockReactionModel struct {
	reactions map[string]data.Reaction
}

func newMockReactionModel() *mockReactionModel {
	return &mockReactionModel{
		reactions: make(map[string]data.Reaction),
	}
}

func reactionKey(reactorUserID, ratingUserID, mediaID uuid.UUID) string {
	return reactorUserID.String() + ":" + ratingUserID.String() + ":" + mediaID.String()
}

func (m *mockReactionModel) UpsertReaction(ctx context.Context, reaction data.Reaction) (data.Reaction, error) {
	now := time.Now()
	key := reactionKey(reaction.ReactorUserID, reaction.RatingUserID, reaction.MediaID)
	if existing, ok := m.reactions[key]; ok {
		reaction.ID = existing.ID
		reaction.CreatedAt = existing.CreatedAt
	} else {
		reaction.ID = uuid.New()
		reaction.CreatedAt = now
	}
	reaction.UpdatedAt = now
	m.reactions[key] = reaction
	return reaction, nil
}

func (m *mockReactionModel) DeleteReaction(ctx context.Context, reactorUserID, ratingUserID, mediaID uuid.UUID) error {
	key := reactionKey(reactorUserID, ratingUserID, mediaID)
	if _, ok := m.reactions[key]; !ok {
		return data.ErrRecordNotFound
	}
	delete(m.reactions, key)
	return nil
}

func (m *mockReactionModel) GetReactionCountForRating(ctx context.Context, ratingUserID, mediaID uuid.UUID) (int64, error) {
	var count int64
	for _, reaction := range m.reactions {
		if reaction.RatingUserID == ratingUserID && reaction.MediaID == mediaID {
			count++
		}
	}
	return count, nil
}

func (m *mockReactionModel) GetUserReactionsForTargetUser(
	ctx context.Context,
	ratingUserID uuid.UUID,
	cursorCreatedAt time.Time,
	cursorReactorUserID uuid.UUID,
	limit int32,
) ([]data.Reaction, error) {
	var reactions []data.Reaction
	for _, reaction := range m.reactions {
		if reaction.RatingUserID != ratingUserID {
			continue
		}
		if reaction.CreatedAt.After(cursorCreatedAt) {
			continue
		}
		if reaction.CreatedAt.Equal(cursorCreatedAt) && reaction.ReactorUserID.String() >= cursorReactorUserID.String() {
			continue
		}
		reactions = append(reactions, reaction)
	}

	sort.Slice(reactions, func(i, j int) bool {
		if reactions[i].CreatedAt.Equal(reactions[j].CreatedAt) {
			return reactions[i].ReactorUserID.String() > reactions[j].ReactorUserID.String()
		}
		return reactions[i].CreatedAt.After(reactions[j].CreatedAt)
	})

	if len(reactions) > int(limit) {
		reactions = reactions[:limit]
	}

	return reactions, nil
}

// --- mock watchlist model ---

type mockWatchlistModel struct {
	items map[string]data.UserWatchlistItem
}

func newMockWatchlistModel() *mockWatchlistModel {
	return &mockWatchlistModel{
		items: make(map[string]data.UserWatchlistItem),
	}
}

func watchlistKey(userID, mediaID uuid.UUID) string {
	return userID.String() + ":" + mediaID.String()
}

func (m *mockWatchlistModel) InsertMediaToWatchlist(ctx context.Context, watchlist data.Watchlist) (data.Watchlist, error) {
	now := time.Now()
	watchlist.CreatedAt = now
	watchlist.Status = string(database.StatusTypeNotWatched)

	key := watchlistKey(watchlist.UserID, watchlist.MediaID)
	m.items[key] = data.UserWatchlistItem{
		Media: data.Media{
			ID: watchlist.MediaID,
		},
		Watchlist: watchlist,
	}

	return watchlist, nil
}

func (m *mockWatchlistModel) UpdateWatchlistItemStatus(
	ctx context.Context,
	userID, mediaID uuid.UUID,
	status string,
) (data.Watchlist, error) {
	key := watchlistKey(userID, mediaID)
	item, ok := m.items[key]
	if !ok {
		return data.Watchlist{}, data.ErrRecordNotFound
	}

	item.Watchlist.Status = status
	m.items[key] = item

	return item.Watchlist, nil
}

func (m *mockWatchlistModel) DeleteMediaFromWatchlist(ctx context.Context, userID, mediaID uuid.UUID) error {
	key := watchlistKey(userID, mediaID)
	if _, ok := m.items[key]; !ok {
		return data.ErrRecordNotFound
	}

	delete(m.items, key)
	return nil
}

func (m *mockWatchlistModel) GetAllItemsInWatchlist(
	ctx context.Context,
	createdAt time.Time,
	limit int32,
	userID, mediaID uuid.UUID,
) ([]data.UserWatchlistItem, error) {
	var items []data.UserWatchlistItem

	for _, item := range m.items {
		if item.Watchlist.UserID != userID {
			continue
		}
		if !createdAt.IsZero() {
			if item.Watchlist.CreatedAt.After(createdAt) {
				continue
			}
			if item.Watchlist.CreatedAt.Equal(createdAt) && item.Watchlist.MediaID.String() >= mediaID.String() {
				continue
			}
		}
		items = append(items, item)
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].Watchlist.CreatedAt.Equal(items[j].Watchlist.CreatedAt) {
			return items[i].Watchlist.MediaID.String() > items[j].Watchlist.MediaID.String()
		}
		return items[i].Watchlist.CreatedAt.After(items[j].Watchlist.CreatedAt)
	})

	if len(items) > int(limit) {
		items = items[:limit]
	}

	return items, nil
}

// --- helpers ---

func newTestApp(tmdbURL string) *application {
	return &application{
		config: &apiConfig{
			tmdbToken:   "test-token",
			tmdbBaseURL: tmdbURL,
		},
		logger: jsonlog.New(io.Discard, jsonlog.LevelOff),
		models: data.Models{
			Movies:    newMockMediaModel(),
			Users:     newMockUserModel(),
			Tokens:    newMockTokenModel(),
			Ratings:   newMockRatingModel(),
			Matches:   &mockMatchModel{},
			Reactions: newMockReactionModel(),
			Watchlist: newMockWatchlistModel(),
		},
	}
}

func newTestUser(name, email, pw string, activated bool) *data.User {
	user := &data.User{
		ID:        uuid.New(),
		Name:      name,
		Email:     email,
		Activated: activated,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Version:   1,
	}
	user.Password.Set(pw)
	return user
}

func seedUser(app *application, user *data.User) {
	m := app.models.Users.(*mockUserModel)
	m.users[user.ID] = user
	m.byEmail[user.Email] = user
}

func withUser(r *http.Request, user *data.User) *http.Request {
	ctx := context.WithValue(r.Context(), userContextKey, user)
	return r.WithContext(ctx)
}
