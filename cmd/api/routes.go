package main

import (
	"expvar"
	"net/http"
)

func (app *application) routes() http.Handler {
	mux := http.NewServeMux()

	// Public endpoints
	mux.HandleFunc("POST /api/v1/users", app.registerUserHandler)
	mux.HandleFunc("POST /api/v1/login", app.loginHandler)
	mux.HandleFunc("POST /api/v1/refresh", app.refreshHandler)
	mux.HandleFunc("PUT /api/v1/users/activated", app.activateUserHandler)
	mux.HandleFunc("GET /api/v1/health", app.healthCheckHandler)

	mux.Handle("GET /api/v1/metrics", expvar.Handler())

	mux.Handle("GET /api/v1/media/{id}",
		app.authenticate(http.HandlerFunc(app.requireActivatedUser(app.getMediaHandler))))
	mux.Handle("GET /api/v1/media/{id}/cast",
		app.authenticate(http.HandlerFunc(app.requireActivatedUser(app.getMediaCast))))
	mux.Handle("GET /api/v1/media/{id}/trailer",
		app.authenticate(http.HandlerFunc(app.requireActivatedUser(app.getMediaVideo))))

	mux.Handle("GET /api/v1/users/profile",
		app.authenticate(http.HandlerFunc(app.requireActivatedUser(app.getUserProfileHandler))))
	mux.Handle("PATCH /api/v1/users/profile",
		app.authenticate(http.HandlerFunc(app.requireActivatedUser(app.updateUserProfileHandler))))
	mux.Handle("GET /api/v1/users/{user_id}/ratings/highest-rated",
		app.authenticate(http.HandlerFunc(app.requireActivatedUser(app.listUserHighestRatedMediaHandler))))
	mux.Handle("GET /api/v1/users/{user_id}/ratings/recent",
		app.authenticate(http.HandlerFunc(app.requireActivatedUser(app.listUserRecentRatedMediaHandler))))

	mux.Handle("GET /api/v1/search",
		app.authenticate(http.HandlerFunc(app.requireActivatedUser(app.searchHandler))))
	mux.Handle("GET /api/v1/discovery/rating-candidates",
		app.authenticate(http.HandlerFunc(app.requireActivatedUser(app.listRandomMediaHandler))))
	mux.Handle("GET /api/v1/onboarding/media",
		app.authenticate(http.HandlerFunc(app.requireActivatedUser(app.listOnboardingMediaHandler))))
	mux.Handle("POST /api/v1/onboarding/ratings",
		app.authenticate(http.HandlerFunc(app.requireActivatedUser(app.submitOnboardingRatingsHandler))))

	mux.Handle("PUT /api/v1/ratings",
		app.authenticate(http.HandlerFunc(app.requireActivatedUser(app.upsertRatingHandler))))
	mux.Handle("GET /api/v1/ratings",
		app.authenticate(http.HandlerFunc(app.requireActivatedUser(app.listRatingsHandler))))
	mux.Handle("GET /api/v1/ratings/tmdb/{tmdb_id}",
		app.authenticate(http.HandlerFunc(app.requireActivatedUser(app.getRatingHandler))))
	mux.Handle("DELETE /api/v1/ratings/tmdb/{tmdb_id}",
		app.authenticate(http.HandlerFunc(app.requireActivatedUser(app.deleteRatingHandler))))
	mux.Handle("GET /api/v1/media/tmdb/{tmdb_id}/ratings",
		app.authenticate(http.HandlerFunc(app.requireActivatedUser(app.listRatingsForMedia))))

	mux.Handle("GET /api/v1/home/top-rated-by-matches",
		app.authenticate(http.HandlerFunc(app.requireActivatedUser(app.getTopRatedByMatchesHandler))))
	mux.Handle("GET /api/v1/home/trending",
		app.authenticate(http.HandlerFunc(app.requireActivatedUser(app.getTrendingHandler))))

	mux.Handle("GET /api/v1/matches",
		app.authenticate(http.HandlerFunc(app.requireActivatedUser(app.listMatchesHandler))))

	mux.Handle("GET /api/v1/recommendations/{user_id}",
		app.authenticate(http.HandlerFunc(app.getUserRecommendations)))

	mux.Handle("PUT /api/v1/reactions",
		app.authenticate(http.HandlerFunc(app.requireActivatedUser(app.upsertReactionHandler))))
	mux.Handle("DELETE /api/v1/reactions/{rating_user_id}/tmdb/{tmdb_id}",
		app.authenticate(http.HandlerFunc(app.requireActivatedUser(app.deleteReactionHandler))))
	mux.Handle("GET /api/v1/users/{user_id}/reactions",
		app.authenticate(http.HandlerFunc(app.requireActivatedUser(app.listUserReactionsHandler))))
	mux.Handle("GET /api/v1/ratings/{rating_user_id}/tmdb/{tmdb_id}/reactions/count",
		app.authenticate(http.HandlerFunc(app.requireActivatedUser(app.getReactionCountForRatingHandler))))

	mux.Handle("GET /api/v1/impact/{user_id}",
		app.authenticate(http.HandlerFunc(app.requireActivatedUser(app.getImpact))))

	mux.Handle("POST /api/v1/watchlist",
		app.authenticate(http.HandlerFunc(app.requireActivatedUser(app.insertToWatchlistHandler))))
	mux.Handle("GET /api/v1/watchlist",
		app.authenticate(http.HandlerFunc(app.requireActivatedUser(app.listWatchlistItemsHandler))))
	mux.Handle("PATCH /api/v1/watchlist/tmdb/{tmdb_id}",
		app.authenticate(http.HandlerFunc(app.requireActivatedUser(app.updateWatchlistStatusHandler))))
	mux.Handle("DELETE /api/v1/watchlist/tmdb/{tmdb_id}",
		app.authenticate(http.HandlerFunc(app.requireActivatedUser(app.deleteWatchlistItemHandler))))

	mux.HandleFunc("/", app.notFoundResponse)

	return app.metrics(app.recoverPanic(app.enableCORS(mux)))
}
