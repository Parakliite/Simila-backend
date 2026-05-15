package data

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/deltron-fr/filmbox/server/internal/database"
	"github.com/google/uuid"
	"gonum.org/v1/gonum/mat"
)

type MatchModel struct {
	DB *sql.DB
	q  *database.Queries
}

func (m MatchModel) mapMediaToIndex() (map[uuid.UUID]int, error) {
	allIds, err := m.q.GetAllMedia(context.Background())
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

func (m MatchModel) GetSimilarities(
	targetUserID uuid.UUID,
	ratings []Rating,
	indexMap map[uuid.UUID]int,
) (map[uuid.UUID]float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	targetRatingsVec := allUsersRatings(ratings, indexMap, len(indexMap))

	similarities := make(map[uuid.UUID]float64)

	allUsers, err := m.q.GetAllUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not get all users: %w", err)
	}

	for _, user := range allUsers {
		otherUserRatings, err := m.q.GetAllUserRatings(ctx, user)
		if err != nil {
			return nil, fmt.Errorf("could not get all user ratings: %w", err)
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
			m.q.UpsertUserMatch(ctx, database.UpsertUserMatchParams{
				TargetUserID:     targetUserID,
				OtherUserID:      user,
				Score:            cosineSim,
				SharedMediaCount: int32(shared),
			})
		}

		similarities[user] = cosineSim
	}

	return similarities, nil
}

func toRating(row database.GetAllUserRatingsRow) Rating {
	return Rating{
		MediaID:     row.MediaID,
		UserID:      row.UserID,
		RatingValue: row.RatingValue,
	}
}
