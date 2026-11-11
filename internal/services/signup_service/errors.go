package signup_service

import (
	"errors"

	"api-aprop/internal/models/user_model"
)

var (
	ErrEventNotFound    = errors.New("event not found")
	ErrInvalidUserID    = user_model.ErrInvalidID
	ErrInvalidEventID   = errors.New("invalid event id")
	ErrInvalidDateRange = errors.New("invalid date range")
	ErrEventExpired     = errors.New("event has alredy ended")
)
