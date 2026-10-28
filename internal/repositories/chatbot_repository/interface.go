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
}
