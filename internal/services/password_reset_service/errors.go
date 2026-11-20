package password_reset_service

import (
	"errors"
	"fmt"
	"time"
)

var (
	// ErrInvalidIdentifier means the identifier is empty, too long or a malformed email.
	ErrInvalidIdentifier = errors.New("invalid identifier")
	// ErrRateLimited means a reset was requested too recently. Use errors.As with *RateLimitedError to
	// get the time to wait.
	ErrRateLimited = errors.New("too many password reset requests")
	// ErrNotConfigured means the server has no Firebase Web API key, so it cannot send emails.
	ErrNotConfigured = errors.New("password reset is not configured")
	// ErrProvider means Firebase failed to send the email.
	ErrProvider = errors.New("password reset provider failed")
)

// RateLimitedError is an ErrRateLimited with the time to wait before trying again.
type RateLimitedError struct {
	RetryAfter time.Duration
}

func (e *RateLimitedError) Error() string {
	return fmt.Sprintf("%v: retry after %s", ErrRateLimited, e.RetryAfter)
}

func (e *RateLimitedError) Is(target error) bool {
	return target == ErrRateLimited
}
