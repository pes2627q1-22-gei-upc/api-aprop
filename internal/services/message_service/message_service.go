package message_service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"unicode/utf8"

	"api-aprop/internal/models/message_model"
	"api-aprop/internal/models/user_model"
	"api-aprop/internal/repositories/message_repository"
)

// TODO(DP-040): page sizes and the text limit are proposed by the backend, pending the frontend.
const (
	DefaultPageSize = 50
	MaxPageSize     = 100
)

var (
	ErrInvalidUserID     = user_model.ErrInvalidID
	ErrInvalidEventID    = errors.New("invalid event id")
	ErrInvalidPagination = errors.New("invalid pagination")
	ErrInvalidText       = fmt.Errorf("text must have between 1 and %d characters", message_model.MaxTextLength)
	ErrEventNotFound     = errors.New("event not found")
	ErrNotSignedUp       = errors.New("user is not signed up to the event") // R11
	ErrDuplicateMessage  = errors.New("message already sent at the same instant")
)

// Service contains the business logic of the event chats.
type Service interface {
	ListMessages(ctx context.Context, userID string, eventID, pageNum, pageSize int) (message_model.MessagePage, error)
	SendMessage(ctx context.Context, userID string, eventID int, text string) (message_model.Message, error)
}

// EventChecker is the part of event_repository.Repository the service needs.
type EventChecker interface {
	EventExists(id int) (bool, error)
}

type MessageService struct {
	messages message_repository.Repository
	events   EventChecker
}

func NewMessageService(messages message_repository.Repository, events EventChecker) *MessageService {
	return &MessageService{messages: messages, events: events}
}

// ListMessages returns a page of the event chat, newest first. Only users signed up to the event can read it.
// TODO(DP-040): reading also requires a sign up (R11 only talks about the author of a message).
func (s *MessageService) ListMessages(ctx context.Context, userID string, eventID, pageNum, pageSize int) (message_model.MessagePage, error) {
	userID, err := user_model.NormalizeID(userID)
	if err != nil {
		return message_model.MessagePage{}, err
	}
	if eventID <= 0 {
		return message_model.MessagePage{}, fmt.Errorf("%w: %d", ErrInvalidEventID, eventID)
	}
	if pageSize == 0 {
		pageSize = DefaultPageSize
	}
	if pageNum < 0 || pageSize < 0 || pageSize > MaxPageSize || pageNum > math.MaxInt32/pageSize {
		return message_model.MessagePage{}, fmt.Errorf("%w: pageNum must be at least 0 and pageSize between 1 and %d", ErrInvalidPagination, MaxPageSize)
	}
	if err := s.checkSignUp(ctx, userID, eventID); err != nil {
		return message_model.MessagePage{}, err
	}

	page := message_model.MessagePage{EventID: eventID, PageNum: pageNum, PageSize: pageSize, Messages: []message_model.Message{}}
	page.Total, err = s.messages.Count(ctx, eventID)
	if err != nil {
		return message_model.MessagePage{}, err
	}
	if page.Total == 0 {
		return page, nil
	}
	messages, err := s.messages.List(ctx, eventID, pageSize, pageNum*pageSize)
	if err != nil {
		return message_model.MessagePage{}, err
	}
	if messages != nil {
		page.Messages = messages
	}
	return page, nil
}

// SendMessage stores a message in the event chat. R11: the author must be signed up to the event.
// The text is trimmed and must have between 1 and message_model.MaxTextLength characters.
func (s *MessageService) SendMessage(ctx context.Context, userID string, eventID int, text string) (message_model.Message, error) {
	userID, err := user_model.NormalizeID(userID)
	if err != nil {
		return message_model.Message{}, err
	}
	if eventID <= 0 {
		return message_model.Message{}, fmt.Errorf("%w: %d", ErrInvalidEventID, eventID)
	}
	text = strings.TrimSpace(text)
	if n := utf8.RuneCountInString(text); n == 0 || n > message_model.MaxTextLength {
		return message_model.Message{}, fmt.Errorf("%w: got %d", ErrInvalidText, n)
	}
	if err := s.checkSignUp(ctx, userID, eventID); err != nil {
		return message_model.Message{}, err
	}

	message, err := s.messages.Create(ctx, userID, eventID, text)
	switch {
	case errors.Is(err, message_repository.ErrEventNotFound):
		return message_model.Message{}, fmt.Errorf("%w: %w", ErrEventNotFound, err)
	case errors.Is(err, message_repository.ErrDuplicateMessage):
		return message_model.Message{}, fmt.Errorf("%w: %w", ErrDuplicateMessage, err)
	case err != nil:
		return message_model.Message{}, err
	}
	return message, nil
}

// checkSignUp returns ErrEventNotFound if the event does not exist and ErrNotSignedUp if the user has no sign up (R11).
func (s *MessageService) checkSignUp(ctx context.Context, userID string, eventID int) error {
	exists, err := s.events.EventExists(eventID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("event %d: %w", eventID, ErrEventNotFound)
	}
	signedUp, err := s.messages.IsSignedUp(ctx, userID, eventID)
	if err != nil {
		return err
	}
	if !signedUp {
		return fmt.Errorf("chat of event %d: %w", eventID, ErrNotSignedUp)
	}
	return nil
}
