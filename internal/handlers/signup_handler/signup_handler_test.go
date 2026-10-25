package signup_handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"api-aprop/internal/models/event_model"
	"api-aprop/internal/models/signup_model"
	"api-aprop/internal/services/signup_service"
)

const validUserID = "3f2504e0-4f89-11d3-9a0c-0305e82c3301"

type fakeService struct {
	signUp      signup_model.SignUp
	signUpErr   error
	unenroll    signup_model.Unenroll
	unenrollErr error
	status      signup_model.EnrollmentStatus
	statusErr   error
	events      []event_model.Event
	listErr     error

	calls         int
	unenrollCalls int
	statusCalls   int
	gotUserID     string
	gotEventID    int
	gotRange      signup_model.DateRange
}

func (f *fakeService) SignUp(_ context.Context, userID string, eventID int) (signup_model.SignUp, error) {
	f.calls++
	f.gotUserID = userID
	f.gotEventID = eventID
	return f.signUp, f.signUpErr
}

func (f *fakeService) Unenroll(_ context.Context, userID string, eventID int) (signup_model.Unenroll, error) {
	f.unenrollCalls++
	f.gotUserID = userID
	f.gotEventID = eventID
	return f.unenroll, f.unenrollErr
}

func (f *fakeService) GetEnrollmentStatus(_ context.Context, userID string, eventID int) (signup_model.EnrollmentStatus, error) {
	f.statusCalls++
	f.gotUserID = userID
	f.gotEventID = eventID
	return f.status, f.statusErr
}

func (f *fakeService) ListUserEvents(_ context.Context, userID string, dateRange signup_model.DateRange) ([]event_model.Event, error) {
	f.calls++
	f.gotUserID = userID
	f.gotRange = dateRange
	return f.events, f.listErr
}

// newMux registers the handler with the same patterns as the router so path values are set.
func newMux(s signup_service.Service) *http.ServeMux {
	h := NewSignUpHandler(s)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	return mux
}

func serve(s signup_service.Service, req *http.Request) *httptest.ResponseRecorder {
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

func TestSignUp(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		userID     string
		service    fakeService
		wantStatus int
		wantCalls  int
		wantError  string
	}{
		{
			name:       "new sign up returns 201",
			path:       "/events/7/signup",
			userID:     validUserID,
			service:    fakeService{signUp: signup_model.SignUp{UserID: validUserID, EventID: 7, Created: true}},
			wantStatus: http.StatusCreated,
			wantCalls:  1,
		},
		{
			name:       "already signed up returns 200",
			path:       "/events/7/signup",
			userID:     validUserID,
			service:    fakeService{signUp: signup_model.SignUp{UserID: validUserID, EventID: 7, Created: false}},
			wantStatus: http.StatusOK,
			wantCalls:  1,
		},
		{
			name:       "missing user header",
			path:       "/events/7/signup",
			wantStatus: http.StatusBadRequest,
			wantError:  "missing X-User-ID header",
		},
		{
			name:       "invalid user header",
			path:       "/events/7/signup",
			userID:     "not-a-uuid",
			service:    fakeService{signUpErr: fmt.Errorf("%w: %q", signup_service.ErrInvalidUserID, "not-a-uuid")},
			wantStatus: http.StatusBadRequest,
			wantCalls:  1,
			wantError:  "invalid user id",
		},
		{
			name:       "non numeric event id",
			path:       "/events/abc/signup",
			userID:     validUserID,
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid event id",
		},
		{
			name:       "non positive event id",
			path:       "/events/0/signup",
			userID:     validUserID,
			service:    fakeService{signUpErr: signup_service.ErrInvalidEventID},
			wantStatus: http.StatusBadRequest,
			wantCalls:  1,
			wantError:  "invalid event id",
		},
		{
			name:       "event not found",
			path:       "/events/99/signup",
			userID:     validUserID,
			service:    fakeService{signUpErr: fmt.Errorf("sign up: %w", signup_service.ErrEventNotFound)},
			wantStatus: http.StatusNotFound,
			wantCalls:  1,
			wantError:  "event not found",
		},
		{
			name:       "repository error",
			path:       "/events/7/signup",
			userID:     validUserID,
			service:    fakeService{signUpErr: errors.New("db down")},
			wantStatus: http.StatusInternalServerError,
			wantCalls:  1,
			wantError:  "internal server error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.service
			req := httptest.NewRequest(http.MethodPost, tt.path, nil)
			if tt.userID != "" {
				req.Header.Set(UserIDHeader, tt.userID)
			}

			rec := serve(&s, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if s.calls != tt.wantCalls {
				t.Errorf("service calls = %d, want %d", s.calls, tt.wantCalls)
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
			if s.gotUserID != tt.userID || s.gotEventID != 7 {
				t.Errorf("service got user %q event %d, want %q 7", s.gotUserID, s.gotEventID, tt.userID)
			}
			var body map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			want := map[string]any{"user-id": validUserID, "event-id": float64(7), "created": tt.service.signUp.Created}
			for k, v := range want {
				if body[k] != v {
					t.Errorf("body[%q] = %v, want %v", k, body[k], v)
				}
			}
		})
	}
}

func TestListUserEvents(t *testing.T) {
	oct1 := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	oct31 := time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)
	ordered := []event_model.Event{
		{ID: 5, Title: "early", StartDate: oct1},
		{ID: 2, Title: "late", StartDate: oct31},
	}
	tests := []struct {
		name       string
		query      string
		userID     string
		service    fakeService
		wantStatus int
		wantCalls  int
		wantRange  signup_model.DateRange
		wantIDs    []int
		wantError  string
	}{
		{
			name:       "without range",
			userID:     validUserID,
			service:    fakeService{events: ordered},
			wantStatus: http.StatusOK,
			wantCalls:  1,
			wantIDs:    []int{5, 2},
		},
		{
			name:       "with from and to",
			query:      "?from=2026-10-01&to=2026-10-31",
			userID:     validUserID,
			service:    fakeService{events: ordered},
			wantStatus: http.StatusOK,
			wantCalls:  1,
			wantRange:  signup_model.DateRange{From: &oct1, To: &oct31},
			wantIDs:    []int{5, 2},
		},
		{
			name:       "only from",
			query:      "?from=2026-10-01",
			userID:     validUserID,
			service:    fakeService{events: ordered},
			wantStatus: http.StatusOK,
			wantCalls:  1,
			wantRange:  signup_model.DateRange{From: &oct1},
			wantIDs:    []int{5, 2},
		},
		{
			name:       "empty list",
			userID:     validUserID,
			service:    fakeService{events: []event_model.Event{}},
			wantStatus: http.StatusOK,
			wantCalls:  1,
			wantIDs:    []int{},
		},
		{
			name:       "invalid from date",
			query:      "?from=01-10-2026",
			userID:     validUserID,
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid from date, expected YYYY-MM-DD",
		},
		{
			name:       "invalid to date",
			query:      "?to=2026-02-30",
			userID:     validUserID,
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid to date, expected YYYY-MM-DD",
		},
		{
			name:       "from after to",
			query:      "?from=2026-10-31&to=2026-10-01",
			userID:     validUserID,
			service:    fakeService{listErr: signup_service.ErrInvalidDateRange},
			wantStatus: http.StatusBadRequest,
			wantCalls:  1,
			wantError:  "invalid date range: from must not be after to",
		},
		{
			name:       "invalid user id",
			userID:     "abc",
			service:    fakeService{listErr: signup_service.ErrInvalidUserID},
			wantStatus: http.StatusBadRequest,
			wantCalls:  1,
			wantError:  "invalid user id",
		},
		{
			name:       "repository error",
			userID:     validUserID,
			service:    fakeService{listErr: errors.New("db down")},
			wantStatus: http.StatusInternalServerError,
			wantCalls:  1,
			wantError:  "internal server error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.service
			req := httptest.NewRequest(http.MethodGet, "/users/"+tt.userID+"/events"+tt.query, nil)

			rec := serve(&s, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if s.calls != tt.wantCalls {
				t.Errorf("service calls = %d, want %d", s.calls, tt.wantCalls)
			}
			if tt.wantError != "" {
				if got := errorMessage(t, rec); got != tt.wantError {
					t.Errorf("error = %q, want %q", got, tt.wantError)
				}
				return
			}
			if s.gotUserID != tt.userID {
				t.Errorf("service got user %q, want %q", s.gotUserID, tt.userID)
			}
			assertDate(t, "from", s.gotRange.From, tt.wantRange.From)
			assertDate(t, "to", s.gotRange.To, tt.wantRange.To)

			if len(tt.wantIDs) == 0 && strings.TrimSpace(rec.Body.String()) != "[]" {
				t.Errorf("body = %q, want []", rec.Body.String())
			}
			var events []event_model.Event
			if err := json.Unmarshal(rec.Body.Bytes(), &events); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if len(events) != len(tt.wantIDs) {
				t.Fatalf("got %d events, want %d", len(events), len(tt.wantIDs))
			}
			for i, id := range tt.wantIDs {
				if events[i].ID != id {
					t.Errorf("event[%d].ID = %d, want %d", i, events[i].ID, id)
				}
			}
		})
	}
}

func assertDate(t *testing.T, name string, got, want *time.Time) {
	t.Helper()
	switch {
	case got == nil && want == nil:
	case got == nil || want == nil:
		t.Errorf("%s = %v, want %v", name, got, want)
	case !got.Equal(*want):
		t.Errorf("%s = %v, want %v", name, *got, *want)
	}
}

func TestUnenroll(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		userID     string
		service    fakeService
		wantStatus int
		wantCalls  int
		wantError  string
	}{
		{
			name:       "successful unenroll DELETE /events/{id}/signup",
			method:     http.MethodDelete,
			path:       "/events/7/signup",
			userID:     validUserID,
			service:    fakeService{unenroll: signup_model.Unenroll{UserID: validUserID, EventID: 7, Deleted: true}},
			wantStatus: http.StatusOK,
			wantCalls:  1,
		},
		{
			name:       "successful unenroll DELETE /events/{id}/enroll",
			method:     http.MethodDelete,
			path:       "/events/7/enroll",
			userID:     validUserID,
			service:    fakeService{unenroll: signup_model.Unenroll{UserID: validUserID, EventID: 7, Deleted: true}},
			wantStatus: http.StatusOK,
			wantCalls:  1,
		},
		{
			name:       "missing user header",
			method:     http.MethodDelete,
			path:       "/events/7/signup",
			wantStatus: http.StatusBadRequest,
			wantError:  "missing X-User-ID header",
		},
		{
			name:       "event not found",
			method:     http.MethodDelete,
			path:       "/events/99/signup",
			userID:     validUserID,
			service:    fakeService{unenrollErr: fmt.Errorf("unenroll: %w", signup_service.ErrEventNotFound)},
			wantStatus: http.StatusNotFound,
			wantCalls:  1,
			wantError:  "event not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.service
			req := httptest.NewRequest(tt.method, tt.path, nil)
			if tt.userID != "" {
				req.Header.Set(UserIDHeader, tt.userID)
			}

			rec := serve(&s, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if s.unenrollCalls != tt.wantCalls {
				t.Errorf("unenroll calls = %d, want %d", s.unenrollCalls, tt.wantCalls)
			}
			if tt.wantError != "" {
				if got := errorMessage(t, rec); got != tt.wantError {
					t.Errorf("error = %q, want %q", got, tt.wantError)
				}
				return
			}
			if s.gotUserID != tt.userID || s.gotEventID != 7 {
				t.Errorf("service got user %q event %d, want %q 7", s.gotUserID, s.gotEventID, tt.userID)
			}
		})
	}
}

func TestGetEnrollmentStatus(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		userID     string
		service    fakeService
		wantStatus int
		wantCalls  int
		wantBody   map[string]any
		wantError  string
	}{
		{
			name:       "enrolled status GET /events/{id}/signup",
			path:       "/events/7/signup",
			userID:     validUserID,
			service:    fakeService{status: signup_model.EnrollmentStatus{EventID: 7, IsEnrolled: true, EnrolledCount: 12}},
			wantStatus: http.StatusOK,
			wantCalls:  1,
			wantBody:   map[string]any{"event-id": float64(7), "is-enrolled": true, "enrolled-count": float64(12)},
		},
		{
			name:       "enrolled status GET /events/{id}/enrollment",
			path:       "/events/7/enrollment",
			userID:     validUserID,
			service:    fakeService{status: signup_model.EnrollmentStatus{EventID: 7, IsEnrolled: true, EnrolledCount: 12}},
			wantStatus: http.StatusOK,
			wantCalls:  1,
			wantBody:   map[string]any{"event-id": float64(7), "is-enrolled": true, "enrolled-count": float64(12)},
		},
		{
			name:       "event not found",
			path:       "/events/99/enrollment",
			service:    fakeService{statusErr: fmt.Errorf("status: %w", signup_service.ErrEventNotFound)},
			wantStatus: http.StatusNotFound,
			wantCalls:  1,
			wantError:  "event not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.service
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			if tt.userID != "" {
				req.Header.Set(UserIDHeader, tt.userID)
			}

			rec := serve(&s, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if s.statusCalls != tt.wantCalls {
				t.Errorf("status calls = %d, want %d", s.statusCalls, tt.wantCalls)
			}
			if tt.wantError != "" {
				if got := errorMessage(t, rec); got != tt.wantError {
					t.Errorf("error = %q, want %q", got, tt.wantError)
				}
				return
			}
			var body map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			for k, v := range tt.wantBody {
				if body[k] != v {
					t.Errorf("body[%q] = %v, want %v", k, body[k], v)
				}
			}
		})
	}
}
