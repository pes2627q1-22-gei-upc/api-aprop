package password_reset_service

import (
	"context"

	"api-aprop/internal/models/user_model"
)

// Service sends the password reset email of an account identified by its email or its username.
type Service interface {
	// RequestReset returns nil whether the account exists or not, so the caller cannot find out which
	// accounts exist.
	RequestReset(ctx context.Context, identifier string) error
}

// UserFinder finds the user of a username to get its email. postgres.UserRepository implements it.
type UserFinder interface {
	GetByUsername(ctx context.Context, username string) (user_model.User, error)
}
