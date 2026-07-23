package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/deltron-fr/filmbox/server/internal/data"
)

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

	var resp map[string]any
	json.Unmarshal(rr.Body.Bytes(), &resp)
	got := resp["media"].(map[string]any)

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

	var resp map[string]any
	json.Unmarshal(rr.Body.Bytes(), &resp)
	got := resp["media"].(map[string]any)

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

	var resp map[string]data.Media
	json.Unmarshal(rr.Body.Bytes(), &resp)
	got := resp["media"]

	if got.Title != "Fight Club" {
		t.Errorf("expected title %q, got %q", "Fight Club", got.Title)
	}
	if got.Genre == nil {
		t.Fatalf("expected genre to serialize as an empty array, got nil")
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

	var resp map[string]data.Media
	json.Unmarshal(rr.Body.Bytes(), &resp)
	got := resp["media"]

	if len(got.Genre) != 2 {
		t.Fatalf("expected 2 genres, got %d", len(got.Genre))
	}
	if got.Genre[0].GenreName != "Drama" {
		t.Errorf("expected first genre %q, got %q", "Drama", got.Genre[0].GenreName)
	}
}

func TestGetMedia_ReturnsEmptyGenreArrayForCachedMediaWithoutGenres(t *testing.T) {
	tmdb := newTMDBServer()
	defer tmdb.Close()
	app := newTestApp(tmdb.URL)

	cached := &data.Media{
		TmdbID:    121,
		Title:     "The Lord of the Rings: The Two Towers",
		MediaType: "movie",
		Runtime:   data.Runtime(179),
	}
	app.models.Movies.(*mockMediaModel).media["121:movie"] = cached

	req := httptest.NewRequest("GET", "/api/v1/media/121?type=movie", nil)
	req.SetPathValue("id", "121")
	rr := httptest.NewRecorder()

	app.getMediaHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]any
	json.Unmarshal(rr.Body.Bytes(), &resp)
	got := resp["media"].(map[string]any)

	genres, ok := got["genre"].([]any)
	if !ok {
		t.Fatalf("expected genre to be an array, got %T", got["genre"])
	}
	if len(genres) != 0 {
		t.Fatalf("expected empty genre array, got %d items", len(genres))
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

// --- searchHandler media result tests ---

func TestSearchMedia_ReturnsFilteredResults(t *testing.T) {
	tmdb := newTMDBServer()
	defer tmdb.Close()
	app := newTestApp(tmdb.URL)

	req := httptest.NewRequest("GET", "/api/v1/search?query=fight", nil)
	rr := httptest.NewRecorder()

	app.searchHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var resp map[string][]json.RawMessage
	json.Unmarshal(rr.Body.Bytes(), &resp)
	results := resp["media_results"]

	if len(results) != 2 {
		t.Fatalf("expected 2 results (person filtered out), got %d", len(results))
	}
}

func TestSearchMedia_EmptyQuery(t *testing.T) {
	app := newTestApp("")

	req := httptest.NewRequest("GET", "/api/v1/search?query=", nil)
	rr := httptest.NewRecorder()

	app.searchHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestSearchMedia_MissingQuery(t *testing.T) {
	app := newTestApp("")

	req := httptest.NewRequest("GET", "/api/v1/search", nil)
	rr := httptest.NewRecorder()

	app.searchHandler(rr, req)

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

	req := httptest.NewRequest("GET", "/api/v1/search?query=test", nil)
	rr := httptest.NewRecorder()

	app.searchHandler(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rr.Code)
	}
}

func TestSearchMedia_ResultsContainExpectedFields(t *testing.T) {
	tmdb := newTMDBServer()
	defer tmdb.Close()
	app := newTestApp(tmdb.URL)

	req := httptest.NewRequest("GET", "/api/v1/search?query=fight", nil)
	rr := httptest.NewRecorder()

	app.searchHandler(rr, req)

	var resp map[string][]struct {
		ID        int    `json:"id"`
		Title     string `json:"title"`
		Name      string `json:"name"`
		MediaType string `json:"media_type"`
	}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	results := resp["media_results"]

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
