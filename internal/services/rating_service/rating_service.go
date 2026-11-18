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
	ErrRatingNotFound = errors.New("rating not found")
)

// Service contains the business logic for rating events.
type Service interface {
	Rate(ctx context.Context, userID string, eventID, score int) (rating_model.Rating, error)
	HasAttended(ctx context.Context, userID string, eventID int) (rating_model.AttendanceStatus, error)
	UpdateRating(ctx context.Context, userID string, eventID, score int) (rating_model.Rating, error)
	DeleteRating(ctx context.Context, userID string, eventID int) error
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

// HasAttended reports whether the user has confirmed attendance (SignUp.Attendance = true) at the event.
func (s *RatingService) HasAttended(ctx context.Context, userID string, eventID int) (rating_model.AttendanceStatus, error) {
	userID, err := user_model.NormalizeID(userID)
	if err != nil {
		return rating_model.AttendanceStatus{}, err
	}
	if eventID <= 0 {
		return rating_model.AttendanceStatus{}, fmt.Errorf("%w: %d", ErrInvalidEventID, eventID)
	}
	exists, err := s.events.EventExists(eventID)
	if err != nil {
		return rating_model.AttendanceStatus{}, err
	}
	if !exists {
		return rating_model.AttendanceStatus{}, fmt.Errorf("check attendance for event %d: %w", eventID, ErrEventNotFound)
	}
	attended, err := s.ratings.HasAttended(ctx, userID, eventID)
	if err != nil {
		return rating_model.AttendanceStatus{}, err
	}
	return rating_model.AttendanceStatus{UserID: userID, EventID: eventID, HasAttended: attended}, nil
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

// UpdateRating changes the score of the user's own rating (R9: 1..5). The organizer's Users.Rating is refreshed by the
// EventRatings trigger (migration 000017), so nothing else needs recalculating.
// TODO(DP-036): R10 is not re-checked on edit/delete, nor whether the event already ended; RatingDate is kept.
func (s *RatingService) UpdateRating(ctx context.Context, userID string, eventID, score int) (rating_model.Rating, error) {
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
	if err := s.requireEvent(eventID); err != nil {
		return rating_model.Rating{}, err
	}
	updated, err := s.ratings.Update(ctx, userID, eventID, score)
	if err != nil {
		return rating_model.Rating{}, err
	}
	if !updated {
		return rating_model.Rating{}, fmt.Errorf("update rating of event %d: %w", eventID, ErrRatingNotFound)
	}
	return rating_model.Rating{UserID: userID, EventID: eventID, Score: score}, nil
}

// DeleteRating removes the user's own rating of the event.
func (s *RatingService) DeleteRating(ctx context.Context, userID string, eventID int) error {
	userID, err := user_model.NormalizeID(userID)
	if err != nil {
		return err
	}
	if eventID <= 0 {
		return fmt.Errorf("%w: %d", ErrInvalidEventID, eventID)
	}
	if err := s.requireEvent(eventID); err != nil {
		return err
	}
	deleted, err := s.ratings.Delete(ctx, userID, eventID)
	if err != nil {
		return err
	}
	if !deleted {
		return fmt.Errorf("delete rating of event %d: %w", eventID, ErrRatingNotFound)
	}
	return nil
}

func (s *RatingService) requireEvent(eventID int) error {
	exists, err := s.events.EventExists(eventID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("event %d: %w", eventID, ErrEventNotFound)
	}
	return nil
}
