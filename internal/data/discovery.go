package data

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/deltron-fr/filmbox/server/internal/database"
	"github.com/google/uuid"
)

type DiscoveryModel struct {
	DB *sql.DB
	q  *database.Queries
}

type Discovery struct {
	Expiry time.Time `json:"expiry"`
}

func (d DiscoveryModel) GetDiscoveryHistoryExpiry(ctx context.Context, userID, mediaID uuid.UUID) (time.Time, error) {
	expTime, err := d.q.GetDiscoveryHistoryExpiry(ctx, database.GetDiscoveryHistoryExpiryParams{
		UserID:  userID,
		MediaID: mediaID,
	})
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return time.Time{}, ErrRecordNotFound
		default:
			return time.Time{}, err
		}
	}

	return expTime, nil
}

func (d DiscoveryModel) SetDiscoveryHistoryExpiry(
	ctx context.Context,
	userID, mediaID uuid.UUID,
	eligibleAt time.Time,
) error {
	return d.q.SetDiscoveryHistoryExpiry(ctx, database.SetDiscoveryHistoryExpiryParams{
		UserID:         userID,
		MediaID:        mediaID,
		NextEligibleAt: eligibleAt,
	})
}
