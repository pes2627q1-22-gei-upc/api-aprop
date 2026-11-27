package chatbot_repository

import "errors"

var (
	ErrConversationNotFound = errors.New("conversation not found")
	ErrMessageNotFound      = errors.New("assistant message not found")
)
