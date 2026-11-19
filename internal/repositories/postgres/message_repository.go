package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"api-aprop/internal/models/message_model"
	"api-aprop/internal/models/user_model"
	"api-aprop/internal/repositories/message_repository"
	"api-aprop/internal/repositories/postgres/sqlc"
)

// MessageRepository stores the event chat messages in Postgres using the sqlc generated queries.
type MessageRepository struct {
	queries *sqlc.Queries
}

var _ message_repository.Repository = (*MessageRepository)(nil)

func NewMessageRepository(pool *pgxpool.Pool) *MessageRepository {
	return &MessageRepository{queries: sqlc.New(pool)}
}

func (r *MessageRepository) Create(ctx context.Context, userID string, eventID int, text string) (message_model.Message, error) {
	uid, err := user_model.ParseID(userID)
	if err != nil {
		return message_model.Message{}, err
	}
	row, err := r.queries.CreateEventMessage(ctx, sqlc.CreateEventMessageParams{UserID: uid, EventID: int32(eventID), Message: text})
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case foreignKeyViolation:
			return message_model.Message{}, fmt.Errorf("send message to event %d: %w", eventID, message_repository.ErrEventNotFound)
		case uniqueViolation:
			return message_model.Message{}, fmt.Errorf("send message to event %d: %w", eventID, message_repository.ErrDuplicateMessage)
		}
	}
	if err != nil {
		return message_model.Message{}, fmt.Errorf("send message to event %d: %w", eventID, err)
	}
	return toMessage(eventID, row.Userid, row.Username, row.Name, row.Profilepicture, row.Message, row.Datesent), nil
}

func (r *MessageRepository) List(ctx context.Context, eventID, limit, offset int) ([]message_model.Message, error) {
	rows, err := r.queries.ListEventMessages(ctx, sqlc.ListEventMessagesParams{
		EventID:    int32(eventID),
		PageSize:   int32(limit),
		PageOffset: int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("list messages of event %d: %w", eventID, err)
	}
	messages := make([]message_model.Message, 0, len(rows))
	for _, row := range rows {
		messages = append(messages, toMessage(eventID, row.Userid, row.Username, row.Name, row.Profilepicture, row.Message, row.Datesent))
	}
	return messages, nil
}

func (r *MessageRepository) Count(ctx context.Context, eventID int) (int64, error) {
	total, err := r.queries.CountEventMessages(ctx, int32(eventID))
	if err != nil {
		return 0, fmt.Errorf("count messages of event %d: %w", eventID, err)
	}
	return total, nil
}

func (r *MessageRepository) IsSignedUp(ctx context.Context, userID string, eventID int) (bool, error) {
	uid, err := user_model.ParseID(userID)
	if err != nil {
		return false, err
	}
	ok, err := r.queries.IsUserSignedUp(ctx, sqlc.IsUserSignedUpParams{UserID: uid, EventID: int32(eventID)})
	if err != nil {
		return false, fmt.Errorf("check sign up of user %s to event %d: %w", userID, eventID, err)
	}
	return ok, nil
}

func toMessage(eventID int, userID pgtype.UUID, username string, name, picture pgtype.Text, text string, sent pgtype.Timestamp) message_model.Message {
	return message_model.Message{
		UserID:         userID.String(),
		Username:       username,
		Name:           name.String,
		ProfilePicture: picture.String,
		EventID:        eventID,
		Text:           text,
		DateSent:       sent.Time,
	}
}
