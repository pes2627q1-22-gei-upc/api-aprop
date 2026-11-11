package event_service

import (
	"errors"

	"api-aprop/internal/models/user_model"
)

var (
	ErrEventNotFound           = errors.New("event not found")
	ErrCategoryNotFound        = errors.New("category not found")
	ErrInvalidFilter           = errors.New("invalid filter or page parameters")
	ErrInvalidEvent            = errors.New("invalid event")
	ErrInvalidEventTitle       = errors.New("event title is required")
	ErrInvalidEventDescription = errors.New("event description is required")
	ErrInvalidEventCategory    = errors.New("event requires at least one category")
	ErrInvalidEventDates       = errors.New("invalid event dates: start date must be in the future and end date after start date")
	ErrInvalidCoordinates      = errors.New("coordinates are out of valid range")
	ErrInvalidBoundingBox      = errors.New("invalid bounding box")
	ErrInvalidUserID           = user_model.ErrInvalidID
	ErrUserNotRegistered       = errors.New("user not registered")
	ErrInvalidContentType      = errors.New("invalid content type, must be image/jpeg, image/png, or image/webp")
	ErrImageUploadFailed       = errors.New("failed to upload event image")
)
