package auth_repository

import (
	"context"

	"api-aprop/internal/models/user_model"
)

// Repository stores the database users linked to Firebase Auth accounts.
type Repository interface {
	GetByFirebaseUID(ctx context.Context, firebaseUID string) (user_model.User, error)
	// Create stores a new user in the active account state with its own settings.
	Create(ctx context.Context, firebaseUID, username, email string) (user_model.User, error)
	UpdateEmail(ctx context.Context, id int, email string) (user_model.User, error)
}
