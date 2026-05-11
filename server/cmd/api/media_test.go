package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/deltron-fr/filmbox/server/internal/data"
	"github.com/deltron-fr/filmbox/server/internal/jsonlog"
	"github.com/google/uuid"
)

// --- mocks ---

type mockMediaModel struct {
	media   map[string]*data.Media // keyed by "tmdbID:type"
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

type mockUserModel struct{}

func (m *mockUserModel) Insert(ctx context.Context, user *data.User) error { return nil }
func (m *mockUserModel) GetByEmail(ctx context.Context, email string) (*data.User, error) {
	return nil, nil
}
func (m *mockUserModel) GetByID(ctx context.Context, id uuid.UUID) (*data.User, error) {
	return nil, nil
}
func (m *mockUserModel) UpdateUser(ctx context.Context, user *data.User) error { return nil }
func (m *mockUserModel) GetForToken(ctx context.Context, tokenScope, tokenPlaintext string) (*data.User, error) {
	return nil, nil
}

type mockTokenModel struct{}

func (m *mockTokenModel) New(
	ctx context.Context,
	userID uuid.UUID,
	ttl time.Duration,
	scope string,
) (*data.Token, error) {
	return nil, nil
}
func (m *mockTokenModel) Insert(ctx context.Context, token *data.Token) error { return nil }
func (m *mockTokenModel) DeleteAllForUser(ctx context.Context, scope string, userID uuid.UUID) error {
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
			Movies: newMockMediaModel(),
			Users:  &mockUserModel{},
			Tokens: &mockTokenModel{},
		},
	}
}

func newTMDBServer() *httptest.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /3/movie/{id}", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"id":             550,
			"title":          "Fight Club",
			"original_title": "Fight Club",
			"overview": "An insomniac office worker and a devil-may-care soap " +
				"maker form an underground fight club.",
			"release_date":  "1999-10-15",
			"runtime":       139,
			"poster_path":   "/pB8BM7pdSp6B6Ih7QZ4DrQ3PmJK.jpg",
			"backdrop_path": "/hZkgoQYus5dXo3H8T7Uef6DNknx.jpg",
			"genres":        []map[string]any{{"name": "Drama"}, {"name": "Thriller"}},
		})
	})

	mux.HandleFunc("GET /3/tv/{id}", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"id":               1396,
			"name":             "Breaking Bad",
			"original_name":    "Breaking Bad",
			"overview":         "A high school chemistry teacher turned meth producer.",
			"first_air_date":   "2008-01-20",
			"episode_run_time": []int{47},
			"poster_path":      "/ggFHVNu6YYI5L9pCfOacjizRGt.jpg",
			"backdrop_path":    "/tsRy63Mu5cu8etL1X7ZLyf7UP1M.jpg",
			"genres":           []map[string]any{{"name": "Drama"}, {"name": "Crime"}},
		})
	})

	mux.HandleFunc("GET /3/search/multi", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")
		if query == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		json.NewEncoder(w).Encode(map[string]any{
			"results": []map[string]any{
				{
					"id":         550,
					"title":      "Fight Club",
					"media_type": "movie",
				},
				{
					"id":         1396,
					"name":       "Breaking Bad",
					"media_type": "tv",
				},
				{
					"id":         999,
					"name":       "Some Actor",
					"media_type": "person",
				},
			},
		})
	})

	return httptest.NewServer(mux)
}

// --- getMediaHandler tests ---

func TestGetMedia_ReturnsMovie(t *testing.T) {
	tmdb := newTMDBServer()
	defer tmdb.Close()
	app := newTestApp(tmdb.URL)

	req := httptest.NewRequest("GET", "/api/v1/media/550?type=movie", nil)
	req.SetPathValue("id", "550")
	rr := httptest.NewRecorder()

	app.getMediaHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var got map[string]any
	json.Unmarshal(rr.Body.Bytes(), &got)

	if got["title"] != "Fight Club" {
		t.Errorf("expected title %q, got %q", "Fight Club", got["title"])
	}
	if got["media_type"] != "movie" {
		t.Errorf("expected media_type %q, got %q", "movie", got["media_type"])
	}
	if int(got["tmdb_id"].(float64)) != 550 {
		t.Errorf("expected tmdb_id 550, got %v", got["tmdb_id"])
	}
	if got["runtime"] != "139 min" {
		t.Errorf("expected runtime %q, got %q", "139 min", got["runtime"])
	}
}

func TestGetMedia_ReturnsTVShow(t *testing.T) {
	tmdb := newTMDBServer()
	defer tmdb.Close()
	app := newTestApp(tmdb.URL)

	req := httptest.NewRequest("GET", "/api/v1/media/1396?type=tv", nil)
	req.SetPathValue("id", "1396")
	rr := httptest.NewRecorder()

	app.getMediaHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var got map[string]any
	json.Unmarshal(rr.Body.Bytes(), &got)

	if got["title"] != "Breaking Bad" {
		t.Errorf("expected title %q, got %q", "Breaking Bad", got["title"])
	}
	if got["media_type"] != "tv" {
		t.Errorf("expected media_type %q, got %q", "tv", got["media_type"])
	}
	if got["runtime"] != "47 min" {
		t.Errorf("expected runtime %q, got %q", "47 min", got["runtime"])
	}
}

func TestGetMedia_ReturnsCachedMedia(t *testing.T) {
	tmdb := newTMDBServer()
	defer tmdb.Close()
	app := newTestApp(tmdb.URL)

	cached := &data.Media{
		TmdbID:    550,
		Title:     "Fight Club",
		MediaType: "movie",
		Runtime:   data.Runtime(139),
	}
	app.models.Movies.(*mockMediaModel).media["550:movie"] = cached

	req := httptest.NewRequest("GET", "/api/v1/media/550?type=movie", nil)
	req.SetPathValue("id", "550")
	rr := httptest.NewRecorder()

	app.getMediaHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var got data.Media
	json.Unmarshal(rr.Body.Bytes(), &got)

	if got.Title != "Fight Club" {
		t.Errorf("expected title %q, got %q", "Fight Club", got.Title)
	}
}

func TestGetMedia_InvalidType(t *testing.T) {
	app := newTestApp("")

	req := httptest.NewRequest("GET", "/api/v1/media/550?type=podcast", nil)
	req.SetPathValue("id", "550")
	rr := httptest.NewRecorder()

	app.getMediaHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestGetMedia_MissingType(t *testing.T) {
	app := newTestApp("")

	req := httptest.NewRequest("GET", "/api/v1/media/550", nil)
	req.SetPathValue("id", "550")
	rr := httptest.NewRecorder()

	app.getMediaHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestGetMedia_InvalidID(t *testing.T) {
	app := newTestApp("")

	req := httptest.NewRequest("GET", "/api/v1/media/abc?type=movie", nil)
	req.SetPathValue("id", "abc")
	rr := httptest.NewRecorder()

	app.getMediaHandler(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rr.Code)
	}
}

func TestGetMedia_NegativeID(t *testing.T) {
	app := newTestApp("")

	req := httptest.NewRequest("GET", "/api/v1/media/-1?type=movie", nil)
	req.SetPathValue("id", "-1")
	rr := httptest.NewRecorder()

	app.getMediaHandler(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rr.Code)
	}
}

func TestGetMedia_FetchesMoveGenres(t *testing.T) {
	tmdb := newTMDBServer()
	defer tmdb.Close()
	app := newTestApp(tmdb.URL)

	req := httptest.NewRequest("GET", "/api/v1/media/550?type=movie", nil)
	req.SetPathValue("id", "550")
	rr := httptest.NewRecorder()

	app.getMediaHandler(rr, req)

	var got data.Media
	json.Unmarshal(rr.Body.Bytes(), &got)

	if len(got.Genre) != 2 {
		t.Fatalf("expected 2 genres, got %d", len(got.Genre))
	}
	if got.Genre[0].GenreName != "Drama" {
		t.Errorf("expected first genre %q, got %q", "Drama", got.Genre[0].GenreName)
	}
}

func TestGetMedia_TMDBDown(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	app := newTestApp(srv.URL)

	req := httptest.NewRequest("GET", "/api/v1/media/550?type=movie", nil)
	req.SetPathValue("id", "550")
	rr := httptest.NewRecorder()

	app.getMediaHandler(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rr.Code)
	}
}

// --- getMediaSearchHandler tests ---

func TestSearchMedia_ReturnsFilteredResults(t *testing.T) {
	tmdb := newTMDBServer()
	defer tmdb.Close()
	app := newTestApp(tmdb.URL)

	req := httptest.NewRequest("GET", "/api/v1/media/search?query=fight", nil)
	rr := httptest.NewRecorder()

	app.getMediaSearchHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var results []json.RawMessage
	json.Unmarshal(rr.Body.Bytes(), &results)

	if len(results) != 2 {
		t.Fatalf("expected 2 results (person filtered out), got %d", len(results))
	}
}

func TestSearchMedia_EmptyQuery(t *testing.T) {
	app := newTestApp("")

	req := httptest.NewRequest("GET", "/api/v1/media/search?query=", nil)
	rr := httptest.NewRecorder()

	app.getMediaSearchHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestSearchMedia_MissingQuery(t *testing.T) {
	app := newTestApp("")

	req := httptest.NewRequest("GET", "/api/v1/media/search", nil)
	rr := httptest.NewRecorder()

	app.getMediaSearchHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestSearchMedia_TMDBDown(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	app := newTestApp(srv.URL)

	req := httptest.NewRequest("GET", "/api/v1/media/search?query=test", nil)
	rr := httptest.NewRecorder()

	app.getMediaSearchHandler(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rr.Code)
	}
}

func TestSearchMedia_ResultsContainExpectedFields(t *testing.T) {
	tmdb := newTMDBServer()
	defer tmdb.Close()
	app := newTestApp(tmdb.URL)

	req := httptest.NewRequest("GET", "/api/v1/media/search?query=fight", nil)
	rr := httptest.NewRecorder()

	app.getMediaSearchHandler(rr, req)

	var results []struct {
		ID        int    `json:"id"`
		Title     string `json:"title"`
		Name      string `json:"name"`
		MediaType string `json:"media_type"`
	}
	json.Unmarshal(rr.Body.Bytes(), &results)

	movie := results[0]
	if movie.ID != 550 || movie.Title != "Fight Club" || movie.MediaType != "movie" {
		t.Errorf("unexpected movie result: %+v", movie)
	}

	tv := results[1]
	if tv.ID != 1396 || tv.Name != "Breaking Bad" || tv.MediaType != "tv" {
		t.Errorf("unexpected tv result: %+v", tv)
	}
}

// --- context propagation tests ---

func TestGetMedia_PassesRequestContext(t *testing.T) {
	tmdb := newTMDBServer()
	defer tmdb.Close()
	app := newTestApp(tmdb.URL)

	cached := &data.Media{
		TmdbID:    550,
		Title:     "Fight Club",
		MediaType: "movie",
		Runtime:   data.Runtime(139),
	}
	mock := app.models.Movies.(*mockMediaModel)
	mock.media["550:movie"] = cached

	type ctxKey string
	req := httptest.NewRequest("GET", "/api/v1/media/550?type=movie", nil)
	req.SetPathValue("id", "550")
	req = req.WithContext(context.WithValue(req.Context(), ctxKey("test"), "marker"))
	rr := httptest.NewRecorder()

	app.getMediaHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	if mock.lastCtx == nil {
		t.Fatal("expected context to be passed to model, got nil")
	}
	if mock.lastCtx.Value(ctxKey("test")) != "marker" {
		t.Error("model did not receive the request context")
	}
}

func TestGetMedia_CancelledContextReturns500(t *testing.T) {
	app := newTestApp("")

	cached := &data.Media{
		TmdbID:    550,
		Title:     "Fight Club",
		MediaType: "movie",
		Runtime:   data.Runtime(139),
	}
	app.models.Movies.(*mockMediaModel).media["550:movie"] = cached

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	req := httptest.NewRequest("GET", "/api/v1/media/550?type=movie", nil)
	req.SetPathValue("id", "550")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	app.getMediaHandler(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500 for cancelled context, got %d", rr.Code)
	}
}
