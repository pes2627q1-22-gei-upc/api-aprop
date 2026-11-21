package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"api-aprop/internal/models/user_model"
	"api-aprop/internal/repositories/postgres/sqlc"
	"api-aprop/internal/repositories/rating_repository"
)

// RatingRepository stores event ratings in Postgres using the sqlc generated queries.
type RatingRepository struct {
	queries *sqlc.Queries
}

var _ rating_repository.Repository = (*RatingRepository)(nil)

func NewRatingRepository(pool *pgxpool.Pool) *RatingRepository {
	return &RatingRepository{queries: sqlc.New(pool)}
}

func (r *RatingRepository) Create(ctx context.Context, userID string, eventID, score int) (bool, error) {
	uid, err := user_model.ParseID(userID)
	if err != nil {
		return false, err
	}
	rows, err := r.queries.CreateEventRating(ctx, sqlc.CreateEventRatingParams{UserID: uid, EventID: int32(eventID), Rating: int32(score)})
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == foreignKeyViolation {
		return false, fmt.Errorf("rate event %d: %w", eventID, rating_repository.ErrEventNotFound)
	}
	if err != nil {
		return false, fmt.Errorf("rate event %d: %w", eventID, err)
	}
	return rows > 0, nil
}

func (r *RatingRepository) CreateUserRating(ctx context.Context, userID pgtype.UUID, targetID pgtype.UUID, eventID int, score int) (bool, error) {
	rows, err := r.queries.CreateUserRating(ctx, sqlc.CreateUserRatingParams{Rater: userID, Rated: targetID, Event: int32(eventID), Rating: int32(score)})
	if err != nil {
		return false, err
	}
	return rows > 0, nil
}

func (r *RatingRepository) UpdateUserRating(ctx context.Context, userID pgtype.UUID, targetID pgtype.UUID, eventID int, score int) (bool, error) {
	rows, err := r.queries.UpdateUserRating(ctx, sqlc.UpdateUserRatingParams{Rater: userID, Rated: targetID, Event: int32(eventID), Rating: int32(score)})
	if err != nil {
		return false, err
	}
	return rows > 0, nil
}

func (r *RatingRepository) DeleteUserRating(ctx context.Context, userID pgtype.UUID, targetID pgtype.UUID, eventID int) (bool, error) {
	rows, err := r.queries.DeleteUserRating(ctx, sqlc.DeleteUserRatingParams{Rater: userID, Rated: targetID, Event: int32(eventID)})
	if err != nil {
		return false, err
	}
	return rows > 0, nil
}

func (r *RatingRepository) Update(ctx context.Context, userID string, eventID, score int) (bool, error) {
	uid, err := user_model.ParseID(userID)
	if err != nil {
		return false, err
	}
	rows, err := r.queries.UpdateEventRating(ctx, sqlc.UpdateEventRatingParams{UserID: uid, EventID: int32(eventID), Rating: int32(score)})
	if err != nil {
		return false, fmt.Errorf("update rating of event %d: %w", eventID, err)
	}
	return rows > 0, nil
}

func (r *RatingRepository) Delete(ctx context.Context, userID string, eventID int) (bool, error) {
	uid, err := user_model.ParseID(userID)
	if err != nil {
		return false, err
	}
	rows, err := r.queries.DeleteEventRating(ctx, sqlc.DeleteEventRatingParams{UserID: uid, EventID: int32(eventID)})
	if err != nil {
		return false, fmt.Errorf("delete rating of event %d: %w", eventID, err)
	}
	return rows > 0, nil
}

func (r *RatingRepository) HasAttended(ctx context.Context, userID string, eventID int) (bool, error) {
	uid, err := user_model.ParseID(userID)
	if err != nil {
		return false, err
	}
	ok, err := r.queries.HasUserAttendedEvent(ctx, sqlc.HasUserAttendedEventParams{UserID: uid, EventID: int32(eventID)})
	if err != nil {
		return false, fmt.Errorf("check attendance of user %s at event %d: %w", userID, eventID, err)
	}
	return ok, nil
}
