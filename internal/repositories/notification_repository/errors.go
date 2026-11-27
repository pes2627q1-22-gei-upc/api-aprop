package notification_repository

import "errors"

var (
	ErrNotificationNotFound = errors.New("notification not found")
	ErrUserNotFound         = errors.New("user not found")
)
