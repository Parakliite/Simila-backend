package data

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/parakliite/simila/internal/database"
)

type RecommendationModel struct {
	DB *sql.DB
	q  *database.Queries
}

type Recommendations struct {
	Media []Media `json:"media"`
}

func (r RecommendationModel) GetUserRecommendations(
	ctx context.Context,
	userID, targetUserID uuid.UUID,
	threshold, limit int32,
) (Recommendations, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	rows, err := r.q.GetUserRecommendations(ctx, database.GetUserRecommendationsParams{
		TargetUserID: targetUserID,
		UserID:       userID,
		Threshold:    threshold,
		Limit:        limit,
	})
	if err != nil {
		return Recommendations{}, err
	}

	var recs Recommendations

	for _, row := range rows {
		recs.Media = append(recs.Media, Media{
			ID:            row.ID,
			TmdbID:        row.TmdbID,
			Title:         row.Title,
			Overview:      row.Overview,
			OriginalTitle: row.OriginalTitle,
			MediaType:     row.MediaType,
			PosterPath:    row.PosterPath,
			ReleaseDate:   row.ReleaseDate,
		})
	}

	return recs, nil
}
