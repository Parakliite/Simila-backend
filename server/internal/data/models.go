package data

import (
	"database/sql"
	"errors"

	"github.com/deltron-fr/filmbox/server/internal/database"
)

var (
	ErrRecordNotFound = errors.New("record not found")
)

type Models struct {
	Movies MovieModel
}

func NewModels(db *sql.DB) Models {

	return Models{
		Movies: MovieModel{
			DB: db,
			q: database.New(db),
		},
	}
}

