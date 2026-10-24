package event_service

import "errors"

var (
	ErrEventNotFound      = errors.New("event not found")
	ErrInvalidEvent       = errors.New("invalid event")
	ErrInvalidBoundingBox = errors.New("invalid bounding box")
	ErrInvalidUserID      = errors.New("invalid user id")
)
