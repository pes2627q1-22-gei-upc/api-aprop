package signup_service

import (
	"context"
	"errors"
	"testing"
	"time"

	"api-aprop/internal/models/event_model"
	"api-aprop/internal/models/signup_model"
	"api-aprop/internal/repositories/signup_repository"
)

const validUserID = "3f2504e0-4f89-11d3-9a0c-0305e82c3301"

type fakeRepository struct {
	deleted     bool
	deleteErr   error
	deleteCalls int
	exists      bool
	existsErr   error
	created     bool
	createErr   error
	events      []event_model.Event
	listErr     error

	createCalls int
	listCalls   int
	gotUserID   string
	gotEventID  int
	gotRange    signup_model.DateRange
}

func (f *fakeRepository) EventExists(_ context.Context, eventID int) (bool, error) {
	f.gotEventID = eventID
	return f.exists, f.existsErr
}

func (f *fakeRepository) Create(_ context.Context, userID string, eventID int) (bool, error) {
	f.createCalls++
	f.gotUserID = userID
	f.gotEventID = eventID
	return f.created, f.createErr
}

func (f *fakeRepository) Delete(_ context.Context, userID string, eventID int) (bool, error) {
	f.deleteCalls++
	f.gotUserID = userID
	f.gotEventID = eventID
	return f.deleted, f.deleteErr
}

func (f *fakeRepository) ListUserEvents(_ context.Context, userID string, dateRange signup_model.DateRange) ([]event_model.Event, error) {
	f.listCalls++
	f.gotUserID = userID
	f.gotRange = dateRange
	return f.events, f.listErr
}

func date(s string) *time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return &t
}

func TestSignUp(t *testing.T) {
	repoErr := errors.New("db down")
	tests := []struct {
		name        string
		userID      string
		eventID     int
		repo        fakeRepository
		wantErr     error
		wantCreated bool
		wantCreate  int
	}{
		{name: "new sign up", userID: validUserID, eventID: 1, repo: fakeRepository{exists: true, created: true}, wantCreated: true, wantCreate: 1},
		{name: "already signed up", userID: validUserID, eventID: 1, repo: fakeRepository{exists: true, created: false}, wantCreate: 1},
		{name: "event not found", userID: validUserID, eventID: 99, repo: fakeRepository{exists: false}, wantErr: ErrEventNotFound},
		{name: "event deleted concurrently", userID: validUserID, eventID: 1, repo: fakeRepository{exists: true, createErr: signup_repository.ErrEventNotFound}, wantErr: ErrEventNotFound, wantCreate: 1},
		{name: "empty user id", userID: "", eventID: 1, repo: fakeRepository{exists: true}, wantErr: ErrInvalidUserID},
		{name: "malformed user id", userID: "not-a-uuid", eventID: 1, repo: fakeRepository{exists: true}, wantErr: ErrInvalidUserID},
		{name: "user id without dashes", userID: "3f2504e04f8911d39a0c0305e82c3301", eventID: 1, repo: fakeRepository{exists: true}, wantErr: ErrInvalidUserID},
		{name: "user id with non hex digit", userID: "3f2504e0-4f89-11d3-9a0c-0305e82c330z", eventID: 1, repo: fakeRepository{exists: true}, wantErr: ErrInvalidUserID},
		{name: "non positive event id", userID: validUserID, eventID: 0, repo: fakeRepository{exists: true}, wantErr: ErrInvalidEventID},
		{name: "exists check fails", userID: validUserID, eventID: 1, repo: fakeRepository{existsErr: repoErr}, wantErr: repoErr},
		{name: "create fails", userID: validUserID, eventID: 1, repo: fakeRepository{exists: true, createErr: repoErr}, wantErr: repoErr, wantCreate: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.repo
			s := NewSignUpService(&repo)

			got, err := s.SignUp(context.Background(), tt.userID, tt.eventID)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if repo.createCalls != tt.wantCreate {
				t.Errorf("Create calls = %d, want %d", repo.createCalls, tt.wantCreate)
			}
			if tt.wantErr != nil {
				return
			}
			want := signup_model.SignUp{UserID: tt.userID, EventID: tt.eventID, Created: tt.wantCreated}
			if got != want {
				t.Errorf("SignUp = %+v, want %+v", got, want)
			}
		})
	}
}

func TestSignUpNormalizesUserID(t *testing.T) {
	repo := fakeRepository{exists: true, created: true}
	s := NewSignUpService(&repo)

	got, err := s.SignUp(context.Background(), "3F2504E0-4F89-11D3-9A0C-0305E82C3301", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.UserID != validUserID || repo.gotUserID != validUserID {
		t.Errorf("user id = %q (repository %q), want %q", got.UserID, repo.gotUserID, validUserID)
	}
}

func TestListUserEvents(t *testing.T) {
	repoErr := errors.New("db down")
	ordered := []event_model.Event{{ID: 2, Title: "first"}, {ID: 1, Title: "second"}}
	tests := []struct {
		name      string
		userID    string
		dateRange signup_model.DateRange
		repo      fakeRepository
		wantErr   error
		want      []event_model.Event
		wantCalls int
	}{
		{name: "no range", userID: validUserID, repo: fakeRepository{events: ordered}, want: ordered, wantCalls: 1},
		{name: "from and to", userID: validUserID, dateRange: signup_model.DateRange{From: date("2026-10-01"), To: date("2026-10-31")}, repo: fakeRepository{events: ordered}, want: ordered, wantCalls: 1},
		{name: "same day range", userID: validUserID, dateRange: signup_model.DateRange{From: date("2026-10-19"), To: date("2026-10-19")}, repo: fakeRepository{events: ordered}, want: ordered, wantCalls: 1},
		{name: "only from", userID: validUserID, dateRange: signup_model.DateRange{From: date("2026-10-01")}, repo: fakeRepository{events: ordered}, want: ordered, wantCalls: 1},
		{name: "only to", userID: validUserID, dateRange: signup_model.DateRange{To: date("2026-10-31")}, repo: fakeRepository{events: ordered}, want: ordered, wantCalls: 1},
		{name: "nil list becomes empty", userID: validUserID, repo: fakeRepository{events: nil}, want: []event_model.Event{}, wantCalls: 1},
		{name: "from after to", userID: validUserID, dateRange: signup_model.DateRange{From: date("2026-11-01"), To: date("2026-10-01")}, wantErr: ErrInvalidDateRange},
		{name: "invalid user id", userID: "abc", wantErr: ErrInvalidUserID},
		{name: "repository error", userID: validUserID, repo: fakeRepository{listErr: repoErr}, wantErr: repoErr, wantCalls: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.repo
			s := NewSignUpService(&repo)

			got, err := s.ListUserEvents(context.Background(), tt.userID, tt.dateRange)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if repo.listCalls != tt.wantCalls {
				t.Errorf("ListUserEvents calls = %d, want %d", repo.listCalls, tt.wantCalls)
			}
			if tt.wantErr != nil {
				return
			}
			if repo.gotRange != tt.dateRange {
				t.Errorf("range passed to repository = %+v, want %+v", repo.gotRange, tt.dateRange)
			}
			if got == nil {
				t.Fatal("events is nil, want non-nil slice")
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %d events, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i].ID != tt.want[i].ID {
					t.Errorf("event[%d].ID = %d, want %d", i, got[i].ID, tt.want[i].ID)
				}
			}
		})
	}
}

func TestUnenroll(t *testing.T) {
	repoErr := errors.New("db down")
	tests := []struct {
		name      string
		userID    string
		eventID   int
		repo      fakeRepository
		want      signup_model.Unenroll
		wantErr   error
		wantCalls int
	}{
		{
			name:      "successful unenroll",
			userID:    validUserID,
			eventID:   42,
			repo:      fakeRepository{exists: true, deleted: true},
			want:      signup_model.Unenroll{UserID: validUserID, EventID: 42, Enrolled: false},
			wantCalls: 1,
		},
		{
			name:      "event not found returns ErrEventNotFound",
			userID:    validUserID,
			eventID:   99,
			repo:      fakeRepository{exists: false},
			wantErr:   ErrEventNotFound,
			wantCalls: 0,
		},
		{
			name:      "invalid user id returns ErrInvalidUserID",
			userID:    "invalid-uuid",
			eventID:   42,
			wantErr:   ErrInvalidUserID,
			wantCalls: 0,
		},
		{
			name:      "invalid event id returns ErrInvalidEventID",
			userID:    validUserID,
			eventID:   0,
			wantErr:   ErrInvalidEventID,
			wantCalls: 0,
		},
		{
			name:      "repository delete error is propagated",
			userID:    validUserID,
			eventID:   42,
			repo:      fakeRepository{exists: true, deleteErr: repoErr},
			wantErr:   repoErr,
			wantCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.repo
			s := NewSignUpService(&repo)

			got, err := s.Unenroll(context.Background(), tt.userID, tt.eventID)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if repo.deleteCalls != tt.wantCalls {
				t.Errorf("delete calls = %d, want %d", repo.deleteCalls, tt.wantCalls)
			}
			if tt.wantErr != nil {
				return
			}
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}
