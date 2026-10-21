package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"
	"strconv"

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

func (r *EventRepository) Create(e event_model.Event, official bool) (event_model.Event, error) {
	row, err := r.queries.CreateEvent(context.Background(), sqlc.CreateEventParams{
		Title:       	e.Title,
		Subtitle:    	toText(e.Subtitle),
		Description: 	toText(e.Description),
		Schedule:    	toText(e.Schedule),
		Url:         	toText(e.URL),
		Tickets:     	toText(e.Tickets),
		Ticketsurl:  	toText(e.TicketsURL),
		Free:        	toPGXBool(e.Free),
		Permanent:   	toPGXBool(e.Permanent),
		Modality:    	toText(e.Modality),
		Relatedurls: 	toText(e.RelatedURLs),
		Urlnames:    	toText(e.URLnames),
		Images:      	toText(e.Images),
		Videos:      	toText(e.Videos),
		Email:       	toText(e.Email),
		Place:     	 	toText(e.Place),
		Latitude:     	toPGXFloat(e.Latitude),
		Longitude:     	toPGXFloat(e.Longitude),
		PostalCode:     toText(e.PostalCode),
		Municipality:   toText(e.Municipality),
		Region:     	toText(e.Region),
		Address:     	toText(e.Address),
		Phone:       	toText(e.Phone),
		Activityurl: 	toText(e.ActivityURL),
		Startdate:   	toDate(e.StartDate),
		Enddate:     	toDate(e.EndDate),
	})
	if err != nil {
		return event_model.Event{}, fmt.Errorf("create event: %w", err)
	}
	if official{
		_, err := r.queries.CreateOfficialEvent(context.Background(), sqlc.CreateOfficialEventParams{ID: int32(row.ID), Code: pgtype.Int8{Int64: e.Code, Valid:true}})
		if err!=nil{
			return event_model.Event{}, fmt.Errorf("create event: %w", err)
		}
	}

	r.SetCategories(int(row.ID), e.Categories, official)

	return toModel(row, e.Code), nil
}

func (r *EventRepository) Update(id int, e event_model.Event, official bool) (event_model.Event, error) {
	row, err := r.queries.UpdateEvent(context.Background(), sqlc.UpdateEventParams{
		ID:          	int32(id),
		Title:       	e.Title,
		Subtitle:    	toText(e.Subtitle),
		Description: 	toText(e.Description),
		Schedule:    	toText(e.Schedule),
		Url:         	toText(e.URL),
		Tickets:     	toText(e.Tickets),
		Ticketsurl:  	toText(e.TicketsURL),
		Free:        	toPGXBool(e.Free),
		Permanent:   	toPGXBool(e.Permanent),
		Modality:    	toText(e.Modality),
		Relatedurls: 	toText(e.RelatedURLs),
		Urlnames:    	toText(e.URLnames),
		Images:      	toText(e.Images),
		Videos:      	toText(e.Videos),
		Email:       	toText(e.Email),
		Place:     	 	toText(e.Place),
		Latitude:     	toPGXFloat(e.Latitude),
		Longitude:     	toPGXFloat(e.Longitude),
		PostalCode:     toText(e.PostalCode),
		Municipality:   toText(e.Municipality),
		Region:     	toText(e.Region),
		Address:     	toText(e.Address),
		Phone:       	toText(e.Phone),
		Activityurl: 	toText(e.ActivityURL),
		Startdate:   	toDate(e.StartDate),
		Enddate:     	toDate(e.EndDate),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return event_model.Event{}, fmt.Errorf("update event %d: %w", id, event_repository.ErrEventNotFound)
	}
	if err != nil {
		return event_model.Event{}, fmt.Errorf("update event %d: %w", id, err)
	}

	r.SetCategories(id, e.Categories, official)

	return toModel(row, e.Code), nil
}

// CreateOrReplace updates the event if its ID already exists, otherwise creates it.
func (r *EventRepository) CreateOrReplace(e event_model.Event) (event_model.Event, error) {
	id, err := r.queries.EqualEventExists(context.Background(), sqlc.EqualEventExistsParams{
		Title:       	e.Title,
		Subtitle:    	toText(e.Subtitle),
		Description: 	toText(e.Description),
		Schedule:    	toText(e.Schedule),
		Url:         	toText(e.URL),
		Tickets:     	toText(e.Tickets),
		Ticketsurl:  	toText(e.TicketsURL),
		Free:        	toPGXBool(e.Free),
		Permanent:   	toPGXBool(e.Permanent),
		Modality:    	toText(e.Modality),
		Relatedurls: 	toText(e.RelatedURLs),
		Urlnames:    	toText(e.URLnames),
		Images:      	toText(e.Images),
		Videos:      	toText(e.Videos),
		Email:       	toText(e.Email),
		Place:     	 	toText(e.Place),
		Latitude:     	toPGXFloat(e.Latitude),
		Longitude:     	toPGXFloat(e.Longitude),
		PostalCode:     toText(e.PostalCode),
		Municipality:   toText(e.Municipality),
		Region:     	toText(e.Region),
		Address:     	toText(e.Address),
		Phone:       	toText(e.Phone),
		Activityurl: 	toText(e.ActivityURL),
		Startdate:   	toDate(e.StartDate),
		Enddate:     	toDate(e.EndDate),
	})
	if errors.Is(err, pgx.ErrNoRows){
		return r.Create(e, true)
	}
	if err != nil {
		return event_model.Event{}, fmt.Errorf("check event %d: %w", e.ID, err)
	}
	return r.Update(int(id), e, true)
}

func (r *EventRepository) SetCategories(id int, cats []string, official bool) (error){
	err := r.queries.RemoveExistingEventCategories(context.Background(), int32(id))
	if err != nil {
		return fmt.Errorf("remove categories from event %d: %w", id, err)
	}

	for _, v := range cats {
		exists, err := r.queries.CategoryExists(context.Background(), v)
		if err!=nil {
			return fmt.Errorf("add categories to event %d: %w", id, err)
		}
		if !exists {
			if official {
				catid, err := r.queries.NewCategory(context.Background(), v)
				if err != nil {
					return fmt.Errorf("add categories to event %d: %w", id, err)
				}
				err = r.queries.AddEventCategory(context.Background(), sqlc.AddEventCategoryParams{Event:int32(id), Category:int32(catid)})
				if err != nil {
					return fmt.Errorf("add categories to event %d: %w", id, err)
				}
			} else{
				return fmt.Errorf("add categories to event %d: category %s does not exist", id, v)
			}
		} else {
			catid, err := r.queries.GetCategoryID(context.Background(), v)
			if err != nil {
				return fmt.Errorf("add categories to event %d: %w", id, err)
			}
			err = r.queries.AddEventCategory(context.Background(), sqlc.AddEventCategoryParams{Event:int32(id), Category:int32(catid)})
			if err != nil {
				return fmt.Errorf("add categories to event %d: %w", id, err)
			}
		}
	}

	return nil
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

func (r *EventRepository) List(params map[string]any) ([]event_model.Event, error) {
	pageSize, err := strconv.Atoi(params["pageSize"].(string))
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	pageNum, err := strconv.Atoi(params["pageNum"].(string))
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	rows, err := r.queries.ListEvents(context.Background(), sqlc.ListEventsParams{Limit: int32(pageSize), Offset: int32(pageNum * pageSize)})
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
		ID:          	int(row.ID),
		Code:        	code,
		Title:       	row.Title,
		Subtitle:    	row.Subtitle.String,
		Schedule:    	row.Schedule.String,
		Tickets:     	row.Tickets.String,
		URL:         	row.Url.String,
		TicketsURL:  	row.Ticketsurl.String,
		Description: 	row.Description.String,
		Free:        	row.Free.Bool,
		Permanent:   	row.Permanent.Bool,
		RelatedURLs: 	row.Relatedurls.String,
		URLnames:    	row.Urlnames.String,
		Modality:    	row.Modality.String,
		Images:      	row.Images.String,
		Videos:      	row.Videos.String,
		Email:       	row.Email.String,
		Place:	     	row.Place.String,
		Latitude:		row.Latitude.Float32,
		Longitude:		row.Longitude.Float32,
		PostalCode:		row.PostalCode.String,
		Municipality:	row.Municipality.String,
		Region:			row.Region.String,
		Address:		row.Address.String,
		Phone:       	row.Phone.String,
		ActivityURL: 	row.Activityurl.String,
		StartDate:   	row.Startdate.Time,
		EndDate:     	row.Enddate.Time,
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

func toPGXFloat(f float32) (pgtype.Float4) {
	return pgtype.Float4{Float32: f, Valid: true}
}

func toPGXBool(b bool) pgtype.Bool{
	return pgtype.Bool{Bool: b, Valid: true}
}