package congestion_service

import "errors"

var (
	ErrNoLocations      = errors.New("at least one location is required")
	ErrTooManyLocations = errors.New("too many locations")
	ErrInvalidLocation  = errors.New("invalid location")
	ErrInvalidTimeframe = errors.New("invalid timeframe")
)
