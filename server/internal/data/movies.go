package data

import (
	"database/sql"
	"time"

	"github.com/deltron-fr/filmbox/server/internal/database"
	"github.com/deltron-fr/filmbox/server/internal/validator"
	"github.com/google/uuid"
)

type Genre struct {
	GenreID   uuid.UUID    `json:"genre_id"`
	CreatedAt   time.Time	  `json:"_created_at"`
	UpdatedAt   time.Time	  `json:"_updated_at"`
	GenreName string `json:"genre_name"`
}

type Rating struct {
	RatingValue int    `json:"rating_value"`
	RatingName  string `json:"rating_name"`
}

type Movie struct {
	ID          uuid.UUID 	  `json:"_id,omitempty"`
	CreatedAt   time.Time	  `json:"created_at"`
	UpdatedAt   time.Time	  `json:"updated_at"`
	ImdbID      string        `json:"imdb_id"`
	Title       string        `json:"title"`
	PosterPath  string        `json:"poster_path"`
	YouTubeID   string        `json:"_"`
	Genre       []Genre       `json:"genre"`
	AdminReview string        `json:"admin_review"`
	Rating     Rating         `json:"ranking"`
}

func ValidateMovie(v *validator.Validator, movie *Movie) {
	v.Check(movie.Title != "", "title", "must be provided")
    v.Check(len(movie.Title) <= 500, "title", "must not be more than 500 bytes long")
}

type MovieModel struct {
	DB *sql.DB
	q *database.Queries
}