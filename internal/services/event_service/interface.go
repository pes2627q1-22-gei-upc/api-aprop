package event_service

import (
	"context"

	"api-aprop/internal/models/event_model"
)

// Service contains the business logic for events.
type Service interface {
	GetEvent(id int) (event_model.Event, error)
	ListEvents() []event_model.Event
	CreateEvent(e event_model.Event) (event_model.Event, error)
	LoadOfficialEvents(ctx context.Context, pageSize int) error
}
