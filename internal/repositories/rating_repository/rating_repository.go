package rating_repository

import (
	"context"
	"errors"
)

var ErrEventNotFound = errors.New("event not found")

// Repository stores event ratings (table EventRatings).
type Repository interface {
	// Create stores the rating; false when the user had already rated the event.
	Create(ctx context.Context, userID string, eventID, score int) (bool, error)
	// HasAttended reports whether the user's sign up to the event has attendance = true.
	HasAttended(ctx context.Context, userID string, eventID int) (bool, error)
}
