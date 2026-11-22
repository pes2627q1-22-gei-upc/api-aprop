package user_service

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"api-aprop/internal/clients/s3_client"
	"api-aprop/internal/models/user_model"
)

// TODO(DP-037): page sizes and query length proposed by the backend, pending the frontend.
const (
	DefaultSearchPageSize = 20
	MaxSearchPageSize     = 50
	MaxSearchQueryLength  = 100
)

func (s *userService) SearchUsers(ctx context.Context, requesterID, query string, pageNum, pageSize int) (user_model.UserSearchResult, error) {
	requester, err := user_model.ParseID(requesterID)
	if err != nil {
		return user_model.UserSearchResult{}, ErrInvalidUserID
	}
	if pageSize == 0 {
		pageSize = DefaultSearchPageSize
	}
	if pageNum < 0 || pageSize < 0 || pageSize > MaxSearchPageSize {
		return user_model.UserSearchResult{}, fmt.Errorf("%w: pageNum must be at least 0 and pageSize between 1 and %d", ErrInvalidInput, MaxSearchPageSize)
	}

	query = strings.TrimSpace(query)
	if utf8.RuneCountInString(query) > MaxSearchQueryLength {
		return user_model.UserSearchResult{}, fmt.Errorf("%w: q must have at most %d characters", ErrInvalidInput, MaxSearchQueryLength)
	}
	result := user_model.UserSearchResult{Query: query, PageNum: pageNum, PageSize: pageSize, Users: []user_model.UserSummary{}}
	// An empty query would list every user
	if query == "" {
		return result, nil
	}

	users, total, err := s.userRepo.SearchUsers(ctx, requester, query, pageSize, pageNum*pageSize)
	if err != nil {
		return user_model.UserSearchResult{}, fmt.Errorf("search users: %w", err)
	}
	if users != nil {
		result.Users = users
	}
	fields := make([]s3_client.AvatarField, len(result.Users))
	for i := range result.Users {
		fields[i] = s3_client.AvatarField{UserID: result.Users[i].ID.String(), Picture: &result.Users[i].ProfilePicture}
	}
	s.resolveAvatars(ctx, fields...)
	result.Total = total
	return result, nil
}
