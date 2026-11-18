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
	"api-aprop/internal/models/user_model"
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
	return pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		deleted, err := deleteEventTx(ctx, r.queries.WithTx(tx), int32(id))
		if err != nil {
			return err
		}
		if deleted == 0 {
			return fmt.Errorf("delete event %d: %w", id, event_repository.ErrEventNotFound)
		}
		return nil
	})
}

// deleteEventTx deletes an event and every row that depends on it with the queries of an open
// transaction. It returns the number of deleted events (0 if it did not exist).
// UserRepository.DeleteUser also uses it to delete the unofficial events of a user.
func deleteEventTx(ctx context.Context, q *sqlc.Queries, eventID int32) (int64, error) {
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
			return 0, fmt.Errorf("delete %s of event %d: %w", d.name, eventID, err)
		}
	}

	deleted, err := q.DeleteEvent(ctx, eventID)
	if err != nil {
		return 0, fmt.Errorf("delete event %d: %w", eventID, err)
	}
	return deleted, nil
}

func (r *EventRepository) UpdateImage(ctx context.Context, id int, imageURL string) error {
	return r.queries.UpdateEventImage(ctx, sqlc.UpdateEventImageParams{
		ID:     int32(id),
		Images: toText(imageURL),
	})
}

func (r *EventRepository) GetUnofficialState(ctx context.Context, id int) (string, error) {
	state, err := r.queries.GetUnofficialEventState(ctx, int32(id))
	if errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("get state of event %d: %w", id, event_repository.ErrEventNotFound)
	}
	if err != nil {
		return "", fmt.Errorf("get state of event %d: %w", id, err)
	}
	return state, nil
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
	// GetEvent returns one row per category, so a missing event is an empty result, not pgx.ErrNoRows
	if len(rows) == 0 {
		return event_model.Event{}, fmt.Errorf("get event %d: %w", id, event_repository.ErrEventNotFound)
	}

	cats := []string{}
	for _, r := range rows {
		cats = appendCategory(cats, r.Category)
	}

	return toModelExtended(rows[0], cats), nil
}

// appendCategory adds the category of a row. Events without categories have a single row with a NULL category.
func appendCategory(cats []string, category pgtype.Text) []string {
	if category.Valid && category.String != "" {
		return append(cats, category.String)
	}
	return cats
}

func regionFilterToDBEntry(s string) (string, error) {
	tmp := strings.Split(s, "_")
	if len(tmp) != 2 {
		return "", fmt.Errorf("list events: %w: filterRegion must be <type>_<name>", event_repository.ErrInvalidFilter)
	}
	return fmt.Sprintf("agenda:ubicacions/%s/%s", tmp[0], tmp[1]), nil
}

func (r *EventRepository) List(params map[string]any) ([]event_model.Event, error) {
	var err error

	pageSize, ok := params["pageSize"].(int)
	if !ok {
		return nil, fmt.Errorf("list events: %w: pageSize", event_repository.ErrInvalidFilter)
	}
	pageNum, ok := params["pageNum"].(int)
	if !ok {
		return nil, fmt.Errorf("list events: %w: pageNum", event_repository.ErrInvalidFilter)
	}
	if pageSize < 0 || pageNum < 0 {
		return nil, fmt.Errorf("list events: %w: pageSize and pageNum must not be negative", event_repository.ErrInvalidFilter)
	}

	filterCategory, ok := params["filterCategory"].(int)
	if !ok {
		return nil, fmt.Errorf("list events: %w: filterCategory", event_repository.ErrInvalidFilter)
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
			return nil, fmt.Errorf("list events: %w: startBefore: %v", event_repository.ErrInvalidFilter, err)
		}
	}
	startAfterTmp, _ := params["startAfter"].(string)
	if startAfterTmp == "" {
		startAfter = time.Time{}
	} else {
		startAfter, err = time.Parse("2006-01-02T15:04:05.000", startAfterTmp)
		if err != nil {
			return nil, fmt.Errorf("list events: %w: startAfter: %v", event_repository.ErrInvalidFilter, err)
		}
	}
	endBeforeTmp, _ := params["endBefore"].(string)
	if endBeforeTmp == "" {
		endBefore = time.Time{}
	} else {
		endBefore, err = time.Parse("2006-01-02T15:04:05.000", endBeforeTmp)
		if err != nil {
			return nil, fmt.Errorf("list events: %w: endBefore: %v", event_repository.ErrInvalidFilter, err)
		}
	}
	endAfterTmp, _ := params["endAfter"].(string)
	if endAfterTmp == "" {
		endAfter = time.Time{}
	} else {
		endAfter, err = time.Parse("2006-01-02T15:04:05.000", endAfterTmp)
		if err != nil {
			return nil, fmt.Errorf("list events: %w: endAfter: %v", event_repository.ErrInvalidFilter, err)
		}
	}

	filterPermanentTmp, _ := params["filterPermanent"].(string)
	if filterPermanentTmp == "" {
		filterPermanent = pgtype.Bool{Valid: false}
	} else {
		filterPermanent.Bool, err = strconv.ParseBool(filterPermanentTmp)
		if err != nil {
			return nil, fmt.Errorf("list events: %w: filterPermanent: %v", event_repository.ErrInvalidFilter, err)
		}
		filterPermanent.Valid = true
	}
	filterFreeTmp, _ := params["filterFree"].(string)
	if filterFreeTmp == "" {
		filterFree = pgtype.Bool{Valid: false}
	} else {
		filterFree.Bool, err = strconv.ParseBool(filterFreeTmp)
		if err != nil {
			return nil, fmt.Errorf("list events: %w: filterFree: %v", event_repository.ErrInvalidFilter, err)
		}
		filterFree.Valid = true
	}
	filterOfficialTmp, _ := params["filterOfficial"].(string)
	if filterOfficialTmp == "" {
		filterOfficial = pgtype.Bool{Valid: false}
	} else {
		filterOfficial.Bool, err = strconv.ParseBool(filterOfficialTmp)
		if err != nil {
			return nil, fmt.Errorf("list events: %w: filterOfficial: %v", event_repository.ErrInvalidFilter, err)
		}
		filterOfficial.Valid = true
	}
	filterModalityTmp, _ := params["filterModality"].(string)
	switch filterModalityTmp {
	case "":
		filterModality = sqlc.NullModalitytype{Valid: false}
	case "Online":
		filterModality = sqlc.NullModalitytype{Modalitytype: sqlc.ModalitytypeOnline, Valid: true}
	case "Face-to-Face":
		filterModality = sqlc.NullModalitytype{Modalitytype: sqlc.ModalitytypeFaceToFace, Valid: true}
	case "Online and Face-to-Face":
		filterModality = sqlc.NullModalitytype{Modalitytype: sqlc.ModalitytypeOnlineandFaceToFace, Valid: true}
	default:
		return nil, fmt.Errorf("list events: %w: filterModality", event_repository.ErrInvalidFilter)
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
		cats = appendCategory(cats, row.Category)
	}
	events = append(events, toModelExtended(sqlc.GetEventRow(currentevent), cats))
	return events, nil
}

func toGetEventRow(r sqlc.SearchEventsWithRelevanceRow) sqlc.GetEventRow {
	return sqlc.GetEventRow{
		ID:           r.ID,
		Title:        r.Title,
		Subtitle:     r.Subtitle,
		Description:  r.Description,
		Schedule:     r.Schedule,
		Tickets:      r.Tickets,
		Url:          r.Url,
		Ticketsurl:   r.Ticketsurl,
		Free:         r.Free,
		Permanent:    r.Permanent,
		Modality:     r.Modality,
		Relatedurls:  r.Relatedurls,
		Urlnames:     r.Urlnames,
		Images:       r.Images,
		Videos:       r.Videos,
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
		cats = appendCategory(cats, row.Category)
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
	if errors.Is(err, pgx.ErrNoRows) {
		return event_model.Category{}, fmt.Errorf("get category %d: %w", id, event_repository.ErrCategoryNotFound)
	}
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
		uid, err := user_model.ParseID(filter.UserID)
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
	lastevent := sqlc.ListMapEventsRow{}
	cats := []string{}
	for _, row := range rows {
		if lastevent == (sqlc.ListMapEventsRow{}) {
			lastevent = row
		}
		if lastevent.ID != row.ID {
			events = append(events, mapEventRowToModel(lastevent, cats))
			lastevent = row
			cats = []string{row.Category.String}
		} else {
			cats = append(cats, row.Category.String)
		}
	}
	if lastevent != (sqlc.ListMapEventsRow{}) {
		events = append(events, mapEventRowToModel(lastevent, cats))
	}

	return events, nil
}

func getPopularToGetEventRow(row sqlc.GetPopularEventsRow) sqlc.GetEventRow {
	return sqlc.GetEventRow{
		ID:           row.ID,
		Title:        row.Title,
		Description:  row.Description,
		Subtitle:     row.Subtitle,
		Schedule:     row.Schedule,
		Tickets:      row.Tickets,
		Url:          row.Url,
		Ticketsurl:   row.Ticketsurl,
		Free:         row.Free,
		Permanent:    row.Permanent,
		Modality:     row.Modality,
		Relatedurls:  row.Relatedurls,
		Urlnames:     row.Urlnames,
		Images:       row.Images,
		Videos:       row.Videos,
		Email:        row.Email,
		Phone:        row.Phone,
		Activityurl:  row.Activityurl,
		Startdate:    row.Startdate,
		Enddate:      row.Enddate,
		Place:        row.Place,
		Latitude:     row.Latitude,
		Longitude:    row.Longitude,
		PostalCode:   row.PostalCode,
		Municipality: row.Municipality,
		Region:       row.Region,
		Address:      row.Address,
		Code:         row.Code,
		Creator:      row.Creator,
		Datecreated:  row.Datecreated,
		Category:     row.Category,
	}
}

func (r *EventRepository) GetPopular(pageSize int, pageNum int) ([]event_model.Event, error) {
	if pageSize < 0 || pageNum < 0 {
		return nil, fmt.Errorf("popular events: %w: pageSize and pageNum must not be negative", event_repository.ErrInvalidFilter)
	}
	rows, err := r.queries.GetPopularEvents(context.Background(), sqlc.GetPopularEventsParams{Limit: int32(pageSize), Offset: int32(pageSize * pageNum)})
	if err != nil {
		return []event_model.Event{}, err
	}
	if len(rows) == 0 {
		return []event_model.Event{}, nil
	}
	events := []event_model.Event{}
	currentevent := rows[0]
	cats := []string{}
	for _, row := range rows {
		if row.ID != currentevent.ID {
			events = append(events, toModelExtended(getPopularToGetEventRow(currentevent), cats))
			cats = []string{}
			currentevent = row
		}
		cats = appendCategory(cats, row.Category)
	}
	events = append(events, toModelExtended(getPopularToGetEventRow(currentevent), cats))
	return events, nil
}

func recommendedRowToGetEventRow(row sqlc.ListRecommendedEventsRow) sqlc.GetEventRow {
	return sqlc.GetEventRow{
		ID:           row.ID,
		Title:        row.Title,
		Subtitle:     row.Subtitle,
		Description:  row.Description,
		Schedule:     row.Schedule,
		Tickets:      row.Tickets,
		Url:          row.Url,
		Ticketsurl:   row.Ticketsurl,
		Free:         row.Free,
		Permanent:    row.Permanent,
		Modality:     row.Modality,
		Relatedurls:  row.Relatedurls,
		Urlnames:     row.Urlnames,
		Images:       row.Images,
		Videos:       row.Videos,
		Email:        row.Email,
		Phone:        row.Phone,
		Activityurl:  row.Activityurl,
		Startdate:    row.Startdate,
		Enddate:      row.Enddate,
		Place:        row.Place,
		Latitude:     row.Latitude,
		Longitude:    row.Longitude,
		PostalCode:   row.PostalCode,
		Municipality: row.Municipality,
		Region:       row.Region,
		Address:      row.Address,
		Code:         row.Code,
		Creator:      row.Creator,
		Datecreated:  row.Datecreated,
		Category:     row.Category,
	}
}

func groupRecommendedEventRows(rows []sqlc.ListRecommendedEventsRow) []event_model.Event {
	if len(rows) == 0 {
		return []event_model.Event{}
	}
	events := make([]event_model.Event, 0, len(rows))
	current := rows[0]
	cats := []string{}
	for _, row := range rows {
		if row.ID != current.ID {
			events = append(events, toModelExtended(recommendedRowToGetEventRow(current), cats))
			cats = []string{}
			current = row
		}
		cats = appendCategory(cats, row.Category)
	}
	events = append(events, toModelExtended(recommendedRowToGetEventRow(current), cats))
	return events
}

// ListRecommended returns a paginated slice of active events ranked in SQL for userID.
func (r *EventRepository) ListRecommended(ctx context.Context, userID pgtype.UUID, pageSize, pageOffset int) ([]event_model.Event, error) {
	rows, err := r.queries.ListRecommendedEvents(ctx, sqlc.ListRecommendedEventsParams{
		UserID:     userID,
		PageOffset: int32(pageOffset),
		PageSize:   int32(pageSize),
	})
	if err != nil {
		return nil, fmt.Errorf("list recommended events: %w", err)
	}
	return groupRecommendedEventRows(rows), nil
}

func (r *EventRepository) EventExists(eventID int) (bool, error) {
	exists, err := r.queries.EventExists(context.Background(), int32(eventID))
	if err != nil {
		return false, fmt.Errorf("check event %d: %w", eventID, err)
	}
	return exists, nil
}

func mapEventRowToModel(row sqlc.ListMapEventsRow, cats []string) event_model.MapEvent {
	return event_model.MapEvent{
		Event: toModelExtended(sqlc.GetEventRow{
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
		}, cats),
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
