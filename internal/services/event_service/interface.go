package event_service

import (
	"context"
	"io"

	"api-aprop/internal/models/event_model"
)

// Service defines the business logic operations on events.
type Service interface {
	GetEvent(id int) (event_model.Event, error)
	ListEvents(params map[string]any) ([]event_model.Event, error)
	SearchEvents(ctx context.Context, query string, pageNum, pageSize int) (event_model.SearchResult, error)
	ListMapEvents(ctx context.Context, filter event_model.MapFilter) ([]event_model.MapEvent, error)

	// ResolveUserID returns the database ID (UUID) of the user identified by a Firebase UID or a UUID.
	ResolveUserID(ctx context.Context, identifier string) (string, error)
	CreateUnofficialEvent(ctx context.Context, userID string, e event_model.Event) (event_model.Event, error)
	UploadEventImage(ctx context.Context, eventID int, filename string, contentType string, reader io.Reader) (string, error)
	LoadOfficialEvents(ctx context.Context, pageSize int) error
	GetCategories() ([]event_model.Category, error)
	GetCategory(id int) (event_model.Category, error)
}
