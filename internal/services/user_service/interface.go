package user_service

import (
	"context"
	"io"
	"time"

	"api-aprop/internal/models/event_model"
	"api-aprop/internal/models/user_model"
)

type UpdateProfileInput struct {
	Name               *string `json:"name"`
	Username           *string `json:"username"`
	Biography          *string `json:"biography"`
	ProfilePicture     *string `json:"profile-picture"`
	ShowPlans          *bool   `json:"show-plans"`
	NotificationBefore *string `json:"notification-before"`
	AllowedNotifTypes  *[]int  `json:"allowed-notif-types"`
  Language           *string `json:"language"` // ca, es or en
}

type Service interface {
	GetUserProfile(ctx context.Context, targetIdentifier string, requesterIdentifier string) (user_model.UserProfile, error)
	GetSettings(ctx context.Context, userId string) (user_model.Settings, error)
	ListCreatedEvents(ctx context.Context, targetIdentifier string) ([]event_model.Event, error)
	ListReceivedRatings(ctx context.Context, targetIdentifier string) ([]user_model.ReceivedRating, error)
	UpdateProfile(ctx context.Context, requesterIdentifier string, input UpdateProfileInput) (user_model.UserProfile, error)
	UploadAvatar(ctx context.Context, requesterIdentifier string, filename string, contentType string, reader io.Reader) (user_model.UserProfile, error)
	GetUserFriends(ctx context.Context, userid string) ([]string, error)
	GetFriendRequests(ctx context.Context, userid string) ([]user_model.FriendRequest, error)
	SendFriendRequest(ctx context.Context, userid string, targetid string) (user_model.FriendRequest, error)
	AcceptFriendRequest(ctx context.Context, userid string, targetid string, date time.Time) (user_model.FriendRequest, error)
	DenyFriendRequest(ctx context.Context, userid string, targetid string, date time.Time) (user_model.FriendRequest, error)
	Unfriend(ctx context.Context, userid string, targetid string) error
	// DeleteAccount erases the requester and its data (RGPD) and deletes its Firebase Auth account.
	DeleteAccount(ctx context.Context, requesterID string) error
	// SearchUsers finds active users by username or name for the requester, without the blocked ones. pageNum
	// starts at 0 and a pageSize of 0 uses DefaultSearchPageSize.
	SearchUsers(ctx context.Context, requesterID, query string, pageNum, pageSize int) (user_model.UserSearchResult, error)
	// BlockUser blocks the target user and removes the friendship and the active friend requests with it (R2, R3).
	BlockUser(ctx context.Context, requesterID, targetID string) (user_model.BlockedUser, error)
	// UnblockUser removes the block of the target user.
	UnblockUser(ctx context.Context, requesterID, targetID string) error
	// ListBlockedUsers returns the users blocked by the requester, the most recent first.
	ListBlockedUsers(ctx context.Context, requesterID string) ([]user_model.BlockedUser, error)
}
