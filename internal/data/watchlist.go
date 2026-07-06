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

type Watchlist struct {
	UserID        uuid.UUID  `json:"user_id"`
	MediaID       uuid.UUID  `json:"media_id"`
	CreatedAt     time.Time  `json:"created_at"`
	Status        string     `json:"status"`
	Source        string     `json:"source"`
	SourceMatchID *uuid.UUID `json:"source_match_id,omitempty"`
}

type UserWatchlistItem struct {
	Media     Media     `json:"media"`
	Watchlist Watchlist `json:"watchlist"`
}

type WatchlistModel struct {
	DB *sql.DB
	q  *database.Queries
}

func (m WatchlistModel) InsertMediaToWatchlist(ctx context.Context, watchlist Watchlist) (Watchlist, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var sourceMatchID uuid.NullUUID
	if watchlist.SourceMatchID != nil {
		sourceMatchID = uuid.NullUUID{
			UUID:  *watchlist.SourceMatchID,
			Valid: true,
		}
	}

	row, err := m.q.InsertMediaToWatchlist(ctx, database.InsertMediaToWatchlistParams{
		UserID:        watchlist.UserID,
		MediaID:       watchlist.MediaID,
		Source:        database.SourceType(watchlist.Source),
		SourceMatchID: sourceMatchID,
	})
	if err != nil {
		return Watchlist{}, fmt.Errorf("insert media to watchlist: %w", err)
	}

	return Watchlist{
		UserID:        row.UserID,
		MediaID:       row.MediaID,
		CreatedAt:     row.CreatedAt,
		Status:        string(row.Status.StatusType),
		Source:        string(row.Source),
		SourceMatchID: nullUUIDToPtr(row.SourceMatchID),
	}, nil
}

func (m WatchlistModel) UpdateWatchlistItemStatus(
	ctx context.Context,
	userID, mediaID uuid.UUID,
	status string,
) (Watchlist, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	row, err := m.q.UpdateWatchlistItemStatus(ctx, database.UpdateWatchlistItemStatusParams{
		UserID:  userID,
		MediaID: mediaID,
		Status: database.NullStatusType{
			StatusType: database.StatusType(status),
			Valid:      true,
		},
	})
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return Watchlist{}, ErrRecordNotFound
		default:
			return Watchlist{}, fmt.Errorf("update watchlist item status: %w", err)
		}
	}

	return Watchlist{
		UserID:        row.UserID,
		MediaID:       row.MediaID,
		CreatedAt:     row.CreatedAt,
		Status:        string(row.Status.StatusType),
		Source:        string(row.Source),
		SourceMatchID: nullUUIDToPtr(row.SourceMatchID),
	}, nil
}

func (m WatchlistModel) DeleteMediaFromWatchlist(ctx context.Context, userID, mediaID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	row, err := m.q.DeleteMediaFromWatchlist(ctx, database.DeleteMediaFromWatchlistParams{
		UserID:  userID,
		MediaID: mediaID,
	})
	if err != nil {
		return fmt.Errorf("delete media from watchlist: %w", err)
	}

	if row == 0 {
		return ErrRecordNotFound
	}

	return nil
}

func (m WatchlistModel) GetAllItemsInWatchlist(
	ctx context.Context,
	createdAt time.Time,
	limit int32, status string,
	userID, mediaID uuid.UUID,
) ([]UserWatchlistItem, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var statusType database.NullStatusType

	if status != "" {
		if status == "watched" {
			statusType = database.NullStatusType{
				StatusType: database.StatusTypeWatched,
				Valid:      true,
			}
		} else {
			statusType = database.NullStatusType{
				StatusType: database.StatusTypeNotWatched,
				Valid:      true,
			}
		}
	}

	rows, err := m.q.GetAllItemsInWatchlist(ctx, database.GetAllItemsInWatchlistParams{
		UserID:          userID,
		Limit:           limit,
		CursorCreatedAt: createdAt,
		CursorMediaID:   mediaID,
		Status:          statusType,
	})
	if err != nil {
		return nil, fmt.Errorf("get all items in watchlist: %w", err)
	}

	var items []UserWatchlistItem
	for _, row := range rows {
		items = append(items, UserWatchlistItem{
			Media: Media{
				ID:            row.MediaID,
				TmdbID:        row.TmdbID,
				Title:         row.Title,
				OriginalTitle: row.OriginalTitle,
				PosterPath:    row.PosterPath,
				BackdropPath:  row.BackdropPath,
				Overview:      row.Overview,
				ReleaseDate:   row.ReleaseDate,
				Runtime:       Runtime(row.Runtime),
				MediaType:     row.MediaType,
			},
			Watchlist: Watchlist{
				UserID:        row.UserID,
				MediaID:       row.MediaID,
				CreatedAt:     row.CreatedAt,
				Status:        string(row.Status.StatusType),
				Source:        string(row.Source),
				SourceMatchID: nullUUIDToPtr(row.SourceMatchID),
			},
		})
	}

	return items, nil
}

func ValidateWatchlistSource(v *validator.Validator, source string) {
	v.Check(source != "", "source", "must be provided")
	v.Check(
		validator.In(
			source,
			string(database.SourceTypeSelf),
			string(database.SourceTypeFromMatch),
		),
		"source",
		"must be a valid source type",
	)
}

func ValidateWatchlistStatus(v *validator.Validator, status string) {
	v.Check(status != "", "status", "must be provided")
	v.Check(
		validator.In(
			status,
			string(database.StatusTypeWatched),
			string(database.StatusTypeNotWatched),
		),
		"status",
		"must be a valid watchlist status",
	)
}
