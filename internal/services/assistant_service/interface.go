package assistant_service

import (
	"context"
	"time"

	"api-aprop/internal/models/chatbot_model"
)

// Service contains the business logic of the chatbot that recommends events.
type Service interface {
	// SendMessage sends a user message to the chatbot and stores it with the reply in the latest conversation.
	SendMessage(ctx context.Context, userID string, text string, byVoice bool) (chatbot_model.Exchange, error)
	// GetConversation returns the latest conversation of the user.
	GetConversation(ctx context.Context, userID string) (chatbot_model.Conversation, error)
	// SendFeedback rates an assistant message as helpful or not; rating it again replaces the rating.
	SendFeedback(ctx context.Context, userID string, conversationStartedAt, sentAt time.Time, helpful bool) error
}
