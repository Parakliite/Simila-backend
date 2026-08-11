package data

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/parakliite/simila/internal/database"
)

type HomeModel struct {
	DB *sql.DB
	q  *database.Queries
}

type MatchRatedMedia struct {
	Media      Media   `json:"media"`
	MatchUser  User    `json:"match_user"`
	MatchScore float64 `json:"match_score"`
	Rating     Rating  `json:"rating"`
}

func (m HomeModel) GetTopRatedByMatches(ctx context.Context, userID uuid.UUID, limit int32) ([]MatchRatedMedia, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	rows, err := m.q.GetTopRatedByMatches(ctx, database.GetTopRatedByMatchesParams{
		TargetUserID: userID,
		Limit:        limit,
	})
	if err != nil {
		return nil, err
	}

	var results []MatchRatedMedia
	seen := make(map[uuid.UUID]bool)

	for _, row := range rows {
		if seen[row.MediaID] {
			continue
		}
		seen[row.MediaID] = true

		results = append(results, MatchRatedMedia{
			Media: Media{
				ID:         row.MediaID,
				Title:      row.Title,
				PosterPath: row.PosterPath,
				MediaType:  row.MediaType,
			},
			MatchUser: User{
				ID:                row.MatchUserID,
				Name:              row.MatchName,
				ProfilePictureURL: row.MatchAvatar.String,
			},
			MatchScore: row.MatchScore,
			Rating: Rating{
				MediaID:     row.MediaID,
				UserID:      row.MatchUserID,
				RatingValue: translateRatingValueOutbound(row.RatingValue),
				CreatedAt:   row.RatedAt,
			},
		})
	}

	return results, nil
}
