package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/deltron-fr/filmbox/server/internal/database"
	"github.com/deltron-fr/filmbox/server/internal/validator"
	"github.com/google/uuid"
)

type RatingModel struct {
	DB *sql.DB
	q  *database.Queries
}

type Rating struct {
	MediaID     uuid.UUID  `json:"media_id"`
	UserID      uuid.UUID  `json:"user_id"`
	RatingValue float64    `json:"rating_value"`
	WatchedDate *time.Time `json:"watched_date"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type UserRating struct {
	Media  Media  `json:"media"`
	Rating Rating `json:"rating"`
}

func translateRatingValueOutbound(value int32) float64 {
	return float64(value) / 2.0
}

func translateRatingValueInbound(value float64) int32 {
	return int32(value * 2)
}

func (r RatingModel) UpsertUserRating(ctx context.Context, rating Rating) (Rating, error) {
	var watchedDate sql.NullTime
	if rating.WatchedDate != nil {
		watchedDate = sql.NullTime{
			Time:  *rating.WatchedDate,
			Valid: true,
		}
	}

	row, err := r.q.UpsertUserRating(ctx, database.UpsertUserRatingParams{
		UserID:      rating.UserID,
		MediaID:     rating.MediaID,
		WatchedDate: watchedDate,
		RatingValue: translateRatingValueInbound(rating.RatingValue),
	})
	if err != nil {
		return Rating{}, fmt.Errorf("upsert user rating: %w", err)
	}

	return Rating{
		MediaID:     row.MediaID,
		UserID:      row.UserID,
		RatingValue: translateRatingValueOutbound(row.RatingValue),
		WatchedDate: nullTimeToPtr(row.WatchedDate),
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}, nil
}

func (r RatingModel) DeleteUserRating(ctx context.Context, userID, mediaID uuid.UUID) error {
	row, err := r.q.SoftDeleteUserRating(ctx, database.SoftDeleteUserRatingParams{
		UserID:  userID,
		MediaID: mediaID,
	})
	if err != nil {
		return fmt.Errorf("soft delete user rating: %w", err)
	}

	if row == 0 {
		return ErrRecordNotFound
	}

	return nil
}

func (r RatingModel) GetUsersRatings(
	ctx context.Context,
	createdAt time.Time,
	limit int32,
	userID, mediaID uuid.UUID,
) ([]UserRating, error) {
	rows, err := r.q.GetUsersRatings(ctx, database.GetUsersRatingsParams{
		UserID:          userID,
		CursorCreatedAt: createdAt,
		CursorMediaID:   mediaID,
		Limit:           limit,
	})
	if err != nil {
		return []UserRating{}, fmt.Errorf("get user ratings: %w", err)
	}

	var userRatings []UserRating

	for _, row := range rows {
		ratingRow := toUserRating(row)
		ratingRow.Rating.RatingValue = translateRatingValueOutbound(int32(ratingRow.Rating.RatingValue))
		userRatings = append(userRatings, ratingRow)

	}

	return userRatings, nil
}

func (r RatingModel) GetUsersRating(
	ctx context.Context,
	userID, mediaID uuid.UUID,
) (UserRating, error) {
	row, err := r.q.GetSingleUserRating(ctx, database.GetSingleUserRatingParams{
		UserID:  userID,
		MediaID: mediaID,
	})
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return UserRating{}, ErrRecordNotFound
		default:
			return UserRating{}, fmt.Errorf("get user rating: %w", err)
		}
	}

	return UserRating{
		Media: Media{
			Title:       row.Title,
			Overview:    row.Overview,
			PosterPath:  row.PosterPath,
			ReleaseDate: row.ReleaseDate,
			MediaType:   row.MediaType,
		},
		Rating: Rating{
			UserID:      userID,
			MediaID:     row.MediaID,
			RatingValue: translateRatingValueOutbound(row.RatingValue),
			WatchedDate: nullTimeToPtr(row.WatchedDate),
			CreatedAt:   row.CreatedAt,
			UpdatedAt:   row.UpdatedAt,
		},
	}, nil
}

func (r RatingModel) GetAllRatingsForSingleMedia(
	ctx context.Context,
	createdAt time.Time,
	mediaID, userID uuid.UUID,
	limit int32,
) ([]UserRating, error) {
	rows, err := r.q.GetAllRatingsForSingleMedia(
		ctx,
		database.GetAllRatingsForSingleMediaParams{
			MediaID:         mediaID,
			CursorCreatedAt: createdAt,
			CursorUserID:    userID,
			Limit:           limit,
		},
	)
	if err != nil {
		return []UserRating{}, fmt.Errorf("ratings for single media: %w", err)
	}

	var usersRating []UserRating
	for _, row := range rows {
		ratingRow := toUserRatingForMedia(row)
		ratingRow.Rating.RatingValue = translateRatingValueOutbound(int32(ratingRow.Rating.RatingValue))

		usersRating = append(usersRating, ratingRow)
	}

	return usersRating, nil
}

func toUserRating(row database.GetUsersRatingsRow) UserRating {
	return UserRating{
		Media: Media{
			Title:       row.Title,
			Overview:    row.Overview,
			PosterPath:  row.PosterPath,
			ReleaseDate: row.ReleaseDate,
			MediaType:   row.MediaType,
		},
		Rating: Rating{
			MediaID:     row.MediaID,
			UserID:      row.UserID,
			WatchedDate: nullTimeToPtr(row.WatchedDate),
			CreatedAt:   row.CreatedAt,
			UpdatedAt:   row.UpdatedAt,
		},
	}
}

func toUserRatingForMedia(row database.GetAllRatingsForSingleMediaRow) UserRating {
	return UserRating{
		Media: Media{
			Title:        row.Title,
			Overview:     row.Overview,
			PosterPath:   row.PosterPath,
			ReleaseDate:  row.ReleaseDate,
			MediaType:    row.MediaType,
			BackdropPath: row.BackdropPath,
			Runtime:      Runtime(row.Runtime),
			TmdbID:       row.TmdbID,
		},
		Rating: Rating{
			MediaID:     row.MediaID,
			UserID:      row.UserID,
			WatchedDate: nullTimeToPtr(row.WatchedDate),
			CreatedAt:   row.CreatedAt,
			UpdatedAt:   row.UpdatedAt,
		},
	}
}

func ValidateRatingValue(v *validator.Validator, rValue float64) {
	v.Check(rValue <= 5.0, "rating_value", "rating value must be between 1 to 10")
	v.Check(rValue >= 1.0, "rating_value", "rating value must be between 1 to 10")
}
