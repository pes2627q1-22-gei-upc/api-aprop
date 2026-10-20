package event_service

import (
	"context"
	"errors"
	"fmt"
	"log"
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
		if len(events)<pageSize{
			end = true
		}
		for _, e := range events {
			if _, err := s.eventRepository.CreateOrReplace(e); err != nil {
				log.Printf("store official event %d: %v", e.Code, err)
			}
		}

		total += len(events)
		log.Printf("events loaded: %d", total)
		i+=1
	}
	return nil
}
