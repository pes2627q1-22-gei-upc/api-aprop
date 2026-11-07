package signup_service

import "errors"

var (
	ErrEventNotFound    = errors.New("event not found")
	ErrInvalidUserID    = errors.New("invalid user id")
	ErrInvalidEventID   = errors.New("invalid event id")
	ErrInvalidDateRange = errors.New("invalid date range")
	ErrEventExpired     = errors.New("event has alredy ended")
)
