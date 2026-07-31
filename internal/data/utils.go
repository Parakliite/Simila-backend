package data

import (
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrRecordNotFound = errors.New("record not found")
	ErrDuplicateEmail = errors.New("duplicate email")
	ErrEditConflict   = errors.New(
		"unable to update the record due to an edit conflict, please try again",
	)
)

func nullTimeToPtr(nt sql.NullTime) *time.Time {
	// Keep nullable timestamp handling in one place so row-to-model mapping stays
	// readable.
	if nt.Valid {
		return &nt.Time
	}
	return nil
}

func nullUUIDToPtr(nu uuid.NullUUID) *uuid.UUID {
	if nu.Valid {
		return &nu.UUID
	}
	return nil
}
