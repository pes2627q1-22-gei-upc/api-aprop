package user_service

import "errors"

var (
	ErrUserNotFound  		= errors.New("user not found")
	ErrUnauthorized  		= errors.New("unauthorized")
	ErrForbidden     		= errors.New("forbidden")
	ErrUsernameTaken 		= errors.New("username already taken")
	ErrInvalidUserID 		= errors.New("invalid user id")
	ErrInvalidInput  		= errors.New("invalid input")
	ErrRequestAlreadyExists = errors.New("pending friend request exists between these users")
	ErrIsBlocked 			= errors.New("requester has blocked or been blocked by target user")
	ErrAlreadyFriends 		= errors.New("requester is already friends with target user")
	ErrRequestNotFound 		= errors.New("request not found")
	ErrNotFriends			= errors.New("requester and target are not friends")
)
