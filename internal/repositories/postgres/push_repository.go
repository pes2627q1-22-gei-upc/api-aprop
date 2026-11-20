package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"api-aprop/internal/models/push_model"
	"api-aprop/internal/models/user_model"
	"api-aprop/internal/repositories/postgres/sqlc"
	"api-aprop/internal/repositories/push_repository"
)

// PushRepository stores the devices of the users and the outbox of the push notifications in Postgres.
type PushRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

var _ push_repository.Repository = (*PushRepository)(nil)

func NewPushRepository(pool *pgxpool.Pool) *PushRepository {
	return &PushRepository{pool: pool, queries: sqlc.New(pool)}
}

func (r *PushRepository) UpsertDevice(ctx context.Context, userID string, device push_model.Device) error {
	uid, err := user_model.ParseID(userID)
	if err != nil {
		return err
	}
	err = r.queries.UpsertDeviceToken(ctx, sqlc.UpsertDeviceTokenParams{Token: device.Token, UserID: uid, Platform: device.Platform})
	if err != nil {
		return fmt.Errorf("upsert device token of user %s: %w", userID, err)
	}
	return nil
}

func (r *PushRepository) DeleteDevice(ctx context.Context, userID, token string) (bool, error) {
	uid, err := user_model.ParseID(userID)
	if err != nil {
		return false, err
	}
	rows, err := r.queries.DeleteUserDeviceToken(ctx, sqlc.DeleteUserDeviceTokenParams{Token: token, UserID: uid})
	if err != nil {
		return false, fmt.Errorf("delete device token of user %s: %w", userID, err)
	}
	return rows > 0, nil
}

func (r *PushRepository) DeleteDeviceTokens(ctx context.Context, tokens []string) error {
	if len(tokens) == 0 {
		return nil
	}
	if err := r.queries.DeleteDeviceTokens(ctx, tokens); err != nil {
		return fmt.Errorf("delete %d device tokens: %w", len(tokens), err)
	}
	return nil
}

func (r *PushRepository) DeviceTokens(ctx context.Context, userIDs []string) (map[string][]string, error) {
	tokens := map[string][]string{}
	if len(userIDs) == 0 {
		return tokens, nil
	}
	uids := make([]pgtype.UUID, 0, len(userIDs))
	for _, id := range userIDs {
		uid, err := user_model.ParseID(id)
		if err != nil {
			return nil, err
		}
		uids = append(uids, uid)
	}
	rows, err := r.queries.GetDeviceTokensByUsers(ctx, uids)
	if err != nil {
		return nil, fmt.Errorf("get device tokens of %d users: %w", len(userIDs), err)
	}
	for _, row := range rows {
		userID := row.Userid.String()
		tokens[userID] = append(tokens[userID], row.Token)
	}
	return tokens, nil
}

func (r *PushRepository) ExpirePending(ctx context.Context, maxAge time.Duration) (int64, error) {
	expired, err := r.queries.ExpirePendingPushes(ctx, pgtype.Interval{Microseconds: maxAge.Microseconds(), Valid: true})
	if err != nil {
		return 0, fmt.Errorf("expire pending pushes: %w", err)
	}
	return expired, nil
}

func (r *PushRepository) ProcessPending(ctx context.Context, limit int, send push_repository.SendFunc) (int, error) {
	claimed := 0
	err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		q := r.queries.WithTx(tx)
		rows, err := q.ClaimPendingPushes(ctx, int32(limit))
		if err != nil {
			return fmt.Errorf("claim pending pushes: %w", err)
		}
		claimed = len(rows)
		if claimed == 0 {
			return nil
		}

		batch := make([]push_model.PendingPush, 0, len(rows))
		for _, row := range rows {
			batch = append(batch, toPendingPush(row))
		}
		resolved, err := send(ctx, batch)
		if err != nil {
			return err
		}
		if len(resolved) == 0 {
			return nil
		}
		ids := make([]int32, 0, len(resolved))
		for _, id := range resolved {
			ids = append(ids, int32(id))
		}
		if err := q.MarkPushed(ctx, ids); err != nil {
			return fmt.Errorf("mark %d notifications as pushed: %w", len(ids), err)
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return claimed, nil
}

func toPendingPush(row sqlc.ClaimPendingPushesRow) push_model.PendingPush {
	p := push_model.PendingPush{
		ID:          int(row.ID),
		UserID:      row.Userid.String(),
		Type:        row.Typename,
		Text:        row.Text,
		PushEnabled: row.Pushenabled,
	}
	if row.Event.Valid {
		eventID := int(row.Event.Int32)
		p.EventID = &eventID
	}
	if row.Friend.Valid {
		friendID := row.Friend.String()
		p.FriendID = &friendID
	}
	return p
}
