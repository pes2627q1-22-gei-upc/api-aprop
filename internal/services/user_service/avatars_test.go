package user_service_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"api-aprop/internal/clients/s3_client"
	"api-aprop/internal/models/user_model"
	"api-aprop/internal/services/user_service"
)

// #247: profile pictures are stored as object paths of a private bucket and served as signed URLs.

func avatarPath(n byte) string { return testID(n).String() + "/avatar-1.png" }

func newAvatarFixture(picture string) (*mockUserRepo, *mockS3Client, user_service.Service) {
	repo := newMockUserRepo()
	repo.addUser(user_model.User{ID: testID(1), FirebaseUID: "fb-alice", Username: "alice", ProfilePicture: picture, Settings: 1}, true)
	s3 := &mockS3Client{}
	return repo, s3, user_service.NewUserService(repo, &mockNotificationRepo{}, s3, nil)
}

func TestGetUserProfile_ProfilePicture(t *testing.T) {
	tests := []struct {
		name    string
		stored  string
		signErr error
		want    string
	}{
		{"stored avatar is signed", avatarPath(1), nil, "signed:" + avatarPath(1)},
		{"external url is kept", "https://lh3.googleusercontent.com/a/photo.jpg", nil, "https://lh3.googleusercontent.com/a/photo.jpg"},
		{"sign error falls back to the generated avatar", avatarPath(1), errors.New("storage down"), "https://ui-avatars.com/api/?name=alice&background=random"},
		{"no picture uses the generated avatar", "", nil, "https://ui-avatars.com/api/?name=alice&background=random"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, s3, svc := newAvatarFixture(tt.stored)
			s3.signErr = tt.signErr

			profile, err := svc.GetUserProfile(context.Background(), "alice", "")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if profile.ProfilePicture != tt.want {
				t.Errorf("profile picture = %q, want %q", profile.ProfilePicture, tt.want)
			}
			if tt.stored != "" && (len(s3.signedRefs) != 1 || s3.signedRefs[0].UserID != testID(1).String()) {
				t.Errorf("signed refs = %+v, want the picture of user 1", s3.signedRefs)
			}
		})
	}
}

func TestGetUserProfile_WithoutStorageHidesStoredAvatars(t *testing.T) {
	repo := newMockUserRepo()
	repo.addUser(user_model.User{ID: testID(1), Username: "alice", ProfilePicture: avatarPath(1), Settings: 1}, true)
	svc := user_service.NewUserService(repo, &mockNotificationRepo{}, nil, nil)

	profile, err := svc.GetUserProfile(context.Background(), "alice", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(profile.ProfilePicture, avatarPath(1)) {
		t.Errorf("profile picture = %q, the object path must not be exposed", profile.ProfilePicture)
	}
}

func TestUploadAvatar_Errors(t *testing.T) {
	tests := []struct {
		name    string
		failErr error
		want    error
	}{
		{"invalid content type", s3_client.ErrInvalidContentType, user_service.ErrInvalidInput},
		{"storage not configured", s3_client.ErrStorageNotConfigured, user_service.ErrStorageUnavailable},
		{"storage error", fmt.Errorf("%w: status 500", s3_client.ErrUploadFailed), user_service.ErrStorageFailed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, s3, svc := newAvatarFixture("")
			s3.failErr = tt.failErr

			_, err := svc.UploadAvatar(context.Background(), "fb-alice", "photo.png", "image/png", strings.NewReader("img"))
			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, want %v", err, tt.want)
			}
			if repo.usersByID[testID(1)].ProfilePicture != "" {
				t.Error("profile picture changed although the upload failed")
			}
		})
	}

	t.Run("without storage", func(t *testing.T) {
		repo := newMockUserRepo()
		repo.addUser(user_model.User{ID: testID(1), FirebaseUID: "fb-alice", Username: "alice", Settings: 1}, true)
		svc := user_service.NewUserService(repo, &mockNotificationRepo{}, nil, nil)
		_, err := svc.UploadAvatar(context.Background(), "fb-alice", "photo.png", "image/png", strings.NewReader("img"))
		if !errors.Is(err, user_service.ErrStorageUnavailable) {
			t.Fatalf("err = %v, want ErrStorageUnavailable", err)
		}
	})
}

func TestUploadAvatar_DeletesTheReplacedAvatar(t *testing.T) {
	old := testID(1).String() + "/avatar-0.png"
	repo, s3, svc := newAvatarFixture(old)

	if _, err := svc.UploadAvatar(context.Background(), "fb-alice", "photo.png", "image/png", strings.NewReader("img")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := repo.usersByID[testID(1)].ProfilePicture; got != avatarPath(1) {
		t.Errorf("stored picture = %q, want %q", got, avatarPath(1))
	}
	if s3.deleteCalls != 1 || s3.deletedAvatarAt != old || s3.deletedUserID != testID(1).String() {
		t.Errorf("delete = %d calls (%q, %q), want 1 call for the old avatar", s3.deleteCalls, s3.deletedUserID, s3.deletedAvatarAt)
	}
}

func TestUpdateProfile_ProfilePicture(t *testing.T) {
	tests := []struct {
		name       string
		picture    string
		wantErr    error
		wantStored string
	}{
		{"external https url", "  https://example.com/me.png ", nil, "https://example.com/me.png"},
		{"empty removes the picture", "", nil, ""},
		{"object path of the bucket", avatarPath(2), user_service.ErrInvalidInput, avatarPath(1)},
		{"other scheme", "javascript:alert(1)", user_service.ErrInvalidInput, avatarPath(1)},
		{"url without host", "https:///me.png", user_service.ErrInvalidInput, avatarPath(1)},
		{"too long", "https://example.com/" + strings.Repeat("a", 2048), user_service.ErrInvalidInput, avatarPath(1)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, s3, svc := newAvatarFixture(avatarPath(1))
			picture := tt.picture

			_, err := svc.UpdateProfile(context.Background(), "fb-alice", user_service.UpdateProfileInput{ProfilePicture: &picture})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if got := repo.usersByID[testID(1)].ProfilePicture; got != tt.wantStored {
				t.Errorf("stored picture = %q, want %q", got, tt.wantStored)
			}
			wantDeletes := 0
			if tt.wantErr == nil {
				wantDeletes = 1 // the avatar of the bucket was replaced
			}
			if s3.deleteCalls != wantDeletes {
				t.Errorf("deleted %d avatars, want %d", s3.deleteCalls, wantDeletes)
			}
		})
	}
}

func TestSearchUsers_SignsTheAvatars(t *testing.T) {
	repo := newMockUserRepo()
	repo.searchResult = []user_model.UserSummary{
		{ID: testID(2), Username: "joan", ProfilePicture: avatarPath(2)},
		{ID: testID(3), Username: "joana", ProfilePicture: "https://example.com/j.png"},
		{ID: testID(4), Username: "joanet"},
	}
	s3 := &mockS3Client{}
	svc := user_service.NewUserService(repo, &mockNotificationRepo{}, s3, nil)

	res, err := svc.SearchUsers(context.Background(), testID(1).String(), "joan", 0, 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"signed:" + avatarPath(2), "https://example.com/j.png", ""}
	for i, u := range res.Users {
		if u.ProfilePicture != want[i] {
			t.Errorf("user %d picture = %q, want %q", i, u.ProfilePicture, want[i])
		}
	}
	if len(s3.signedRefs) != 3 || s3.signedRefs[0].UserID != testID(2).String() {
		t.Errorf("signed refs = %+v, want one batch with the owners", s3.signedRefs)
	}
}

func TestListBlockedUsers_SignsTheAvatars(t *testing.T) {
	repo, _, svc := newAvatarFixture("")
	repo.blockedList = []user_model.BlockedUser{{UserSummary: user_model.UserSummary{ID: testID(2), Username: "bob", ProfilePicture: avatarPath(2)}}}

	got, err := svc.ListBlockedUsers(context.Background(), testID(1).String())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].ProfilePicture != "signed:"+avatarPath(2) {
		t.Errorf("blocked users = %+v, want the signed avatar", got)
	}
}
