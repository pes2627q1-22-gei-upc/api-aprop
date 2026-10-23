package auth_service

import "errors"

var (
	ErrInvalidUsername = errors.New("invalid username")
	ErrUsernameTaken   = errors.New("username already taken")
)
