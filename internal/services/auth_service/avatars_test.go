package auth_service

import (
	"context"
	"testing"

	"api-aprop/internal/clients/s3_client"
	"api-aprop/internal/models/user_model"
)

type fakeSigner struct{ refs []s3_client.AvatarRef }

func (f *fakeSigner) AvatarURLs(_ context.Context, avatars []s3_client.AvatarRef) ([]string, error) {
	f.refs = append(f.refs, avatars...)
	urls := make([]string, len(avatars))
	for i, a := range avatars {
		urls[i] = "signed:" + a.Stored
	}
	return urls, nil
}

// #247: the avatar of the private bucket is returned as a signed URL, never as the object path.
func TestSync_SignsTheStoredAvatar(t *testing.T) {
	stored := testID(7).String() + "/avatar-1.png"
	existing := user_model.User{ID: testID(7), FirebaseUID: firebaseUID, Username: "anna", Email: "anna@example.com", ProfilePicture: stored}

	signer := &fakeSigner{}
	result, err := NewAuthService(newFakeRepository(existing)).WithAvatarSigner(signer).Sync(context.Background(), firebaseUID, "anna@example.com", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.User.ProfilePicture != "signed:"+stored {
		t.Errorf("profile picture = %q, want the signed URL", result.User.ProfilePicture)
	}
	if len(signer.refs) != 1 || signer.refs[0].UserID != testID(7).String() {
		t.Errorf("signed refs = %+v, want the avatar of user 7", signer.refs)
	}

	// Without a signer the object path is hidden
	result, err = NewAuthService(newFakeRepository(existing)).Sync(context.Background(), firebaseUID, "anna@example.com", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.User.ProfilePicture != "" {
		t.Errorf("profile picture = %q, want empty without a signer", result.User.ProfilePicture)
	}
}
