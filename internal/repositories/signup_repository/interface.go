package signup_repository

import (
	"context"

	"api-aprop/internal/models/event_model"
	"api-aprop/internal/models/signup_model"
)

// Repository stores sign ups in Postgres.
type Repository interface {
	EventExists(ctx context.Context, eventID int) (bool, error)
	Create(ctx context.Context, userID string, eventID int) (bool, error)
	Delete(ctx context.Context, userID string, eventID int) (bool, error)
	ListUserEvents(ctx context.Context, userID string, dateRange signup_model.DateRange) ([]event_model.Event, error)
	CountSignUps(ctx context.Context, eventID int) (int, error)
	IsSignedUp(ctx context.Context, userID string, eventID int) (bool, error)
}
