package preference_service

import (
	"errors"

	"api-aprop/internal/models/user_model"
)

var (
	ErrInvalidUserID  = user_model.ErrInvalidID
	ErrEmptyAnswers   = errors.New("answers cannot be empty")
	ErrTooManyAnswers = errors.New("too many answers")
	ErrInvalidEventID = errors.New("invalid event id")
)
