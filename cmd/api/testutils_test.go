package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/parakliite/simila/internal/data"
	"github.com/parakliite/simila/internal/database"
	"github.com/parakliite/simila/internal/jsonlog"
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
		return nil, data.ErrRecordNotFound
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

func (m *mockMediaModel) GetOrCreateMediaByTmdbID(ctx context.Context, tmdbID int32, mediaType string) (uuid.UUID, error) {
	if err := ctx.Err(); err != nil {
		return uuid.UUID{}, err
	}
	key := fmt.Sprintf("%d:%s", tmdbID, mediaType)
	if media, ok := m.media[key]; ok {
		return media.ID, nil
	}
	media := &data.Media{
		ID:        uuid.New(),
		TmdbID:    tmdbID,
		MediaType: mediaType,
	}
	m.media[key] = media
	return media.ID, nil
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

func (m *mockUserModel) GetUserForToken(
	ctx context.Context,
	tokenScope, tokenPlaintext string,
) (*data.User, error) {
	key := tokenScope + ":" + tokenPlaintext
	user, ok := m.forToken[key]
	if !ok {
		return nil, data.ErrRecordNotFound
	}
	return user, nil
}

func (m *mockUserModel) SearchUsers(ctx context.Context, query string, limit int32) ([]data.User, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if query == "" {
		return nil, fmt.Errorf("empty query")
	}
	if limit <= 0 {
		return []data.User{}, nil
	}

	query = strings.ToLower(query)
	users := make([]data.User, 0, len(m.users))
	for _, user := range m.users {
		if strings.Contains(strings.ToLower(user.Name), query) {
			users = append(users, *user)
		}
	}
	sort.Slice(users, func(i, j int) bool {
		if users[i].Name == users[j].Name {
			return users[i].ID.String() < users[j].ID.String()
		}
		return users[i].Name < users[j].Name
	})
	if int32(len(users)) > limit {
		users = users[:limit]
	}

	return users, nil
}

func (m *mockUserModel) GetAllUsers(ctx context.Context) ([]uuid.UUID, error) {
	ids := make([]uuid.UUID, 0, len(m.users))
	for id := range m.users {
		ids = append(ids, id)
	}

	sort.Slice(ids, func(i, j int) bool {
		return ids[i].String() < ids[j].String()
	})

	return ids, nil
}

func (m *mockUserModel) RevokeAllTokensForSession(
	ctx context.Context,
	sessionID uuid.NullUUID,
	scope string,
) error {
	return nil
}

// --- mock token model ---

type mockTokenModel struct {
	tokens  map[uuid.UUID][]*data.Token
	revoked map[string]bool
	counter int
}

func newMockTokenModel() *mockTokenModel {
	return &mockTokenModel{
		tokens:  make(map[uuid.UUID][]*data.Token),
		revoked: make(map[string]bool),
	}
}

func (m *mockTokenModel) New(
	ctx context.Context,
	userID uuid.UUID,
	sessionID uuid.NullUUID,
	ttl time.Duration,
	scope string,
) (*data.Token, error) {
	m.counter++
	plaintext := fmt.Sprintf("%026d", m.counter)
	hash := sha256.Sum256([]byte(plaintext))

	token := &data.Token{
		Plaintext: plaintext,
		Hash:      hash[:],
		SessionID: sessionID,
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

func (m *mockTokenModel) GetUserIDFromToken(
	ctx context.Context,
	scope string,
	tokenHash []byte,
) (uuid.UUID, uuid.UUID, error) {
	for userID, tokens := range m.tokens {
		for _, token := range tokens {
			if token.Scope == scope && string(token.Hash) == string(tokenHash) {
				if m.revoked[tokenKey(scope, tokenHash)] || time.Now().After(token.Expiry) {
					return uuid.Nil, uuid.Nil, data.ErrRecordNotFound
				}

				if token.SessionID.Valid {
					return userID, token.SessionID.UUID, nil
				}

				return userID, uuid.Nil, nil
			}
		}
	}
	return uuid.Nil, uuid.Nil, data.ErrRecordNotFound
}

func (m *mockTokenModel) RevokePreviousToken(ctx context.Context, scope string, tokenHash []byte) error {
	for _, tokens := range m.tokens {
		for _, token := range tokens {
			if token.Scope == scope && string(token.Hash) == string(tokenHash) {
				m.revoked[tokenKey(scope, tokenHash)] = true
				return nil
			}
		}
	}
	return data.ErrRecordNotFound
}

func (m *mockTokenModel) RevokeAllPreviousTokens(ctx context.Context, scope string, userID uuid.UUID) error {
	for _, token := range m.tokens[userID] {
		if token.Scope == scope {
			m.revoked[tokenKey(scope, token.Hash)] = true
		}
	}
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

func tokenKey(scope string, tokenHash []byte) string {
	return scope + ":" + string(tokenHash)
}

// --- mock rating model ---

type mockRatingModel struct {
	ratings     map[string]data.UserRating
	randomMedia []data.Media
}

func newMockRatingModel() *mockRatingModel {
	return &mockRatingModel{
		ratings: make(map[string]data.UserRating),
	}
}

func ratingKey(userID, mediaID uuid.UUID) string {
	return userID.String() + ":" + mediaID.String()
}

func translateMockRatingValueInbound(value float64) float64 {
	return value * 2
}

func translateMockRatingValueOutbound(value float64) float64 {
	return value / 2.0
}

func translateMockUserRatingOutbound(userRating data.UserRating) data.UserRating {
	userRating.Rating.RatingValue = translateMockRatingValueOutbound(userRating.Rating.RatingValue)
	return userRating
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

	storedRating := rating
	storedRating.RatingValue = translateMockRatingValueInbound(rating.RatingValue)
	m.ratings[key] = data.UserRating{Rating: storedRating}

	rating.RatingValue = translateMockRatingValueOutbound(storedRating.RatingValue)
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

func (m *mockRatingModel) GetUsersRating(
	ctx context.Context,
	userID, mediaID uuid.UUID,
) (data.UserRating, error) {
	key := ratingKey(userID, mediaID)
	ur, ok := m.ratings[key]
	if !ok {
		return data.UserRating{}, data.ErrRecordNotFound
	}
	return translateMockUserRatingOutbound(ur), nil
}

func (m *mockRatingModel) GetUsersRatings(
	ctx context.Context,
	createdAt time.Time,
	limit int32,
	userID, mediaID uuid.UUID,
) ([]data.UserRating, error) {
	var result []data.UserRating
	for _, ur := range m.ratings {
		if ur.Rating.UserID == userID {
			result = append(result, translateMockUserRatingOutbound(ur))
		}
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Rating.CreatedAt.Equal(result[j].Rating.CreatedAt) {
			return result[i].Rating.MediaID.String() > result[j].Rating.MediaID.String()
		}
		return result[i].Rating.CreatedAt.After(result[j].Rating.CreatedAt)
	})

	return result, nil
}

func (m *mockRatingModel) GetAllRatingsForSingleMedia(
	ctx context.Context,
	createdAt time.Time,
	mediaID, userID uuid.UUID,
	limit int32,
) ([]data.UserRating, error) {
	var result []data.UserRating
	for _, ur := range m.ratings {
		if ur.Rating.MediaID == mediaID {
			result = append(result, translateMockUserRatingOutbound(ur))
		}
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Rating.CreatedAt.Equal(result[j].Rating.CreatedAt) {
			return result[i].Rating.UserID.String() > result[j].Rating.UserID.String()
		}
		return result[i].Rating.CreatedAt.After(result[j].Rating.CreatedAt)
	})

	return result, nil
}

func (m *mockRatingModel) GetRandomMedia(
	ctx context.Context,
	userID uuid.UUID,
	limit int32,
) ([]data.Media, error) {
	var result []data.Media
	for _, media := range m.randomMedia {
		if _, ok := m.ratings[ratingKey(userID, media.ID)]; ok {
			continue
		}
		result = append(result, media)
		if len(result) == int(limit) {
			break
		}
	}

	return result, nil
}

// --- mock match model ---

type mockMatchModel struct {
	matches          []data.Match
	getMatchesErr    error
	matchExists      bool
	checkMatchErr    error
	lastTargetUserID uuid.UUID
	lastLimit        int32
	lastCursorUserID uuid.NullUUID
	lastCursorScore  sql.NullFloat64
}

func (m *mockMatchModel) CalculateSimilarities(
	ctx context.Context,
	targetUserID uuid.UUID,
	ratings []data.Rating,
	indexMap map[uuid.UUID]int,
) error {
	return nil
}

func (m *mockMatchModel) MapMediaToIndex(ctx context.Context) (map[uuid.UUID]int, error) {
	return make(map[uuid.UUID]int), nil
}

func (m *mockMatchModel) GetAllUserRatingsForMatches(
	ctx context.Context,
	userID uuid.UUID,
) ([]data.Rating, error) {
	return []data.Rating{}, nil
}

func (m *mockMatchModel) GetUserMatches(
	ctx context.Context,
	targetUserID uuid.UUID,
	limit int32,
	cursorOtherUserID uuid.NullUUID,
	cursorScore sql.NullFloat64,
	cursorSharedMediaCount sql.NullInt32,
	cursorLastRecalculatedAt sql.NullTime,
) ([]data.Match, error) {
	m.lastTargetUserID = targetUserID
	m.lastLimit = limit
	m.lastCursorUserID = cursorOtherUserID
	m.lastCursorScore = cursorScore
	if m.getMatchesErr != nil {
		return nil, m.getMatchesErr
	}
	return m.matches, nil
}

func (m *mockMatchModel) CheckMatchExists(
	ctx context.Context,
	userID, otherUserID uuid.UUID,
) (bool, error) {
	if m.checkMatchErr != nil {
		return false, m.checkMatchErr
	}
	return m.matchExists, nil
}

// --- mock recommendation model ---

type mockRecommendationModel struct {
	recommendations  data.Recommendations
	err              error
	lastUserID       uuid.UUID
	lastTargetUserID uuid.UUID
	lastThreshold    int32
	lastLimit        int32
}

func (m *mockRecommendationModel) GetUserRecommendations(
	ctx context.Context,
	userID, targetUserID uuid.UUID,
	threshold, limit int32,
) (data.Recommendations, error) {
	m.lastUserID = userID
	m.lastTargetUserID = targetUserID
	m.lastThreshold = threshold
	m.lastLimit = limit
	if m.err != nil {
		return data.Recommendations{}, m.err
	}
	return m.recommendations, nil
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

func mockReactionWithDetails(reaction data.Reaction) data.ReactionWithDetails {
	return data.ReactionWithDetails{
		ID:        reaction.ID,
		Reaction:  reaction.Reaction,
		CreatedAt: reaction.CreatedAt,
		UpdatedAt: reaction.UpdatedAt,
		Reactor: data.ReactionUser{
			ID: reaction.ReactorUserID,
		},
		RatingUser: data.ReactionUser{
			ID: reaction.RatingUserID,
		},
		Media: data.ReactionMedia{
			ID: reaction.MediaID,
		},
		Rating: data.ReactionRating{},
	}
}

func (m *mockReactionModel) UpsertReaction(
	ctx context.Context,
	reaction data.Reaction,
) (data.ReactionWithDetails, error) {
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
	return mockReactionWithDetails(reaction), nil
}

func (m *mockReactionModel) DeleteReaction(
	ctx context.Context,
	reactorUserID, ratingUserID, mediaID uuid.UUID,
) error {
	key := reactionKey(reactorUserID, ratingUserID, mediaID)
	if _, ok := m.reactions[key]; !ok {
		return data.ErrRecordNotFound
	}
	delete(m.reactions, key)
	return nil
}

func (m *mockReactionModel) GetReactionCountForRating(
	ctx context.Context,
	ratingUserID, mediaID uuid.UUID,
) (int64, error) {
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
) ([]data.ReactionWithDetails, error) {
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

	detailedReactions := make([]data.ReactionWithDetails, 0, len(reactions))
	for _, reaction := range reactions {
		detailedReactions = append(detailedReactions, mockReactionWithDetails(reaction))
	}

	return detailedReactions, nil
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

func (m *mockWatchlistModel) InsertMediaToWatchlist(
	ctx context.Context,
	watchlist data.Watchlist,
) (data.Watchlist, error) {
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
	limit int32, status string,
	userID, mediaID uuid.UUID,
) ([]data.UserWatchlistItem, error) {
	var items []data.UserWatchlistItem

	for _, item := range m.items {
		if item.Watchlist.UserID != userID {
			continue
		}
		if status != "" && item.Watchlist.Status != status {
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

// --- mock discovery model ---

type mockDiscoveryModel struct {
	history map[string]time.Time
}

func newMockDiscoveryModel() *mockDiscoveryModel {
	return &mockDiscoveryModel{
		history: make(map[string]time.Time),
	}
}

func discoveryKey(userID, mediaID uuid.UUID) string {
	return userID.String() + ":" + mediaID.String()
}

func (m *mockDiscoveryModel) GetDiscoveryHistoryExpiry(
	ctx context.Context,
	userID, mediaID uuid.UUID,
) (time.Time, error) {
	expiry, ok := m.history[discoveryKey(userID, mediaID)]
	if !ok {
		return time.Time{}, data.ErrRecordNotFound
	}

	return expiry, nil
}

func (m *mockDiscoveryModel) SetDiscoveryHistoryExpiry(
	ctx context.Context,
	userID, mediaID uuid.UUID,
	eligibleAt time.Time,
) error {
	m.history[discoveryKey(userID, mediaID)] = eligibleAt
	return nil
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
			Movies:          newMockMediaModel(),
			Users:           newMockUserModel(),
			Tokens:          newMockTokenModel(),
			Ratings:         newMockRatingModel(),
			Matches:         &mockMatchModel{},
			Reactions:       newMockReactionModel(),
			Watchlist:       newMockWatchlistModel(),
			Discovery:       newMockDiscoveryModel(),
			Recommendations: &mockRecommendationModel{},
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

func withUser(req *http.Request, user *data.User) *http.Request {
	ctx := context.WithValue(req.Context(), userContextKey, user)
	return req.WithContext(ctx)
}
