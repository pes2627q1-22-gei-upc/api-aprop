package event_repository

import (
	"api-aprop/internal/models/event_model"
)

// Repository stores and retrieves events.
type Repository interface {
	Create(e event_model.Event, official bool) (event_model.Event, error)
	CreateOrReplace(e event_model.Event) (event_model.Event, error)
	Update(id int, e event_model.Event, official bool) (event_model.Event, error)
	SetCategories(id int, cats []string, official bool) (error)
	Get(id int) (event_model.Event, error)
	List(params map[string]any) ([]event_model.Event, error)
	GetCategories() ([]event_model.Category, error)
	GetCategory(id int) (event_model.Category, error)
}
