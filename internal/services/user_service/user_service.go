package user_service

import (
	"context"
	"errors"
	"net/url"
	"strconv"

	"api-aprop/internal/models/event_model"
	"api-aprop/internal/models/user_model"
	"api-aprop/internal/repositories/user_repository"
)

type userService struct {
	userRepo user_repository.Repository
}

var _ Service = (*userService)(nil)

func NewUserService(userRepo user_repository.Repository) Service {
	return &userService{userRepo: userRepo}
}

func (s *userService) GetUserProfile(ctx context.Context, targetIdentifier string, requesterIdentifier string) (user_model.UserProfile, error) {
	var (
		target    user_model.User
		showPlans bool
		isSelf    bool
		err       error
	)

	if targetIdentifier == "" || targetIdentifier == "me" {
		if requesterIdentifier == "" {
			return user_model.UserProfile{}, ErrUnauthorized
		}
		target, showPlans, err = s.findUser(ctx, requesterIdentifier)
		if err != nil {
			return user_model.UserProfile{}, ErrUserNotFound
		}
		isSelf = true
	} else {
		target, showPlans, err = s.findUser(ctx, targetIdentifier)
		if err != nil {
			return user_model.UserProfile{}, ErrUserNotFound
		}
		if requesterIdentifier != "" {
			if requester, _, rErr := s.findUser(ctx, requesterIdentifier); rErr == nil {
				if requester.ID == target.ID || (requester.UUID != "" && requester.UUID == target.UUID) {
					isSelf = true
				}
			}
		}
	}

	var (
		friendsCount        int
		plansCount          int
		completedPlansCount int
		reliability         *float32
	)

	if target.UUID != "" {
		friendsCount, _ = s.userRepo.CountFriends(ctx, target.UUID)
		plansCount, _ = s.userRepo.CountPlans(ctx, target.UUID)
		completedPlansCount, _ = s.userRepo.CountCompletedPlans(ctx, target.UUID)
		reliability, _ = s.userRepo.GetReliability(ctx, target.UUID)
	}
	if reliability == nil && target.Rating != nil {
		reliability = target.Rating
	}

	var (
		isFriend     *bool
		plansVisible bool
	)

	if isSelf {
		plansVisible = true
		isFriend = nil
	} else {
		if requesterIdentifier != "" {
			if requester, _, rErr := s.findUser(ctx, requesterIdentifier); rErr == nil && requester.UUID != "" && target.UUID != "" {
				areFriends, _ := s.userRepo.AreFriends(ctx, requester.UUID, target.UUID)
				isFriend = &areFriends
				plansVisible = areFriends && showPlans
			} else {
				plansVisible = false
			}
		} else {
			plansVisible = false
		}
	}

	displayName := target.Name
	if displayName == "" {
		displayName = target.Username
	}

	profilePic := target.ProfilePicture
	if profilePic == "" {
		profilePic = "https://ui-avatars.com/api/?name=" + url.QueryEscape(displayName) + "&background=random"
	}

	profile := user_model.UserProfile{
		ID:                  target.ID,
		UUID:                target.UUID,
		FirebaseUID:         target.FirebaseUID,
		Username:            target.Username,
		Name:                displayName,
		ProfilePicture:      profilePic,
		Biography:           target.Biography,
		DateRegistered:      target.DateRegistered,
		AccountState:        target.AccountState,
		Experience:          target.Experience,
		Level:               target.Level,
		Rating:              target.Rating,
		Reliability:         reliability,
		FriendsCount:        friendsCount,
		PlansCount:          plansCount,
		CompletedPlansCount: completedPlansCount,
		ShowPlans:           showPlans,
		IsFriend:            isFriend,
		PlansVisible:        plansVisible,
	}

	if isSelf {
		profile.Email = target.Email
	}

	return profile, nil
}

func (s *userService) ListCreatedEvents(ctx context.Context, targetIdentifier string) ([]event_model.Event, error) {
	target, _, err := s.findUser(ctx, targetIdentifier)
	if err != nil {
		return nil, ErrUserNotFound
	}
	if target.UUID == "" {
		return []event_model.Event{}, nil
	}
	events, err := s.userRepo.ListCreatedEvents(ctx, target.UUID)
	if err != nil {
		return nil, err
	}
	if events == nil {
		events = []event_model.Event{}
	}
	return events, nil
}

func (s *userService) UpdateProfile(ctx context.Context, requesterIdentifier string, input UpdateProfileInput) (user_model.UserProfile, error) {
	if requesterIdentifier == "" {
		return user_model.UserProfile{}, ErrUnauthorized
	}
	user, _, err := s.findUser(ctx, requesterIdentifier)
	if err != nil {
		return user_model.UserProfile{}, ErrUserNotFound
	}

	if input.ShowPlans != nil {
		if err := s.userRepo.UpdateSettings(ctx, user.Settings, *input.ShowPlans); err != nil {
			return user_model.UserProfile{}, err
		}
	}

	if input.Name != nil || input.Username != nil || input.Biography != nil || input.ProfilePicture != nil {
		updatedUser, err := s.userRepo.UpdateProfile(ctx, user.ID, input.Name, input.Username, input.Biography, input.ProfilePicture)
		if errors.Is(err, user_repository.ErrUsernameTaken) {
			return user_model.UserProfile{}, ErrUsernameTaken
		}
		if err != nil {
			return user_model.UserProfile{}, err
		}
		user = updatedUser
	}

	return s.GetUserProfile(ctx, strconv.Itoa(user.ID), requesterIdentifier)
}

func (s *userService) findUser(ctx context.Context, identifier string) (user_model.User, bool, error) {
	if identifier == "" {
		return user_model.User{}, false, ErrUserNotFound
	}
	// Try integer ID first
	if id, err := strconv.Atoi(identifier); err == nil && id > 0 {
		u, showPlans, err := s.userRepo.GetByID(ctx, id)
		if err == nil {
			return u, showPlans, nil
		}
	}
	// Try UUID
	if u, showPlans, err := s.userRepo.GetByUUID(ctx, identifier); err == nil {
		return u, showPlans, nil
	}
	// Try FirebaseUID
	if u, showPlans, err := s.userRepo.GetByFirebaseUID(ctx, identifier); err == nil {
		return u, showPlans, nil
	}
	// Try Username
	if u, showPlans, err := s.userRepo.GetByUsername(ctx, identifier); err == nil {
		return u, showPlans, nil
	}
	return user_model.User{}, false, ErrUserNotFound
}
