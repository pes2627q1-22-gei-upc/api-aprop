package signup_service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"api-aprop/internal/models/event_model"
	"api-aprop/internal/models/signup_model"
	"api-aprop/internal/repositories/event_repository"
	"api-aprop/internal/repositories/notification_repository"
	"api-aprop/internal/repositories/signup_repository"
)

type SignUpService struct {
	signUpRepository       signup_repository.Repository
	eventRepository        event_repository.Repository
	notificationRepository notification_repository.Repository
}

func NewSignUpService(signUpRepository signup_repository.Repository, eventRepository event_repository.Repository, notificationRepository notification_repository.Repository) *SignUpService {
	return &SignUpService{signUpRepository: signUpRepository, eventRepository: eventRepository, notificationRepository: notificationRepository}
}

// SignUp signs the user up to the event. Signing up twice is not an error: Created is false.
func (s *SignUpService) SignUp(ctx context.Context, userID string, eventID int) (signup_model.SignUp, error) {
	userID, err := normalizeUserID(userID)
	if err != nil {
		return signup_model.SignUp{}, err
	}
	if eventID <= 0 {
		return signup_model.SignUp{}, fmt.Errorf("%w: %d", ErrInvalidEventID, eventID)
	}

	exists, err := s.eventRepository.EventExists(eventID)
	if err != nil {
		return signup_model.SignUp{}, err
	}
	if !exists {
		return signup_model.SignUp{}, fmt.Errorf("sign up to event %d: %w", eventID, ErrEventNotFound)
	}
	event, err := s.eventRepository.Get(eventID)
	if err != nil {
		return signup_model.SignUp{}, err
	}
	if event.EndDate.Before(time.Now()) {
		return signup_model.SignUp{}, fmt.Errorf("sign up to event %d: %w", eventID, ErrEventExpired)
	}

	created, err := s.signUpRepository.Create(ctx, userID, eventID)
	if errors.Is(err, signup_repository.ErrEventNotFound) {
		return signup_model.SignUp{}, fmt.Errorf("%w: %w", ErrEventNotFound, err)
	}
	if err != nil {
		return signup_model.SignUp{}, err
	} else {
		uuid, err := uuid.FromString(userID)
		if err == nil {
			s.notificationRepository.CreateFriendSignupNotifications(pgtype.UUID{Bytes: [16]byte(uuid), Valid: true}, eventID, "apuntat a")
		}
	}
	return signup_model.SignUp{UserID: userID, EventID: eventID, Created: created}, nil
}

// Unenroll removes the user's sign up from the event.
func (s *SignUpService) Unenroll(ctx context.Context, userID string, eventID int) (signup_model.Unenroll, error) {
	userID, err := normalizeUserID(userID)
	if err != nil {
		return signup_model.Unenroll{}, err
	}
	if eventID <= 0 {
		return signup_model.Unenroll{}, fmt.Errorf("%w: %d", ErrInvalidEventID, eventID)
	}

	exists, err := s.eventRepository.EventExists(eventID)
	if err != nil {
		return signup_model.Unenroll{}, err
	}
	if !exists {
		return signup_model.Unenroll{}, fmt.Errorf("unenroll from event %d: %w", eventID, ErrEventNotFound)
	}

	deleted, err := s.signUpRepository.Delete(ctx, userID, eventID)
	if err != nil {
		return signup_model.Unenroll{}, err
	} else {
		uuid, err := uuid.FromString(userID)
		if err == nil {
			s.notificationRepository.CreateFriendSignupNotifications(pgtype.UUID{Bytes: [16]byte(uuid), Valid: true}, eventID, "desapuntat de")
		}
	}
	return signup_model.Unenroll{UserID: userID, EventID: eventID, Enrolled: false, Deleted: deleted}, nil
}

// GetEnrollmentStatus returns whether the user is enrolled and the total count of sign ups.
func (s *SignUpService) GetEnrollmentStatus(ctx context.Context, userID string, eventID int) (signup_model.EnrollmentStatus, error) {
	if eventID <= 0 {
		return signup_model.EnrollmentStatus{}, fmt.Errorf("%w: %d", ErrInvalidEventID, eventID)
	}

	exists, err := s.eventRepository.EventExists(eventID)
	if err != nil {
		return signup_model.EnrollmentStatus{}, err
	}
	if !exists {
		return signup_model.EnrollmentStatus{}, fmt.Errorf("event %d: %w", eventID, ErrEventNotFound)
	}

	count, err := s.signUpRepository.CountSignUps(ctx, eventID)
	if err != nil {
		return signup_model.EnrollmentStatus{}, err
	}

	var isEnrolled bool
	if userID != "" {
		normalizedUID, normErr := normalizeUserID(userID)
		if normErr == nil {
			isEnrolled, err = s.signUpRepository.IsSignedUp(ctx, normalizedUID, eventID)
			if err != nil {
				return signup_model.EnrollmentStatus{}, err
			}
		}
	}

	return signup_model.EnrollmentStatus{
		EventID:       eventID,
		IsEnrolled:    isEnrolled,
		EnrolledCount: count,
	}, nil
}

func (s *SignUpService) ListUserEvents(ctx context.Context, userID string, dateRange signup_model.DateRange) ([]event_model.Event, error) {
	userID, err := normalizeUserID(userID)
	if err != nil {
		return nil, err
	}
	if dateRange.From != nil && dateRange.To != nil && dateRange.From.After(*dateRange.To) {
		return nil, fmt.Errorf("%w: from is after to", ErrInvalidDateRange)
	}

	events, err := s.signUpRepository.ListUserEvents(ctx, userID, dateRange)
	if err != nil {
		return nil, err
	}
	if events == nil {
		events = []event_model.Event{}
	}
	return events, nil
}

// normalizeUserID accepts only the canonical UUID form (8-4-4-4-12 hex digits) and lowercases it.
func normalizeUserID(userID string) (string, error) {
	if len(userID) != 36 {
		return "", fmt.Errorf("%w: %q", ErrInvalidUserID, userID)
	}
	for i, c := range userID {
		switch {
		case i == 8 || i == 13 || i == 18 || i == 23:
			if c != '-' {
				return "", fmt.Errorf("%w: %q", ErrInvalidUserID, userID)
			}
		case !isHex(c):
			return "", fmt.Errorf("%w: %q", ErrInvalidUserID, userID)
		}
	}
	return strings.ToLower(userID), nil
}

func isHex(c rune) bool {
	return ('0' <= c && c <= '9') || ('a' <= c && c <= 'f') || ('A' <= c && c <= 'F')
}
