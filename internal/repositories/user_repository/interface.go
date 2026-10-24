package user_repository

import (
	"context"
	"errors"

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
	GetByID(ctx context.Context, id int) (user_model.User, bool, error)
	GetByUsername(ctx context.Context, username string) (user_model.User, bool, error)
	GetByFirebaseUID(ctx context.Context, firebaseUID string) (user_model.User, bool, error)
	GetByUUID(ctx context.Context, uuid string) (user_model.User, bool, error)
	CountFriends(ctx context.Context, userUUID string) (int, error)
	AreFriends(ctx context.Context, user1UUID, user2UUID string) (bool, error)
	CountPlans(ctx context.Context, userUUID string) (int, error)
	CountCompletedPlans(ctx context.Context, userUUID string) (int, error)
	GetReliability(ctx context.Context, creatorUUID string) (*float32, error)
	ListCreatedEvents(ctx context.Context, creatorUUID string) ([]event_model.Event, error)
	UpdateProfile(ctx context.Context, id int, name, username, biography, profilePicture *string) (user_model.User, error)
	UpdateSettings(ctx context.Context, settingsID int, showPlans bool) error
}
