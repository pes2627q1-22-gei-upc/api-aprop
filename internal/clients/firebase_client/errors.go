package firebase_client

import "errors"

var (
	ErrInvalidToken = errors.New("invalid or expired id token")
	ErrTokenRevoked = errors.New("id token has been revoked")
	ErrUserNotFound = errors.New("user does not exist")
	ErrUserDisabled = errors.New("user is disabled")

	// Errors of the password reset email
	ErrEmailNotFound   = errors.New("no account with this email")
	ErrInvalidEmail    = errors.New("invalid email")
	ErrTooManyAttempts = errors.New("too many password reset attempts")
)
