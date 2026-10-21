package event_service

import (
	"context"

	"api-aprop/internal/models/event_model"
)

// Service contains the business logic for events.
type Service interface {
	GetEvent(id int) (event_model.Event, error)
	ListEvents(params map[string]any) ([]event_model.Event, error)
	ListMapEvents(ctx context.Context, filter event_model.MapFilter) ([]event_model.MapEvent, error)
	CreateEvent(e event_model.Event) (event_model.Event, error)
	LoadOfficialEvents(ctx context.Context, pageSize int) error
}

