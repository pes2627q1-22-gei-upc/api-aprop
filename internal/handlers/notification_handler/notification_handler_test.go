package notification_handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"api-aprop/internal/handlers/requester"
	"api-aprop/internal/models/notification_model"
	"api-aprop/internal/models/user_model"
	"api-aprop/internal/services/notification_service"
)

const (
	validUserID = "3f2504e0-4f89-11d3-9a0c-0305e82c3301"
	otherUserID = "9b2e6c1a-1d3f-4c5e-8a7b-6f0e2d4c8a19"
)

type fakeNotificationService struct {
	notifications map[int]notification_model.Notification

	getNotificationError       error
	getUserNotificationsError  error
	readNotificationError      error
	readUserNotificationsError error
	getNotificationTypesError  error
}

func (s *fakeNotificationService) GetNotifications() map[int]notification_model.Notification {
	return s.notifications
}

func (s *fakeNotificationService) GetNotification(id int) (notification_model.Notification, error) {
	return s.notifications[id], s.getNotificationError
}
func (s *fakeNotificationService) GetUserNotifications(id pgtype.UUID, params map[string]any) ([]notification_model.Notification, error) {
	notifs := []notification_model.Notification{}
	for _, n := range s.notifications {
		if n.UserID == id {
			notifs = append(notifs, n)
		}
	}
	slices.SortFunc(notifs, func(a notification_model.Notification, b notification_model.Notification) int { return a.ID - b.ID })
	return notifs, s.getUserNotificationsError
}
func (s *fakeNotificationService) CreateNotification(n notification_model.Notification) (int, error) {
	return 0, nil
}
func (s *fakeNotificationService) ReadNotification(id int) error {
	if s.notifications[id] != (notification_model.Notification{}) {
		tmp := s.notifications[id]
		tmp.Read = true
		s.notifications[id] = tmp
	}
	return s.readNotificationError
}
func (s *fakeNotificationService) ReadUserNotifications(id pgtype.UUID) error {
	for i, n := range s.notifications {
		if n.UserID == id {
			tmp := s.notifications[i]
			tmp.Read = true
			s.notifications[i] = tmp
		}
	}
	return s.readUserNotificationsError
}
func (s *fakeNotificationService) GetNotificationTypes() ([]notification_model.NotificationType, error) {
	return []notification_model.NotificationType{
		{ID: 1, Name: "Social"},
		{ID: 2, Name: "Recordatori"},
		{ID: 3, Name: "Recomanacio"},
	}, s.getNotificationTypesError
}
func (s *fakeNotificationService) CreateReminders() ([]int32, error) { return []int32{}, nil }

func newMux(s notification_service.Service) *http.ServeMux {
	h := NewNotificationHandler(s)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	return mux
}

func serve(s notification_service.Service, req *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	newMux(s).ServeHTTP(rec, req)
	return rec
}

func errorMessage(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error body %q: %v", rec.Body.String(), err)
	}
	return body.Error
}

func TestGetNotification(t *testing.T) {
	notifications := make(map[int]notification_model.Notification)
	user1, _ := user_model.ParseID(validUserID)
	user2, _ := user_model.ParseID(otherUserID)
	notifications[1] = notification_model.Notification{ID: 1, UserID: user1, NotificationDate: pgtype.Timestamp{Time: time.Now(), Valid: true}, Friend: user2, Text: "user2 ha acceptat la teva sol·licitud d'amistat"}
	notifications[2] = notification_model.Notification{ID: 2, UserID: user1, NotificationDate: pgtype.Timestamp{Time: time.Now(), Valid: true}, Friend: user2, Event: pgtype.Int4{Int32: 1, Valid: true}, Text: "user2 s'ha apuntat a Esdeveniment 1"}

	tests := []struct {
		name              string
		service           fakeNotificationService
		request           string
		wantStatus        int
		wantError         string
		wantNotifications []notification_model.Notification
	}{
		{"success", fakeNotificationService{notifications: notifications}, "/notifications/1", 200, "", []notification_model.Notification{notifications[1]}},
		{"invalid notification id", fakeNotificationService{notifications: notifications}, "/notifications/abc", 400, "invalid notification id", []notification_model.Notification{}},
		{"notification not found", fakeNotificationService{notifications: notifications, getNotificationError: notification_service.ErrNotificationNotFound}, "/notifications/4", 404, "notification not found", []notification_model.Notification{}},
		{"success, user has notifications", fakeNotificationService{notifications: notifications}, fmt.Sprintf("/notifications/user/%s", validUserID), 200, "", []notification_model.Notification{notifications[1], notifications[2]}},
		{"success, user has no notifications", fakeNotificationService{notifications: notifications}, fmt.Sprintf("/notifications/user/%s", otherUserID), 200, "", []notification_model.Notification{}},
		{"invalid user id", fakeNotificationService{notifications: notifications}, "/notifications/user/abc", 400, "invalid user id", []notification_model.Notification{}},
		{"user not found", fakeNotificationService{notifications: notifications, getUserNotificationsError: notification_service.ErrUserNotFound}, fmt.Sprintf("/notifications/user/%s", validUserID), 404, "user not found", []notification_model.Notification{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.service
			req := httptest.NewRequest(http.MethodGet, tt.request, nil)
			req.Header.Set(requester.UserIDHeader, validUserID)

			rec := serve(&s, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if got := rec.Header().Get("Content-Type"); got != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", got)
			}
			if tt.wantError != "" {
				if got := errorMessage(t, rec); got != tt.wantError {
					t.Errorf("error = %q, want %q", got, tt.wantError)
				}
				return
			}
			var notifs []notification_model.Notification
			if err := json.Unmarshal(rec.Body.Bytes(), &notifs); err != nil {
				var notif notification_model.Notification
				if err := json.Unmarshal(rec.Body.Bytes(), &notif); err != nil {
					t.Fatalf("decode body: %v", err)
				}
				notifs = []notification_model.Notification{notif}
			}
			if len(notifs) != len(tt.wantNotifications) {
				t.Fatalf("got %d notifications, want %d", len(notifs), len(tt.wantNotifications))
			}
			for i, n := range tt.wantNotifications {
				if notifs[i].ID != n.ID {
					t.Errorf("notification[%d].ID = %d, want %d", i, notifs[i].ID, n.ID)
				}
				if notifs[i].UserID != n.UserID {
					t.Errorf("notification[%d].UserID = %s, want %s", i, notifs[i].UserID, n.UserID)
				}
				if notifs[i].Text != n.Text {
					t.Errorf("notification[%d].Text = %s, want %s", i, notifs[i].Text, n.Text)
				}
			}
		})
	}
}

func createNotificationsForTest() map[int]notification_model.Notification {
	notifications := make(map[int]notification_model.Notification)
	user1, _ := user_model.ParseID(validUserID)
	user2, _ := user_model.ParseID(otherUserID)
	notifications[1] = notification_model.Notification{ID: 1, UserID: user1, NotificationDate: pgtype.Timestamp{Time: time.Now(), Valid: true}, Friend: user2, Text: "user2 ha acceptat la teva sol·licitud d'amistat", Read: false}
	notifications[2] = notification_model.Notification{ID: 2, UserID: user1, NotificationDate: pgtype.Timestamp{Time: time.Now(), Valid: true}, Friend: user2, Event: pgtype.Int4{Int32: 1, Valid: true}, Text: "user2 s'ha apuntat a Esdeveniment 1", Read: true}
	notifications[3] = notification_model.Notification{ID: 3, UserID: user1, NotificationDate: pgtype.Timestamp{Time: time.Now(), Valid: true}, Friend: user2, Text: "user2 ha eliminat la seva amistat amb tu", Read: false}
	return notifications
}

func TestReadNotification(t *testing.T) {
	tests := []struct {
		name       string
		service    fakeNotificationService
		request    string
		wantStatus int
		wantError  string
		wantRead   []int
	}{
		{"success", fakeNotificationService{notifications: createNotificationsForTest()}, "/notifications/1/read", 200, "", []int{1, 2}},
		{"invalid notification id", fakeNotificationService{notifications: createNotificationsForTest()}, "/notifications/abc/read", 400, "invalid notification id", []int{2}},
		{"notification not found", fakeNotificationService{notifications: createNotificationsForTest(), readNotificationError: notification_service.ErrNotificationNotFound}, "/notifications/5/read", 404, "notification not found", []int{2}},
		{"already read", fakeNotificationService{notifications: createNotificationsForTest()}, "/notifications/2/read", 200, "", []int{2}},
		{"user read success", fakeNotificationService{notifications: createNotificationsForTest()}, fmt.Sprintf("/notifications/user/%s/read", validUserID), 200, "", []int{1, 2, 3}},
		{"user read with no notifs", fakeNotificationService{notifications: createNotificationsForTest()}, fmt.Sprintf("/notifications/user/%s/read", otherUserID), 200, "", []int{2}},
		{"invalid user id", fakeNotificationService{notifications: createNotificationsForTest()}, "/notifications/user/abc/read", 400, "invalid user id", []int{2}},
		{"user not found", fakeNotificationService{notifications: createNotificationsForTest(), readUserNotificationsError: notification_service.ErrUserNotFound}, fmt.Sprintf("/notifications/user/%s/read", otherUserID), 404, "user not found", []int{2}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.service
			req := httptest.NewRequest(http.MethodPost, tt.request, nil)
			req.Header.Set(requester.UserIDHeader, validUserID)

			rec := serve(&s, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if got := rec.Header().Get("Content-Type"); got != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", got)
			}
			if tt.wantError != "" {
				if got := errorMessage(t, rec); got != tt.wantError {
					t.Errorf("error = %q, want %q", got, tt.wantError)
				}
				return
			}
			notifs := s.GetNotifications()
			for i, n := range notifs {
				if n.Read && !slices.Contains(tt.wantRead, n.ID) {
					t.Errorf("notification[%d].Read = %t, want false", i, notifs[i].Read)
				} else if !n.Read && slices.Contains(tt.wantRead, n.ID) {
					t.Errorf("notification[%d].Read = %t, want true", i, notifs[i].Read)
				}
			}
		})
	}
}

func TestGetNotificationTypes(t *testing.T) {
	dberr := errors.New("databse error")

	tests := []struct {
		name       string
		service    fakeNotificationService
		request    string
		wantStatus int
		wantError  string
		wantTypes  []notification_model.NotificationType
	}{
		{"success", fakeNotificationService{}, "/notificationtypes", 200, "", []notification_model.NotificationType{
			{ID: 1, Name: "Social"},
			{ID: 2, Name: "Recordatori"},
			{ID: 3, Name: "Recomanacio"},
		}},
		{"database error", fakeNotificationService{getNotificationTypesError: dberr}, "/notificationtypes", 500, "internal server error", []notification_model.NotificationType{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.service
			req := httptest.NewRequest(http.MethodGet, tt.request, nil)
			req.Header.Set(requester.UserIDHeader, validUserID)

			rec := serve(&s, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if got := rec.Header().Get("Content-Type"); got != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", got)
			}
			if tt.wantError != "" {
				if got := errorMessage(t, rec); got != tt.wantError {
					t.Errorf("error = %q, want %q", got, tt.wantError)
				}
				return
			}
			var notiftypes []notification_model.NotificationType
			if err := json.Unmarshal(rec.Body.Bytes(), &notiftypes); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if len(notiftypes) != len(tt.wantTypes) {
				t.Fatalf("got %d notification types, want %d", len(notiftypes), len(tt.wantTypes))
			}
			for i, n := range tt.wantTypes {
				if notiftypes[i] != n {
					t.Errorf("notificationtype[%d] = %+v, want %+v", i, notiftypes[i], n)
				}
			}
		})
	}
}
