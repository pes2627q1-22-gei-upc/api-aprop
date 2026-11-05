package user_service

import (
	"context"
	"io"

	"api-aprop/internal/models/event_model"
	"api-aprop/internal/models/user_model"
)

type UpdateProfileInput struct {
	Name           *string `json:"name"`
	Username       *string `json:"username"`
	Biography      *string `json:"biography"`
	ProfilePicture *string `json:"profile-picture"`
	ShowPlans      *bool   `json:"show-plans"`
}

type Service interface {
	GetUserProfile(ctx context.Context, targetIdentifier string, requesterIdentifier string) (user_model.UserProfile, error)
	ListCreatedEvents(ctx context.Context, targetIdentifier string) ([]event_model.Event, error)
	UpdateProfile(ctx context.Context, requesterIdentifier string, input UpdateProfileInput) (user_model.UserProfile, error)
	UploadAvatar(ctx context.Context, requesterIdentifier string, filename string, contentType string, reader io.Reader) (user_model.UserProfile, error)
	GetUserFriends(ctx context.Context, userid string) ([]string, error)
	GetFriendRequests(ctx context.Context, userid string) ([]user_model.FriendRequest, error)
	SendFriendRequest(ctx context.Context, userid string, targetid string) (user_model.FriendRequest, error)
}
