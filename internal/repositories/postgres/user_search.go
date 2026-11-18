package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"

	"api-aprop/internal/models/user_model"
	"api-aprop/internal/repositories/postgres/sqlc"
)

// likeEscaper escapes the LIKE wildcards, so a query like "joan_p" or "50%" matches those characters.
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

func (r *UserRepository) SearchUsers(ctx context.Context, requester pgtype.UUID, query string, limit, offset int) ([]user_model.UserSummary, int64, error) {
	pattern := likeEscaper.Replace(query)

	total, err := r.queries.CountSearchUsers(ctx, sqlc.CountSearchUsersParams{Requester: requester, Pattern: pattern})
	if err != nil {
		return nil, 0, fmt.Errorf("count users matching %q: %w", query, err)
	}
	if total == 0 {
		return []user_model.UserSummary{}, 0, nil
	}

	rows, err := r.queries.SearchUsers(ctx, sqlc.SearchUsersParams{
		Requester:  requester,
		Pattern:    pattern,
		PageSize:   int32(limit),
		PageOffset: int32(offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("search users matching %q: %w", query, err)
	}

	users := make([]user_model.UserSummary, 0, len(rows))
	for _, row := range rows {
		users = append(users, user_model.UserSummary{
			ID:             row.ID,
			Username:       row.Username,
			Name:           row.Name.String,
			ProfilePicture: row.Profilepicture.String,
		})
	}
	return users, total, nil
}
