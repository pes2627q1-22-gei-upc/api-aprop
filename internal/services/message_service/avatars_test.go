package message_service

import (
	"context"
	"testing"

	"api-aprop/internal/clients/s3_client"
	"api-aprop/internal/models/message_model"
)

type fakeSigner struct{ calls int }

func (f *fakeSigner) AvatarURLs(_ context.Context, avatars []s3_client.AvatarRef) ([]string, error) {
	f.calls++
	urls := make([]string, len(avatars))
	for i, a := range avatars {
		if a.Stored != "" {
			urls[i] = "signed:" + a.UserID + ":" + a.Stored
		}
	}
	return urls, nil
}

// #247: the avatars of the authors are signed in a single request.
func TestListMessages_SignsTheAvatars(t *testing.T) {
	repo := &fakeMessages{signedUp: true, total: 2, listed: []message_model.Message{
		{UserID: validUserID, ProfilePicture: validUserID + "/avatar-1.png"},
		{UserID: validUserID},
	}}
	signer := &fakeSigner{}
	svc := newService(repo, true).WithAvatarSigner(signer)

	page, err := svc.ListMessages(context.Background(), validUserID, 7, 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := "signed:" + validUserID + ":" + validUserID + "/avatar-1.png"; page.Messages[0].ProfilePicture != want {
		t.Errorf("picture = %q, want %q", page.Messages[0].ProfilePicture, want)
	}
	if page.Messages[1].ProfilePicture != "" {
		t.Errorf("picture = %q, want empty", page.Messages[1].ProfilePicture)
	}
	if signer.calls != 1 {
		t.Errorf("sign calls = %d, want 1", signer.calls)
	}
}

func TestListMessages_WithoutSignerHidesStoredAvatars(t *testing.T) {
	repo := &fakeMessages{signedUp: true, total: 2, listed: []message_model.Message{
		{UserID: validUserID, ProfilePicture: validUserID + "/avatar-1.png"},
		{UserID: validUserID, ProfilePicture: "https://example.com/me.png"},
	}}

	page, err := newService(repo, true).ListMessages(context.Background(), validUserID, 7, 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if page.Messages[0].ProfilePicture != "" || page.Messages[1].ProfilePicture != "https://example.com/me.png" {
		t.Errorf("pictures = %q, %q; want the object path hidden and the external URL kept",
			page.Messages[0].ProfilePicture, page.Messages[1].ProfilePicture)
	}
}
