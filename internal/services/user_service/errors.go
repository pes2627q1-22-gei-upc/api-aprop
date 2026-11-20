package user_service

import "errors"

var (
	ErrUserNotFound         = errors.New("user not found")
	ErrUnauthorized         = errors.New("unauthorized")
	ErrForbidden            = errors.New("forbidden")
	ErrUsernameTaken        = errors.New("username already taken")
	ErrInvalidUserID        = errors.New("invalid user id")
	ErrInvalidInput         = errors.New("invalid input")
	ErrRequestAlreadyExists = errors.New("pending friend request exists between these users")
	ErrIsBlocked            = errors.New("requester has blocked or been blocked by target user")
	ErrAlreadyFriends       = errors.New("requester is already friends with target user")
	ErrRequestNotFound      = errors.New("request not found")
	ErrNotFriends           = errors.New("requester and target are not friends")
	ErrSameUser             = errors.New("requester and target cannot be the same")
	ErrAlreadyBlocked       = errors.New("requester has already blocked the target user")
	ErrNotBlocked           = errors.New("requester has not blocked the target user")
	// ErrAuthProvider means Firebase Auth could not delete the account, so nothing was deleted.
	ErrAuthProvider = errors.New("could not delete the authentication account")
	// ErrStorageUnavailable means the avatar storage is not configured.
	ErrStorageUnavailable = errors.New("avatar storage is not available")
	// ErrStorageFailed means the avatar storage answered with an error.
	ErrStorageFailed = errors.New("could not store the profile picture")
)
