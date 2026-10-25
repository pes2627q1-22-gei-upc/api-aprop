package event_service

import "errors"

var (
	ErrEventNotFound           = errors.New("event not found")
	ErrInvalidEvent            = errors.New("invalid event")
	ErrInvalidEventTitle       = errors.New("event title is required")
	ErrInvalidEventDescription = errors.New("event description is required")
	ErrInvalidEventCategory    = errors.New("event requires at least one category")
	ErrInvalidEventDates       = errors.New("invalid event dates: start date must be in the future and end date after start date")
	ErrInvalidCoordinates      = errors.New("coordinates are out of valid range")
	ErrInvalidBoundingBox      = errors.New("invalid bounding box")
	ErrInvalidUserID           = errors.New("invalid user id")
	ErrInvalidContentType      = errors.New("invalid content type, must be image/jpeg, image/png, or image/webp")
	ErrImageUploadFailed       = errors.New("failed to upload event image")
)
