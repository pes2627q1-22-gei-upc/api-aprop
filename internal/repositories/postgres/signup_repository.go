package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"api-aprop/internal/models/event_model"
	"api-aprop/internal/models/signup_model"
	"api-aprop/internal/repositories/postgres/sqlc"
	"api-aprop/internal/repositories/signup_repository"
)

const foreignKeyViolation = "23503"

// SignUpRepository stores sign ups in Postgres using the sqlc generated queries.
type SignUpRepository struct {
	queries *sqlc.Queries
}

var _ signup_repository.Repository = (*SignUpRepository)(nil)

func NewSignUpRepository(pool *pgxpool.Pool) *SignUpRepository {
	return &SignUpRepository{queries: sqlc.New(pool)}
}

func (r *SignUpRepository) EventExists(ctx context.Context, eventID int) (bool, error) {
	exists, err := r.queries.EventExists(ctx, int32(eventID))
	if err != nil {
		return false, fmt.Errorf("check event %d: %w", eventID, err)
	}
	return exists, nil
}

// Create inserts the sign up unless the user is already signed up to the event.
// TODO(DP-012): without a UNIQUE (UserID, Event) constraint concurrent requests can still duplicate the sign up.
func (r *SignUpRepository) Create(ctx context.Context, userID string, eventID int) (bool, error) {
	uid, err := toUUID(userID)
	if err != nil {
		return false, err
	}
	rows, err := r.queries.CreateSignUp(ctx, sqlc.CreateSignUpParams{UserID: uid, EventID: int32(eventID)})
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == foreignKeyViolation {
		return false, fmt.Errorf("sign up to event %d: %w", eventID, signup_repository.ErrEventNotFound)
	}
	if err != nil {
		return false, fmt.Errorf("sign up to event %d: %w", eventID, err)
	}
	return rows > 0, nil
}

func (r *SignUpRepository) Delete(ctx context.Context, userID string, eventID int) (bool, error) {
	uid, err := toUUID(userID)
	if err != nil {
		return false, err
	}
	rows, err := r.queries.DeleteSignUp(ctx, sqlc.DeleteSignUpParams{UserID: uid, EventID: int32(eventID)})
	if err != nil {
		return false, fmt.Errorf("unenroll from event %d: %w", eventID, err)
	}
	return rows > 0, nil
}

func (r *SignUpRepository) CountSignUps(ctx context.Context, eventID int) (int, error) {
	count, err := r.queries.CountEventSignUps(ctx, int32(eventID))
	if err != nil {
		return 0, fmt.Errorf("count sign ups for event %d: %w", eventID, err)
	}
	return int(count), nil
}

func (r *SignUpRepository) IsSignedUp(ctx context.Context, userID string, eventID int) (bool, error) {
	uid, err := toUUID(userID)
	if err != nil {
		return false, err
	}
	signedUp, err := r.queries.IsUserSignedUp(ctx, sqlc.IsUserSignedUpParams{UserID: uid, EventID: int32(eventID)})
	if err != nil {
		return false, fmt.Errorf("check sign up for user %s and event %d: %w", userID, eventID, err)
	}
	return signedUp, nil
}

func (r *SignUpRepository) ListUserEvents(ctx context.Context, userID string, dateRange signup_model.DateRange) ([]event_model.Event, error) {
	uid, err := toUUID(userID)
	if err != nil {
		return nil, err
	}
	params := sqlc.ListUserSignedUpEventsParams{UserID: uid}
	if dateRange.From != nil {
		params.FromDate = toDate(*dateRange.From)
	}
	if dateRange.To != nil {
		params.ToDate = toDate(*dateRange.To)
	}
	rows, err := r.queries.ListUserSignedUpEvents(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("list events of user %s: %w", userID, err)
	}
	events := make([]event_model.Event, 0, len(rows))
	for _, row := range rows {
		events = append(events, signedUpEventToModel(row))
	}
	return events, nil
}

// signedUpEventToModel reuses toModel so both queries map events the same way.
func signedUpEventToModel(row sqlc.ListUserSignedUpEventsRow) event_model.Event {
	return toModel(sqlc.Event{
		ID:           row.ID,
		Title:        row.Title,
		Subtitle:     row.Subtitle,
		Description:  row.Description,
		Schedule:     row.Schedule,
		Url:          row.Url,
		Tickets:      row.Tickets,
		Ticketsurl:   row.Ticketsurl,
		Free:         row.Free,
		Permanent:    row.Permanent,
		Modality:     toText(row.Modality),
		Relatedurls:  row.Relatedurls,
		Urlnames:     row.Urlnames,
		Images:       row.Images,
		Videos:       row.Videos,
		Email:        row.Email,
		Place:        row.Place,
		Latitude:     row.Latitude,
		Longitude:    row.Longitude,
		PostalCode:   row.PostalCode,
		Municipality: row.Municipality,
		Region:       row.Region,
		Address:      row.Address,
		Phone:        row.Phone,
		Activityurl:  row.Activityurl,
		Startdate:    row.Startdate,
		Enddate:      row.Enddate,
	}, 0)
}

func toUUID(s string) (pgtype.UUID, error) {
	var uid pgtype.UUID
	if err := uid.Scan(s); err != nil {
		return pgtype.UUID{}, fmt.Errorf("invalid user id %q: %w", s, err)
	}
	return uid, nil
}
