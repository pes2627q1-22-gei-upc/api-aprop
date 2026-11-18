package user_service_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"api-aprop/internal/models/user_model"
	"api-aprop/internal/services/user_service"
)

func newSearchService(repo *mockUserRepo) user_service.Service {
	return user_service.NewUserService(repo, &mockNotificationRepo{}, &mockS3Client{}, nil)
}

func TestSearchUsers_ReturnsThePageAndTotal(t *testing.T) {
	repo := newMockUserRepo()
	repo.searchResult = []user_model.UserSummary{{ID: testID(2), Username: "joanpuig", Name: "Joan Puig"}}
	repo.searchTotal = 41
	requester := testID(1)

	res, err := newSearchService(repo).SearchUsers(context.Background(), requester.String(), "  joan ", 2, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.searchRequester != requester || repo.searchQuery != "joan" || repo.searchLimit != 10 || repo.searchOffset != 20 {
		t.Errorf("repository got requester=%v query=%q limit=%d offset=%d, want %v \"joan\" 10 20",
			repo.searchRequester, repo.searchQuery, repo.searchLimit, repo.searchOffset, requester)
	}
	if res.Query != "joan" || res.Total != 41 || res.PageNum != 2 || res.PageSize != 10 || len(res.Users) != 1 {
		t.Errorf("result = %+v", res)
	}
}

func TestSearchUsers_DefaultPageSize(t *testing.T) {
	repo := newMockUserRepo()
	res, err := newSearchService(repo).SearchUsers(context.Background(), testID(1).String(), "joan", 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.searchLimit != user_service.DefaultSearchPageSize || res.PageSize != user_service.DefaultSearchPageSize {
		t.Errorf("page size = %d (repo %d), want %d", res.PageSize, repo.searchLimit, user_service.DefaultSearchPageSize)
	}
	if res.Users == nil {
		t.Error("users must be an empty list, not null")
	}
}

func TestSearchUsers_EmptyQueryDoesNotListEveryone(t *testing.T) {
	repo := newMockUserRepo()
	res, err := newSearchService(repo).SearchUsers(context.Background(), testID(1).String(), "   ", 0, 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.searchCalls != 0 {
		t.Errorf("repository called %d times, want 0", repo.searchCalls)
	}
	if res.Total != 0 || res.Users == nil || len(res.Users) != 0 {
		t.Errorf("result = %+v, want an empty page", res)
	}
}

func TestSearchUsers_InvalidInput(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		pageNum  int
		pageSize int
	}{
		{"negative page", "joan", -1, 20},
		{"negative page size", "joan", 0, -5},
		{"page size too big", "joan", 0, user_service.MaxSearchPageSize + 1},
		{"query too long", strings.Repeat("a", user_service.MaxSearchQueryLength+1), 0, 20},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockUserRepo()
			_, err := newSearchService(repo).SearchUsers(context.Background(), testID(1).String(), tt.query, tt.pageNum, tt.pageSize)
			if !errors.Is(err, user_service.ErrInvalidInput) {
				t.Errorf("error = %v, want ErrInvalidInput", err)
			}
			if repo.searchCalls != 0 {
				t.Errorf("repository called with invalid input")
			}
		})
	}
}

func TestSearchUsers_MaxPageSizeIsAllowed(t *testing.T) {
	repo := newMockUserRepo()
	if _, err := newSearchService(repo).SearchUsers(context.Background(), testID(1).String(), "joan", 0, user_service.MaxSearchPageSize); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSearchUsers_InvalidRequester(t *testing.T) {
	_, err := newSearchService(newMockUserRepo()).SearchUsers(context.Background(), "not-a-uuid", "joan", 0, 20)
	if !errors.Is(err, user_service.ErrInvalidUserID) {
		t.Errorf("error = %v, want ErrInvalidUserID", err)
	}
}

func TestSearchUsers_RepositoryError(t *testing.T) {
	repo := newMockUserRepo()
	repo.searchErr = errors.New("db down")
	if _, err := newSearchService(repo).SearchUsers(context.Background(), testID(1).String(), "joan", 0, 20); !errors.Is(err, repo.searchErr) {
		t.Errorf("error = %v, want the repository error", err)
	}
}
