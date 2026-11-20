package message_repository

import (
	"context"
	"errors"

	"api-aprop/internal/models/message_model"
)

var (
	ErrEventNotFound = errors.New("event not found")
	// ErrDuplicateMessage is returned when the user already sent a message to the event at the same instant (R1 key).
	ErrDuplicateMessage = errors.New("duplicate message")
)

// Repository stores the chat messages of the events (table EventMessages).
type Repository interface {
	// Create stores the message and returns it with its author and the date it was sent.
	Create(ctx context.Context, userID string, eventID int, text string) (message_model.Message, error)
	// List returns a page of the event messages, newest first.
	List(ctx context.Context, eventID, limit, offset int) ([]message_model.Message, error)
	// Count returns the number of messages of the event.
	Count(ctx context.Context, eventID int) (int64, error)
	// IsSignedUp reports whether the user has a sign up to the event (R11).
	IsSignedUp(ctx context.Context, userID string, eventID int) (bool, error)
}
