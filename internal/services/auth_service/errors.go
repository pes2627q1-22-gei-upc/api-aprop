package auth_service

import "errors"

var (
	ErrInvalidUsername = errors.New("invalid username")
	ErrUsernameTaken   = errors.New("username already taken")
	// ErrUserNotRegistered means the Firebase account has no database user yet (POST /auth/sync was not called).
	ErrUserNotRegistered = errors.New("user not registered")
)
