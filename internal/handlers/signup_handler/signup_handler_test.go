package signup_handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"api-aprop/internal/clients/firebase_client"
	"api-aprop/internal/handlers/requester"
	"api-aprop/internal/middleware/auth_middleware"
	"api-aprop/internal/models/event_model"
	"api-aprop/internal/models/signup_model"
	"api-aprop/internal/services/auth_service"
	"api-aprop/internal/services/signup_service"
	"api-aprop/internal/services/translation_service"
)

const validUserID = "3f2504e0-4f89-11d3-9a0c-0305e82c3301"

type fakeService struct {
	signUp          signup_model.SignUp
	signUpErr       error
	unenroll        signup_model.Unenroll
	unenrollErr     error
	status          signup_model.EnrollmentStatus
	statusErr       error
	events          []event_model.Event
	listErr         error
	attendanceErr   error
	attendanceCalls int

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
	h := NewSignUpHandler(s, nil)
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
			wantStatus: http.StatusUnauthorized,
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
				req.Header.Set(requester.UserIDHeader, tt.userID)
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
			wantStatus: http.StatusUnauthorized,
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
				req.Header.Set(requester.UserIDHeader, tt.userID)
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
				req.Header.Set(requester.UserIDHeader, tt.userID)
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

type fakeResolver struct {
	id  string
	err error
}

func (f fakeResolver) ResolveUserID(context.Context, string) (string, error) { return f.id, f.err }

func withFirebaseUser(req *http.Request) *http.Request {
	return req.WithContext(auth_middleware.ContextWithUser(req.Context(), firebase_client.User{UID: "Vw3kU1pZ8rQ2hXyT6mN9aB4cD7e1"}))
}

// A Firebase user is translated to its database UUID before the service is called.
func TestSignUp_TranslatesFirebaseUID(t *testing.T) {
	s := &fakeService{signUp: signup_model.SignUp{UserID: validUserID, EventID: 7, Created: true}}
	mux := http.NewServeMux()
	NewSignUpHandler(s, fakeResolver{id: validUserID}).RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, withFirebaseUser(httptest.NewRequest(http.MethodPost, "/events/7/signup", nil)))

	if rec.Code != http.StatusCreated || s.gotUserID != validUserID {
		t.Fatalf("status = %d, user passed to service = %q; want 201 and %q", rec.Code, s.gotUserID, validUserID)
	}

	rec = httptest.NewRecorder()
	h := NewSignUpHandler(s, fakeResolver{err: auth_service.ErrUserNotRegistered})
	mux = http.NewServeMux()
	h.RegisterRoutes(mux)
	mux.ServeHTTP(rec, withFirebaseUser(httptest.NewRequest(http.MethodDelete, "/events/7/signup", nil)))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("not registered: status = %d, want 401", rec.Code)
	}
}

func (f *fakeService) SetAttendance(_ context.Context, userID string, eventID int, attendance *bool) (signup_model.Attendance, error) {
	f.attendanceCalls++
	f.gotUserID = userID
	f.gotEventID = eventID
	return signup_model.Attendance{UserID: userID, EventID: eventID, Attendance: attendance}, f.attendanceErr
}

func TestSetAttendance(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		userID     string
		err        error
		wantStatus int
		wantCalls  int
	}{
		{"true", `{"attendance": true}`, validUserID, nil, http.StatusOK, 1},
		{"null clears", `{"attendance": null}`, validUserID, nil, http.StatusOK, 1},
		{"missing field", `{}`, validUserID, nil, http.StatusBadRequest, 0},
		{"bad json", `nope`, validUserID, nil, http.StatusBadRequest, 0},
		{"non boolean", `{"attendance": "yes"}`, validUserID, nil, http.StatusBadRequest, 0},
		{"no user", `{"attendance": true}`, "", nil, http.StatusUnauthorized, 0},
		{"not enrolled", `{"attendance": true}`, validUserID, signup_service.ErrNotEnrolled, http.StatusNotFound, 1},
		{"R8 not started", `{"attendance": true}`, validUserID, signup_service.ErrEventNotStarted, http.StatusConflict, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeService{attendanceErr: tt.err}
			req := httptest.NewRequest(http.MethodPatch, "/events/7/enrollment/attendance", strings.NewReader(tt.body))
			if tt.userID != "" {
				req.Header.Set(requester.UserIDHeader, tt.userID)
			}
			rec := serve(svc, req)
			if rec.Code != tt.wantStatus || svc.attendanceCalls != tt.wantCalls {
				t.Fatalf("status = %d calls = %d, want %d / %d (%s)", rec.Code, svc.attendanceCalls, tt.wantStatus, tt.wantCalls, rec.Body.String())
			}
		})
	}
}

// langTranslator prefixes the title with the language it was asked for.
type langTranslator struct{ translation_service.Service }

func (langTranslator) TranslateEvents(_ context.Context, lang string, events []event_model.Event) []event_model.Event {
	out := slices.Clone(events)
	for i := range out {
		out[i].Title = "[" + lang + "] " + out[i].Title
	}
	return out
}

func TestListUserEvents_Lang(t *testing.T) {
	s := &fakeService{events: []event_model.Event{{ID: 1, Title: "Festa"}}}
	mux := http.NewServeMux()
	NewSignUpHandler(s, fakeResolver{id: validUserID}).WithTranslator(langTranslator{}).RegisterRoutes(mux)
	base := "/users/" + validUserID + "/events?from=2026-01-01&to=2026-12-31"

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, base+"&lang=en", nil))
	var got []event_model.Event
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || rec.Code != http.StatusOK || got[0].Title != "[en] Festa" {
		t.Errorf("status %d, body %s, err %v", rec.Code, rec.Body.String(), err)
	}
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, base+"&lang=fr", nil))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("lang=fr: status = %d, want 400", rec.Code)
	}
}
