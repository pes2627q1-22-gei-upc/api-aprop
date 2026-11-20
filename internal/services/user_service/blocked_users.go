package user_service

import (
	"context"
	"errors"
	"fmt"

	"api-aprop/internal/models/user_model"
	"api-aprop/internal/repositories/user_repository"
)

// BlockUser blocks the target user (its id, Firebase UID or username, like Unfriend). R2: nobody can block themselves.
// R3: the friendship and the Pending or Accepted friend requests between them are removed in the same transaction.
// TODO(DP-042): the blocked user is not notified, and blocking does not hide events, chats or notifications.
func (s *userService) BlockUser(ctx context.Context, requesterID, targetID string) (user_model.BlockedUser, error) {
	requester, target, err := s.findBlockPair(ctx, requesterID, targetID)
	if err != nil {
		return user_model.BlockedUser{}, err
	}
	date, err := s.userRepo.BlockUser(ctx, requester.ID, target.ID)
	if errors.Is(err, user_repository.ErrAlreadyBlocked) {
		return user_model.BlockedUser{}, fmt.Errorf("%w: %w", ErrAlreadyBlocked, err)
	}
	if err != nil {
		return user_model.BlockedUser{}, err
	}
	return user_model.BlockedUser{
		UserSummary: user_model.UserSummary{
			ID:             target.ID,
			Username:       target.Username,
			Name:           target.Name,
			ProfilePicture: target.ProfilePicture,
		},
		DateBlocked: date,
	}, nil
}

// UnblockUser removes the block of the target user. The friendship removed when blocking is not restored.
func (s *userService) UnblockUser(ctx context.Context, requesterID, targetID string) error {
	requester, target, err := s.findBlockPair(ctx, requesterID, targetID)
	if err != nil {
		return err
	}
	unblocked, err := s.userRepo.UnblockUser(ctx, requester.ID, target.ID)
	if err != nil {
		return err
	}
	if !unblocked {
		return ErrNotBlocked
	}
	return nil
}

// ListBlockedUsers returns the users blocked by the requester, the most recent first.
// TODO(DP-042): no pagination, the list of blocked users is expected to be short.
func (s *userService) ListBlockedUsers(ctx context.Context, requesterID string) ([]user_model.BlockedUser, error) {
	requester, err := s.findUser(ctx, requesterID)
	if err != nil {
		return nil, ErrUserNotFound
	}
	users, err := s.userRepo.ListBlockedUsers(ctx, requester.ID)
	if err != nil {
		return nil, err
	}
	if users == nil {
		users = []user_model.BlockedUser{}
	}
	return users, nil
}

// findBlockPair finds the requester and the target of a block. R2: they cannot be the same user.
func (s *userService) findBlockPair(ctx context.Context, requesterID, targetID string) (user_model.User, user_model.User, error) {
	requester, err := s.findUser(ctx, requesterID)
	if err != nil {
		return user_model.User{}, user_model.User{}, ErrUserNotFound
	}
	target, err := s.findUser(ctx, targetID)
	if err != nil {
		return user_model.User{}, user_model.User{}, ErrUserNotFound
	}
	if requester.ID == target.ID {
		return user_model.User{}, user_model.User{}, ErrSameUser
	}
	return requester, target, nil
}
