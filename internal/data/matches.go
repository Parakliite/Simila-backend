package data

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/parakliite/simila/internal/database"
	"gonum.org/v1/gonum/mat"
)

type MatchModel struct {
	DB *sql.DB
	q  *database.Queries
}

type Match struct {
	User               User      `json:"user"`
	Score              float64   `json:"score"`
	SharedMedia        int32     `json:"shared_media_count"`
	LastRecalculatedAt time.Time `json:"-"`
}

func (m MatchModel) MapMediaToIndex(ctx context.Context) (map[uuid.UUID]int, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	allIds, err := m.q.GetAllMedia(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not get all media: %w", err)
	}

	indexMap := make(map[uuid.UUID]int, len(allIds))
	for i, id := range allIds {
		indexMap[id] = i
	}

	return indexMap, nil
}

func cosineSimilarity(vecA, vecB *mat.VecDense) float64 {
	dotProduct := mat.Dot(vecA, vecB)
	normA := mat.Norm(vecA, 2)
	normB := mat.Norm(vecB, 2)

	if normA == 0 || normB == 0 {
		return 0.0
	}

	return dotProduct / (normA * normB)
}

func allUsersRatings(ratings []Rating, indexMap map[uuid.UUID]int, size int) []float64 {
	data := make([]float64, size)

	for _, rating := range ratings {
		i := indexMap[rating.MediaID]
		data[i] = float64(rating.RatingValue)
	}

	return data
}

func (m MatchModel) CalculateSimilarities(
	ctx context.Context,
	targetUserID uuid.UUID,
	ratings []Rating,
	indexMap map[uuid.UUID]int,
) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	targetRatingsVec := allUsersRatings(ratings, indexMap, len(indexMap))

	allUsers, err := m.q.GetAllUsers(ctx)
	if err != nil {
		return fmt.Errorf("could not get all users: %w", err)
	}

	for _, user := range allUsers {
		if user == targetUserID {
			continue
		}

		otherUserRatings, err := m.q.GetAllUserRatings(ctx, user)
		if err != nil {
			return fmt.Errorf("could not get all user ratings: %w", err)
		}

		var otherUserRatingsParsed []Rating

		for _, row := range otherUserRatings {
			otherUserRatingsParsed = append(otherUserRatingsParsed, toRating(row))
		}

		OtherRatingsVec := allUsersRatings(otherUserRatingsParsed, indexMap, len(indexMap))

		cosineSim := cosineSimilarity(
			mat.NewVecDense(len(targetRatingsVec), targetRatingsVec),
			mat.NewVecDense(len(OtherRatingsVec), OtherRatingsVec),
		)

		shared := 0
		for i := range targetRatingsVec {
			if targetRatingsVec[i] != 0 && OtherRatingsVec[i] != 0 {
				shared++
			}
		}

		if shared > 4 && cosineSim >= 0.5 {
			_, err := m.q.UpsertUserMatch(ctx, database.UpsertUserMatchParams{
				TargetUserID:     targetUserID,
				OtherUserID:      user,
				Score:            cosineSim,
				SharedMediaCount: int32(shared),
			})
			if err != nil {
				return fmt.Errorf("could not upsert user match: %w", err)
			}
		}

	}

	return nil
}

func (m MatchModel) GetAllUserRatingsForMatches(ctx context.Context, userID uuid.UUID) ([]Rating, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	rows, err := m.q.GetAllUserRatings(ctx, userID)
	if err != nil {
		return nil, err
	}

	var ratings []Rating

	for _, row := range rows {
		ratings = append(ratings, toRating(row))
	}

	return ratings, nil
}

func (m MatchModel) GetUserMatches(
	ctx context.Context,
	targetUserID uuid.UUID,
	limit int32,
	cursorOtherUserID uuid.NullUUID,
	cursorScore sql.NullFloat64,
	cursorSharedMediaCount sql.NullInt32,
	cursorLastRecalculatedAt sql.NullTime,
) ([]Match, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	rows, err := m.q.GetUserMatches(ctx, database.GetUserMatchesParams{
		TargetUserID:             targetUserID,
		Limit:                    limit,
		CursorScore:              cursorScore,
		CursorSharedMediaCount:   cursorSharedMediaCount,
		CursorLastRecalculatedAt: cursorLastRecalculatedAt,
		CursorOtherUserID:        cursorOtherUserID,
	})
	if err != nil {
		return nil, err
	}

	var matches []Match

	for _, row := range rows {
		matches = append(matches, Match{
			User: User{
				Name:              row.UserName,
				ProfilePictureURL: row.UserProfilePictureUrl.String,
				ID:                row.UserID,
				About:             row.UserAbout.String,
				CreatedAt:         row.UserCreatedAt,
			},
			Score:              row.Score,
			SharedMedia:        row.SharedMediaCount,
			LastRecalculatedAt: row.LastRecalculatedAt,
		})
	}

	return matches, nil
}

func (m MatchModel) CheckMatchExists(
	ctx context.Context,
	userID, otherUserID uuid.UUID,
) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	return m.q.CheckMatchExists(ctx, database.CheckMatchExistsParams{
		TargetUserID: userID,
		OtherUserID: otherUserID,
	})
}

func toRating(row database.GetAllUserRatingsRow) Rating {
	return Rating{
		MediaID:     row.MediaID,
		UserID:      row.UserID,
		RatingValue: translateRatingValueOutbound(row.RatingValue),
	}
}
