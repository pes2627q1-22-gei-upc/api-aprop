package event_service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"strings"

	"api-aprop/internal/clients/gencat_client"
	"api-aprop/internal/models/event_model"
	"api-aprop/internal/repositories/event_repository"
)

type EventService struct {
	eventRepository event_repository.Repository
	gencatClient    gencat_client.Client
}

func NewEventService(eventRepository event_repository.Repository, gencatClient gencat_client.Client) *EventService {
	return &EventService{
		eventRepository: eventRepository,
		gencatClient:    gencatClient,
	}
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

func (s *EventService) ListEvents(params map[string]any) ([]event_model.Event, error) {
	return s.eventRepository.List(params)
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

func (s *EventService) CreateEvent(e event_model.Event) (event_model.Event, error) {
	if strings.TrimSpace(e.Title) == "" {
		return event_model.Event{}, fmt.Errorf("create event: %w: title is required", ErrInvalidEvent)
	}
	// The ID is always assigned by the repository
	e.ID = 0
	return s.eventRepository.Create(e, false)
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
	return s.eventRepository.GetCategory(id)
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
