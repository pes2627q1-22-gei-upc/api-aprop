package auth_repository

import "errors"

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists for this firebase account")
	ErrUsernameTaken     = errors.New("username already taken")
)
