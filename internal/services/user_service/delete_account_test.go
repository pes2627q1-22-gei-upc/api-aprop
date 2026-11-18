package user_service_test

import (
	"context"
	"errors"
	"testing"

	"api-aprop/internal/models/user_model"
	"api-aprop/internal/services/user_service"
)

type fakeAccountDeleter struct {
	err   error
	calls []string
}

func (f *fakeAccountDeleter) DeleteUser(_ context.Context, uid string) error {
	f.calls = append(f.calls, uid)
	return f.err
}

const deleteTestAvatar = "https://my-bucket.s3.amazonaws.com/avatars/x/avatar-1.png"

func newDeleteAccountFixture() (*mockUserRepo, *mockS3Client, *fakeAccountDeleter, string) {
	repo := newMockUserRepo()
	user := user_model.User{ID: testID(7), FirebaseUID: "fb-user-7", Username: "carla", ProfilePicture: deleteTestAvatar, Settings: 7}
	repo.addUser(user, true)
	return repo, &mockS3Client{}, &fakeAccountDeleter{}, user.ID.String()
}

func TestDeleteAccount_Success(t *testing.T) {
	repo, s3, deleter, id := newDeleteAccountFixture()
	svc := user_service.NewUserService(repo, &mockNotificationRepo{}, s3, deleter)

	if err := svc.DeleteAccount(context.Background(), id); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := repo.usersByID[testID(7)]; ok {
		t.Errorf("user still in the repository")
	}
	if len(deleter.calls) != 1 || deleter.calls[0] != "fb-user-7" {
		t.Errorf("firebase deletions = %v, want [fb-user-7]", deleter.calls)
	}
	if s3.deleteCalls != 1 || s3.deletedUserID != id || s3.deletedAvatarAt != deleteTestAvatar {
		t.Errorf("avatar deletion = %d calls (%q, %q), want 1 call (%q, %q)", s3.deleteCalls, s3.deletedUserID, s3.deletedAvatarAt, id, deleteTestAvatar)
	}
}

func TestDeleteAccount_WithoutAvatarOrFirebase(t *testing.T) {
	repo, s3, _, id := newDeleteAccountFixture()
	repo.usersByID[testID(7)].ProfilePicture = ""
	svc := user_service.NewUserService(repo, &mockNotificationRepo{}, s3, nil)

	if err := svc.DeleteAccount(context.Background(), id); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := repo.usersByID[testID(7)]; ok {
		t.Errorf("user still in the repository")
	}
	if s3.deleteCalls != 0 {
		t.Errorf("DeleteAvatar called %d times without avatar", s3.deleteCalls)
	}
}

func TestDeleteAccount_FirebaseFailsRollsBack(t *testing.T) {
	repo, s3, deleter, id := newDeleteAccountFixture()
	deleter.err = errors.New("firebase unavailable")
	svc := user_service.NewUserService(repo, &mockNotificationRepo{}, s3, deleter)

	err := svc.DeleteAccount(context.Background(), id)
	if !errors.Is(err, user_service.ErrAuthProvider) {
		t.Fatalf("err = %v, want ErrAuthProvider", err)
	}
	if _, ok := repo.usersByID[testID(7)]; !ok {
		t.Errorf("user deleted although Firebase failed")
	}
	if s3.deleteCalls != 0 {
		t.Errorf("avatar deleted although the transaction was rolled back")
	}
}

func TestDeleteAccount_AvatarErrorIsIgnored(t *testing.T) {
	repo, s3, deleter, id := newDeleteAccountFixture()
	s3.deleteErr = errors.New("storage down")
	svc := user_service.NewUserService(repo, &mockNotificationRepo{}, s3, deleter)

	if err := svc.DeleteAccount(context.Background(), id); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := repo.usersByID[testID(7)]; ok {
		t.Errorf("user still in the repository")
	}
}

func TestDeleteAccount_Errors(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		repoErr error
		want    error
	}{
		{"invalid id", "not-a-uuid", nil, user_service.ErrInvalidUserID},
		{"unknown user", testID(99).String(), nil, user_service.ErrUserNotFound},
		{"repository error", testID(7).String(), errDatabase, errDatabase},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, s3, deleter, _ := newDeleteAccountFixture()
			repo.deleteUserErr = tt.repoErr
			svc := user_service.NewUserService(repo, &mockNotificationRepo{}, s3, deleter)

			err := svc.DeleteAccount(context.Background(), tt.id)
			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, want %v", err, tt.want)
			}
			if len(deleter.calls) != 0 || s3.deleteCalls != 0 {
				t.Errorf("firebase calls = %d, avatar calls = %d, want none", len(deleter.calls), s3.deleteCalls)
			}
		})
	}
}
