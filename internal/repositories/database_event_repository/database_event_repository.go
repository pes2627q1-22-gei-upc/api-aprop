package database_event_repository

import (
	"errors"
	. "api-aprop/internal/models/event_model"
	"context"
	"github.com/jackc/pgx/v5"
)

var (
	ErrEventNotFound = errors.New("event not found")
	ErrInvalidEvent  = errors.New("invalid event")
)

type databaseStore struct {
	connection *pgx.Conn
}

func NewDatabaseEventRepository(c *pgx.Conn) *databaseStore {
	return &databaseStore{connection: c}
}

func (s *databaseStore) Create(e Event) (Event, error){
	var id int

	err := s.connection.QueryRow(context.Background(), `INSERT INTO events (title, subtitle, description, schedule, url, tickets, ticketsURL, free, permanent, online, relatedURLs, urlnames, images, videos, email, placeID, phone, activityURL, startdate, enddate)
												VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20) RETURNING id`,
												e.Title, e.Subtitle, e.Description, e.Schedule, e.URL, e.Tickets, e.TicketsURL, e.Free, e.Permanent, e.Online, e.RelatedURLs, e.URLnames, e.Images, e.Videos, e.Email, e.PlaceID, e.Phone, e.ActivityURL, e.StartDate, e.EndDate).Scan(&id)
    if err != nil {
        return Event{}, err
    }
	return e, nil
}

func (s *databaseStore) Update(id int, e Event) (Event, error){
	var eventid int

	err := s.connection.QueryRow(context.Background(), `UPDATE events SET title = $1, subtitle = $2, description = $3, schedule = $4, url = $5, tickets = $6, ticketsURL = $7, free = $8, permanent = $9, online = $10,
												relatedURLs = $11, urlnames = $12 images = $13, videos = $14, email = $15, placeID = $16, phone = $17, activityURL = $18, startdate = $19, enddate = $20
												WHERE id = $21 RETURNING id`,
												e.Title, e.Subtitle, e.Description, e.Schedule, e.URL, e.Tickets, e.TicketsURL, e.Free, e.Permanent, e.Online, e.RelatedURLs, e.URLnames, e.Images, e.Videos, e.Email, e.PlaceID, e.Phone, e.ActivityURL, e.StartDate, e.EndDate, id).Scan(&eventid)
    if err != nil {
        return Event{}, err
    }
	return e, nil
}

func (s *databaseStore) CreateOrReplace(e Event) (Event, error){
	var exists bool
	err := s.connection.QueryRow(context.Background(), `SELECT (EXISTS (SELECT * FROM events WHERE id = $1))`, e.ID).Scan(&exists)

	if err != nil {
		return Event{}, err
    }
	if exists {
		_, err := s.Update(e.ID, e)
		if err!=nil{
			return Event{}, err
		}
		return e, nil
	} else {
		newev, err := s.Create(e)
		if err!=nil{
			return Event{}, err
		}
		return newev, nil
	}
}

func (s *databaseStore) Get(id int) (Event, error){
	var e Event
	err := s.connection.QueryRow(context.Background(), `SELECT * FROM events WHERE id = $1`, id).Scan(&e.ID, &e.Title, &e.Subtitle, &e.Description, &e.Schedule, &e.URL, &e.Tickets, &e.TicketsURL, &e.Free, &e.Permanent, &e.Online, &e.RelatedURLs, &e.URLnames, &e.Images, &e.Videos, &e.Email, &e.PlaceID, &e.Phone, &e.ActivityURL, &e.StartDate, &e.EndDate)
    if err != nil {
        return Event{}, err
    }
    
	return e, nil
}

func (s *databaseStore) List() ([]Event, error){
	rows, err := s.connection.Query(context.Background(), `SELECT * FROM events LIMIT 10`)	// Will have to update in future to add paging and filters
    if err != nil {
        return []Event{}, err
    }
	defer rows.Close()

	var events []Event
	for rows.Next(){
		var e Event
		err = rows.Scan(&e.ID, &e.Title, &e.Subtitle, &e.Description, &e.Schedule, &e.URL, &e.Tickets, &e.TicketsURL, &e.Free, &e.Permanent, &e.Online, &e.RelatedURLs, &e.URLnames, &e.Images, &e.Videos, &e.Email, &e.PlaceID, &e.Phone, &e.ActivityURL, &e.StartDate, &e.EndDate)
		if err != nil {
			return []Event{}, err
		}
		events=append(events, e)
	}
	return events, nil
}