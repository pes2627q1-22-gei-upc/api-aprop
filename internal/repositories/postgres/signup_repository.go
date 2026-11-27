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
	"api-aprop/internal/models/user_model"
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

// Create inserts the sign up unless the user is already signed up to the event.
// TODO(DP-012): without a UNIQUE (UserID, Event) constraint concurrent requests can still duplicate the sign up.
func (r *SignUpRepository) Create(ctx context.Context, userID string, eventID int) (bool, error) {
	uid, err := user_model.ParseID(userID)
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
	uid, err := user_model.ParseID(userID)
	if err != nil {
		return false, err
	}
	rows, err := r.queries.DeleteSignUp(ctx, sqlc.DeleteSignUpParams{UserID: uid, EventID: int32(eventID)})
	if err != nil {
		return false, fmt.Errorf("unenroll from event %d: %w", eventID, err)
	}
	return rows > 0, nil
}

func (r *SignUpRepository) SetAttendance(ctx context.Context, userID string, eventID int, attendance *bool) (bool, error) {
	uid, err := user_model.ParseID(userID)
	if err != nil {
		return false, err
	}
	params := sqlc.SetSignUpAttendanceParams{UserID: uid, EventID: int32(eventID)}
	if attendance != nil {
		params.Attendance = pgtype.Bool{Bool: *attendance, Valid: true}
	}
	rows, err := r.queries.SetSignUpAttendance(ctx, params)
	if err != nil {
		return false, fmt.Errorf("set attendance for event %d: %w", eventID, err)
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
	uid, err := user_model.ParseID(userID)
	if err != nil {
		return false, err
	}
	signedUp, err := r.queries.IsUserSignedUp(ctx, sqlc.IsUserSignedUpParams{UserID: uid, EventID: int32(eventID)})
	if err != nil {
		return false, fmt.Errorf("check sign up for user %s and event %d: %w", userID, eventID, err)
	}
	return signedUp, nil
}

func (r *SignUpRepository) GetAttendance(ctx context.Context, userID string, eventID int) (*bool, error) {
	uid, err := user_model.ParseID(userID)
	if err != nil {
		return nil, err
	}
	attendance, err := r.queries.GetAttendance(ctx, sqlc.GetAttendanceParams{Userid: uid, Event: int32(eventID)})
	if err != nil {
		return nil, fmt.Errorf("check attendance for user %s and event %d: %w", userID, eventID, err)
	}
	var ret *bool
	if attendance.Valid {
		b := attendance.Bool
		ret = &b
	}
	return ret, nil
}

func (r *SignUpRepository) ListUserEvents(ctx context.Context, userID string, dateRange signup_model.DateRange) ([]event_model.Event, error) {
	uid, err := user_model.ParseID(userID)
	if err != nil {
		return nil, err
	}
	params := sqlc.ListUserSignedUpEventsParams{UserID: uid}
	if dateRange.From != nil {
		params.FromDate = pgtype.Timestamptz{
			Time:  *dateRange.From,
			Valid: true,
		}
	}
	if dateRange.To != nil {
		params.ToDate = pgtype.Timestamptz{
			Time:  *dateRange.To,
			Valid: true,
		}
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
		Modality:     row.Modality,
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
