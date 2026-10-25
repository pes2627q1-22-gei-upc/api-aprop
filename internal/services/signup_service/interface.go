package signup_service

import (
	"context"

	"api-aprop/internal/models/event_model"
	"api-aprop/internal/models/signup_model"
)

// Service contains the business logic for signing users up to events.
type Service interface {
	SignUp(ctx context.Context, userID string, eventID int) (signup_model.SignUp, error)
	Unenroll(ctx context.Context, userID string, eventID int) (signup_model.Unenroll, error)
	GetEnrollmentStatus(ctx context.Context, userID string, eventID int) (signup_model.EnrollmentStatus, error)
	ListUserEvents(ctx context.Context, userID string, dateRange signup_model.DateRange) ([]event_model.Event, error)
}
