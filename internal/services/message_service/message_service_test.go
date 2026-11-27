package message_service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"api-aprop/internal/models/message_model"
	"api-aprop/internal/repositories/message_repository"
)

const validUserID = "3f2504e0-4f89-11d3-9a0c-0305e82c3301"

type fakeEvents struct {
	exists bool
	err    error
}

func (f fakeEvents) EventExists(int) (bool, error) { return f.exists, f.err }

type fakeMessages struct {
	signedUp    bool
	signUpErr   error
	createErr   error
	createCalls int
	gotText     string
	listed      []message_model.Message
	listCalls   int
	gotLimit    int
	gotOffset   int
	total       int64
}

func (f *fakeMessages) Create(_ context.Context, userID string, eventID int, text string) (message_model.Message, error) {
	f.createCalls++
	f.gotText = text
	if f.createErr != nil {
		return message_model.Message{}, f.createErr
	}
	return message_model.Message{UserID: userID, EventID: eventID, Text: text, DateSent: time.Unix(0, 0).UTC()}, nil
}

func (f *fakeMessages) List(_ context.Context, _, limit, offset int) ([]message_model.Message, error) {
	f.listCalls++
	f.gotLimit, f.gotOffset = limit, offset
	return f.listed, nil
}

func (f *fakeMessages) Count(context.Context, int) (int64, error) { return f.total, nil }

func (f *fakeMessages) IsSignedUp(context.Context, string, int) (bool, error) {
	return f.signedUp, f.signUpErr
}

func newService(repo *fakeMessages, eventExists bool) *MessageService {
	return NewMessageService(repo, fakeEvents{exists: eventExists})
}

func TestR11_ChatRequiresSignUp(t *testing.T) {
	repo := &fakeMessages{signedUp: false, total: 3}
	svc := newService(repo, true)

	if _, err := svc.SendMessage(context.Background(), validUserID, 7, "hola"); !errors.Is(err, ErrNotSignedUp) {
		t.Fatalf("SendMessage err = %v, want ErrNotSignedUp", err)
	}
	if _, err := svc.ListMessages(context.Background(), validUserID, 7, 0, 0); !errors.Is(err, ErrNotSignedUp) {
		t.Fatalf("ListMessages err = %v, want ErrNotSignedUp", err)
	}
	if repo.createCalls != 0 || repo.listCalls != 0 {
		t.Fatalf("repository used without a sign up: create=%d list=%d", repo.createCalls, repo.listCalls)
	}
}

func TestR11_ChatAllowsSignedUpUser(t *testing.T) {
	repo := &fakeMessages{signedUp: true}
	svc := newService(repo, true)

	msg, err := svc.SendMessage(context.Background(), validUserID, 7, "  Ens veiem a l'entrada  ")
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	if repo.gotText != "Ens veiem a l'entrada" || msg.Text != "Ens veiem a l'entrada" {
		t.Errorf("text not trimmed: stored %q, returned %q", repo.gotText, msg.Text)
	}
	if msg.UserID != validUserID || msg.EventID != 7 {
		t.Errorf("message = %+v", msg)
	}
}

func TestR11_SignUpCheckError(t *testing.T) {
	repo := &fakeMessages{signUpErr: errors.New("db down")}
	if _, err := newService(repo, true).SendMessage(context.Background(), validUserID, 7, "hola"); err == nil || errors.Is(err, ErrNotSignedUp) {
		t.Fatalf("err = %v, want the repository error", err)
	}
	if repo.createCalls != 0 {
		t.Fatal("message stored although the sign up could not be checked")
	}
}

func TestR1_DuplicateMessage(t *testing.T) {
	repo := &fakeMessages{signedUp: true, createErr: message_repository.ErrDuplicateMessage}
	if _, err := newService(repo, true).SendMessage(context.Background(), validUserID, 7, "hola"); !errors.Is(err, ErrDuplicateMessage) {
		t.Fatalf("err = %v, want ErrDuplicateMessage", err)
	}
}

func TestSendMessage_InvalidText(t *testing.T) {
	for name, text := range map[string]string{
		"empty":       "",
		"only spaces": " \n\t ",
		"too long":    strings.Repeat("a", message_model.MaxTextLength+1),
	} {
		t.Run(name, func(t *testing.T) {
			repo := &fakeMessages{signedUp: true}
			if _, err := newService(repo, true).SendMessage(context.Background(), validUserID, 7, text); !errors.Is(err, ErrInvalidText) {
				t.Fatalf("err = %v, want ErrInvalidText", err)
			}
			if repo.createCalls != 0 {
				t.Fatal("invalid message stored")
			}
		})
	}
}

func TestSendMessage_MaxLengthCountsCharacters(t *testing.T) {
	repo := &fakeMessages{signedUp: true}
	text := strings.Repeat("à", message_model.MaxTextLength) // 2 bytes per character
	if _, err := newService(repo, true).SendMessage(context.Background(), validUserID, 7, text); err != nil {
		t.Fatalf("err = %v, want nil for %d characters", err, message_model.MaxTextLength)
	}
}

func TestMessages_UnknownEvent(t *testing.T) {
	repo := &fakeMessages{signedUp: true}
	svc := newService(repo, false)
	if _, err := svc.SendMessage(context.Background(), validUserID, 7, "hola"); !errors.Is(err, ErrEventNotFound) {
		t.Fatalf("SendMessage err = %v, want ErrEventNotFound", err)
	}
	if _, err := svc.ListMessages(context.Background(), validUserID, 7, 0, 0); !errors.Is(err, ErrEventNotFound) {
		t.Fatalf("ListMessages err = %v, want ErrEventNotFound", err)
	}
}

func TestSendMessage_EventDeletedBetweenChecks(t *testing.T) {
	repo := &fakeMessages{signedUp: true, createErr: message_repository.ErrEventNotFound}
	if _, err := newService(repo, true).SendMessage(context.Background(), validUserID, 7, "hola"); !errors.Is(err, ErrEventNotFound) {
		t.Fatalf("err = %v, want ErrEventNotFound", err)
	}
}

func TestMessages_InvalidIDs(t *testing.T) {
	svc := newService(&fakeMessages{signedUp: true}, true)
	if _, err := svc.SendMessage(context.Background(), "not-a-uuid", 7, "hola"); !errors.Is(err, ErrInvalidUserID) {
		t.Errorf("bad user: err = %v", err)
	}
	if _, err := svc.SendMessage(context.Background(), validUserID, 0, "hola"); !errors.Is(err, ErrInvalidEventID) {
		t.Errorf("bad event: err = %v", err)
	}
	if _, err := svc.ListMessages(context.Background(), validUserID, -1, 0, 0); !errors.Is(err, ErrInvalidEventID) {
		t.Errorf("bad event on list: err = %v", err)
	}
}

func TestListMessages_Pagination(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		repo := &fakeMessages{signedUp: true, total: 2, listed: []message_model.Message{{Text: "b"}, {Text: "a"}}}
		page, err := newService(repo, true).ListMessages(context.Background(), validUserID, 7, 0, 0)
		if err != nil {
			t.Fatal(err)
		}
		if page.PageSize != DefaultPageSize || page.PageNum != 0 || page.Total != 2 || len(page.Messages) != 2 || page.EventID != 7 {
			t.Errorf("page = %+v", page)
		}
		if repo.gotLimit != DefaultPageSize || repo.gotOffset != 0 {
			t.Errorf("limit/offset = %d/%d", repo.gotLimit, repo.gotOffset)
		}
	})
	t.Run("offset", func(t *testing.T) {
		repo := &fakeMessages{signedUp: true, total: 30}
		if _, err := newService(repo, true).ListMessages(context.Background(), validUserID, 7, 2, 10); err != nil {
			t.Fatal(err)
		}
		if repo.gotLimit != 10 || repo.gotOffset != 20 {
			t.Errorf("limit/offset = %d/%d, want 10/20", repo.gotLimit, repo.gotOffset)
		}
	})
	t.Run("empty chat skips the list query", func(t *testing.T) {
		repo := &fakeMessages{signedUp: true}
		page, err := newService(repo, true).ListMessages(context.Background(), validUserID, 7, 0, 0)
		if err != nil {
			t.Fatal(err)
		}
		if page.Messages == nil || len(page.Messages) != 0 || repo.listCalls != 0 {
			t.Errorf("page = %+v, listCalls = %d", page, repo.listCalls)
		}
	})
	for name, p := range map[string][2]int{
		"negative page":   {-1, 10},
		"negative size":   {0, -1},
		"size over max":   {0, MaxPageSize + 1},
		"offset overflow": {1 << 40, MaxPageSize},
	} {
		t.Run(name, func(t *testing.T) {
			repo := &fakeMessages{signedUp: true}
			if _, err := newService(repo, true).ListMessages(context.Background(), validUserID, 7, p[0], p[1]); !errors.Is(err, ErrInvalidPagination) {
				t.Fatalf("err = %v, want ErrInvalidPagination", err)
			}
		})
	}
}
