package event_repository

import "errors"

var (
	ErrEventNotFound    = errors.New("event not found")
	ErrCategoryNotFound = errors.New("category not found")
	// ErrInvalidFilter is returned when a list parameter (filter or page) cannot be used.
	ErrInvalidFilter = errors.New("invalid filter")
)
