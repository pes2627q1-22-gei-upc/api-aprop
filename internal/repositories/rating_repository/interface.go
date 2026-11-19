package rating_repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgtype"
)

var ErrEventNotFound = errors.New("event not found")

// Repository stores event ratings (table EventRatings).
type Repository interface {
	// Create stores the rating; false when the user had already rated the event.
	Create(ctx context.Context, userID string, eventID, score int) (bool, error)
	// Update changes the score of the user's rating; false when the user had not rated the event.
	Update(ctx context.Context, userID string, eventID, score int) (bool, error)
	// Delete removes the user's rating; false when the user had not rated the event.
	Delete(ctx context.Context, userID string, eventID int) (bool, error)
	CreateUserRating(ctx context.Context, userID pgtype.UUID, targetID pgtype.UUID, eventID int, score int) (bool, error)
	// HasAttended reports whether the user's sign up to the event has attendance = true.
	HasAttended(ctx context.Context, userID string, eventID int) (bool, error)
}
