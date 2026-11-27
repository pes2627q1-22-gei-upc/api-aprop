package rating_handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"api-aprop/internal/handlers/requester"
	"api-aprop/internal/models/rating_model"
	"api-aprop/internal/services/rating_service"
)

const validUserID = "3f2504e0-4f89-11d3-9a0c-0305e82c3301"

type fakeService struct {
	err         error
	calls       int
	gotScore    int
	hasAttended bool
	attendCalls int
}

func (f *fakeService) Rate(_ context.Context, userID string, eventID, score int) (rating_model.Rating, error) {
	f.calls++
	f.gotScore = score
	return rating_model.Rating{UserID: userID, EventID: eventID, Score: score}, f.err
}

func (f *fakeService) UpdateRating(_ context.Context, userID string, eventID, score int) (rating_model.Rating, error) {
	f.calls++
	f.gotScore = score
	return rating_model.Rating{UserID: userID, EventID: eventID, Score: score}, f.err
}

func (f *fakeService) DeleteRating(context.Context, string, int) error {
	f.calls++
	return f.err
}

func (f *fakeService) RateUser(_ context.Context, userID string, eventID, score int) (rating_model.UserRating, error) {
	return rating_model.UserRating{}, nil
}

func (f *fakeService) UpdateUserRating(_ context.Context, userID string, eventID, score int) (rating_model.UserRating, error) {
	return rating_model.UserRating{}, nil
}

func (f *fakeService) DeleteUserRating(_ context.Context, userID string, eventID int) error {
	return nil
}

func (f *fakeService) HasAttended(_ context.Context, userID string, eventID int) (rating_model.AttendanceStatus, error) {
	f.attendCalls++
	return rating_model.AttendanceStatus{UserID: userID, EventID: eventID, HasAttended: f.hasAttended}, f.err
}

func send(svc *fakeService, method, path, body, userID string) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	NewRatingHandler(svc, nil).RegisterRoutes(mux)
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if userID != "" {
		req.Header.Set(requester.UserIDHeader, userID)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func post(svc *fakeService, path, body, userID string) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	NewRatingHandler(svc, nil).RegisterRoutes(mux)
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	if userID != "" {
		req.Header.Set(requester.UserIDHeader, userID)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func get(svc *fakeService, path, userID string) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	NewRatingHandler(svc, nil).RegisterRoutes(mux)
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if userID != "" {
		req.Header.Set(requester.UserIDHeader, userID)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestR9_RateStatusCodes(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		body       string
		userID     string
		err        error
		wantStatus int
		wantCalls  int
	}{
		{"created", "/events/7/ratings", `{"score": 5}`, validUserID, nil, http.StatusCreated, 1},
		{"R9 out of range", "/events/7/ratings", `{"score": 6}`, validUserID, rating_service.ErrInvalidScore, http.StatusBadRequest, 1},
		{"score missing", "/events/7/ratings", `{}`, validUserID, nil, http.StatusBadRequest, 0},
		{"score not an integer", "/events/7/ratings", `{"score": 4.5}`, validUserID, nil, http.StatusBadRequest, 0},
		{"bad json", "/events/7/ratings", `nope`, validUserID, nil, http.StatusBadRequest, 0},
		{"bad event id", "/events/x/ratings", `{"score": 3}`, validUserID, nil, http.StatusBadRequest, 0},
		{"no user", "/events/7/ratings", `{"score": 3}`, "", nil, http.StatusUnauthorized, 0},
		{"R10 not attended", "/events/7/ratings", `{"score": 3}`, validUserID, rating_service.ErrNotAttended, http.StatusForbidden, 1},
		{"already rated", "/events/7/ratings", `{"score": 3}`, validUserID, rating_service.ErrAlreadyRated, http.StatusConflict, 1},
		{"event not found", "/events/7/ratings", `{"score": 3}`, validUserID, rating_service.ErrEventNotFound, http.StatusNotFound, 1},
		{"internal", "/events/7/ratings", `{"score": 3}`, validUserID, context.DeadlineExceeded, http.StatusInternalServerError, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeService{err: tt.err}
			rec := post(svc, tt.path, tt.body, tt.userID)
			if rec.Code != tt.wantStatus || svc.calls != tt.wantCalls {
				t.Fatalf("status = %d calls = %d, want %d / %d (%s)", rec.Code, svc.calls, tt.wantStatus, tt.wantCalls, rec.Body.String())
			}
		})
	}
}

func TestR9_CreatedBody(t *testing.T) {
	rec := post(&fakeService{}, "/events/7/ratings", `{"score": 4}`, validUserID)
	want := `{"user-id":"` + validUserID + `","event-id":7,"score":4}`
	if strings.TrimSpace(rec.Body.String()) != want {
		t.Fatalf("body = %s, want %s", rec.Body.String(), want)
	}
}

func TestHasAttendedHandler(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		userID     string
		attended   bool
		err        error
		wantStatus int
		wantCalls  int
		wantBody   string
	}{
		{
			name:       "attended enrollment route",
			path:       "/events/7/enrollment/attendance",
			userID:     validUserID,
			attended:   true,
			wantStatus: http.StatusOK,
			wantCalls:  1,
			wantBody:   `{"user-id":"` + validUserID + `","event-id":7,"has-attended":true}`,
		},
		{
			name:       "not attended signup route",
			path:       "/events/7/signup/attendance",
			userID:     validUserID,
			attended:   false,
			wantStatus: http.StatusOK,
			wantCalls:  1,
			wantBody:   `{"user-id":"` + validUserID + `","event-id":7,"has-attended":false}`,
		},
		{name: "bad event id", path: "/events/x/enrollment/attendance", userID: validUserID, wantStatus: http.StatusBadRequest, wantCalls: 0},
		{name: "no user", path: "/events/7/enrollment/attendance", wantStatus: http.StatusUnauthorized, wantCalls: 0},
		{name: "event not found", path: "/events/7/enrollment/attendance", userID: validUserID, err: rating_service.ErrEventNotFound, wantStatus: http.StatusNotFound, wantCalls: 1},
		{name: "internal error", path: "/events/7/enrollment/attendance", userID: validUserID, err: context.DeadlineExceeded, wantStatus: http.StatusInternalServerError, wantCalls: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeService{hasAttended: tt.attended, err: tt.err}
			rec := get(svc, tt.path, tt.userID)
			if rec.Code != tt.wantStatus || svc.attendCalls != tt.wantCalls {
				t.Fatalf("status = %d calls = %d, want %d / %d (%s)", rec.Code, svc.attendCalls, tt.wantStatus, tt.wantCalls, rec.Body.String())
			}
			if tt.wantBody != "" && strings.TrimSpace(rec.Body.String()) != tt.wantBody {
				t.Fatalf("body = %s, want %s", rec.Body.String(), tt.wantBody)
			}
		})
	}
}

func TestUpdateMine(t *testing.T) {
	const path = "/events/7/ratings/me"
	tests := []struct {
		name       string
		path       string
		body       string
		userID     string
		err        error
		wantStatus int
		wantCalls  int
	}{
		{"ok", path, `{"score": 2}`, validUserID, nil, http.StatusOK, 1},
		{"R9 out of range", path, `{"score": 9}`, validUserID, rating_service.ErrInvalidScore, http.StatusBadRequest, 1},
		{"score missing", path, `{}`, validUserID, nil, http.StatusBadRequest, 0},
		{"bad json", path, `nope`, validUserID, nil, http.StatusBadRequest, 0},
		{"bad event id", "/events/x/ratings/me", `{"score": 3}`, validUserID, nil, http.StatusBadRequest, 0},
		{"no user", path, `{"score": 3}`, "", nil, http.StatusUnauthorized, 0},
		{"event not found", path, `{"score": 3}`, validUserID, rating_service.ErrEventNotFound, http.StatusNotFound, 1},
		{"rating not found", path, `{"score": 3}`, validUserID, rating_service.ErrRatingNotFound, http.StatusNotFound, 1},
		{"internal", path, `{"score": 3}`, validUserID, context.DeadlineExceeded, http.StatusInternalServerError, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeService{err: tt.err}
			rec := send(svc, http.MethodPut, tt.path, tt.body, tt.userID)
			if rec.Code != tt.wantStatus || svc.calls != tt.wantCalls {
				t.Fatalf("status = %d calls = %d, want %d / %d (%s)", rec.Code, svc.calls, tt.wantStatus, tt.wantCalls, rec.Body.String())
			}
		})
	}
	rec := send(&fakeService{}, http.MethodPut, path, `{"score": 4}`, validUserID)
	want := `{"user-id":"` + validUserID + `","event-id":7,"score":4}`
	if strings.TrimSpace(rec.Body.String()) != want {
		t.Fatalf("body = %s, want %s", rec.Body.String(), want)
	}
}

func TestDeleteMine(t *testing.T) {
	const path = "/events/7/ratings/me"
	tests := []struct {
		name       string
		path       string
		userID     string
		err        error
		wantStatus int
		wantCalls  int
	}{
		{"ok", path, validUserID, nil, http.StatusNoContent, 1},
		{"bad event id", "/events/x/ratings/me", validUserID, nil, http.StatusBadRequest, 0},
		{"no user", path, "", nil, http.StatusUnauthorized, 0},
		{"event not found", path, validUserID, rating_service.ErrEventNotFound, http.StatusNotFound, 1},
		{"rating not found", path, validUserID, rating_service.ErrRatingNotFound, http.StatusNotFound, 1},
		{"internal", path, validUserID, context.DeadlineExceeded, http.StatusInternalServerError, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeService{err: tt.err}
			rec := send(svc, http.MethodDelete, tt.path, "", tt.userID)
			if rec.Code != tt.wantStatus || svc.calls != tt.wantCalls {
				t.Fatalf("status = %d calls = %d, want %d / %d (%s)", rec.Code, svc.calls, tt.wantStatus, tt.wantCalls, rec.Body.String())
			}
		})
	}
}
