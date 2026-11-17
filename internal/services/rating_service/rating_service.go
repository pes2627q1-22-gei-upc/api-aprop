package rating_service

import (
	"context"
	"errors"
	"fmt"

	"api-aprop/internal/models/rating_model"
	"api-aprop/internal/models/user_model"
	"api-aprop/internal/repositories/rating_repository"
)

var (
	ErrInvalidUserID  = user_model.ErrInvalidID
	ErrInvalidEventID = errors.New("invalid event id")
	ErrInvalidScore   = errors.New("score must be between 1 and 5") // R9
	ErrEventNotFound  = errors.New("event not found")
	ErrNotAttended    = errors.New("user has not attended the event") // R10
	ErrAlreadyRated   = errors.New("user has already rated the event")
)

// Service contains the business logic for rating events.
type Service interface {
	Rate(ctx context.Context, userID string, eventID, score int) (rating_model.Rating, error)
}

// EventChecker is the part of event_repository.Repository the service needs.
type EventChecker interface {
	EventExists(id int) (bool, error)
}

type RatingService struct {
	ratings rating_repository.Repository
	events  EventChecker
}

func NewRatingService(ratings rating_repository.Repository, events EventChecker) *RatingService {
	return &RatingService{ratings: ratings, events: events}
}

// Rate stores the user's rating of the event.
// R9: score in [1, 5]. R10: the user must have attended (SignUp.Attendance = true, which R8 only allows once the event started).
func (s *RatingService) Rate(ctx context.Context, userID string, eventID, score int) (rating_model.Rating, error) {
	userID, err := user_model.NormalizeID(userID)
	if err != nil {
		return rating_model.Rating{}, err
	}
	if eventID <= 0 {
		return rating_model.Rating{}, fmt.Errorf("%w: %d", ErrInvalidEventID, eventID)
	}
	if score < 1 || score > 5 {
		return rating_model.Rating{}, fmt.Errorf("%w: %d", ErrInvalidScore, score)
	}
	exists, err := s.events.EventExists(eventID)
	if err != nil {
		return rating_model.Rating{}, err
	}
	if !exists {
		return rating_model.Rating{}, fmt.Errorf("rate event %d: %w", eventID, ErrEventNotFound)
	}
	attended, err := s.ratings.HasAttended(ctx, userID, eventID)
	if err != nil {
		return rating_model.Rating{}, err
	}
	if !attended {
		return rating_model.Rating{}, fmt.Errorf("rate event %d: %w", eventID, ErrNotAttended)
	}
	created, err := s.ratings.Create(ctx, userID, eventID, score)
	if errors.Is(err, rating_repository.ErrEventNotFound) {
		return rating_model.Rating{}, fmt.Errorf("%w: %w", ErrEventNotFound, err)
	}
	if err != nil {
		return rating_model.Rating{}, err
	}
	if !created {
		return rating_model.Rating{}, fmt.Errorf("rate event %d: %w", eventID, ErrAlreadyRated)
	}
	return rating_model.Rating{UserID: userID, EventID: eventID, Score: score}, nil
}
