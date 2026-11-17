package event_service

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"slices"
	"strings"
	"time"

	"api-aprop/internal/clients/gencat_client"
	"api-aprop/internal/clients/s3_client"
	"api-aprop/internal/models/event_model"
	"api-aprop/internal/models/signup_model"
	"api-aprop/internal/models/user_model"
	"api-aprop/internal/repositories/event_repository"
	"api-aprop/internal/repositories/preference_repository"
	"api-aprop/internal/repositories/signup_repository"
)

const (
	defaultPreferenceWeight = 1.0
	signUpCategoryStep      = 0.5
	maxSignUpCategoryBoost  = 2.0
	popularityWeight        = 0.1
	maxPopularityBoost      = 0.5
)

type EventService struct {
	eventRepository      event_repository.Repository
	gencatClient         gencat_client.Client
	preferenceRepository preference_repository.Repository
	signUpRepository     signup_repository.Repository
	s3Client             s3_client.Client
}

func NewEventService(eventRepository event_repository.Repository, preferenceRepository preference_repository.Repository, gencatClient gencat_client.Client, s3Client s3_client.Client) *EventService {
	return &EventService{
		eventRepository:      eventRepository,
		preferenceRepository: preferenceRepository,
		gencatClient:         gencatClient,
		s3Client:             s3Client,
	}
}

// WithSignUpRepository attaches the sign-up repository so recommendations incorporate user enrollments.
func (s *EventService) WithSignUpRepository(signUpRepository signup_repository.Repository) *EventService {
	s.signUpRepository = signUpRepository
	return s
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

// GetRecommendedEvents returns active events ranked for userID using UserPreferences (Interès./pes),
// the user's SignUp category affinity (excluding events already joined), and global sign-up popularity,
// applying pagination after global ranking.
func (s *EventService) GetRecommendedEvents(ctx context.Context, userid string, pageSize int, pageNum int) ([]event_model.Event, error) {
	normalizedUID, err := user_model.NormalizeID(userid)
	if err != nil {
		return nil, err
	}
	if pageSize <= 0 || pageNum < 0 {
		return nil, fmt.Errorf("recommended events: %w: pageSize must be > 0 and pageNum >= 0", ErrInvalidFilter)
	}

	activeEvents, err := s.eventRepository.ListActive(ctx)
	if err != nil {
		return nil, err
	}
	if len(activeEvents) == 0 {
		return []event_model.Event{}, nil
	}

	prefWeights := make(map[string]float64)
	if s.preferenceRepository != nil {
		prefs, err := s.preferenceRepository.GetUserPreferences(ctx, normalizedUID)
		if err != nil {
			return nil, err
		}
		for _, p := range prefs {
			prefWeights[p.CategoryName] = float64(p.Weight)
		}
	}

	signedUpIDs, signUpCategoryCounts, err := s.userSignUpSignals(ctx, normalizedUID, activeEvents)
	if err != nil {
		return nil, err
	}

	// TODO(DP-030): confirm the exact combination formula (UserPreferences + SignUp category boost + popularity)
	// and whether already-enrolled events should be excluded rather than penalized.
	type scoredEvent struct {
		event       event_model.Event
		score       float64
		signUpCount int
	}
	candidates := make([]scoredEvent, 0, len(activeEvents))
	for _, ae := range activeEvents {
		if signedUpIDs[ae.Event.ID] {
			continue
		}
		candidates = append(candidates, scoredEvent{
			event:       ae.Event,
			score:       scoreRecommendedEvent(ae.Event, ae.SignUpCount, prefWeights, signUpCategoryCounts),
			signUpCount: ae.SignUpCount,
		})
	}

	slices.SortFunc(candidates, func(a, b scoredEvent) int {
		if a.score != b.score {
			if a.score > b.score {
				return -1
			}
			return 1
		}
		if a.signUpCount != b.signUpCount {
			return cmp.Compare(b.signUpCount, a.signUpCount)
		}
		aZero, bZero := a.event.EndDate.IsZero(), b.event.EndDate.IsZero()
		if aZero != bZero {
			if aZero {
				return 1
			}
			return -1
		}
		if !aZero && !a.event.EndDate.Equal(b.event.EndDate) {
			if a.event.EndDate.Before(b.event.EndDate) {
				return -1
			}
			return 1
		}
		return cmp.Compare(a.event.ID, b.event.ID)
	})

	if pageNum > len(candidates)/pageSize {
		return []event_model.Event{}, nil
	}
	offset := pageNum * pageSize
	if offset >= len(candidates) {
		return []event_model.Event{}, nil
	}
	end := min(offset+pageSize, len(candidates))

	ret := make([]event_model.Event, 0, end-offset)
	for _, c := range candidates[offset:end] {
		ret = append(ret, c.event)
	}
	return ret, nil
}

func (s *EventService) userSignUpSignals(ctx context.Context, userID string, activeEvents []event_repository.ActiveEvent) (map[int]bool, map[string]int, error) {
	signedUpIDs := make(map[int]bool)
	categoryCounts := make(map[string]int)
	if s.signUpRepository == nil {
		return signedUpIDs, categoryCounts, nil
	}

	signUps, err := s.signUpRepository.ListUserEvents(ctx, userID, signup_model.DateRange{})
	if err != nil {
		return nil, nil, err
	}
	if len(signUps) == 0 {
		return signedUpIDs, categoryCounts, nil
	}

	activeCatsByID := make(map[int][]string, len(activeEvents))
	for _, ae := range activeEvents {
		activeCatsByID[ae.Event.ID] = ae.Event.Categories
	}

	var missingCatEventIDs []int
	for _, su := range signUps {
		signedUpIDs[su.ID] = true
		cats := su.Categories
		if len(cats) == 0 {
			cats = activeCatsByID[su.ID]
		}
		if len(cats) > 0 {
			for _, cat := range cats {
				categoryCounts[cat]++
			}
		} else {
			missingCatEventIDs = append(missingCatEventIDs, su.ID)
		}
	}

	if len(missingCatEventIDs) > 0 && s.preferenceRepository != nil {
		evCats, err := s.preferenceRepository.GetEventCategories(ctx, missingCatEventIDs)
		if err != nil {
			return nil, nil, err
		}
		if len(evCats) > 0 {
			allCats, err := s.preferenceRepository.ListAllCategories(ctx)
			if err != nil {
				return nil, nil, err
			}
			catNames := make(map[int]string, len(allCats))
			for _, c := range allCats {
				catNames[c.ID] = c.Name
			}
			for _, id := range missingCatEventIDs {
				for _, catID := range evCats[id] {
					if name, ok := catNames[catID]; ok {
						categoryCounts[name]++
					}
				}
			}
		}
	}

	return signedUpIDs, categoryCounts, nil
}

func scoreRecommendedEvent(ev event_model.Event, signUpCount int, prefWeights map[string]float64, signUpCategoryCounts map[string]int) float64 {
	categoryScore := defaultPreferenceWeight
	if len(ev.Categories) > 0 {
		sum := 0.0
		for _, cat := range ev.Categories {
			w := defaultPreferenceWeight
			if pw, ok := prefWeights[cat]; ok {
				w = pw
			}
			if cnt := signUpCategoryCounts[cat]; cnt > 0 {
				w += min(float64(cnt)*signUpCategoryStep, maxSignUpCategoryBoost)
			}
			sum += w
		}
		categoryScore = sum / float64(len(ev.Categories))
	}

	popularityBoost := 0.0
	if signUpCount > 0 {
		popularityBoost = min(popularityWeight*math.Log1p(float64(signUpCount)), maxPopularityBoost)
	}
	return categoryScore + popularityBoost
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
		uid, err := user_model.NormalizeID(filter.UserID)
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
	creatorUUID, err := user_model.ParseID(userID)
	if err != nil {
		return event_model.Event{}, err
	}

	now := time.Now().UTC()
	if err := validateUnofficialEvent(e, now); err != nil {
		return event_model.Event{}, err
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
