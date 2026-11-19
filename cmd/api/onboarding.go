package main

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/parakliite/simila/internal/data"
	"github.com/parakliite/simila/internal/validator"
)

const (
	onboardingPopularityThreshold float64 = 200
	onboardingMediaLimit          int32   = 25
	onboardingRatingsMin          int     = 10
	onboardingRatingsMax          int     = 50
)

type onboardingRatingInput struct {
	TmdbID      int32      `json:"tmdb_id"`
	MediaType   string     `json:"media_type"`
	RatingValue float64    `json:"rating_value"`
	WatchedDate *time.Time `json:"watched_date,omitempty"`
}

func (app *application) listOnboardingMediaHandler(w http.ResponseWriter, req *http.Request) {
	user := app.contextGetUser(req)

	media, err := app.models.Onboarding.GetMedia(
		req.Context(),
		user.ID,
		onboardingPopularityThreshold,
		onboardingMediaLimit,
	)
	if err != nil {
		app.serverErrorResponse(w, req, err)
		return
	}

	if len(media) < int(onboardingMediaLimit) {
		app.errorResponse(w, req, http.StatusServiceUnavailable, "not enough popular media is available for onboarding")
		return
	}
	media = media[:onboardingMediaLimit]

	err = app.writeJSON(w, http.StatusOK, envelope{"media": media}, nil)
	if err != nil {
		app.serverErrorResponse(w, req, err)
	}
}

func (app *application) submitOnboardingRatingsHandler(w http.ResponseWriter, req *http.Request) {
	user := app.contextGetUser(req)

	var input struct {
		Ratings []onboardingRatingInput `json:"ratings"`
	}

	if err := app.readJSON(w, req, &input); err != nil {
		app.badRequestResponse(w, req, err)
		return
	}

	v := validator.New()
	if len(input.Ratings) < onboardingRatingsMin {
		v.AddError("ratings", fmt.Sprintf("must contain at least %d ratings", onboardingRatingsMin))
	}
	if len(input.Ratings) > onboardingRatingsMax {
		v.AddError("ratings", fmt.Sprintf("must not contain more than %d ratings", onboardingRatingsMax))
	}

	type mediaKey struct {
		tmdbID    int32
		mediaType string
	}
	seen := make(map[mediaKey]struct{}, len(input.Ratings))
	ratings := make([]data.OnboardingRating, 0, len(input.Ratings))
	for index, rating := range input.Ratings {
		field := fmt.Sprintf("ratings[%d]", index)
		if rating.TmdbID <= 0 {
			v.AddError(field+".tmdb_id", "must be a positive integer")
		}
		if rating.MediaType != "movie" && rating.MediaType != "tv" {
			v.AddError(field+".media_type", "must be 'movie' or 'tv'")
		}
		key := mediaKey{tmdbID: rating.TmdbID, mediaType: rating.MediaType}
		if _, exists := seen[key]; exists {
			v.AddError(field, "must not duplicate another rating")
		}
		seen[key] = struct{}{}
		data.ValidateRatingValue(v, rating.RatingValue)

		ratings = append(ratings, data.OnboardingRating{
			TmdbID:      rating.TmdbID,
			MediaType:   rating.MediaType,
			RatingValue: rating.RatingValue,
			WatchedDate: rating.WatchedDate,
		})
	}

	if !v.Valid() {
		app.failedValidationResponse(w, req, v.Errors)
		return
	}

	updatedUser, savedRatings, err := app.models.Onboarding.Complete(req.Context(), user.ID, ratings)
	if err != nil {
		if errors.Is(err, data.ErrRecordNotFound) {
			app.notFoundResponse(w, req)
			return
		}
		app.serverErrorResponse(w, req, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{
		"user":    updatedUser,
		"ratings": savedRatings,
	}, nil)
	if err != nil {
		app.serverErrorResponse(w, req, err)
	}
}
