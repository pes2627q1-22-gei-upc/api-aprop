package event_repository

import (
	"context"

	"api-aprop/internal/models/event_model"
)

// Repository stores and retrieves events.
type Repository interface {
	Create(e event_model.Event) (event_model.Event, error)
	CreateOrReplace(event_model.Event) (event_model.Event, error)
	Update(int, event_model.Event) (event_model.Event, error)
	Get(id int) (event_model.Event, error)
	List(params map[string]any) ([]event_model.Event, error)
	ListMapEvents(ctx context.Context, filter event_model.MapFilter) ([]event_model.MapEvent, error)
}

