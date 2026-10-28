package preference_handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"api-aprop/internal/clients/firebase_client"
	"api-aprop/internal/handlers/requester"
	"api-aprop/internal/middleware/auth_middleware"
	"api-aprop/internal/models/preference_model"
	"api-aprop/internal/services/event_service"
	"api-aprop/internal/services/preference_service"
)

const validUserID = "3f2504e0-4f89-11d3-9a0c-0305e82c3301"

type fakePreferenceService struct {
	questions      []preference_model.TestQuestion
	questionsErr   error
	submitResponse preference_model.SubmitTestResponse
	submitErr      error
	preferences    []preference_model.UserPreference
	preferencesErr error

	calls      int
	gotUserID  string
	gotAnswers []preference_model.TestAnswer
}

func (f *fakePreferenceService) GetQuestions(_ context.Context, userID string) ([]preference_model.TestQuestion, error) {
	f.calls++
	f.gotUserID = userID
	return f.questions, f.questionsErr
}

func (f *fakePreferenceService) SubmitTest(_ context.Context, userID string, answers []preference_model.TestAnswer) (preference_model.SubmitTestResponse, error) {
	f.calls++
	f.gotUserID = userID
	f.gotAnswers = answers
	return f.submitResponse, f.submitErr
}

func (f *fakePreferenceService) GetUserPreferences(_ context.Context, userID string) ([]preference_model.UserPreference, error) {
	f.calls++
	f.gotUserID = userID
	return f.preferences, f.preferencesErr
}

func newMux(s preference_service.Service) *http.ServeMux {
	h := NewPreferenceHandler(s, nil)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /preferences/test", h.GetQuestions)
	mux.HandleFunc("POST /preferences/test", h.SubmitTest)
	mux.HandleFunc("GET /preferences", h.GetUserPreferences)
	return mux
}

func serve(s preference_service.Service, req *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	newMux(s).ServeHTTP(rec, req)
	return rec
}

func errorMessage(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Message string `json:"message"`
		Error   string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal error response %q: %v", rec.Body.String(), err)
	}
	if body.Error != "" {
		return body.Error
	}
	return body.Message
}

func TestGetQuestions(t *testing.T) {
	tests := []struct {
		name       string
		userHeader string
		service    fakePreferenceService
		wantStatus int
		wantCalls  int
		wantError  string
		wantCount  int
	}{
		{
			name:       "success returns questions",
			userHeader: validUserID,
			service: fakePreferenceService{
				questions: []preference_model.TestQuestion{
					{EventID: 1, Title: "Event 1"},
					{EventID: 2, Title: "Event 2"},
				},
			},
			wantStatus: http.StatusOK,
			wantCalls:  1,
			wantCount:  2,
		},
		{
			name:       "missing X-User-ID header returns 400",
			wantStatus: http.StatusBadRequest,
			wantCalls:  0,
			wantError:  "missing X-User-ID header",
		},
		{
			name:       "invalid user id returns 400",
			userHeader: "invalid-uuid",
			service:    fakePreferenceService{questionsErr: preference_service.ErrInvalidUserID},
			wantStatus: http.StatusBadRequest,
			wantCalls:  1,
			wantError:  "invalid user id",
		},
		{
			name:       "service error returns 500",
			userHeader: validUserID,
			service:    fakePreferenceService{questionsErr: errors.New("db down")},
			wantStatus: http.StatusInternalServerError,
			wantCalls:  1,
			wantError:  "internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.service
			req := httptest.NewRequest(http.MethodGet, "/preferences/test", nil)
			if tt.userHeader != "" {
				req.Header.Set(requester.UserIDHeader, tt.userHeader)
			}

			rec := serve(&s, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if s.calls != tt.wantCalls {
				t.Errorf("calls = %d, want %d", s.calls, tt.wantCalls)
			}
			if tt.wantError != "" {
				if got := errorMessage(t, rec); got != tt.wantError {
					t.Errorf("error = %q, want %q", got, tt.wantError)
				}
				return
			}
			var list []preference_model.TestQuestion
			if err := json.NewDecoder(rec.Body).Decode(&list); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if len(list) != tt.wantCount {
				t.Errorf("got %d questions, want %d", len(list), tt.wantCount)
			}
		})
	}
}

func TestSubmitTest(t *testing.T) {
	fixedTime := time.Date(2026, 10, 24, 18, 0, 0, 0, time.UTC)
	tests := []struct {
		name       string
		userHeader string
		body       string
		service    fakePreferenceService
		wantStatus int
		wantCalls  int
		wantError  string
	}{
		{
			name:       "success submits test",
			userHeader: validUserID,
			body:       `{"answers":[{"event-id":1,"likes":true}]}`,
			service: fakePreferenceService{
				submitResponse: preference_model.SubmitTestResponse{
					UserID:    validUserID,
					DateTaken: fixedTime,
					Preferences: []preference_model.UserPreference{
						{CategoryID: 1, CategoryName: "Música", Weight: 1.2},
					},
				},
			},
			wantStatus: http.StatusOK,
			wantCalls:  1,
		},
		{
			name:       "missing X-User-ID header returns 400",
			body:       `{"answers":[{"event-id":1,"likes":true}]}`,
			wantStatus: http.StatusBadRequest,
			wantCalls:  0,
			wantError:  "missing X-User-ID header",
		},
		{
			name:       "invalid json body returns 400",
			userHeader: validUserID,
			body:       `{bad json}`,
			wantStatus: http.StatusBadRequest,
			wantCalls:  0,
			wantError:  "invalid request body",
		},
		{
			name:       "empty answers returns 400",
			userHeader: validUserID,
			body:       `{"answers":[]}`,
			service:    fakePreferenceService{submitErr: preference_service.ErrEmptyAnswers},
			wantStatus: http.StatusBadRequest,
			wantCalls:  1,
			wantError:  "answers cannot be empty",
		},
		{
			name:       "service error returns 500",
			userHeader: validUserID,
			body:       `{"answers":[{"event-id":1,"likes":true}]}`,
			service:    fakePreferenceService{submitErr: errors.New("db error")},
			wantStatus: http.StatusInternalServerError,
			wantCalls:  1,
			wantError:  "internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.service
			req := httptest.NewRequest(http.MethodPost, "/preferences/test", bytes.NewBufferString(tt.body))
			if tt.userHeader != "" {
				req.Header.Set(requester.UserIDHeader, tt.userHeader)
			}

			rec := serve(&s, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if s.calls != tt.wantCalls {
				t.Errorf("calls = %d, want %d", s.calls, tt.wantCalls)
			}
			if tt.wantError != "" {
				if got := errorMessage(t, rec); got != tt.wantError {
					t.Errorf("error = %q, want %q", got, tt.wantError)
				}
			}
		})
	}
}

func TestGetUserPreferences(t *testing.T) {
	tests := []struct {
		name       string
		userHeader string
		service    fakePreferenceService
		wantStatus int
		wantCalls  int
		wantError  string
		wantCount  int
	}{
		{
			name:       "success returns preferences",
			userHeader: validUserID,
			service: fakePreferenceService{
				preferences: []preference_model.UserPreference{
					{CategoryID: 1, CategoryName: "Música", Weight: 1.5},
				},
			},
			wantStatus: http.StatusOK,
			wantCalls:  1,
			wantCount:  1,
		},
		{
			name:       "missing header returns 400",
			wantStatus: http.StatusBadRequest,
			wantCalls:  0,
			wantError:  "missing X-User-ID header",
		},
		{
			name:       "service error returns 500",
			userHeader: validUserID,
			service:    fakePreferenceService{preferencesErr: errors.New("db error")},
			wantStatus: http.StatusInternalServerError,
			wantCalls:  1,
			wantError:  "internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.service
			req := httptest.NewRequest(http.MethodGet, "/preferences", nil)
			if tt.userHeader != "" {
				req.Header.Set(requester.UserIDHeader, tt.userHeader)
			}

			rec := serve(&s, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if s.calls != tt.wantCalls {
				t.Errorf("calls = %d, want %d", s.calls, tt.wantCalls)
			}
			if tt.wantError != "" {
				if got := errorMessage(t, rec); got != tt.wantError {
					t.Errorf("error = %q, want %q", got, tt.wantError)
				}
				return
			}
			var prefs []preference_model.UserPreference
			if err := json.NewDecoder(rec.Body).Decode(&prefs); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if len(prefs) != tt.wantCount {
				t.Errorf("got %d preferences, want %d", len(prefs), tt.wantCount)
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
func TestGetUserPreferences_TranslatesFirebaseUID(t *testing.T) {
	s := &fakePreferenceService{}
	rec := httptest.NewRecorder()
	NewPreferenceHandler(s, fakeResolver{id: validUserID}).GetUserPreferences(rec, withFirebaseUser(httptest.NewRequest(http.MethodGet, "/preferences", nil)))
	if rec.Code != http.StatusOK || s.gotUserID != validUserID {
		t.Fatalf("status = %d, user passed to service = %q; want 200 and %q", rec.Code, s.gotUserID, validUserID)
	}

	rec = httptest.NewRecorder()
	NewPreferenceHandler(s, fakeResolver{err: event_service.ErrUserNotRegistered}).GetUserPreferences(rec, withFirebaseUser(httptest.NewRequest(http.MethodGet, "/preferences", nil)))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("not registered: status = %d, want 401", rec.Code)
	}
}
