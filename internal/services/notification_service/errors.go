package notification_service

import "errors"

var (
	ErrNotificationNotFound = errors.New("notification not found")
	ErrUserNotFound         = errors.New("user not found")
)
