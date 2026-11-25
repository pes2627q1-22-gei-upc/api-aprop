package event_repository

import "errors"

var (
	ErrEventNotFound    	= errors.New("event not found")
	ErrCategoryNotFound 	= errors.New("category not found")
	// ErrInvalidFilter is returned when a list parameter (filter or page) cannot be used.
	ErrInvalidFilter 		= errors.New("invalid filter")
	// ErrEventNotActive is returned when an unofficial event is no longer Active (cancelled or withdrawn).
	ErrEventNotActive 		= errors.New("event is not active")
	ErrCategoryNotAllowed 	= errors.New("you are not allowed to add this category")
)
