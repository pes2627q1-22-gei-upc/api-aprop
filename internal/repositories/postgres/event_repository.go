package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"api-aprop/internal/models/event_model"
	"api-aprop/internal/repositories/event_repository"
	"api-aprop/internal/repositories/postgres/sqlc"
)

// EventRepository stores events in Postgres using the sqlc generated queries.
type EventRepository struct {
	queries *sqlc.Queries
}

var _ event_repository.Repository = (*EventRepository)(nil)

func NewEventRepository(pool *pgxpool.Pool) *EventRepository {
	return &EventRepository{queries: sqlc.New(pool)}
}

func (r *EventRepository) Create(e event_model.Event) (event_model.Event, error) {
	row, err := r.queries.CreateEvent(context.Background(), sqlc.CreateEventParams{
		Title:       e.Title,
		Subtitle:    toText(e.Subtitle),
		Description: e.Description,
		Schedule:    toText(e.Schedule),
		Url:         toText(e.URL),
		Tickets:     toText(e.Tickets),
		Ticketsurl:  toText(e.TicketsURL),
		Free:        e.Free,
		Permanent:   e.Permanent,
		Online:      e.Online,
		Relatedurls: toText(e.RelatedURLs),
		Urlnames:    toText(e.URLnames),
		Images:      toText(e.Images),
		Videos:      toText(e.Videos),
		Email:       toText(e.Email),
		Placeid:     e.PlaceID,
		Phone:       toText(e.Phone),
		Activityurl: toText(e.ActivityURL),
		Startdate:   toDate(e.StartDate),
		Enddate:     toDate(e.EndDate),
	})
	if err != nil {
		return event_model.Event{}, fmt.Errorf("create event: %w", err)
	}
	return toModel(row, e.Code), nil
}

func (r *EventRepository) Update(id int, e event_model.Event) (event_model.Event, error) {
	row, err := r.queries.UpdateEvent(context.Background(), sqlc.UpdateEventParams{
		ID:          int32(id),
		Title:       e.Title,
		Subtitle:    toText(e.Subtitle),
		Description: e.Description,
		Schedule:    toText(e.Schedule),
		Url:         toText(e.URL),
		Tickets:     toText(e.Tickets),
		Ticketsurl:  toText(e.TicketsURL),
		Free:        e.Free,
		Permanent:   e.Permanent,
		Online:      e.Online,
		Relatedurls: toText(e.RelatedURLs),
		Urlnames:    toText(e.URLnames),
		Images:      toText(e.Images),
		Videos:      toText(e.Videos),
		Email:       toText(e.Email),
		Placeid:     e.PlaceID,
		Phone:       toText(e.Phone),
		Activityurl: toText(e.ActivityURL),
		Startdate:   toDate(e.StartDate),
		Enddate:     toDate(e.EndDate),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return event_model.Event{}, fmt.Errorf("update event %d: %w", id, event_repository.ErrEventNotFound)
	}
	if err != nil {
		return event_model.Event{}, fmt.Errorf("update event %d: %w", id, err)
	}
	return toModel(row, e.Code), nil
}

// CreateOrReplace updates the event if its ID already exists, otherwise creates it.
func (r *EventRepository) CreateOrReplace(e event_model.Event) (event_model.Event, error) {
	exists, err := r.queries.EventExists(context.Background(), int32(e.ID))
	if err != nil {
		return event_model.Event{}, fmt.Errorf("check event %d: %w", e.ID, err)
	}
	if exists {
		return r.Update(e.ID, e)
	}
	return r.Create(e)
}

func (r *EventRepository) Get(id int) (event_model.Event, error) {
	row, err := r.queries.GetEvent(context.Background(), int32(id))
	if errors.Is(err, pgx.ErrNoRows) {
		return event_model.Event{}, fmt.Errorf("get event %d: %w", id, event_repository.ErrEventNotFound)
	}
	if err != nil {
		return event_model.Event{}, fmt.Errorf("get event %d: %w", id, err)
	}
	return toModel(row, 0), nil
}

func (r *EventRepository) List() ([]event_model.Event, error) {
	rows, err := r.queries.ListEvents(context.Background())
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	events := make([]event_model.Event, 0, len(rows))
	for _, row := range rows {
		events = append(events, toModel(row, 0))
	}
	return events, nil
}

// toModel converts a database row to the domain model.
// The code is not stored in the events table, so it is passed separately.
func toModel(row sqlc.Event, code int64) event_model.Event {
	return event_model.Event{
		ID:          int(row.ID),
		Code:        code,
		Title:       row.Title,
		Subtitle:    row.Subtitle.String,
		Schedule:    row.Schedule.String,
		Tickets:     row.Tickets.String,
		URL:         row.Url.String,
		TicketsURL:  row.Ticketsurl.String,
		Description: row.Description,
		Free:        row.Free,
		Permanent:   row.Permanent,
		RelatedURLs: row.Relatedurls.String,
		URLnames:    row.Urlnames.String,
		Online:      row.Online,
		Images:      row.Images.String,
		Videos:      row.Videos.String,
		Email:       row.Email.String,
		PlaceID:     row.Placeid,
		Phone:       row.Phone.String,
		ActivityURL: row.Activityurl.String,
		StartDate:   row.Startdate.Time,
		EndDate:     row.Enddate.Time,
	}
}

// toText stores empty strings as NULL.
func toText(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: s != ""}
}

// toDate stores the zero time as NULL.
func toDate(t time.Time) pgtype.Date {
	return pgtype.Date{Time: t, Valid: !t.IsZero()}
}
