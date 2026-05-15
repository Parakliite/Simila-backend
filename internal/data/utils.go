package data

import (
	"database/sql"
	"errors"
	"time"
)

var ErrRecordNotFound = errors.New("record not found")

func nullTimeToPtr(nt sql.NullTime) *time.Time {
	// Keep nullable timestamp handling in one place so row-to-model mapping stays
	// readable.
	if nt.Valid {
		return &nt.Time
	}
	return nil
}
