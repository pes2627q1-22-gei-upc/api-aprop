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
	err      error
	calls    int
	gotScore int
}

func (f *fakeService) Rate(_ context.Context, userID string, eventID, score int) (rating_model.Rating, error) {
	f.calls++
	f.gotScore = score
	return rating_model.Rating{UserID: userID, EventID: eventID, Score: score}, f.err
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
