package user_service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"api-aprop/internal/clients/s3_client"
	"api-aprop/internal/models/event_model"
	"api-aprop/internal/models/notification_model"
	"api-aprop/internal/models/user_model"
	"api-aprop/internal/repositories/notification_repository"
	"api-aprop/internal/repositories/user_repository"
)

var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]{3,30}$`)

const (
	maxNameLength = 100
	maxBioLength  = 500
)

type userService struct {
	userRepo         user_repository.Repository
	notificationRepo notification_repository.Repository
	s3Client         s3_client.Client
}

var _ Service = (*userService)(nil)

func NewUserService(userRepo user_repository.Repository, notificationRepo notification_repository.Repository, s3Client s3_client.Client) Service {
	return &userService{
		userRepo:         userRepo,
		notificationRepo: notificationRepo,
		s3Client:         s3Client,
	}
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
				if requester.ID == target.ID {
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

	friendsCount, _ = s.userRepo.CountFriends(ctx, target.ID)
	plansCount, _ = s.userRepo.CountPlans(ctx, target.ID)
	completedPlansCount, _ = s.userRepo.CountCompletedPlans(ctx, target.ID)
	reliability, _ = s.userRepo.GetReliability(ctx, target.ID)

	var (
		isFriend     *bool
		plansVisible bool
	)

	if isSelf {
		plansVisible = true
		isFriend = nil
	} else {
		if requesterIdentifier != "" {
			if requester, _, rErr := s.findUser(ctx, requesterIdentifier); rErr == nil {
				areFriends, _ := s.userRepo.AreFriends(ctx, requester.ID, target.ID)
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
	events, err := s.userRepo.ListCreatedEvents(ctx, target.ID)
	if err != nil {
		return nil, err
	}
	if events == nil {
		events = []event_model.Event{}
	}
	return events, nil
}

func (s *userService) ListReceivedRatings(ctx context.Context, targetIdentifier string) ([]user_model.ReceivedRating, error) {
	target, _, err := s.findUser(ctx, targetIdentifier)
	if err != nil {
		return nil, ErrUserNotFound
	}
	return s.userRepo.ListReceivedRatings(ctx, target.ID)
}

func (s *userService) UpdateProfile(ctx context.Context, requesterIdentifier string, input UpdateProfileInput) (user_model.UserProfile, error) {
	if requesterIdentifier == "" {
		return user_model.UserProfile{}, ErrUnauthorized
	}
	user, _, err := s.findUser(ctx, requesterIdentifier)
	if err != nil {
		return user_model.UserProfile{}, ErrUserNotFound
	}

	// Validation R1: Username
	if input.Username != nil {
		trimmedUser := strings.TrimSpace(*input.Username)
		if !usernameRegex.MatchString(trimmedUser) {
			return user_model.UserProfile{}, fmt.Errorf("%w: username must be 3-30 characters long and contain only letters, numbers and underscores", ErrInvalidInput)
		}
		input.Username = &trimmedUser
	}

	// Validation R1: Name
	if input.Name != nil {
		trimmedName := strings.TrimSpace(*input.Name)
		if len(trimmedName) > maxNameLength {
			return user_model.UserProfile{}, fmt.Errorf("%w: name cannot exceed %d characters", ErrInvalidInput, maxNameLength)
		}
		input.Name = &trimmedName
	}

	// Validation R1: Biography
	if input.Biography != nil {
		trimmedBio := strings.TrimSpace(*input.Biography)
		if len(trimmedBio) > maxBioLength {
			return user_model.UserProfile{}, fmt.Errorf("%w: biography cannot exceed %d characters", ErrInvalidInput, maxBioLength)
		}
		input.Biography = &trimmedBio
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

	return s.GetUserProfile(ctx, user.ID.String(), requesterIdentifier)
}

func (s *userService) UploadAvatar(ctx context.Context, requesterIdentifier string, filename string, contentType string, reader io.Reader) (user_model.UserProfile, error) {
	if requesterIdentifier == "" {
		return user_model.UserProfile{}, ErrUnauthorized
	}
	user, _, err := s.findUser(ctx, requesterIdentifier)
	if err != nil {
		return user_model.UserProfile{}, ErrUserNotFound
	}

	if s.s3Client == nil {
		return user_model.UserProfile{}, fmt.Errorf("%w: storage service unconfigured", ErrInvalidInput)
	}

	targetUID := user.ID

	avatarURL, err := s.s3Client.UploadAvatar(ctx, targetUID.String(), filename, contentType, reader)
	if err != nil {
		return user_model.UserProfile{}, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}

	updatedUser, err := s.userRepo.UpdateProfile(ctx, user.ID, nil, nil, nil, &avatarURL)
	if err != nil {
		return user_model.UserProfile{}, err
	}

	return s.GetUserProfile(ctx, updatedUser.ID.String(), requesterIdentifier)
}

func (s *userService) GetUserFriends(ctx context.Context, userid string) ([]string, error) {
	user, _, err := s.findUser(ctx, userid)
	if err != nil {
		return []string{}, ErrUserNotFound
	}

	id := user.ID

	friends, err := s.userRepo.GetUserFriends(ctx, id)
	if err != nil {
		return []string{}, err
	}
	return friends, nil
}

func (s *userService) GetFriendRequests(ctx context.Context, userid string) ([]user_model.FriendRequest, error) {
	user, _, err := s.findUser(ctx, userid)
	if err != nil {
		return []user_model.FriendRequest{}, ErrUserNotFound
	}

	id := user.ID

	friendreqs, err := s.userRepo.GetFriendRequests(ctx, id)
	if err != nil {
		return []user_model.FriendRequest{}, err
	}
	return friendreqs, nil
}

func (s *userService) SendFriendRequest(ctx context.Context, userid string, targetid string) (user_model.FriendRequest, error) {
	user, _, err := s.findUser(ctx, userid)
	if err != nil {
		return user_model.FriendRequest{}, ErrUserNotFound
	}

	id := user.ID
	targetuser, _, err := s.findUser(ctx, targetid)
	if err != nil {
		return user_model.FriendRequest{}, ErrUserNotFound
	}
	targetuuid := targetuser.ID

	if id == targetuuid {
		return user_model.FriendRequest{}, ErrSameUser
	}

	arefriends, err := s.userRepo.AreFriends(ctx, id, targetuuid)
	if err != nil {
		return user_model.FriendRequest{}, err
	}
	if arefriends {
		return user_model.FriendRequest{}, ErrAlreadyFriends
	}

	existingrequest, err := s.userRepo.PendingRequestExists(ctx, id, targetuuid)
	if err != nil {
		return user_model.FriendRequest{}, err
	}
	if existingrequest {
		return user_model.FriendRequest{}, ErrRequestAlreadyExists
	}

	isblocked, err := s.userRepo.IsBlocked(ctx, id, targetuuid)
	if err != nil {
		return user_model.FriendRequest{}, err
	}
	if isblocked {
		return user_model.FriendRequest{}, ErrIsBlocked
	}

	friendreq, err := s.userRepo.SendFriendRequest(ctx, id, targetuuid)
	if err != nil {
		return user_model.FriendRequest{}, err
	}
	return friendreq, nil
}

func (s *userService) AcceptFriendRequest(ctx context.Context, userid string, targetid string, date time.Time) (user_model.FriendRequest, error) {
	user, _, err := s.findUser(ctx, userid)
	if err != nil {
		return user_model.FriendRequest{}, ErrUserNotFound
	}

	id := user.ID
	targetuser, _, err := s.findUser(ctx, targetid)
	if err != nil {
		return user_model.FriendRequest{}, ErrUserNotFound
	}
	targetuuid := targetuser.ID

	if id == targetuuid {
		return user_model.FriendRequest{}, ErrSameUser
	}

	friendreq, err := s.userRepo.AcceptFriendRequest(ctx, id, targetuuid, date)
	if errors.Is(err, pgx.ErrNoRows) {
		return user_model.FriendRequest{}, ErrRequestNotFound
	} else if err != nil {
		return user_model.FriendRequest{}, err
	} else {
		// If it fails to create the notification, the AcceptFriendRequest operation should still return properly if accepting succeeded
		nottype, err := s.notificationRepo.GetNotificationTypeByName("Social")
		if err == nil {
			_, _ = s.notificationRepo.Create(notification_model.Notification{
				UserID:      targetuuid,
				Friend:      id,
				Type:        nottype,
				RequestDate: pgtype.Timestamp{Time: date, Valid: true},
				Text:        fmt.Sprintf("%s ha acceptat la teva sol·licitud d'amistat", user.Username),
			})
		}
	}

	return friendreq, nil
}

func (s *userService) DenyFriendRequest(ctx context.Context, userid string, targetid string, date time.Time) (user_model.FriendRequest, error) {
	user, _, err := s.findUser(ctx, userid)
	if err != nil {
		return user_model.FriendRequest{}, ErrUserNotFound
	}

	id := user.ID
	targetuser, _, err := s.findUser(ctx, targetid)
	if err != nil {
		return user_model.FriendRequest{}, ErrUserNotFound
	}
	targetuuid := targetuser.ID

	if id == targetuuid {
		return user_model.FriendRequest{}, ErrSameUser
	}

	friendreq, err := s.userRepo.DenyFriendRequest(ctx, id, targetuuid, date)
	if errors.Is(err, pgx.ErrNoRows) {
		return user_model.FriendRequest{}, ErrRequestNotFound
	} else if err != nil {
		return user_model.FriendRequest{}, err
	} else {
		// If it fails to create the notification, the AcceptFriendRequest operation should still return properly if accepting succeeded
		nottype, err := s.notificationRepo.GetNotificationTypeByName("Social")
		if err == nil {
			_, _ = s.notificationRepo.Create(notification_model.Notification{
				UserID:      targetuuid,
				Friend:      id,
				Type:        nottype,
				RequestDate: pgtype.Timestamp{Time: date, Valid: true},
				Text:        fmt.Sprintf("%s ha rebutjat la teva sol·licitud d'amistat", user.Username),
			})
		}
	}

	return friendreq, nil
}

func (s *userService) Unfriend(ctx context.Context, userid string, targetid string) error {
	user, _, err := s.findUser(ctx, userid)
	if err != nil {
		return ErrUserNotFound
	}

	id := user.ID
	targetuser, _, err := s.findUser(ctx, targetid)
	if err != nil {
		return ErrUserNotFound
	}
	targetuuid := targetuser.ID

	if id == targetuuid {
		return ErrSameUser
	}

	arefriends, err := s.userRepo.AreFriends(ctx, id, targetuuid)
	if err != nil {
		return err
	}
	if !arefriends {
		return ErrNotFriends
	}

	err = s.userRepo.Unfriend(ctx, id, targetuuid)

	if err == nil {
		// If it fails to create the notification, the Unfriend operation should still return properly if unfriending succeeded
		nottype, err := s.notificationRepo.GetNotificationTypeByName("Social")
		if err == nil {
			_, _ = s.notificationRepo.Create(notification_model.Notification{
				UserID: targetuuid,
				Friend: id,
				Type:   nottype,
				Text:   fmt.Sprintf("%s ha eliminat la seva amistat amb tu", user.Username),
			})
		}
	}

	return err
}

func (s *userService) findUser(ctx context.Context, identifier string) (user_model.User, bool, error) {
	if identifier == "" {
		return user_model.User{}, false, ErrUserNotFound
	}
	// Try ID first
	if id, err := user_model.ParseID(identifier); err == nil {
		u, showPlans, err := s.userRepo.GetByID(ctx, id)
		if err == nil {
			return u, showPlans, nil
		}
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
