package notification_service

import (
	"errors"
	"reflect"
	"testing"

	"api-aprop/internal/models/notification_model"
	"api-aprop/internal/repositories/notification_repository"
	"github.com/jackc/pgx/v5/pgtype"
)

type fakeNotificationRepository struct {
	getFunc                       func(int) (notification_model.Notification, error)
	createFunc                    func(notification_model.Notification) (int, error)
	getUserNotificationsFunc      func(pgtype.UUID, map[string]any) ([]notification_model.Notification, error)
	readNotificationFunc          func(int) error
	readUserNotificationsFunc     func(pgtype.UUID) error
	getNotificationTypesFunc      func() ([]notification_model.NotificationType, error)
	getNotificationTypeByNameFunc func(string) (int, error)
	createRemindersFunc           func() ([]int32, error)
	createFriendSignupFunc        func(pgtype.UUID, int, string) ([]int32, error)
}

func (f *fakeNotificationRepository) Get(id int) (notification_model.Notification, error) {
	if f.getFunc != nil {
		return f.getFunc(id)
	}
	return notification_model.Notification{}, nil
}

func (f *fakeNotificationRepository) Create(n notification_model.Notification) (int, error) {
	if f.createFunc != nil {
		return f.createFunc(n)
	}
	return 0, nil
}

func (f *fakeNotificationRepository) GetUserNotifications(id pgtype.UUID, params map[string]any) ([]notification_model.Notification, error) {
	if f.getUserNotificationsFunc != nil {
		return f.getUserNotificationsFunc(id, params)
	}
	return nil, nil
}

func (f *fakeNotificationRepository) ReadNotification(id int) error {
	if f.readNotificationFunc != nil {
		return f.readNotificationFunc(id)
	}
	return nil
}

func (f *fakeNotificationRepository) ReadUserNotifications(id pgtype.UUID) error {
	if f.readUserNotificationsFunc != nil {
		return f.readUserNotificationsFunc(id)
	}
	return nil
}

func (f *fakeNotificationRepository) GetNotificationTypes() ([]notification_model.NotificationType, error) {
	if f.getNotificationTypesFunc != nil {
		return f.getNotificationTypesFunc()
	}
	return nil, nil
}

func (f *fakeNotificationRepository) GetNotificationTypeByName(name string) (int, error) {
	if f.getNotificationTypeByNameFunc != nil {
		return f.getNotificationTypeByNameFunc(name)
	}
	return 0, nil
}

func (f *fakeNotificationRepository) CreateReminders() ([]int32, error) {
	if f.createRemindersFunc != nil {
		return f.createRemindersFunc()
	}
	return nil, nil
}

func (f *fakeNotificationRepository) CreateFriendSignupNotifications(id pgtype.UUID, event int, text string) ([]int32, error) {
	if f.createFriendSignupFunc != nil {
		return f.createFriendSignupFunc(id, event, text)
	}
	return nil, nil
}

var (
	errRepository     = errors.New("repository error")
	validUUID         = pgtype.UUID{Bytes: [16]byte{1, 2, 3}, Valid: true}
	validNotification = notification_model.Notification{
		ID:     1,
		UserID: validUUID,
		Type:   1,
		Read:   false,
		Text:   "Test notification",
	}
)

// --------------------------------------------------
// GetNotification
// --------------------------------------------------

func TestGetNotification(t *testing.T) {
	tests := []struct {
		name    string
		id      int
		repo    *fakeNotificationRepository
		want    notification_model.Notification
		wantErr error
	}{
		{
			name: "valid notification",
			id:   1,
			repo: &fakeNotificationRepository{
				getFunc: func(id int) (notification_model.Notification, error) {
					return validNotification, nil
				},
			},
			want: validNotification,
		},
		{
			name: "notification not found",
			id:   999,
			repo: &fakeNotificationRepository{
				getFunc: func(id int) (notification_model.Notification, error) {
					return notification_model.Notification{}, notification_repository.ErrNotificationNotFound
				},
			},
			wantErr: ErrNotificationNotFound,
		},
		{
			name: "repository error",
			id:   1,
			repo: &fakeNotificationRepository{
				getFunc: func(id int) (notification_model.Notification, error) {
					return notification_model.Notification{}, errRepository
				},
			},
			wantErr: errRepository,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewNotificationService(tt.repo)

			got, err := svc.GetNotification(tt.id)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}

			if tt.wantErr != nil {
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("notification = %+v, want %+v", got, tt.want)
			}
		})
	}
}

// --------------------------------------------------
// GetNotificationTypes
// --------------------------------------------------

func TestGetNotificationTypes(t *testing.T) {
	validTypes := []notification_model.NotificationType{
		{ID: 1, Name: "Friend request"},
		{ID: 2, Name: "Event reminder"},
	}

	tests := []struct {
		name    string
		repo    *fakeNotificationRepository
		want    []notification_model.NotificationType
		wantErr error
	}{
		{
			name: "valid notification types",
			repo: &fakeNotificationRepository{
				getNotificationTypesFunc: func() ([]notification_model.NotificationType, error) {
					return validTypes, nil
				},
			},
			want: validTypes,
		},
		{
			name: "empty notification types",
			repo: &fakeNotificationRepository{
				getNotificationTypesFunc: func() ([]notification_model.NotificationType, error) {
					return []notification_model.NotificationType{}, nil
				},
			},
			want: []notification_model.NotificationType{},
		},
		{
			name: "repository error",
			repo: &fakeNotificationRepository{
				getNotificationTypesFunc: func() ([]notification_model.NotificationType, error) {
					return nil, errRepository
				},
			},
			want:    []notification_model.NotificationType{},
			wantErr: errRepository,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewNotificationService(tt.repo)

			got, err := svc.GetNotificationTypes()

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("notification types = %+v, want %+v", got, tt.want)
			}
		})
	}
}

// --------------------------------------------------
// GetUserNotifications
// --------------------------------------------------

func TestGetUserNotifications(t *testing.T) {
	params := map[string]any{
		"limit":  10,
		"offset": 0,
	}

	validNotifications := []notification_model.Notification{
		validNotification,
		{ID: 2, UserID: validUUID, Type: 2, Text: "Another notification"},
	}

	tests := []struct {
		name    string
		userID  pgtype.UUID
		params  map[string]any
		repo    *fakeNotificationRepository
		want    []notification_model.Notification
		wantErr error
	}{
		{
			name:   "valid user notifications",
			userID: validUUID,
			params: params,
			repo: &fakeNotificationRepository{
				getUserNotificationsFunc: func(id pgtype.UUID, p map[string]any) ([]notification_model.Notification, error) {
					return validNotifications, nil
				},
			},
			want: validNotifications,
		},
		{
			name:   "user with no notifications",
			userID: validUUID,
			params: params,
			repo: &fakeNotificationRepository{
				getUserNotificationsFunc: func(id pgtype.UUID, p map[string]any) ([]notification_model.Notification, error) {
					return []notification_model.Notification{}, nil
				},
			},
			want: []notification_model.Notification{},
		},
		{
			name:   "user not found",
			userID: validUUID,
			params: params,
			repo: &fakeNotificationRepository{
				getUserNotificationsFunc: func(id pgtype.UUID, p map[string]any) ([]notification_model.Notification, error) {
					return nil, notification_repository.ErrUserNotFound
				},
			},
			want:    []notification_model.Notification{},
			wantErr: ErrUserNotFound,
		},
		{
			name:   "repository error",
			userID: validUUID,
			params: params,
			repo: &fakeNotificationRepository{
				getUserNotificationsFunc: func(id pgtype.UUID, p map[string]any) ([]notification_model.Notification, error) {
					return nil, errRepository
				},
			},
			want:    []notification_model.Notification{},
			wantErr: errRepository,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewNotificationService(tt.repo)

			got, err := svc.GetUserNotifications(tt.userID, tt.params)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("notifications = %+v, want %+v", got, tt.want)
			}
		})
	}
}

// --------------------------------------------------
// CreateNotification
// --------------------------------------------------

func TestCreateNotification(t *testing.T) {
	tests := []struct {
		name    string
		input   notification_model.Notification
		repo    *fakeNotificationRepository
		want    int
		wantErr error
	}{
		{
			name:  "valid notification",
			input: validNotification,
			repo: &fakeNotificationRepository{
				createFunc: func(n notification_model.Notification) (int, error) {
					return 123, nil
				},
			},
			want: 123,
		},
		{
			name:  "repository error",
			input: validNotification,
			repo: &fakeNotificationRepository{
				createFunc: func(n notification_model.Notification) (int, error) {
					return 0, errRepository
				},
			},
			wantErr: errRepository,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewNotificationService(tt.repo)

			got, err := svc.CreateNotification(tt.input)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}

			if got != tt.want {
				t.Errorf("notification ID = %d, want %d", got, tt.want)
			}
		})
	}
}

// --------------------------------------------------
// ReadNotification
// --------------------------------------------------

func TestReadNotification(t *testing.T) {
	tests := []struct {
		name    string
		id      int
		repo    *fakeNotificationRepository
		wantErr error
	}{
		{
			name: "valid notification read",
			id:   1,
			repo: &fakeNotificationRepository{
				getFunc: func(id int) (notification_model.Notification, error) {
					return validNotification, nil
				},
				readNotificationFunc: func(id int) error {
					return nil
				},
			},
		},
		{
			name: "notification not found",
			id:   999,
			repo: &fakeNotificationRepository{
				getFunc: func(id int) (notification_model.Notification, error) {
					return notification_model.Notification{}, notification_repository.ErrNotificationNotFound
				},
			},
			wantErr: ErrNotificationNotFound,
		},
		{
			name: "error retrieving notification",
			id:   1,
			repo: &fakeNotificationRepository{
				getFunc: func(id int) (notification_model.Notification, error) {
					return notification_model.Notification{}, errRepository
				},
			},
			wantErr: errRepository,
		},
		{
			name: "error marking notification as read",
			id:   1,
			repo: &fakeNotificationRepository{
				getFunc: func(id int) (notification_model.Notification, error) {
					return validNotification, nil
				},
				readNotificationFunc: func(id int) error {
					return errRepository
				},
			},
			wantErr: errRepository,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewNotificationService(tt.repo)

			err := svc.ReadNotification(tt.id)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

// --------------------------------------------------
// ReadUserNotifications
// --------------------------------------------------

func TestReadUserNotifications(t *testing.T) {
	tests := []struct {
		name    string
		userID  pgtype.UUID
		repo    *fakeNotificationRepository
		wantErr error
	}{
		{
			name:   "valid user notifications read",
			userID: validUUID,
			repo: &fakeNotificationRepository{
				readUserNotificationsFunc: func(id pgtype.UUID) error {
					return nil
				},
			},
		},
		{
			name:   "user not found",
			userID: validUUID,
			repo: &fakeNotificationRepository{
				readUserNotificationsFunc: func(id pgtype.UUID) error {
					return notification_repository.ErrUserNotFound
				},
			},
			wantErr: ErrUserNotFound,
		},
		{
			name:   "repository error",
			userID: validUUID,
			repo: &fakeNotificationRepository{
				readUserNotificationsFunc: func(id pgtype.UUID) error {
					return errRepository
				},
			},
			wantErr: errRepository,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewNotificationService(tt.repo)

			err := svc.ReadUserNotifications(tt.userID)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

// --------------------------------------------------
// CreateReminders
// --------------------------------------------------

func TestCreateReminders(t *testing.T) {
	validIDs := []int32{1, 2, 3}

	tests := []struct {
		name    string
		repo    *fakeNotificationRepository
		want    []int32
		wantErr error
	}{
		{
			name: "valid reminders created",
			repo: &fakeNotificationRepository{
				createRemindersFunc: func() ([]int32, error) {
					return validIDs, nil
				},
			},
			want: validIDs,
		},
		{
			name: "no reminders created",
			repo: &fakeNotificationRepository{
				createRemindersFunc: func() ([]int32, error) {
					return []int32{}, nil
				},
			},
			want: []int32{},
		},
		{
			name: "repository error",
			repo: &fakeNotificationRepository{
				createRemindersFunc: func() ([]int32, error) {
					return nil, errRepository
				},
			},
			want:    []int32{},
			wantErr: errRepository,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewNotificationService(tt.repo)

			got, err := svc.CreateReminders()

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("reminder IDs = %v, want %v", got, tt.want)
			}
		})
	}
}
