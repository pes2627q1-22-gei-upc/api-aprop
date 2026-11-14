package signup_repository

import (
	"context"

	"api-aprop/internal/models/event_model"
	"api-aprop/internal/models/signup_model"
)

// Repository stores sign ups in Postgres.
type Repository interface {
	Create(ctx context.Context, userID string, eventID int) (bool, error)
	Delete(ctx context.Context, userID string, eventID int) (bool, error)
	ListUserEvents(ctx context.Context, userID string, dateRange signup_model.DateRange) ([]event_model.Event, error)
	CountSignUps(ctx context.Context, eventID int) (int, error)
	// SetAttendance stores the attendance (nil = unknown); false when the user is not signed up.
	SetAttendance(ctx context.Context, userID string, eventID int, attendance *bool) (bool, error)
	IsSignedUp(ctx context.Context, userID string, eventID int) (bool, error)
}
