package data

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/parakliite/simila/internal/database"
)

type OnboardingModel struct {
	DB *sql.DB
	q  *database.Queries
}

func (m OnboardingModel) GetMedia(
	ctx context.Context,
	userID uuid.UUID,
	popularityThreshold float64,
	limit int32,
) ([]Media, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	rows, err := m.q.GetOnboardingMedia(ctx, database.GetOnboardingMediaParams{
		Popularity: popularityThreshold,
		UserID:     userID,
		Limit:      limit,
	})
	if err != nil {
		return nil, fmt.Errorf("get onboarding media: %w", err)
	}

	media := make([]Media, 0, len(rows))
	for _, row := range rows {
		var names []string
		if err := json.Unmarshal([]byte(row.Genres), &names); err != nil {
			return nil, fmt.Errorf("decode onboarding media genres: %w", err)
		}
		genres := make([]Genre, 0, len(names))
		for _, name := range names {
			genres = append(genres, Genre{GenreName: name})
		}

		media = append(media, Media{
			ID:            row.ID,
			CreatedAt:     row.CreatedAt,
			UpdatedAt:     row.UpdatedAt,
			TmdbID:        row.TmdbID,
			Title:         row.Title,
			OriginalTitle: row.OriginalTitle,
			PosterPath:    row.PosterPath,
			BackdropPath:  row.BackdropPath,
			Overview:      row.Overview,
			ReleaseDate:   row.ReleaseDate,
			Runtime:       Runtime(row.Runtime),
			Popularity:    row.Popularity,
			MediaType:     row.MediaType,
			Genre:         genres,
		})
	}

	return media, nil
}

func (m OnboardingModel) Complete(
	ctx context.Context,
	userID uuid.UUID,
	ratings []OnboardingRating,
) (*User, []Rating, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	tx, err := m.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("begin onboarding transaction: %w", err)
	}

	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	q := m.q.WithTx(tx)
	savedRatings := make([]Rating, 0, len(ratings))
	for _, input := range ratings {
		mediaID, err := q.GetOrCreateMediaID(ctx, database.GetOrCreateMediaIDParams{
			TmdbID:    input.TmdbID,
			MediaType: input.MediaType,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("get or create onboarding media: %w", err)
		}

		var watchedDate sql.NullTime
		if input.WatchedDate != nil {
			watchedDate = sql.NullTime{Time: *input.WatchedDate, Valid: true}
		}

		row, err := q.UpsertUserRating(ctx, database.UpsertUserRatingParams{
			UserID:      userID,
			MediaID:     mediaID,
			RatingValue: translateRatingValueInbound(input.RatingValue),
			WatchedDate: watchedDate,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("save onboarding rating: %w", err)
		}

		savedRatings = append(savedRatings, Rating{
			MediaID:     row.MediaID,
			UserID:      row.UserID,
			RatingValue: translateRatingValueOutbound(row.RatingValue),
			WatchedDate: nullTimeToPtr(row.WatchedDate),
			CreatedAt:   row.CreatedAt,
			UpdatedAt:   row.UpdatedAt,
		})
	}

	row, err := q.MarkUserOnboarded(ctx, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("mark user onboarded: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, fmt.Errorf("commit onboarding transaction: %w", err)
	}
	committed = true

	return &User{
		ID:                row.ID,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
		Name:              row.Name,
		Email:             row.Email,
		ProfilePictureURL: row.ProfilePictureUrl.String,
		About:             row.About.String,
		Activated:         row.Activated,
		Onboarded:         row.Onboarded,
		Version:           row.Version,
	}, savedRatings, nil
}
