package password_reset_service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"api-aprop/internal/clients/firebase_client"
	"api-aprop/internal/repositories/user_repository"
)

// TODO(DP-033): window, locale and anti-enumeration behaviour proposed by the backend, pending the frontend.
const (
	// Window is the minimum time between two reset emails for the same identifier or email.
	Window = time.Minute
	// Locale is the language of the email that Firebase sends.
	Locale = "ca"

	maxIdentifierLength = 254
	// pruneThreshold is the number of remembered requests from which old ones are removed.
	pruneThreshold = 1024
)

var emailPattern = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

type PasswordResetService struct {
	users  UserFinder
	sender firebase_client.PasswordResetSender
	now    func() time.Time

	mu       sync.Mutex
	requests map[string]time.Time
}

var _ Service = (*PasswordResetService)(nil)

// NewPasswordResetService creates the service. A nil sender (no Web API key) makes every request fail
// with ErrNotConfigured.
func NewPasswordResetService(users UserFinder, sender firebase_client.PasswordResetSender, now func() time.Time) *PasswordResetService {
	if now == nil {
		now = time.Now
	}
	return &PasswordResetService{users: users, sender: sender, now: now, requests: map[string]time.Time{}}
}

// RequestReset accepts an email or a username. Unknown accounts are not an error, and the limit of one
// request per Window is checked before looking for the account, so it behaves the same whether the
// account exists or not.
func (s *PasswordResetService) RequestReset(ctx context.Context, identifier string) error {
	identifier = strings.ToLower(strings.TrimSpace(identifier))
	if identifier == "" || utf8.RuneCountInString(identifier) > maxIdentifierLength {
		return ErrInvalidIdentifier
	}
	isEmail := strings.Contains(identifier, "@")
	if isEmail && !emailPattern.MatchString(identifier) {
		return ErrInvalidIdentifier
	}
	if s.sender == nil {
		return ErrNotConfigured
	}

	key := "username:" + identifier
	if isEmail {
		key = "email:" + identifier
	}
	if retryAfter, ok := s.allow(key); !ok {
		return &RateLimitedError{RetryAfter: retryAfter}
	}

	email := identifier
	if !isEmail {
		user, err := s.users.GetByUsername(ctx, identifier)
		if errors.Is(err, user_repository.ErrUserNotFound) || (err == nil && user.Email == "") {
			slog.Debug("password reset for an unknown username")
			return nil
		}
		if err != nil {
			s.forget(key)
			return fmt.Errorf("find user to reset the password: %w", err)
		}
		email = strings.ToLower(user.Email)
		// The email may have been requested directly: the limit is not reported, because a 429 here
		// and a 204 for an unknown username would tell that the username exists
		if _, ok := s.allow("email:" + email); !ok {
			return nil
		}
	}

	err := s.sender.SendPasswordResetEmail(ctx, email, Locale)
	switch {
	case err == nil, errors.Is(err, firebase_client.ErrEmailNotFound):
		return nil
	case !isEmail && (errors.Is(err, firebase_client.ErrInvalidEmail) || errors.Is(err, firebase_client.ErrTooManyAttempts)):
		// An unknown username never reaches Firebase: reporting these would tell that the username exists
		slog.Warn("password reset email of a username not sent", "error", err)
		return nil
	case errors.Is(err, firebase_client.ErrInvalidEmail):
		return ErrInvalidIdentifier
	case errors.Is(err, firebase_client.ErrTooManyAttempts):
		return &RateLimitedError{RetryAfter: Window}
	default:
		// The email was not sent: the user can try again without waiting
		s.forget(key)
		if !isEmail {
			s.forget("email:" + email)
		}
		return fmt.Errorf("%w: %w", ErrProvider, err)
	}
}

// allow records a request for key and reports whether the previous one was at least Window ago. If
// not, it returns the time left.
func (s *PasswordResetService) allow(key string) (time.Duration, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now()
	if last, ok := s.requests[key]; ok {
		if elapsed := now.Sub(last); elapsed < Window {
			return Window - elapsed, false
		}
	}
	if len(s.requests) >= pruneThreshold {
		for k, t := range s.requests {
			if now.Sub(t) >= Window {
				delete(s.requests, k)
			}
		}
	}
	s.requests[key] = now
	return 0, true
}

func (s *PasswordResetService) forget(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.requests, key)
}
