package data

import (
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
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

func nullUUIDToPtr(nu uuid.NullUUID) *uuid.UUID {
	if nu.Valid {
		return &nu.UUID
	}
	return nil
}
