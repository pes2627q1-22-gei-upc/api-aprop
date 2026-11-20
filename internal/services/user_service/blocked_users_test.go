package user_service_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"api-aprop/internal/models/user_model"
	"api-aprop/internal/repositories/user_repository"
	"api-aprop/internal/services/user_service"
)

func newBlockService() (*mockUserRepo, user_service.Service) {
	repo := newMockUserRepo()
	repo.addUser(user_model.User{ID: testID(1), FirebaseUID: "fb-alice", Username: "alice"}, true)
	repo.addUser(user_model.User{ID: testID(2), FirebaseUID: "fb-bob", Username: "bob", Name: "Bob Puig", ProfilePicture: "https://img/bob.png"}, true)
	return repo, user_service.NewUserService(repo, &mockNotificationRepo{}, &mockS3Client{}, nil)
}

func TestBlockUser(t *testing.T) {
	date := time.Date(2026, 11, 19, 10, 0, 0, 0, time.UTC)
	repo, svc := newBlockService()
	repo.blockDate = date

	got, err := svc.BlockUser(context.Background(), testID(1).String(), "bob")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.blockCalls != 1 || repo.gotBlocker != testID(1) || repo.gotBlocked != testID(2) {
		t.Errorf("repository got blocker=%v blocked=%v calls=%d", repo.gotBlocker, repo.gotBlocked, repo.blockCalls)
	}
	want := user_model.BlockedUser{
		UserSummary: user_model.UserSummary{ID: testID(2), Username: "bob", Name: "Bob Puig", ProfilePicture: "https://img/bob.png"},
		DateBlocked: date,
	}
	if got != want {
		t.Errorf("blocked = %+v, want %+v", got, want)
	}
}

func TestR2_CannotBlockYourself(t *testing.T) {
	repo, svc := newBlockService()
	if _, err := svc.BlockUser(context.Background(), testID(1).String(), "alice"); !errors.Is(err, user_service.ErrSameUser) {
		t.Fatalf("block: err = %v, want ErrSameUser", err)
	}
	if err := svc.UnblockUser(context.Background(), testID(1).String(), "fb-alice"); !errors.Is(err, user_service.ErrSameUser) {
		t.Fatalf("unblock: err = %v, want ErrSameUser", err)
	}
	if repo.blockCalls != 0 || repo.unblockCalls != 0 {
		t.Errorf("repository used: block=%d unblock=%d", repo.blockCalls, repo.unblockCalls)
	}
}

func TestBlockUser_Errors(t *testing.T) {
	tests := []struct {
		name      string
		requester string
		target    string
		repoErr   error
		wantErr   error
		wantCalls int
	}{
		{"requester not found", "unknown", "bob", nil, user_service.ErrUserNotFound, 0},
		{"target not found", testID(1).String(), "unknown", nil, user_service.ErrUserNotFound, 0},
		{"already blocked", testID(1).String(), "bob", fmt.Errorf("x: %w", user_repository.ErrAlreadyBlocked), user_service.ErrAlreadyBlocked, 1},
		{"database error", testID(1).String(), "bob", errDatabase, errDatabase, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, svc := newBlockService()
			repo.blockErr = tt.repoErr
			if _, err := svc.BlockUser(context.Background(), tt.requester, tt.target); !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if repo.blockCalls != tt.wantCalls {
				t.Errorf("repository calls = %d, want %d", repo.blockCalls, tt.wantCalls)
			}
		})
	}
}

func TestUnblockUser(t *testing.T) {
	tests := []struct {
		name      string
		target    string
		unblocked bool
		repoErr   error
		wantErr   error
	}{
		{"unblocked", "bob", true, nil, nil},
		{"by firebase uid", "fb-bob", true, nil, nil},
		{"was not blocked", "bob", false, nil, user_service.ErrNotBlocked},
		{"target not found", "unknown", false, nil, user_service.ErrUserNotFound},
		{"database error", "bob", false, errDatabase, errDatabase},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, svc := newBlockService()
			repo.unblocked, repo.unblockErr = tt.unblocked, tt.repoErr
			err := svc.UnblockUser(context.Background(), testID(1).String(), tt.target)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil && (repo.gotBlocker != testID(1) || repo.gotBlocked != testID(2)) {
				t.Errorf("repository got blocker=%v blocked=%v", repo.gotBlocker, repo.gotBlocked)
			}
		})
	}
}

func TestListBlockedUsers(t *testing.T) {
	t.Run("list", func(t *testing.T) {
		repo, svc := newBlockService()
		repo.blockedList = []user_model.BlockedUser{{UserSummary: user_model.UserSummary{ID: testID(2), Username: "bob"}}}
		got, err := svc.ListBlockedUsers(context.Background(), testID(1).String())
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got[0].Username != "bob" || repo.gotBlocker != testID(1) {
			t.Errorf("got %+v, blocker %v", got, repo.gotBlocker)
		}
	})
	t.Run("empty list is not null", func(t *testing.T) {
		_, svc := newBlockService()
		got, err := svc.ListBlockedUsers(context.Background(), testID(1).String())
		if err != nil || got == nil || len(got) != 0 {
			t.Fatalf("got %v, %v; want an empty list", got, err)
		}
	})
	t.Run("requester not found", func(t *testing.T) {
		_, svc := newBlockService()
		if _, err := svc.ListBlockedUsers(context.Background(), "unknown"); !errors.Is(err, user_service.ErrUserNotFound) {
			t.Fatalf("err = %v, want ErrUserNotFound", err)
		}
	})
	t.Run("database error", func(t *testing.T) {
		repo, svc := newBlockService()
		repo.blockedErr = errDatabase
		if _, err := svc.ListBlockedUsers(context.Background(), testID(1).String()); !errors.Is(err, errDatabase) {
			t.Fatalf("err = %v, want errDatabase", err)
		}
	})
}

// R3: once blocked, a friend request in either direction is refused.
func TestR3_BlockedUsersCannotSendFriendRequests(t *testing.T) {
	repo, svc := newBlockService()
	repo.isBlockedResult = true
	if _, err := svc.SendFriendRequest(context.Background(), testID(2).String(), "alice"); !errors.Is(err, user_service.ErrIsBlocked) {
		t.Fatalf("err = %v, want ErrIsBlocked", err)
	}
}
