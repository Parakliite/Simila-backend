package data

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/parakliite/simila/internal/database"
)

func (m UserModel) SearchUsers(ctx context.Context, query string, limit int32) ([]User, error) {
	if query == "" {
		return nil, fmt.Errorf("empty query")
	}

	rows, err := m.q.SearchUsers(ctx, database.SearchUsersParams{
		Query: sql.NullString{
			String: query,
			Valid:  true,
		},
		Limit: limit,
	})
	if err != nil {
		return nil, err
	}

	var users []User

	if len(rows) == 0 {
		return users, nil
	}

	for _, row := range rows {
		users = append(users, User{
			ID:                row.ID,
			Name:              row.Name,
			Email:             row.Email,
			CreatedAt:         row.CreatedAt,
			UpdatedAt:         row.UpdatedAt,
			ProfilePictureURL: row.ProfilePictureUrl.String,
			About:             row.About.String,
		})
	}

	return users, nil
}
