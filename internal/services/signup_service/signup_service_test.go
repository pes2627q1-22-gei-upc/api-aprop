package signup_service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"api-aprop/internal/models/event_model"
	"api-aprop/internal/models/notification_model"
	"api-aprop/internal/models/signup_model"
	"api-aprop/internal/repositories/signup_repository"
)

const validUserID = "3f2504e0-4f89-11d3-9a0c-0305e82c3301"

type fakeEventRepository struct {
	exists    bool
	existsErr error

	gotEventID int
}

type fakeRepository struct {
	created      bool
	createErr    error
	deleted      bool
	deleteErr    error
	countSignUps int
	countErr     error
	isSignedUp   bool
	isSignedErr  error
	events       []event_model.Event
	listErr      error

	createCalls int
	deleteCalls int
	listCalls   int
	gotUserID   string
	gotEventID  int
	gotRange    signup_model.DateRange
}

func (f *fakeEventRepository) EventExists(eventID int) (bool, error) {
	f.gotEventID = eventID
	return f.exists, f.existsErr
}

func (f *fakeEventRepository) Create(e event_model.Event, official bool) (event_model.Event, error) {
	return event_model.Event{}, nil
}
func (f *fakeEventRepository) CreateOrReplace(e event_model.Event) (event_model.Event, error) {
	return event_model.Event{}, nil
}
func (f *fakeEventRepository) Update(id int, e event_model.Event, official bool) (event_model.Event, error) {
	return event_model.Event{}, nil
}
func (f *fakeEventRepository) UpdateImage(ctx context.Context, id int, imageURL string) error {
	return nil
}
func (f *fakeEventRepository) Delete(ctx context.Context, id int) error                 { return nil }
func (f *fakeEventRepository) SetCategories(id int, cats []string, official bool) error { return nil }
func (f *fakeEventRepository) Get(id int) (event_model.Event, error) {
	return event_model.Event{EndDate: time.Now().Add(time.Hour * time.Duration(24))}, nil
}
func (f *fakeEventRepository) List(params map[string]any) ([]event_model.Event, error) {
	return []event_model.Event{}, nil
}
func (f *fakeEventRepository) GetPopular(pageSize int, pageNum int) ([]event_model.Event, error) {
	return []event_model.Event{}, nil
}
func (f *fakeEventRepository) Search(ctx context.Context, query string, pageNum, pageSize int) (event_model.SearchResult, error) {
	return event_model.SearchResult{}, nil
}
func (f *fakeEventRepository) GetCategories() ([]event_model.Category, error) {
	return []event_model.Category{}, nil
}
func (f *fakeEventRepository) GetCategory(id int) (event_model.Category, error) {
	return event_model.Category{}, nil
}
func (f *fakeEventRepository) ListMapEvents(ctx context.Context, filter event_model.MapFilter) ([]event_model.MapEvent, error) {
	return []event_model.MapEvent{}, nil
}

type fakeNotificationRepository struct {
	created_on_signup   int
	created_on_unenroll int
}

func (f *fakeNotificationRepository) Get(id int) (notification_model.Notification, error) {
	return notification_model.Notification{}, nil
}
func (f *fakeNotificationRepository) Create(notification_model.Notification) (int, error) {
	return 0, nil
}
func (f *fakeNotificationRepository) GetUserNotifications(id pgtype.UUID, params map[string]any) ([]notification_model.Notification, error) {
	return []notification_model.Notification{}, nil
}
func (f *fakeNotificationRepository) ReadNotification(id int) error              { return nil }
func (f *fakeNotificationRepository) ReadUserNotifications(id pgtype.UUID) error { return nil }
func (f *fakeNotificationRepository) GetNotificationTypes() ([]notification_model.NotificationType, error) {
	return []notification_model.NotificationType{}, nil
}
func (f *fakeNotificationRepository) GetNotificationTypeByName(name string) (int, error) {
	return 0, nil
}
func (f *fakeNotificationRepository) CreateReminders() ([]int32, error) { return []int32{}, nil }
func (f *fakeNotificationRepository) CreateFriendSignupNotifications(id pgtype.UUID, event int, text string) ([]int32, error) {
	if text == "apuntat a" {
		f.created_on_signup++
	} else if text == "desapuntat de" {
		f.created_on_unenroll++
	}
	return []int32{}, nil
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

func (f *fakeRepository) CountSignUps(_ context.Context, eventID int) (int, error) {
	f.gotEventID = eventID
	return f.countSignUps, f.countErr
}

func (f *fakeRepository) IsSignedUp(_ context.Context, userID string, eventID int) (bool, error) {
	f.gotUserID = userID
	f.gotEventID = eventID
	return f.isSignedUp, f.isSignedErr
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
		eventRepo   fakeEventRepository
		wantErr     error
		wantCreated bool
		wantCreate  int
	}{
		{name: "new sign up", userID: validUserID, eventID: 1, repo: fakeRepository{created: true}, eventRepo: fakeEventRepository{exists: true}, wantCreated: true, wantCreate: 1},
		{name: "already signed up", userID: validUserID, eventID: 1, repo: fakeRepository{created: false}, eventRepo: fakeEventRepository{exists: true}, wantCreate: 1},
		{name: "event not found", userID: validUserID, eventID: 99, repo: fakeRepository{}, eventRepo: fakeEventRepository{exists: false}, wantErr: ErrEventNotFound},
		{name: "event deleted concurrently", userID: validUserID, eventID: 1, repo: fakeRepository{createErr: signup_repository.ErrEventNotFound}, eventRepo: fakeEventRepository{exists: true}, wantErr: ErrEventNotFound, wantCreate: 1},
		{name: "empty user id", userID: "", eventID: 1, repo: fakeRepository{}, eventRepo: fakeEventRepository{exists: true}, wantErr: ErrInvalidUserID},
		{name: "malformed user id", userID: "not-a-uuid", eventID: 1, repo: fakeRepository{}, eventRepo: fakeEventRepository{exists: true}, wantErr: ErrInvalidUserID},
		{name: "user id without dashes", userID: "3f2504e04f8911d39a0c0305e82c3301", eventID: 1, repo: fakeRepository{}, eventRepo: fakeEventRepository{exists: true}, wantErr: ErrInvalidUserID},
		{name: "user id with non hex digit", userID: "3f2504e0-4f89-11d3-9a0c-0305e82c330z", eventID: 1, repo: fakeRepository{}, eventRepo: fakeEventRepository{exists: true}, wantErr: ErrInvalidUserID},
		{name: "non positive event id", userID: validUserID, eventID: 0, repo: fakeRepository{}, eventRepo: fakeEventRepository{exists: true}, wantErr: ErrInvalidEventID},
		{name: "exists check fails", userID: validUserID, eventID: 1, repo: fakeRepository{}, eventRepo: fakeEventRepository{existsErr: repoErr}, wantErr: repoErr},
		{name: "create fails", userID: validUserID, eventID: 1, repo: fakeRepository{createErr: repoErr}, eventRepo: fakeEventRepository{exists: true}, wantErr: repoErr, wantCreate: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.repo
			eventRepo := tt.eventRepo
			s := NewSignUpService(&repo, &eventRepo, &fakeNotificationRepository{})

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
	repo := fakeRepository{created: true}
	eventRepo := fakeEventRepository{exists: true}
	s := NewSignUpService(&repo, &eventRepo, &fakeNotificationRepository{})

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
		eventRepo fakeEventRepository
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
			eventRepo := fakeEventRepository{}
			s := NewSignUpService(&repo, &eventRepo, &fakeNotificationRepository{})

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
		name        string
		userID      string
		eventID     int
		repo        fakeRepository
		eventRepo   fakeEventRepository
		wantErr     error
		wantDeleted bool
		wantDelete  int
	}{
		{name: "successful unenroll", userID: validUserID, eventID: 1, repo: fakeRepository{deleted: true}, eventRepo: fakeEventRepository{exists: true}, wantDeleted: true, wantDelete: 1},
		{name: "not signed up", userID: validUserID, eventID: 1, repo: fakeRepository{deleted: false}, eventRepo: fakeEventRepository{exists: true}, wantDeleted: false, wantDelete: 1},
		{name: "event not found", userID: validUserID, eventID: 99, repo: fakeRepository{}, eventRepo: fakeEventRepository{exists: false}, wantErr: ErrEventNotFound},
		{name: "empty user id", userID: "", eventID: 1, repo: fakeRepository{}, eventRepo: fakeEventRepository{exists: true}, wantErr: ErrInvalidUserID},
		{name: "malformed user id", userID: "invalid", eventID: 1, repo: fakeRepository{}, eventRepo: fakeEventRepository{exists: true}, wantErr: ErrInvalidUserID},
		{name: "invalid event id", userID: validUserID, eventID: 0, repo: fakeRepository{}, eventRepo: fakeEventRepository{exists: true}, wantErr: ErrInvalidEventID},
		{name: "delete repository error", userID: validUserID, eventID: 1, repo: fakeRepository{deleteErr: repoErr}, eventRepo: fakeEventRepository{exists: true}, wantErr: repoErr, wantDelete: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.repo
			eventRepo := tt.eventRepo
			s := NewSignUpService(&repo, &eventRepo, &fakeNotificationRepository{})

			got, err := s.Unenroll(context.Background(), tt.userID, tt.eventID)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if repo.deleteCalls != tt.wantDelete {
				t.Errorf("Delete calls = %d, want %d", repo.deleteCalls, tt.wantDelete)
			}
			if tt.wantErr != nil {
				return
			}
			want := signup_model.Unenroll{UserID: tt.userID, EventID: tt.eventID, Deleted: tt.wantDeleted}
			if got != want {
				t.Errorf("Unenroll = %+v, want %+v", got, want)
			}
		})
	}
}

func TestGetEnrollmentStatus(t *testing.T) {
	repoErr := errors.New("db down")
	tests := []struct {
		name       string
		userID     string
		eventID    int
		repo       fakeRepository
		eventRepo  fakeEventRepository
		wantErr    error
		wantStatus signup_model.EnrollmentStatus
	}{
		{
			name:       "enrolled with count",
			userID:     validUserID,
			eventID:    1,
			repo:       fakeRepository{countSignUps: 42, isSignedUp: true},
			eventRepo:  fakeEventRepository{exists: true},
			wantStatus: signup_model.EnrollmentStatus{EventID: 1, IsEnrolled: true, EnrolledCount: 42},
		},
		{
			name:       "not enrolled with count",
			userID:     validUserID,
			eventID:    1,
			repo:       fakeRepository{countSignUps: 15, isSignedUp: false},
			eventRepo:  fakeEventRepository{exists: true},
			wantStatus: signup_model.EnrollmentStatus{EventID: 1, IsEnrolled: false, EnrolledCount: 15},
		},
		{
			name:       "unauthenticated request gets count only",
			userID:     "",
			eventID:    1,
			repo:       fakeRepository{countSignUps: 10},
			eventRepo:  fakeEventRepository{exists: true},
			wantStatus: signup_model.EnrollmentStatus{EventID: 1, IsEnrolled: false, EnrolledCount: 10},
		},
		{
			name:      "event not found",
			userID:    validUserID,
			eventID:   999,
			repo:      fakeRepository{},
			eventRepo: fakeEventRepository{exists: false},
			wantErr:   ErrEventNotFound,
		},
		{
			name:      "invalid event id",
			userID:    validUserID,
			eventID:   -1,
			repo:      fakeRepository{},
			eventRepo: fakeEventRepository{exists: true},
			wantErr:   ErrInvalidEventID,
		},
		{
			name:      "count error",
			userID:    validUserID,
			eventID:   1,
			repo:      fakeRepository{countErr: repoErr},
			eventRepo: fakeEventRepository{exists: true},
			wantErr:   repoErr,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.repo
			eventRepo := tt.eventRepo
			s := NewSignUpService(&repo, &eventRepo, &fakeNotificationRepository{})

			got, err := s.GetEnrollmentStatus(context.Background(), tt.userID, tt.eventID)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				return
			}
			if got != tt.wantStatus {
				t.Errorf("GetEnrollmentStatus = %+v, want %+v", got, tt.wantStatus)
			}
		})
	}
}

func TestSignupNotifications(t *testing.T) {
	tests := []struct {
		name               string
		userID             string
		eventID            int
		repo               fakeRepository
		eventRepo          fakeEventRepository
		wantErr            error
		wantCreated        bool
		wantCreate         int
		notifRepo          fakeNotificationRepository
		wantSignupNotifs   int
		wantUnenrollNotifs int
	}{
		{name: "new sign up", userID: validUserID, eventID: 1, repo: fakeRepository{created: true}, eventRepo: fakeEventRepository{exists: true}, wantCreated: true, wantCreate: 1, notifRepo: fakeNotificationRepository{}, wantSignupNotifs: 1, wantUnenrollNotifs: 0},
		{name: "signup fails", userID: "bad-id", eventID: 1, repo: fakeRepository{created: true}, eventRepo: fakeEventRepository{exists: true}, wantErr: ErrInvalidUserID, wantCreated: false, wantCreate: 0, notifRepo: fakeNotificationRepository{}, wantSignupNotifs: 0, wantUnenrollNotifs: 0},
		{name: "signup already exists", userID: validUserID, eventID: 1, repo: fakeRepository{created: false}, eventRepo: fakeEventRepository{exists: true}, wantCreated: false, wantCreate: 1, notifRepo: fakeNotificationRepository{}, wantSignupNotifs: 0, wantUnenrollNotifs: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.repo
			eventRepo := tt.eventRepo
			s := NewSignUpService(&repo, &eventRepo, &tt.notifRepo)

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
			if tt.notifRepo.created_on_signup != tt.wantSignupNotifs {
				t.Errorf("Create notification calls for signup = %d, want %d", tt.notifRepo.created_on_signup, tt.wantSignupNotifs)
			}
			if tt.notifRepo.created_on_unenroll != tt.wantUnenrollNotifs {
				t.Errorf("Create notification calls for unenroll = %d, want %d", tt.notifRepo.created_on_unenroll, tt.wantUnenrollNotifs)
			}
		})
	}
}

func TestUnenrollNotifications(t *testing.T) {
	tests := []struct {
		name               string
		userID             string
		eventID            int
		repo               fakeRepository
		eventRepo          fakeEventRepository
		wantErr            error
		wantDeleted        bool
		wantDelete         int
		notifRepo          fakeNotificationRepository
		wantSignupNotifs   int
		wantUnenrollNotifs int
	}{
		{name: "new unenroll", userID: validUserID, eventID: 1, repo: fakeRepository{deleted: true}, eventRepo: fakeEventRepository{exists: true}, wantDeleted: true, wantDelete: 1, notifRepo: fakeNotificationRepository{}, wantSignupNotifs: 0, wantUnenrollNotifs: 1},
		{name: "unenroll fails", userID: "bad-id", eventID: 1, repo: fakeRepository{deleted: true}, eventRepo: fakeEventRepository{exists: true}, wantErr: ErrInvalidUserID, wantDeleted: false, wantDelete: 0, notifRepo: fakeNotificationRepository{}, wantSignupNotifs: 0, wantUnenrollNotifs: 0},
		{name: "not enrolled", userID: validUserID, eventID: 1, repo: fakeRepository{deleted: false}, eventRepo: fakeEventRepository{exists: true}, wantDeleted: false, wantDelete: 1, notifRepo: fakeNotificationRepository{}, wantSignupNotifs: 0, wantUnenrollNotifs: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.repo
			eventRepo := tt.eventRepo
			s := NewSignUpService(&repo, &eventRepo, &tt.notifRepo)

			got, err := s.Unenroll(context.Background(), tt.userID, tt.eventID)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if repo.deleteCalls != tt.wantDelete {
				t.Errorf("Delete calls = %d, want %d", repo.deleteCalls, tt.wantDelete)
			}
			if tt.wantErr != nil {
				return
			}
			want := signup_model.Unenroll{UserID: tt.userID, EventID: tt.eventID, Deleted: tt.wantDeleted}
			if got != want {
				t.Errorf("Unenroll = %+v, want %+v", got, want)
			}
			if tt.notifRepo.created_on_signup != tt.wantSignupNotifs {
				t.Errorf("Create notification calls for signup = %d, want %d", tt.notifRepo.created_on_signup, tt.wantSignupNotifs)
			}
			if tt.notifRepo.created_on_unenroll != tt.wantUnenrollNotifs {
				t.Errorf("Create notification calls for unenroll = %d, want %d", tt.notifRepo.created_on_unenroll, tt.wantUnenrollNotifs)
			}
		})
	}
}
