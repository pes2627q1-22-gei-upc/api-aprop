package user_repository

import "errors"

var (
	ErrInvalidInterval         = errors.New("invalid format for time interval")
	ErrInvalidNotificationType = errors.New("notification type does not exist")
)
