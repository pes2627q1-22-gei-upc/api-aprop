package preference_service

import "errors"

var (
	ErrInvalidUserID  = errors.New("invalid user id")
	ErrEmptyAnswers   = errors.New("answers cannot be empty")
	ErrTooManyAnswers = errors.New("too many answers")
	ErrInvalidEventID = errors.New("invalid event id")
)
