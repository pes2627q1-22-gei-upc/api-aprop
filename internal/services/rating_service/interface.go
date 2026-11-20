package rating_service

import (
	"api-aprop/internal/models/rating_model"
	"context"
)

// Service contains the business logic for rating events.
type Service interface {
	Rate(ctx context.Context, userID string, eventID, score int) (rating_model.Rating, error)
	HasAttended(ctx context.Context, userID string, eventID int) (rating_model.AttendanceStatus, error)
	UpdateRating(ctx context.Context, userID string, eventID, score int) (rating_model.Rating, error)
	DeleteRating(ctx context.Context, userID string, eventID int) error
	RateUser(ctx context.Context, userID string, eventID int, score int) (rating_model.UserRating, error)
	UpdateUserRating(ctx context.Context, userID string, eventID int, score int) (rating_model.UserRating, error)
	DeleteUserRating(ctx context.Context, userID string, eventID int) error
}
