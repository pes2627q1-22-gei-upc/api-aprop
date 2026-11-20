package rating_service

import (
	"api-aprop/internal/models/user_model"
	"errors"
)

var (
	ErrInvalidUserID    = user_model.ErrInvalidID
	ErrInvalidEventID   = errors.New("invalid event id")
	ErrInvalidScore     = errors.New("score must be between 1 and 5") // R9
	ErrEventNotFound    = errors.New("event not found")
	ErrNotAttended      = errors.New("user has not attended the event") // R10
	ErrAlreadyRated     = errors.New("user has already rated the event")
	ErrUserAlreadyRated = errors.New("user has already rated the event's creator")
	ErrRatingNotFound   = errors.New("rating not found")
	ErrCreatorNotFound  = errors.New("creator not found")
	ErrRateOwnEvent     = errors.New("cannot rate own event")
)
