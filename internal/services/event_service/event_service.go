package event_service

import (
	"container/heap"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"api-aprop/internal/clients/gencat_client"
	"api-aprop/internal/clients/s3_client"
	"api-aprop/internal/models/event_model"
	"api-aprop/internal/models/preference_model"
	"api-aprop/internal/repositories/auth_repository"
	"api-aprop/internal/repositories/event_repository"
	"api-aprop/internal/repositories/preference_repository"
)

type Item struct {
	Value    event_model.Event
	Priority float32
	Index    int
}

type PriorityQueue []*Item

// heap.Interface implementation

func (pq PriorityQueue) Len() int {
	return len(pq)
}

func (pq PriorityQueue) Less(i, j int) bool {
	return pq[i].Priority > pq[j].Priority
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].Index = i
	pq[j].Index = j
}

func (pq *PriorityQueue) Push(x any) {
	item := x.(*Item)
	item.Index = len(*pq)
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() any {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil // Avoid retaining the item in memory
	item.Index = -1
	*pq = old[:n-1]
	return item
}

type EventService struct {
	eventRepository      event_repository.Repository
	authRepository       auth_repository.Repository
	gencatClient         gencat_client.Client
	preferenceRepository preference_repository.Repository
	s3Client             s3_client.Client
}

func NewEventService(eventRepository event_repository.Repository, authRepository auth_repository.Repository, preferenceRepository preference_repository.Repository, gencatClient gencat_client.Client, s3Client s3_client.Client) *EventService {
	return &EventService{
		eventRepository:      eventRepository,
		authRepository:       authRepository,
		preferenceRepository: preferenceRepository,
		gencatClient:         gencatClient,
		s3Client:             s3Client,
	}
}

// ResolveUserID returns the database ID (UUID) of the requester. The identifier is the Firebase UID set by
// the auth middleware, or directly a UUID when the API runs without authentication (local development and tests).
func (s *EventService) ResolveUserID(ctx context.Context, identifier string) (string, error) {
	if identifier == "" {
		return "", ErrUserNotRegistered
	}
	if uid, err := normalizeUserID(identifier); err == nil {
		return uid, nil
	}
	if s.authRepository == nil {
		return "", ErrUserNotRegistered
	}
	user, err := s.authRepository.GetByFirebaseUID(ctx, identifier)
	if errors.Is(err, auth_repository.ErrUserNotFound) {
		return "", fmt.Errorf("%w: %w", ErrUserNotRegistered, err)
	}
	if err != nil {
		return "", fmt.Errorf("resolve user: %w", err)
	}
	return user.ID.String(), nil
}

func (s *EventService) GetEvent(id int) (event_model.Event, error) {
	e, err := s.eventRepository.Get(id)
	if errors.Is(err, event_repository.ErrEventNotFound) {
		return event_model.Event{}, fmt.Errorf("%w: %w", ErrEventNotFound, err)
	}
	if err != nil {
		return event_model.Event{}, err
	}
	return e, nil
}

func (s *EventService) GetRecommendedEvents(ctx context.Context, userid string, pageSize int, pageNum int) ([]event_model.Event, error) {
	events, err := s.eventRepository.List(map[string]any{"pageSize": pageSize, "pageNum": pageNum, "filterCategory": 0})
	if errors.Is(err, event_repository.ErrInvalidFilter) {
		return []event_model.Event{}, fmt.Errorf("%w: %w", ErrInvalidFilter, err)
	}
	if err != nil {
		return []event_model.Event{}, err
	}

	prefs, err := s.preferenceRepository.GetUserPreferences(ctx, userid)
	if err != nil {
		return []event_model.Event{}, err
	}

	ordered_events := &PriorityQueue{}
	heap.Init(ordered_events)
	for _, ev := range events {
		prio := float32(1.0)
		for _, cat := range ev.Categories {
			// A category without preference has weight 1, the default weight of UserPreferences
			ind := slices.IndexFunc(prefs, func(p preference_model.UserPreference) bool { return p.CategoryName == cat })
			if ind != -1 {
				prio *= prefs[ind].Weight
			}
		}
		heap.Push(ordered_events, &Item{Value: ev, Priority: prio})
	}

	ret := []event_model.Event{}
	for ordered_events.Len() > 0 {
		tmp, _ := heap.Pop(ordered_events).(*Item)
		ret = append(ret, tmp.Value)
	}

	return ret, nil
}

func (s *EventService) ListEvents(params map[string]any) ([]event_model.Event, error) {
	if q, ok := params["search"].(string); ok && strings.TrimSpace(q) != "" {
		pageNum := 0
		if pn, ok := params["pageNum"].(int); ok {
			pageNum = pn
		}
		pageSize := 20
		if ps, ok := params["pageSize"].(int); ok && ps > 0 {
			pageSize = ps
		}
		res, err := s.eventRepository.Search(context.Background(), strings.TrimSpace(q), pageNum, pageSize)
		if err != nil {
			return nil, err
		}
		return res.Events, nil
	}
	events, err := s.eventRepository.List(params)
	if errors.Is(err, event_repository.ErrInvalidFilter) {
		return nil, fmt.Errorf("%w: %w", ErrInvalidFilter, err)
	}
	return events, err
}

func (s *EventService) SearchEvents(ctx context.Context, query string, pageNum, pageSize int) (event_model.SearchResult, error) {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return event_model.SearchResult{
			Query:    query,
			Total:    0,
			PageNum:  pageNum,
			PageSize: pageSize,
			Events:   []event_model.Event{},
		}, nil
	}
	return s.eventRepository.Search(ctx, trimmed, pageNum, pageSize)
}

func (s *EventService) ListMapEvents(ctx context.Context, filter event_model.MapFilter) ([]event_model.MapEvent, error) {
	if err := validateBoundingBox(filter.MinLat, filter.MaxLat, filter.MinLng, filter.MaxLng); err != nil {
		return nil, err
	}
	if filter.UserID != "" {
		uid, err := normalizeUserID(filter.UserID)
		if err != nil {
			return nil, err
		}
		filter.UserID = uid
	}
	events, err := s.eventRepository.ListMapEvents(ctx, filter)
	if err != nil {
		return nil, err
	}
	if events == nil {
		events = []event_model.MapEvent{}
	}
	return events, nil
}

func (s *EventService) GetPopularEvents(pageSize int, pageNum int) ([]event_model.Event, error) {
	events, err := s.eventRepository.GetPopular(pageSize, pageNum)
	if errors.Is(err, event_repository.ErrInvalidFilter) {
		return nil, fmt.Errorf("%w: %w", ErrInvalidFilter, err)
	}
	return events, err
}

// CreateUnofficialEvent validates and publishes a user-created unofficial event adhering to rule R5.
func (s *EventService) CreateUnofficialEvent(ctx context.Context, userID string, e event_model.Event) (event_model.Event, error) {
	uid, err := normalizeUserID(userID)
	if err != nil {
		return event_model.Event{}, err
	}

	now := time.Now().UTC()
	if err := validateUnofficialEvent(e, now); err != nil {
		return event_model.Event{}, err
	}

	var creatorUUID pgtype.UUID
	if err := creatorUUID.Scan(uid); err != nil {
		return event_model.Event{}, fmt.Errorf("%w: %v", ErrInvalidUserID, err)
	}

	e.ID = 0
	e.Code = 0
	e.IsOfficial = false
	e.Creator = creatorUUID
	e.DateCreated = now

	return s.eventRepository.Create(e, false)
}

// UpdateUnofficialEvent replaces the fields of an unofficial event. The creator and the creation date are kept,
// and rule R5 is checked against the original creation date. The handler checks that the requester is the creator.
func (s *EventService) UpdateUnofficialEvent(ctx context.Context, id int, e event_model.Event) (event_model.Event, error) {
	stored, err := s.GetEvent(id)
	if err != nil {
		return event_model.Event{}, err
	}
	if stored.IsOfficial || !stored.Creator.Valid {
		return event_model.Event{}, fmt.Errorf("update event %d: %w: official events cannot be modified", id, ErrInvalidEvent)
	}

	if err := validateUnofficialEvent(e, stored.DateCreated); err != nil {
		return event_model.Event{}, err
	}

	e.ID = id
	e.Code = 0
	e.IsOfficial = false
	e.Creator = stored.Creator
	e.DateCreated = stored.DateCreated
	// The image is uploaded with POST /events/{id}/image, so a body without it keeps the current one
	// TODO(DP-026): there is no way to remove the image of an event.
	if e.Images == "" {
		e.Images = stored.Images
	}

	updated, err := s.eventRepository.Update(id, e, false)
	if errors.Is(err, event_repository.ErrEventNotFound) {
		return event_model.Event{}, fmt.Errorf("%w: %w", ErrEventNotFound, err)
	}
	if err != nil {
		return event_model.Event{}, err
	}
	return updated, nil
}

// DeleteUnofficialEvent deletes an unofficial event and everything that references it.
// The handler checks that the requester is the creator.
func (s *EventService) DeleteUnofficialEvent(ctx context.Context, id int) error {
	stored, err := s.GetEvent(id)
	if err != nil {
		return err
	}
	if stored.IsOfficial || !stored.Creator.Valid {
		return fmt.Errorf("delete event %d: %w: official events cannot be deleted", id, ErrInvalidEvent)
	}

	err = s.eventRepository.Delete(ctx, id)
	if errors.Is(err, event_repository.ErrEventNotFound) {
		return fmt.Errorf("%w: %w", ErrEventNotFound, err)
	}
	return err
}

// UploadEventImage uploads an event poster/image to S3 storage and associates it with the event.
func (s *EventService) UploadEventImage(ctx context.Context, eventID int, filename string, contentType string, reader io.Reader) (string, error) {
	if eventID <= 0 {
		return "", ErrEventNotFound
	}

	// Verify event exists
	_, err := s.eventRepository.Get(eventID)
	if err != nil {
		if errors.Is(err, event_repository.ErrEventNotFound) {
			return "", ErrEventNotFound
		}
		return "", err
	}

	if s.s3Client == nil {
		return "", fmt.Errorf("%w: storage client not configured", ErrImageUploadFailed)
	}

	url, err := s.s3Client.UploadEventImage(ctx, eventID, filename, contentType, reader)
	if err != nil {
		if errors.Is(err, s3_client.ErrInvalidContentType) {
			return "", ErrInvalidContentType
		}
		return "", fmt.Errorf("%w: %v", ErrImageUploadFailed, err)
	}

	if err := s.eventRepository.UpdateImage(ctx, eventID, url); err != nil {
		return "", fmt.Errorf("update event image: %w", err)
	}

	return url, nil
}

// LoadOfficialEvents fetches the most recent events from the gencat open data and stores them.
func (s *EventService) LoadOfficialEvents(ctx context.Context, pageSize int) error {
	i := 1
	total := 0
	end := false
	for !end {
		events, err := s.gencatClient.FetchEvents(ctx, i, pageSize)
		if err != nil {
			return fmt.Errorf("load official events: %w", err)
		}
		if len(events) < pageSize {
			end = true
		}
		for _, e := range events {
			if _, err := s.eventRepository.CreateOrReplace(e); err != nil {
				log.Printf("store official event %d: %v", e.Code, err)
			}
		}

		total += len(events)
		log.Printf("events loaded: %d", total)
		i += 1
	}
	return nil
}

func (s *EventService) GetCategories() ([]event_model.Category, error) {
	return s.eventRepository.GetCategories()
}

func (s *EventService) GetCategory(id int) (event_model.Category, error) {
	cat, err := s.eventRepository.GetCategory(id)
	if errors.Is(err, event_repository.ErrCategoryNotFound) {
		return event_model.Category{}, fmt.Errorf("%w: %w", ErrCategoryNotFound, err)
	}
	return cat, err
}

// validateUnofficialEvent checks the fields required to publish an unofficial event and rule R5
// against the date the event was created.
func validateUnofficialEvent(e event_model.Event, createdAt time.Time) error {
	if strings.TrimSpace(e.Title) == "" {
		return ErrInvalidEventTitle
	}
	if strings.TrimSpace(e.Description) == "" {
		return ErrInvalidEventDescription
	}
	if len(e.Categories) == 0 {
		return ErrInvalidEventCategory
	}

	// The dates are stored without the time of day, so R5 compares days
	// R5: dataInici >= dataCreacio
	if e.StartDate.IsZero() || day(e.StartDate).Before(day(createdAt.UTC())) {
		return fmt.Errorf("%w: start date must not be before the creation date", ErrInvalidEventDates)
	}
	// R5: dataFi >= dataInici
	if !e.EndDate.IsZero() && day(e.EndDate).Before(day(e.StartDate)) {
		return fmt.Errorf("%w: end date must not be before start date", ErrInvalidEventDates)
	}

	// Validate coordinates if provided
	if e.Latitude != 0 || e.Longitude != 0 {
		if e.Latitude < -90 || e.Latitude > 90 || e.Longitude < -180 || e.Longitude > 180 {
			return ErrInvalidCoordinates
		}
	}
	return nil
}

// day returns the date of t without the time of day, the same date the database stores for it.
func day(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func validateBoundingBox(minLat, maxLat, minLng, maxLng float32) error {
	if math.IsNaN(float64(minLat)) || math.IsNaN(float64(maxLat)) || math.IsNaN(float64(minLng)) || math.IsNaN(float64(maxLng)) {
		return fmt.Errorf("%w: coordinates must not be NaN", ErrInvalidBoundingBox)
	}
	if minLat < -90 || maxLat > 90 || minLat > maxLat {
		return fmt.Errorf("%w: latitude range [%v, %v] must be within [-90, 90]", ErrInvalidBoundingBox, minLat, maxLat)
	}
	if minLng < -180 || maxLng > 180 || minLng > maxLng {
		return fmt.Errorf("%w: longitude range [%v, %v] must be within [-180, 180]", ErrInvalidBoundingBox, minLng, maxLng)
	}
	return nil
}

// normalizeUserID accepts only the canonical UUID form (8-4-4-4-12 hex digits) and lowercases it.
func normalizeUserID(userID string) (string, error) {
	if len(userID) != 36 {
		return "", fmt.Errorf("%w: %q", ErrInvalidUserID, userID)
	}
	for i, c := range userID {
		switch {
		case i == 8 || i == 13 || i == 18 || i == 23:
			if c != '-' {
				return "", fmt.Errorf("%w: %q", ErrInvalidUserID, userID)
			}
		case !isHex(c):
			return "", fmt.Errorf("%w: %q", ErrInvalidUserID, userID)
		}
	}
	return strings.ToLower(userID), nil
}

func isHex(c rune) bool {
	return ('0' <= c && c <= '9') || ('a' <= c && c <= 'f') || ('A' <= c && c <= 'F')
}
