package rating_service

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"

	"api-aprop/internal/models/event_model"
	"api-aprop/internal/models/rating_model"
	"api-aprop/internal/models/user_model"
	"api-aprop/internal/repositories/rating_repository"
)

// EventChecker is the part of event_repository.Repository the service needs.
type EventChecker interface {
	EventExists(id int) (bool, error)
	Get(id int) (event_model.Event, error)
}

type UserChecker interface {
	GetByID(ctx context.Context, userID pgtype.UUID) (user_model.User, error)
}

type RatingService struct {
	ratings rating_repository.Repository
	users   UserChecker
	events  EventChecker
}

func NewRatingService(ratings rating_repository.Repository, events EventChecker, users UserChecker) *RatingService {
	return &RatingService{ratings: ratings, events: events, users: users}
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

func (s *RatingService) RateUser(ctx context.Context, userID string, eventID int, score int) (rating_model.UserRating, error) {
	userID, err := user_model.NormalizeID(userID)
	if err != nil {
		return rating_model.UserRating{}, ErrInvalidUserID
	}

	if score < 1 || score > 5 {
		return rating_model.UserRating{}, fmt.Errorf("%w: %d", ErrInvalidScore, score)
	}

	rateruid, err := user_model.ParseID(userID)
	if err != nil {
		return rating_model.UserRating{}, err
	}

	rater, err := s.users.GetByID(ctx, rateruid)
	if err != nil {
		return rating_model.UserRating{}, err
	}

	exists, err := s.events.EventExists(eventID)
	if err != nil {
		return rating_model.UserRating{}, err
	}
	if !exists {
		return rating_model.UserRating{}, fmt.Errorf("rate organizer of event %d: %w", eventID, ErrEventNotFound)
	}
	attended, err := s.ratings.HasAttended(ctx, userID, eventID)
	if err != nil {
		return rating_model.UserRating{}, err
	}
	if !attended {
		return rating_model.UserRating{}, fmt.Errorf("rate organizer of event %d: %w", eventID, ErrNotAttended)
	}

	event, err := s.events.Get(eventID)
	if err != nil {
		return rating_model.UserRating{}, err
	}
	creator := event.Creator
	if !creator.Valid {
		return rating_model.UserRating{}, ErrCreatorNotFound
	}

	created, err := s.ratings.CreateUserRating(ctx, rater.ID, creator, eventID, score)
	if err != nil {
		return rating_model.UserRating{}, err
	}
	if !created {
		return rating_model.UserRating{}, ErrUserAlreadyRated
	}
	return rating_model.UserRating{Rater: userID, Rated: UUIDToString(creator), Event: eventID, Score: score}, nil
}

func (s *RatingService) UpdateUserRating(ctx context.Context, userID string, eventID int, score int) (rating_model.UserRating, error) {
	userID, err := user_model.NormalizeID(userID)
	if err != nil {
		return rating_model.UserRating{}, err
	}
	if eventID <= 0 {
		return rating_model.UserRating{}, fmt.Errorf("%w: %d", ErrInvalidEventID, eventID)
	}
	if score < 1 || score > 5 {
		return rating_model.UserRating{}, fmt.Errorf("%w: %d", ErrInvalidScore, score)
	}
	if err := s.requireEvent(eventID); err != nil {
		return rating_model.UserRating{}, err
	}

	rateruid, err := user_model.ParseID(userID)
	if err != nil {
		return rating_model.UserRating{}, err
	}
	rater, err := s.users.GetByID(ctx, rateruid)
	if err != nil {
		return rating_model.UserRating{}, err
	}

	event, err := s.events.Get(eventID)
	if err != nil {
		return rating_model.UserRating{}, ErrEventNotFound
	}
	creator := event.Creator
	if !creator.Valid {
		return rating_model.UserRating{}, ErrCreatorNotFound
	}

	updated, err := s.ratings.UpdateUserRating(ctx, rater.ID, creator, eventID, score)
	if err != nil {
		return rating_model.UserRating{}, err
	}
	if !updated {
		return rating_model.UserRating{}, fmt.Errorf("update rating of creator of event %d: %w", eventID, ErrRatingNotFound)
	}
	return rating_model.UserRating{Rater: userID, Rated: UUIDToString(creator), Event: eventID, Score: score}, nil
}

func (s *RatingService) DeleteUserRating(ctx context.Context, userID string, eventID int) error {
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

	rateruid, err := user_model.ParseID(userID)
	if err != nil {
		return err
	}
	rater, err := s.users.GetByID(ctx, rateruid)
	if err != nil {
		return err
	}

	event, err := s.events.Get(eventID)
	if err != nil {
		return ErrEventNotFound
	}
	creator := event.Creator
	if !creator.Valid {
		return ErrCreatorNotFound
	}

	updated, err := s.ratings.DeleteUserRating(ctx, rater.ID, creator, eventID)
	if err != nil {
		return err
	}
	if !updated {
		return fmt.Errorf("delete rating of creator of event %d: %w", eventID, ErrRatingNotFound)
	}
	return nil
}

// UpdateRating changes the score of the user's own rating (R9: 1..5). The event's Event.Rating is refreshed by the
// EventRatings trigger (migration 000020), so nothing else needs recalculating.
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

func UUIDToString(id pgtype.UUID) string {
	if !id.Valid {
		return ""
	}

	b := id.Bytes
	return fmt.Sprintf(
		"%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16],
	)
}
