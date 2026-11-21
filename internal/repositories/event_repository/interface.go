package event_repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"

	"api-aprop/internal/models/event_model"
)

// Repository stores and retrieves events.
type Repository interface {
	Create(e event_model.Event, official bool) (event_model.Event, error)
	CreateOrReplace(e event_model.Event) (event_model.Event, error)
	Update(id int, e event_model.Event, official bool) (event_model.Event, error)
	UpdateImage(ctx context.Context, id int, imageURL string) error
	// Delete removes the event and every row that references it.
	Delete(ctx context.Context, id int) error
	SetCategories(id int, cats []string, official bool) error
	Get(id int) (event_model.Event, error)
	List(params map[string]any) ([]event_model.Event, error)
	ListRecommended(ctx context.Context, userID pgtype.UUID, pageSize, pageOffset int) ([]event_model.Event, error)
	GetPopular(pageSize int, pageNum int) ([]event_model.Event, error)
	Search(ctx context.Context, query string, pageNum, pageSize int) (event_model.SearchResult, error)
	GetCategories() ([]event_model.Category, error)
	GetCategory(id int) (event_model.Category, error)
	ListMapEvents(ctx context.Context, filter event_model.MapFilter) ([]event_model.MapEvent, error)
	EventExists(id int) (bool, error)
	// GetUnofficialState returns the state of an unofficial event (event_model.UnofficialState*).
	// It returns ErrEventNotFound if the event does not exist or is official.
	GetUnofficialState(ctx context.Context, id int) (string, error)
	GetReceivedRatings(id int) ([]event_model.ReceivedRating, error)
	// Cancel marks an Active unofficial event as Cancelled and notifies the users signed up to it, except its creator,
	// in a single transaction. It returns the number of notified users, ErrEventNotFound if the event does not exist
	// or is official, and ErrEventNotActive if it is no longer Active.
	Cancel(ctx context.Context, id int) (int64, error)
}
