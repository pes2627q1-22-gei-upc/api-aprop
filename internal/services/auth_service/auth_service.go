package auth_service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"api-aprop/internal/models/user_model"
	"api-aprop/internal/repositories/auth_repository"
)

const (
	minUsernameLength = 3
	maxUsernameLength = 30
)

type AuthService struct {
	authRepository auth_repository.Repository
}

func NewAuthService(authRepository auth_repository.Repository) *AuthService {
	return &AuthService{authRepository: authRepository}
}

// Sync returns the database user of the Firebase account, creating it the first time the user signs in.
// The username is only needed (and used) when the user is created. The email always follows Firebase.
func (s *AuthService) Sync(ctx context.Context, firebaseUID, email, username string) (user_model.SyncResult, error) {
	user, err := s.authRepository.GetByFirebaseUID(ctx, firebaseUID)
	if err == nil {
		user, err = s.syncEmail(ctx, user, email)
		if err != nil {
			return user_model.SyncResult{}, err
		}
		return user_model.SyncResult{User: user, Created: false}, nil
	}
	if !errors.Is(err, auth_repository.ErrUserNotFound) {
		return user_model.SyncResult{}, err
	}

	username, err = normalizeUsername(username)
	if err != nil {
		return user_model.SyncResult{}, err
	}

	user, err = s.authRepository.Create(ctx, firebaseUID, username, email)
	switch {
	case errors.Is(err, auth_repository.ErrUsernameTaken):
		return user_model.SyncResult{}, fmt.Errorf("%w: %q", ErrUsernameTaken, username)
	case errors.Is(err, auth_repository.ErrUserAlreadyExists):
		// Another request created the user at the same time (e.g. two tabs signing in): use that one
		user, err = s.authRepository.GetByFirebaseUID(ctx, firebaseUID)
		if err != nil {
			return user_model.SyncResult{}, err
		}
		return user_model.SyncResult{User: user, Created: false}, nil
	case err != nil:
		return user_model.SyncResult{}, err
	}
	return user_model.SyncResult{User: user, Created: true}, nil
}

// ResolveUserID returns the database ID (UUID) of the requester. The identifier is the Firebase UID set by
// the auth middleware, or directly a UUID when the API runs without authentication (local development and tests).
func (s *AuthService) ResolveUserID(ctx context.Context, identifier string) (string, error) {
	if identifier == "" {
		return "", ErrUserNotRegistered
	}
	if uid, err := user_model.NormalizeID(identifier); err == nil {
		return uid, nil
	}
	user, err := s.authRepository.GetByFirebaseUID(ctx, identifier)
	if errors.Is(err, auth_repository.ErrUserNotFound) {
		return "", fmt.Errorf("%w: %w", ErrUserNotRegistered, err)
	}
	if err != nil {
		return "", fmt.Errorf("resolve user: %w", err)
	}
	return user.ID.String(), nil
}

// syncEmail stores the Firebase email when the user changed it there.
func (s *AuthService) syncEmail(ctx context.Context, user user_model.User, email string) (user_model.User, error) {
	if email == "" || email == user.Email {
		return user, nil
	}
	return s.authRepository.UpdateEmail(ctx, user.ID, email)
}

// normalizeUsername trims the username and checks its length.
func normalizeUsername(username string) (string, error) {
	username = strings.TrimSpace(username)
	length := utf8.RuneCountInString(username)
	if length < minUsernameLength || length > maxUsernameLength {
		return "", fmt.Errorf("%w: must have between %d and %d characters", ErrInvalidUsername, minUsernameLength, maxUsernameLength)
	}
	return username, nil
}
