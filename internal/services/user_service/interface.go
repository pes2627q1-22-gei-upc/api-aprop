package user_service

import (
	"context"

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
}
