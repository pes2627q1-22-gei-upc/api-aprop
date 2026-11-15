package notification_handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"api-aprop/internal/models/user_model"
	"api-aprop/internal/models/notification_model"
	"api-aprop/internal/services/notification_service"
	"api-aprop/internal/handlers/requester"
)

const (
	validUserID = "3f2504e0-4f89-11d3-9a0c-0305e82c3301"
	otherUserID = "9b2e6c1a-1d3f-4c5e-8a7b-6f0e2d4c8a19"
)

type fakeNotificationService struct{
	notifications				map[int]notification_model.Notification
	gotReminderIDs				[]int32

	getNotificationError		error
	getUserNotificationsError	error
	readNotificationError		error
	readUserNotificationsError	error
	getNotificationTypesError	error
	createRemindersError		error
}

func (s *fakeNotificationService) GetNotification(id int) (notification_model.Notification, error) {return s.notifications[id], s.getNotificationError}
func (s *fakeNotificationService) GetUserNotifications(id pgtype.UUID, params map[string]any) ([]notification_model.Notification, error) {
	notifs := []notification_model.Notification{}
	for _, n := range(s.notifications){
		if n.UserID == id{
			notifs=append(notifs, n)
		}
	}
	return notifs, s.getUserNotificationsError
}
func (s *fakeNotificationService) CreateNotification(n notification_model.Notification) (int, error){return 0, nil}
func (s *fakeNotificationService) ReadNotification(id int) error {return s.readNotificationError}
func (s *fakeNotificationService) ReadUserNotifications(id pgtype.UUID) error {return s.readUserNotificationsError}
func (s *fakeNotificationService) GetNotificationTypes() ([]notification_model.NotificationType, error) {return []notification_model.NotificationType{
																																						notification_model.NotificationType{ID: 1, Name: "Social"},
																																						notification_model.NotificationType{ID: 2, Name: "Recordatori"},
																																						notification_model.NotificationType{ID: 3, Name: "Recomanacio"},
																																					  }, s.getNotificationTypesError}
func (s *fakeNotificationService) CreateReminders() ([]int32, error) {return s.gotReminderIDs, s.createRemindersError}


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


func TestGetNotification(t *testing.T){
	notifications := make(map[int]notification_model.Notification)
	user1, _ := user_model.ParseID(validUserID)
	user2, _ := user_model.ParseID(otherUserID)
	notifications[1]=notification_model.Notification{ID: 1, UserID: user1, NotificationDate: pgtype.Timestamp{Time:time.Now(), Valid:true}, Friend: user2, Text: "user2 ha acceptat la teva sol·licitud d'amistat"}
	notifications[2]=notification_model.Notification{ID: 2, UserID: user1, NotificationDate: pgtype.Timestamp{Time:time.Now(), Valid:true}, Friend: user2, Event: pgtype.Int4{Int32:1, Valid:true}, Text: "user2 s'ha apuntat a Esdeveniment 1"}

	tests := []struct{
		name				string
		service				fakeNotificationService
		request				string
		wantStatus			int
		wantError			string
		wantNotifications	[]notification_model.Notification
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
				if err := json.Unmarshal(rec.Body.Bytes(), &notif); err != nil{
					t.Fatalf("decode body: %v", err)
				}
				notifs = []notification_model.Notification{notif}
			}
			if len(notifs)!=len(tt.wantNotifications){
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