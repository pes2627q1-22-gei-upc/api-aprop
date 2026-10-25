package user_repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgtype"

	"api-aprop/internal/models/event_model"
	"api-aprop/internal/models/user_model"
)

var (
	ErrUserNotFound     = errors.New("user not found")
	ErrUsernameTaken    = errors.New("username already taken")
	ErrInvalidUserID    = errors.New("invalid user id")
	ErrCannotFriendSelf = errors.New("cannot check friendship with self")
)

type Repository interface {
	GetByID(ctx context.Context, id string) (user_model.User, bool, error)
	GetByUsername(ctx context.Context, username string) (user_model.User, bool, error)
	GetByFirebaseUID(ctx context.Context, firebaseUID string) (user_model.User, bool, error)
	CountFriends(ctx context.Context, userID string) (int, error)
	AreFriends(ctx context.Context, user1ID, user2ID string) (bool, error)
	CountPlans(ctx context.Context, userID string) (int, error)
	CountCompletedPlans(ctx context.Context, userID string) (int, error)
	GetReliability(ctx context.Context, creatorID string) (*float32, error)
	ListCreatedEvents(ctx context.Context, creatorID string) ([]event_model.Event, error)
	UpdateProfile(ctx context.Context, id pgtype.UUID, name, username, biography, profilePicture *string) (user_model.User, error)
	UpdateSettings(ctx context.Context, settingsID int, showPlans bool) error
}
