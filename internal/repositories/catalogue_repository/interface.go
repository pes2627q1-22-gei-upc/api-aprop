package catalogue_repository

import (
	"context"
	"time"

	"api-aprop/internal/models/event_model"
)

// SearchFilter selects catalogue events. Empty strings and nil dates are not applied.
type SearchFilter struct {
	Category     string
	Municipality string
	From         *time.Time
	To           *time.Time
	Limit        int
}

// Repository reads the events catalogue for the chatbot.
type Repository interface {
	// Search returns the events matching the filter that overlap the date range,
	// ordered by start date and id.
	Search(ctx context.Context, filter SearchFilter) ([]event_model.Event, error)
	// ListByIDs returns the existing events in the order of ids. Missing ids are skipped.
	ListByIDs(ctx context.Context, ids []int) ([]event_model.Event, error)
}
