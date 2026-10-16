package event_repository

import (
	"api-aprop/internal/models/event_model"
)

// Repository stores and retrieves events.
type Repository interface {
	Create(e event_model.Event) (event_model.Event, error)
	CreateOrReplace(event_model.Event) (event_model.Event, error)
	Update(int, event_model.Event) (event_model.Event, error)
	Get(id int) (event_model.Event, error)
	List() ([]event_model.Event, error)
}
