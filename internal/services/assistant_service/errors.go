package assistant_service

import "errors"

var (
	ErrInvalidUserID        = errors.New("invalid user id")
	ErrInvalidMessage       = errors.New("invalid message")
	ErrConversationNotFound = errors.New("conversation not found")
	ErrMessageNotFound      = errors.New("assistant message not found")
	ErrAssistantUnavailable = errors.New("assistant unavailable")
	ErrAssistantTimeout     = errors.New("assistant timeout")
	ErrTooManyMessages      = errors.New("too many messages")
)
