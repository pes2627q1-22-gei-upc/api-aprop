package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"api-aprop/internal/models/user_model"
	"api-aprop/internal/repositories/postgres/sqlc"
	"api-aprop/internal/repositories/user_repository"
)

func (r *UserRepository) BlockUser(ctx context.Context, blocker, blocked pgtype.UUID) (time.Time, error) {
	var date pgtype.Timestamp
	err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		q := r.queries.WithTx(tx)
		var err error
		date, err = q.BlockUser(ctx, sqlc.BlockUserParams{Blocker: blocker, Blocked: blocked})
		if errors.Is(err, pgx.ErrNoRows) {
			return user_repository.ErrAlreadyBlocked
		}
		if err != nil {
			return fmt.Errorf("block user %s: %w", blocked, err)
		}
		if err := q.RemoveFriendship(ctx, sqlc.RemoveFriendshipParams{User1: blocker, User2: blocked}); err != nil {
			return fmt.Errorf("remove friendship with blocked user %s: %w", blocked, err)
		}
		if err := q.RejectActiveFriendRequests(ctx, sqlc.RejectActiveFriendRequestsParams{User1: blocker, User2: blocked}); err != nil {
			return fmt.Errorf("reject friend requests with blocked user %s: %w", blocked, err)
		}
		return nil
	})
	if err != nil {
		return time.Time{}, err
	}
	return date.Time, nil
}

func (r *UserRepository) UnblockUser(ctx context.Context, blocker, blocked pgtype.UUID) (bool, error) {
	rows, err := r.queries.UnblockUser(ctx, sqlc.UnblockUserParams{Blocker: blocker, Blocked: blocked})
	if err != nil {
		return false, fmt.Errorf("unblock user %s: %w", blocked, err)
	}
	return rows > 0, nil
}

func (r *UserRepository) ListBlockedUsers(ctx context.Context, blocker pgtype.UUID) ([]user_model.BlockedUser, error) {
	rows, err := r.queries.ListBlockedUsers(ctx, blocker)
	if err != nil {
		return nil, fmt.Errorf("list users blocked by %s: %w", blocker, err)
	}
	users := make([]user_model.BlockedUser, 0, len(rows))
	for _, row := range rows {
		users = append(users, user_model.BlockedUser{
			UserSummary: user_model.UserSummary{
				ID:             row.ID,
				Username:       row.Username,
				Name:           row.Name.String,
				ProfilePicture: row.Profilepicture.String,
			},
			DateBlocked: row.Dateblocked.Time,
		})
	}
	return users, nil
}
