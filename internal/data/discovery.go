package data

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/parakliite/simila/internal/database"
)

type DiscoveryModel struct {
	DB *sql.DB
	q  *database.Queries
}

type Discovery struct {
	Expiry time.Time `json:"expiry"`
}

func nullableTime(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}

func (d DiscoveryModel) GetDiscoverySyncState(
	ctx context.Context,
	mediaType string,
) (DiscoverySyncState, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	state, err := d.q.GetDiscoverySyncState(ctx, mediaType)
	if err != nil {
		return DiscoverySyncState{}, err
	}

	return DiscoverySyncState{
		MediaType:           state.MediaType,
		NextPage:            state.NextPage,
		ConsecutiveFailures: state.ConsecutiveFailures,
		PausedUntil:         nullableTime(state.PausedUntil),
		LastSuccessAt:       nullableTime(state.LastSuccessAt),
	}, nil
}

func (d DiscoveryModel) UpdateDiscoverySyncSuccess(
	ctx context.Context,
	mediaType string,
	nextPage int32,
) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	return d.q.UpdateDiscoverySyncSuccess(ctx, database.UpdateDiscoverySyncSuccessParams{
		MediaType: mediaType,
		NextPage:  nextPage,
	})
}

func (d DiscoveryModel) UpdateDiscoverySyncFailure(
	ctx context.Context,
	mediaType string,
	consecutiveFailures int32,
	pausedUntil *time.Time,
) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var nullablePausedUntil sql.NullTime
	if pausedUntil != nil {
		nullablePausedUntil = sql.NullTime{Time: *pausedUntil, Valid: true}
	}

	return d.q.UpdateDiscoverySyncFailure(ctx, database.UpdateDiscoverySyncFailureParams{
		MediaType:           mediaType,
		ConsecutiveFailures: consecutiveFailures,
		PausedUntil:         nullablePausedUntil,
	})
}

func (d DiscoveryModel) GetDiscoveryHistoryExpiry(
	ctx context.Context,
	userID, mediaID uuid.UUID,
) (time.Time, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

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
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	return d.q.SetDiscoveryHistoryExpiry(ctx, database.SetDiscoveryHistoryExpiryParams{
		UserID:         userID,
		MediaID:        mediaID,
		NextEligibleAt: eligibleAt,
	})
}
