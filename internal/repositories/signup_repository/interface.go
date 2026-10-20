package signup_repository

import (
	"context"

	"api-aprop/internal/models/event_model"
	"api-aprop/internal/models/signup_model"
)

// Repository stores and retrieves the events users are signed up to.
type Repository interface {
	EventExists(ctx context.Context, eventID int) (bool, error)
	// Create signs the user up to the event and reports whether a new sign up was stored.
	Create(ctx context.Context, userID string, eventID int) (bool, error)
	// ListUserEvents returns the events the user is signed up to that overlap the range,
	// ordered by start date and id.
	ListUserEvents(ctx context.Context, userID string, dateRange signup_model.DateRange) ([]event_model.Event, error)
}
