package signup_service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"api-aprop/internal/models/event_model"
	"api-aprop/internal/models/signup_model"
	"api-aprop/internal/models/user_model"
	"api-aprop/internal/repositories/event_repository"
	"api-aprop/internal/repositories/signup_repository"
)

type SignUpService struct {
	signUpRepository signup_repository.Repository
	eventRepository  event_repository.Repository
}

func NewSignUpService(signUpRepository signup_repository.Repository, eventRepository event_repository.Repository) *SignUpService {
	return &SignUpService{signUpRepository: signUpRepository, eventRepository: eventRepository}
}

// SignUp signs the user up to the event. Signing up twice is not an error: Created is false.
func (s *SignUpService) SignUp(ctx context.Context, userID string, eventID int) (signup_model.SignUp, error) {
	userID, err := user_model.NormalizeID(userID)
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
	}
	return signup_model.SignUp{UserID: userID, EventID: eventID, Created: created}, nil
}

// Unenroll removes the user's sign up from the event.
func (s *SignUpService) Unenroll(ctx context.Context, userID string, eventID int) (signup_model.Unenroll, error) {
	userID, err := user_model.NormalizeID(userID)
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
		normalizedUID, normErr := user_model.NormalizeID(userID)
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
	userID, err := user_model.NormalizeID(userID)
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
