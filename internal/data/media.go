package data

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/deltron-fr/filmbox/server/internal/database"
	"github.com/deltron-fr/filmbox/server/internal/validator"
	"github.com/google/uuid"
)

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

func (m MediaModel) GetMedia(ctx context.Context, id int32, mediaType string) (*Media, error) {
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
