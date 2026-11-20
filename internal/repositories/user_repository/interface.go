package user_repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"api-aprop/internal/models/event_model"
	"api-aprop/internal/models/user_model"
)

var (
	ErrUserNotFound     = errors.New("user not found")
	ErrUsernameTaken    = errors.New("username already taken")
	ErrInvalidUserID    = errors.New("invalid user id")
	ErrCannotFriendSelf = errors.New("cannot check friendship with self")
	ErrAlreadyBlocked   = errors.New("user already blocked")
)

type Repository interface {
	GetByID(ctx context.Context, id pgtype.UUID) (user_model.User, error)
	GetByUsername(ctx context.Context, username string) (user_model.User, error)
	GetByFirebaseUID(ctx context.Context, firebaseUID string) (user_model.User, error)
	GetSettings(ctx context.Context, id int) (user_model.Settings, error)
	CountFriends(ctx context.Context, userUUID pgtype.UUID) (int, error)
	AreFriends(ctx context.Context, user1UUID pgtype.UUID, user2UUID pgtype.UUID) (bool, error)
	CountPlans(ctx context.Context, userUUID pgtype.UUID) (int, error)
	CountCompletedPlans(ctx context.Context, userUUID pgtype.UUID) (int, error)
	ListReceivedRatings(ctx context.Context, creatorUUID pgtype.UUID) ([]user_model.ReceivedRating, error)
	ListCreatedEvents(ctx context.Context, creatorUUID pgtype.UUID) ([]event_model.Event, error)
	UpdateProfile(ctx context.Context, id pgtype.UUID, name, username, biography, profilePicture *string) (user_model.User, error)
	UpdateSettings(ctx context.Context, settings user_model.Settings) error
	GetUserFriends(ctx context.Context, userid pgtype.UUID) ([]string, error)
	GetFriendRequests(ctx context.Context, userid pgtype.UUID) ([]user_model.FriendRequest, error)
	PendingRequestExists(ctx context.Context, userid pgtype.UUID, targetid pgtype.UUID) (bool, error)
	// Checks if the user is blocked by the target
	IsBlocked(ctx context.Context, userid pgtype.UUID, targetid pgtype.UUID) (bool, error)
	SendFriendRequest(ctx context.Context, userid pgtype.UUID, targetid pgtype.UUID) (user_model.FriendRequest, error)
	AcceptFriendRequest(ctx context.Context, userid pgtype.UUID, targetid pgtype.UUID, date time.Time) (user_model.FriendRequest, error)
	DenyFriendRequest(ctx context.Context, userid pgtype.UUID, targetid pgtype.UUID, date time.Time) (user_model.FriendRequest, error)
	Unfriend(ctx context.Context, userid pgtype.UUID, targetuuid pgtype.UUID) error
	// DeleteUser erases the user, the unofficial events it created and every row that references
	// them in a single transaction. beforeCommit runs inside the transaction once everything is
	// deleted: if it fails nothing is deleted. Returns ErrUserNotFound if the user does not exist.
	DeleteUser(ctx context.Context, id pgtype.UUID, beforeCommit func(context.Context) error) error
	// SearchUsers returns the active users whose username or name contains query, except the requester and the
	// users that block or are blocked by the requester, and the total number of matches.
	SearchUsers(ctx context.Context, requester pgtype.UUID, query string, limit, offset int) ([]user_model.UserSummary, int64, error)
	// BlockUser blocks the user and, in the same transaction, removes their friendship and rejects the Pending or
	// Accepted friend requests between them (R3). It returns the date of the block, or ErrAlreadyBlocked.
	BlockUser(ctx context.Context, blocker, blocked pgtype.UUID) (time.Time, error)
	// UnblockUser removes the block; false if the user was not blocked. The friendship is not restored.
	UnblockUser(ctx context.Context, blocker, blocked pgtype.UUID) (bool, error)
	// ListBlockedUsers returns the users blocked by the blocker, the most recent first.
	ListBlockedUsers(ctx context.Context, blocker pgtype.UUID) ([]user_model.BlockedUser, error)
}
