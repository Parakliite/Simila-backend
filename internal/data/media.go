package data

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/parakliite/simila/internal/database"
	"github.com/parakliite/simila/internal/validator"
)

var placeholderDate = time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)

type Genre struct {
	GenreName string `json:"name"`
}

type Media struct {
	ID            uuid.UUID `json:"_id,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	TmdbID        int32     `json:"tmdb_id"`
	Title         string    `json:"title"`
	ReleaseDate   time.Time `json:"release_date"`
	Runtime       Runtime   `json:"runtime"`
	Overview      string    `json:"overview"`
	OriginalTitle string    `json:"original_title"`
	PosterPath    string    `json:"poster_path"`
	BackdropPath  string    `json:"backdrop_path"`
	MediaType     string    `json:"media_type"`
	Genre         []Genre   `json:"genre"`
}

func ValidateMedia(v *validator.Validator, media *Media) {
	v.Check(media.Title != "", "title", "must be provided")
	v.Check(len(media.Title) <= 500, "title", "must not be more than 500 bytes long")
}

type MediaModel struct {
	DB *sql.DB
	q  *database.Queries
}

func (m MediaModel) InsertMedia(ctx context.Context, media *Media) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	row, err := m.q.CreateMedia(ctx,
		database.CreateMediaParams{
			TmdbID:        media.TmdbID,
			Title:         media.Title,
			OriginalTitle: media.OriginalTitle,
			PosterPath:    media.PosterPath,
			BackdropPath:  media.BackdropPath,
			Overview:      media.Overview,
			ReleaseDate:   media.ReleaseDate,
			Runtime:       int32(media.Runtime),
			MediaType:     media.MediaType,
		},
	)
	if err != nil {
		return err
	}

	media.ID = row.ID
	media.CreatedAt = row.CreatedAt
	media.UpdatedAt = row.UpdatedAt

	for _, g := range media.Genre {
		id, err := m.q.GetGenre(ctx, g.GenreName)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				continue
			} else {
				return err
			}
		}
		err = m.q.CreateMediaGenre(ctx, database.CreateMediaGenreParams{
			GenreID: id,
			MediaID: media.ID,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (m MediaModel) GetOrCreateMediaByTmdbID(ctx context.Context, tmdbID int32, mediaType string) (uuid.UUID, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	existing, err := m.q.GetMedia(ctx, database.GetMediaParams{
		TmdbID:    tmdbID,
		MediaType: mediaType,
	})
	if err == nil {
		return existing.ID, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return uuid.UUID{}, err
	}

	row, err := m.q.CreateMedia(ctx, database.CreateMediaParams{
		TmdbID:        tmdbID,
		Title:         "",
		OriginalTitle: "",
		PosterPath:    "",
		BackdropPath:  "",
		Overview:      "",
		ReleaseDate:   placeholderDate,
		Runtime:       1,
		MediaType:     mediaType,
	})
	if err != nil {
		return uuid.UUID{}, err
	}

	return row.ID, nil
}

func (m MediaModel) GetMedia(ctx context.Context, id int32, mediaType string) (*Media, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	media, err := m.q.GetMedia(ctx, database.GetMediaParams{
		TmdbID:    id,
		MediaType: mediaType,
	})
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	genres := make([]Genre, 0)
	if len(media.Genres) > 0 {
		var genreNames []string
		if err := json.Unmarshal([]byte(media.Genres), &genreNames); err != nil {
			return nil, err
		}

		genres = make([]Genre, 0, len(genreNames))
		for _, name := range genreNames {
			genres = append(genres, Genre{GenreName: name})
		}
	}

	return &Media{
		ID:            media.ID,
		Genre:         genres,
		TmdbID:        media.TmdbID,
		Title:         media.Title,
		OriginalTitle: media.OriginalTitle,
		PosterPath:    media.PosterPath,
		BackdropPath:  media.BackdropPath,
		Overview:      media.Overview,
		ReleaseDate:   media.ReleaseDate,
		Runtime:       Runtime(media.Runtime),
		MediaType:     media.MediaType,
	}, nil
}
