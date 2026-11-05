package user_service

import "errors"

var (
	ErrUserNotFound  = errors.New("user not found")
	ErrUnauthorized  = errors.New("unauthorized")
	ErrForbidden     = errors.New("forbidden")
	ErrUsernameTaken = errors.New("username already taken")
	ErrInvalidUserID = errors.New("invalid user id")
	ErrInvalidInput  = errors.New("invalid input")
	ErrRequestAlreadyExists = errors.New("pending friend request exists between these users")
	ErrIsBlocked = errors.New("requester is blocked by target user")
)
