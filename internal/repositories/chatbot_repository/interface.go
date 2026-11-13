package chatbot_repository

import (
	"context"
	"time"

	"api-aprop/internal/models/chatbot_model"
)

// Repository stores the chatbot conversations of the users.
type Repository interface {
	// LatestConversation returns the start of the most recent conversation of the user,
	// or ErrConversationNotFound if the user has none.
	LatestConversation(ctx context.Context, userID string) (time.Time, error)
	// SaveMessages stores the messages and their recommended events in a single transaction,
	// creating the conversation if it does not exist yet.
	SaveMessages(ctx context.Context, userID string, conversationStartedAt time.Time, messages ...chatbot_model.Message) error
	// ListMessages returns the messages of the conversation ordered by date, with their recommended events.
	ListMessages(ctx context.Context, userID string, conversationStartedAt time.Time) ([]chatbot_model.Message, error)
	// SaveFeedback stores (or replaces) the rating of an assistant message,
	// or returns ErrMessageNotFound if the user has no assistant message with that key.
	SaveFeedback(ctx context.Context, userID string, conversationStartedAt, sentAt time.Time, helpful bool) error
}
