package postgres

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
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
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

var _ event_repository.Repository = (*EventRepository)(nil)

func NewEventRepository(pool *pgxpool.Pool) *EventRepository {
	return &EventRepository{pool: pool, queries: sqlc.New(pool)}
}

// Create stores the event, its official or unofficial row and its categories in a single transaction.
func (r *EventRepository) Create(e event_model.Event, official bool) (event_model.Event, error) {
	ctx := context.Background()
	var row sqlc.Event
	err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		q := r.queries.WithTx(tx)
		var err error
		row, err = createEvent(ctx, q, e, official)
		if err != nil {
			return err
		}
		return setCategories(ctx, q, int(row.ID), e.Categories)
	})
	if err != nil {
		return event_model.Event{}, err
	}

	res := toModel(row, e.Code)
	res.IsOfficial = official
	res.Creator = e.Creator
	res.DateCreated = e.DateCreated
	res.Categories = e.Categories
	return res, nil
}

func createEvent(ctx context.Context, q *sqlc.Queries, e event_model.Event, official bool) (sqlc.Event, error) {
	row, err := q.CreateEvent(ctx, sqlc.CreateEventParams{
		Title:        e.Title,
		Subtitle:     toText(e.Subtitle),
		Description:  toText(e.Description),
		Schedule:     toText(e.Schedule),
		Url:          toText(e.URL),
		Tickets:      toText(e.Tickets),
		Ticketsurl:   toText(e.TicketsURL),
		Free:         e.Free,
		Permanent:    e.Permanent,
		Modality:     toText(e.Modality),
		Relatedurls:  toText(e.RelatedURLs),
		Urlnames:     toText(e.URLnames),
		Images:       toText(e.Images),
		Videos:       toText(e.Videos),
		Email:        toText(e.Email),
		Place:        toText(e.Place),
		Latitude:     toPGXFloat(e.Latitude),
		Longitude:    toPGXFloat(e.Longitude),
		PostalCode:   toText(e.PostalCode),
		Municipality: toText(e.Municipality),
		Region:       toText(e.Region),
		Address:      toText(e.Address),
		Phone:        toText(e.Phone),
		Activityurl:  toText(e.ActivityURL),
		Startdate:    toDate(e.StartDate),
		Enddate:      toDate(e.EndDate),
	})
	if err != nil {
		return sqlc.Event{}, fmt.Errorf("create event: %w", err)
	}
	if official {
		_, err := q.CreateOfficialEvent(ctx, sqlc.CreateOfficialEventParams{ID: row.ID, Code: pgtype.Int8{Int64: e.Code, Valid: true}})
		if err != nil {
			return sqlc.Event{}, fmt.Errorf("create official event: %w", err)
		}
		return row, nil
	}

	dateCreated := e.DateCreated
	if dateCreated.IsZero() {
		dateCreated = time.Now().UTC()
	}
	_, err = q.CreateUnofficialEvent(ctx, sqlc.CreateUnofficialEventParams{
		ID:          row.ID,
		Creator:     e.Creator,
		Datecreated: pgtype.Timestamp{Time: dateCreated, Valid: true},
	})
	if err != nil {
		return sqlc.Event{}, fmt.Errorf("create unofficial event: %w", err)
	}
	return row, nil
}

// Update replaces the event fields and its categories in a single transaction.
// The official/unofficial row is not changed: its creator and creation date stay the same.
func (r *EventRepository) Update(id int, e event_model.Event, official bool) (event_model.Event, error) {
	ctx := context.Background()
	var row sqlc.Event
	err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		q := r.queries.WithTx(tx)
		var err error
		row, err = updateEvent(ctx, q, id, e)
		if err != nil {
			return err
		}
		return setCategories(ctx, q, id, e.Categories)
	})
	if err != nil {
		return event_model.Event{}, err
	}

	res := toModel(row, e.Code)
	res.IsOfficial = official
	res.Creator = e.Creator
	res.DateCreated = e.DateCreated
	res.Categories = e.Categories
	return res, nil
}

func updateEvent(ctx context.Context, q *sqlc.Queries, id int, e event_model.Event) (sqlc.Event, error) {
	row, err := q.UpdateEvent(ctx, sqlc.UpdateEventParams{
		ID:           int32(id),
		Title:        e.Title,
		Subtitle:     toText(e.Subtitle),
		Description:  toText(e.Description),
		Schedule:     toText(e.Schedule),
		Url:          toText(e.URL),
		Tickets:      toText(e.Tickets),
		Ticketsurl:   toText(e.TicketsURL),
		Free:         e.Free,
		Permanent:    e.Permanent,
		Modality:     toText(e.Modality),
		Relatedurls:  toText(e.RelatedURLs),
		Urlnames:     toText(e.URLnames),
		Images:       toText(e.Images),
		Videos:       toText(e.Videos),
		Email:        toText(e.Email),
		Place:        toText(e.Place),
		Latitude:     toPGXFloat(e.Latitude),
		Longitude:    toPGXFloat(e.Longitude),
		PostalCode:   toText(e.PostalCode),
		Municipality: toText(e.Municipality),
		Region:       toText(e.Region),
		Address:      toText(e.Address),
		Phone:        toText(e.Phone),
		Activityurl:  toText(e.ActivityURL),
		Startdate:    toDate(e.StartDate),
		Enddate:      toDate(e.EndDate),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return sqlc.Event{}, fmt.Errorf("update event %d: %w", id, event_repository.ErrEventNotFound)
	}
	if err != nil {
		return sqlc.Event{}, fmt.Errorf("update event %d: %w", id, err)
	}
	return row, nil
}

// CreateOrReplace updates the event if its ID already exists, otherwise creates it.
func (r *EventRepository) CreateOrReplace(e event_model.Event) (event_model.Event, error) {
	id, err := r.queries.EqualEventExists(context.Background(), sqlc.EqualEventExistsParams{
		Title:        e.Title,
		Subtitle:     toText(e.Subtitle),
		Description:  toText(e.Description),
		Schedule:     toText(e.Schedule),
		Url:          toText(e.URL),
		Tickets:      toText(e.Tickets),
		Ticketsurl:   toText(e.TicketsURL),
		Free:         e.Free,
		Permanent:    e.Permanent,
		Modality:     toText(e.Modality),
		Relatedurls:  toText(e.RelatedURLs),
		Urlnames:     toText(e.URLnames),
		Images:       toText(e.Images),
		Videos:       toText(e.Videos),
		Email:        toText(e.Email),
		Place:        toText(e.Place),
		Latitude:     toPGXFloat(e.Latitude),
		Longitude:    toPGXFloat(e.Longitude),
		PostalCode:   toText(e.PostalCode),
		Municipality: toText(e.Municipality),
		Region:       toText(e.Region),
		Address:      toText(e.Address),
		Phone:        toText(e.Phone),
		Activityurl:  toText(e.ActivityURL),
		Startdate:    toDate(e.StartDate),
		Enddate:      toDate(e.EndDate),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return r.Create(e, true)
	}
	if err != nil {
		return event_model.Event{}, fmt.Errorf("check event %d: %w", e.ID, err)
	}
	return r.Update(int(id), e, true)
}

// Delete removes the event together with its sign-ups, ratings, messages, notifications, categories,
// preference test choices, chatbot recommendations and its official/unofficial row, in a single transaction.
func (r *EventRepository) Delete(ctx context.Context, id int) error {
	eventID := int32(id)
	return pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		q := r.queries.WithTx(tx)
		dependents := []struct {
			name   string
			delete func(context.Context, int32) error
		}{
			{"sign-ups", q.DeleteEventSignUps},
			{"event ratings", q.DeleteEventRatings},
			{"user ratings", q.DeleteEventUserRatings},
			{"messages", q.DeleteEventMessages},
			{"notifications", q.DeleteEventNotifications},
			{"preference test choices", q.DeleteEventPreferenceTestChoices},
			{"chatbot recommendations", q.DeleteEventChatbotRecommendations},
			{"categories", q.RemoveExistingEventCategories},
			{"unofficial event", q.DeleteUnofficialEvent},
		}
		for _, d := range dependents {
			if err := d.delete(ctx, eventID); err != nil {
				return fmt.Errorf("delete %s of event %d: %w", d.name, id, err)
			}
		}

		deleted, err := q.DeleteEvent(ctx, eventID)
		if err != nil {
			return fmt.Errorf("delete event %d: %w", id, err)
		}
		if deleted == 0 {
			return fmt.Errorf("delete event %d: %w", id, event_repository.ErrEventNotFound)
		}
		return nil
	})
}

func (r *EventRepository) UpdateImage(ctx context.Context, id int, imageURL string) error {
	return r.queries.UpdateEventImage(ctx, sqlc.UpdateEventImageParams{
		ID:     int32(id),
		Images: toText(imageURL),
	})
}

func (r *EventRepository) SetCategories(id int, cats []string, _ bool) error {
	return setCategories(context.Background(), r.queries, id, cats)
}

// setCategories replaces the categories of the event, creating the ones that do not exist yet.
func setCategories(ctx context.Context, q *sqlc.Queries, id int, cats []string) error {
	if len(cats) == 0 {
		return nil
	}

	if err := q.RemoveExistingEventCategories(ctx, int32(id)); err != nil {
		return fmt.Errorf("remove categories from event %d: %w", id, err)
	}

	for _, v := range cats {
		exists, err := q.CategoryExists(ctx, v)
		if err != nil {
			return fmt.Errorf("add categories to event %d: %w", id, err)
		}
		var catid int32
		if exists {
			catid, err = q.GetCategoryID(ctx, v)
		} else {
			catid, err = q.NewCategory(ctx, v)
		}
		if err != nil {
			return fmt.Errorf("add categories to event %d: %w", id, err)
		}
		if err := q.AddEventCategory(ctx, sqlc.AddEventCategoryParams{Event: int32(id), Category: catid}); err != nil {
			return fmt.Errorf("add categories to event %d: %w", id, err)
		}
	}

	return nil
}

func (r *EventRepository) Get(id int) (event_model.Event, error) {
	rows, err := r.queries.GetEvent(context.Background(), int32(id))
	if errors.Is(err, pgx.ErrNoRows) {
		return event_model.Event{}, fmt.Errorf("get event %d: %w", id, event_repository.ErrEventNotFound)
	}
	if err != nil {
		return event_model.Event{}, fmt.Errorf("get event %d: %w", id, err)
	}

	cats := []string{}
	for _, r := range rows {
		cats = append(cats, r.Category.String)
	}

	return toModelExtended(rows[0], cats), nil
}

func regionFilterToDBEntry(s string) (string, error) {
	tmp := strings.Split(s, "_")
	if len(tmp) != 2 {
		return "", fmt.Errorf("list events: error converting filterRegion parameter")
	}
	return fmt.Sprintf("agenda:ubicacions/%s/%s", tmp[0], tmp[1]), nil
}

func (r *EventRepository) List(params map[string]any) ([]event_model.Event, error) {
	var err error

	pageSize, ok := params["pageSize"].(int)
	if !ok {
		return nil, fmt.Errorf("list events: error converting pageSize parameter")
	}
	pageNum, ok := params["pageNum"].(int)
	if !ok {
		return nil, fmt.Errorf("list events: error converting pageNum parameter")
	}

	filterCategory, ok := params["filterCategory"].(int)
	if !ok {
		return nil, fmt.Errorf("list events: error converting filterCategory parameter")
	}

	filterRegion, _ := params["filterRegion"].(string)
	if filterRegion != "" {
		filterRegion, err = regionFilterToDBEntry(string(filterRegion))
		if err != nil {
			return nil, err
		}
	}

	var startBefore time.Time
	var startAfter time.Time
	var endBefore time.Time
	var endAfter time.Time
	var filterPermanent pgtype.Bool
	var filterFree pgtype.Bool
	var filterOfficial pgtype.Bool
	var filterModality sqlc.NullModalitytype

	startBeforeTmp, _ := params["startBefore"].(string)
	if startBeforeTmp == "" {
		startBefore = time.Time{}
	} else {
		startBefore, err = time.Parse("2006-01-02T15:04:05.000", startBeforeTmp)
		if err != nil {
			return nil, fmt.Errorf("list events: error converting startBefore parameter: %w", err)
		}
	}
	startAfterTmp, _ := params["startAfter"].(string)
	if startAfterTmp == "" {
		startAfter = time.Time{}
	} else {
		startAfter, err = time.Parse("2006-01-02T15:04:05.000", startAfterTmp)
		if err != nil {
			return nil, fmt.Errorf("list events: error converting startAfter parameter: %w", err)
		}
	}
	endBeforeTmp, _ := params["endBefore"].(string)
	if endBeforeTmp == "" {
		endBefore = time.Time{}
	} else {
		endBefore, err = time.Parse("2006-01-02T15:04:05.000", endBeforeTmp)
		if err != nil {
			return nil, fmt.Errorf("list events: error converting endBefore parameter: %w", err)
		}
	}
	endAfterTmp, _ := params["endAfter"].(string)
	if endAfterTmp == "" {
		endAfter = time.Time{}
	} else {
		endAfter, err = time.Parse("2006-01-02T15:04:05.000", endAfterTmp)
		if err != nil {
			return nil, fmt.Errorf("list events: error converting endAfter parameter: %w", err)
		}
	}

	filterPermanentTmp, _ := params["filterPermanent"].(string)
	if filterPermanentTmp == "" {
		filterPermanent = pgtype.Bool{Valid: false}
	} else {
		filterPermanent.Bool, err = strconv.ParseBool(filterPermanentTmp)
		if err != nil {
			return nil, fmt.Errorf("list events: error converting filterPermanent parameter: %w", err)
		}
		filterPermanent.Valid = true
	}
	filterFreeTmp, _ := params["filterFree"].(string)
	if filterFreeTmp == "" {
		filterFree = pgtype.Bool{Valid: false}
	} else {
		filterFree.Bool, err = strconv.ParseBool(filterFreeTmp)
		if err != nil {
			return nil, fmt.Errorf("list events: error converting filterFree parameter: %w", err)
		}
		filterFree.Valid = true
	}
	filterOfficialTmp, _ := params["filterOfficial"].(string)
	if filterOfficialTmp == "" {
		filterOfficial = pgtype.Bool{Valid: false}
	} else {
		filterOfficial.Bool, err = strconv.ParseBool(filterOfficialTmp)
		if err != nil {
			return nil, fmt.Errorf("list events: error converting filterOfficial parameter: %w", err)
		}
		filterOfficial.Valid = true
	}
	filterModalityTmp, _ := params["filterModality"].(string)
	if filterModalityTmp == "" {
		filterModality = sqlc.NullModalitytype{Valid: false}
	} else if filterModalityTmp == "Online" {
		filterModality = sqlc.NullModalitytype{Modalitytype: sqlc.ModalitytypeOnline, Valid: true}
	} else if filterModalityTmp == "Face-to-Face" {
		filterModality = sqlc.NullModalitytype{Modalitytype: sqlc.ModalitytypeFaceToFace, Valid: true}
	} else if filterModalityTmp == "Online and Face-to-Face" {
		filterModality = sqlc.NullModalitytype{Modalitytype: sqlc.ModalitytypeOnlineandFaceToFace, Valid: true}
	} else {
		return nil, fmt.Errorf("list events: error converting filterModality parameter")
	}

	rows, err := r.queries.ListEvents(context.Background(), sqlc.ListEventsParams{
		Limit:       int32(pageSize),
		Offset:      int32(pageNum * pageSize),
		Category:    int32(filterCategory),
		Region:      filterRegion,
		Startbefore: toDate(startBefore),
		Startafter:  toDate(startAfter),
		Endbefore:   toDate(endBefore),
		Endafter:    toDate(endAfter),
		Permanent:   filterPermanent,
		Official:    filterOfficial,
		Free:        filterFree,
		Modality:    filterModality,
	})
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}

	if len(rows) == 0 {
		return []event_model.Event{}, nil
	}

	events := []event_model.Event{}
	currentevent := rows[0]
	cats := []string{}
	for _, row := range rows {
		if row.ID != currentevent.ID {
			events = append(events, toModelExtended(sqlc.GetEventRow(currentevent), cats))
			cats = []string{}
			currentevent = row
		}
		cats = append(cats, row.Category.String)
	}
	events = append(events, toModelExtended(sqlc.GetEventRow(currentevent), cats))
	return events, nil
}

func toGetEventRow (r sqlc.SearchEventsWithRelevanceRow) sqlc.GetEventRow {
	return sqlc.GetEventRow{
		ID:           r.ID,
		Title:        r.Title,
		Subtitle:     r.Subtitle,
		Description:  r.Description,
		Schedule:     r.Schedule,
		Tickets:      r.Tickets,
		Url:          r.Url,
		Ticketsurl:   r.Url,
		Free:         r.Free,
		Permanent:    r.Permanent,
		Modality:     r.Modality,
		Relatedurls:  r.Relatedurls,
		Urlnames:     r.Urlnames,
		Images:       r.Images,
		Videos:		  r.Videos,
		Email:        r.Email,
		Phone:        r.Phone,
		Activityurl:  r.Activityurl,
		Startdate:    r.Startdate,
		Enddate:      r.Enddate,
		Place:        r.Place,
		Latitude:     r.Latitude,
		Longitude:    r.Longitude,
		PostalCode:   r.PostalCode,
		Municipality: r.Municipality,
		Region:       r.Region,
		Address:      r.Address,
		Code:         r.Code,
		Creator:      r.Creator,
		Datecreated:  r.Datecreated,
		Category:     r.Category,
	}    
}

func (r *EventRepository) Search(ctx context.Context, query string, pageNum, pageSize int) (event_model.SearchResult, error) {
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageNum < 0 {
		pageNum = 0
	}

	total, err := r.queries.CountSearchEvents(ctx, query)
	if err != nil {
		return event_model.SearchResult{}, fmt.Errorf("count search events %q: %w", query, err)
	}

	rows, err := r.queries.SearchEventsWithRelevance(ctx, sqlc.SearchEventsWithRelevanceParams{
		Query:  query,
		Limit:  int32(pageSize),
		Offset: int32(pageNum * pageSize),
	})
	if err != nil {
		return event_model.SearchResult{}, fmt.Errorf("search events %q: %w", query, err)
	}

	if len(rows) == 0 {
		return event_model.SearchResult{
			Query:    query,
			Total:    total,
			PageNum:  pageNum,
			PageSize: pageSize,
			Events:   []event_model.Event{},
		}, nil
	}

	events := []event_model.Event{}
	currentevent := rows[0]
	cats := []string{}
	for _, row := range rows {
		if row.ID != currentevent.ID {
			events = append(events, toModelExtended(toGetEventRow(currentevent), cats))
			cats = []string{}
			currentevent = row
		}
		if row.Category.Valid && row.Category.String != "" {
			cats = append(cats, row.Category.String)
		}
	}
	events = append(events, toModelExtended(toGetEventRow(currentevent), cats))

	return event_model.SearchResult{
		Query:    query,
		Total:    total,
		PageNum:  pageNum,
		PageSize: pageSize,
		Events:   events,
	}, nil
}

func (r *EventRepository) GetCategories() ([]event_model.Category, error) {
	rows, err := r.queries.GetCategories(context.Background())
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}

	cats := []event_model.Category{}
	for _, row := range rows {
		cat := toCategory(row)
		cats = append(cats, cat)
	}
	return cats, nil
}

func (r *EventRepository) GetCategory(id int) (event_model.Category, error) {
	row, err := r.queries.GetCategory(context.Background(), int32(id))
	if err != nil {
		return event_model.Category{}, fmt.Errorf("get category %d: %w", id, err)
	}

	return toCategory(row), nil
}

func toCategory(cat sqlc.Category) event_model.Category {
	return event_model.Category{ID: int(cat.ID), Name: cat.Name}
}
func (r *EventRepository) ListMapEvents(ctx context.Context, filter event_model.MapFilter) ([]event_model.MapEvent, error) {
	params := sqlc.ListMapEventsParams{
		MinLat: filter.MinLat,
		MaxLat: filter.MaxLat,
		MinLng: filter.MinLng,
		MaxLng: filter.MaxLng,
	}
	if filter.UserID != "" {
		uid, err := toUUID(filter.UserID)
		if err != nil {
			return nil, err
		}
		params.UserID = uid
	}
	rows, err := r.queries.ListMapEvents(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("list map events: %w", err)
	}
	events := make([]event_model.MapEvent, 0, len(rows))
	for _, row := range rows {
		events = append(events, mapEventRowToModel(row))
	}
	return events, nil
}

func mapEventRowToModel(row sqlc.ListMapEventsRow) event_model.MapEvent {
	return event_model.MapEvent{
		Event: toModel(sqlc.Event{
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
		}, 0),
		IsOfficial: row.IsOfficial,
		HasFriends: row.HasFriends,
	}
}

// toModel converts a database row to the domain model.
// The code is not stored in the events table, so it is passed separately.
func toModel(row sqlc.Event, code int64) event_model.Event {
	return event_model.Event{
		ID:           int(row.ID),
		Code:         code,
		IsOfficial:   code > 0,
		Title:        row.Title,
		Subtitle:     row.Subtitle.String,
		Schedule:     row.Schedule.String,
		Tickets:      row.Tickets.String,
		URL:          row.Url.String,
		TicketsURL:   row.Ticketsurl.String,
		Description:  row.Description.String,
		Free:         row.Free,
		Permanent:    row.Permanent,
		RelatedURLs:  row.Relatedurls.String,
		URLnames:     row.Urlnames.String,
		Modality:     row.Modality.String,
		Images:       row.Images.String,
		Videos:       row.Videos.String,
		Email:        row.Email.String,
		Place:        row.Place.String,
		Latitude:     row.Latitude.Float32,
		Longitude:    row.Longitude.Float32,
		PostalCode:   row.PostalCode.String,
		Municipality: row.Municipality.String,
		Region:       row.Region.String,
		Address:      row.Address.String,
		Phone:        row.Phone.String,
		ActivityURL:  row.Activityurl.String,
		StartDate:    row.Startdate.Time,
		EndDate:      row.Enddate.Time,
	}
}

func toModelExtended(row sqlc.GetEventRow, cats []string) event_model.Event {
	return event_model.Event{
		ID:           int(row.ID),
		Code:         row.Code.Int64,
		IsOfficial:   (row.Code.Valid && row.Code.Int64 > 0) && !row.Creator.Valid,
		Title:        row.Title,
		Subtitle:     row.Subtitle.String,
		Schedule:     row.Schedule.String,
		Tickets:      row.Tickets.String,
		URL:          row.Url.String,
		TicketsURL:   row.Ticketsurl.String,
		Description:  row.Description.String,
		Free:         row.Free,
		Permanent:    row.Permanent,
		RelatedURLs:  row.Relatedurls.String,
		URLnames:     row.Urlnames.String,
		Modality:     row.Modality.String,
		Images:       row.Images.String,
		Videos:       row.Videos.String,
		Email:        row.Email.String,
		Place:        row.Place.String,
		Latitude:     row.Latitude.Float32,
		Longitude:    row.Longitude.Float32,
		PostalCode:   row.PostalCode.String,
		Municipality: row.Municipality.String,
		Region:       row.Region.String,
		Address:      row.Address.String,
		Phone:        row.Phone.String,
		ActivityURL:  row.Activityurl.String,
		StartDate:    row.Startdate.Time,
		EndDate:      row.Enddate.Time,
		Creator:      row.Creator,
		DateCreated:  row.Datecreated.Time,
		Categories:   cats,
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

func toPGXFloat(f float32) pgtype.Float4 {
	return pgtype.Float4{Float32: f, Valid: true}
}

func toPGXBool(b bool) pgtype.Bool {
	return pgtype.Bool{Bool: b, Valid: true}
}
