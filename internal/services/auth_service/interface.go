package auth_service

import (
	"context"

	"api-aprop/internal/models/user_model"
)

// Service keeps the database users in sync with the Firebase Auth accounts.
type Service interface {
	Sync(ctx context.Context, firebaseUID, email, username string) (user_model.SyncResult, error)
}
