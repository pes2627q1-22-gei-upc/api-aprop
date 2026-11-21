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
	"api-aprop/internal/models/event_model"
	"api-aprop/internal/models/preference_model"
	"api-aprop/internal/services/auth_service"
	"api-aprop/internal/services/preference_service"
)

const validUserID = "3f2504e0-4f89-11d3-9a0c-0305e82c3301"

type fakePreferenceService struct {
	questions       []preference_model.TestQuestion
	questionsErr    error
	status          preference_model.TestStatus
	statusErr       error
	markPromptedErr error
	submitResponse  preference_model.SubmitTestResponse
	submitErr       error
	partialResponse preference_model.SubmitTestResponse
	partialErr      error
	preferences     []preference_model.UserPreference
	preferencesErr  error

	calls        int
	partialCalls int
	promptCalls  int
	gotUserID    string
	gotAnswers   []preference_model.TestAnswer
}

func (f *fakePreferenceService) GetQuestions(_ context.Context, userID string) ([]preference_model.TestQuestion, error) {
	f.calls++
	f.gotUserID = userID
	return f.questions, f.questionsErr
}

func (f *fakePreferenceService) GetTestStatus(_ context.Context, userID string) (preference_model.TestStatus, error) {
	f.calls++
	f.gotUserID = userID
	return f.status, f.statusErr
}

func (f *fakePreferenceService) MarkTestPrompted(_ context.Context, userID string) error {
	f.calls++
	f.promptCalls++
	f.gotUserID = userID
	return f.markPromptedErr
}

func (f *fakePreferenceService) SubmitTest(_ context.Context, userID string, answers []preference_model.TestAnswer) (preference_model.SubmitTestResponse, error) {
	f.calls++
	f.gotUserID = userID
	f.gotAnswers = answers
	return f.submitResponse, f.submitErr
}

func (f *fakePreferenceService) SavePartialTest(_ context.Context, userID string, answers []preference_model.TestAnswer) (preference_model.SubmitTestResponse, error) {
	f.calls++
	f.partialCalls++
	f.gotUserID = userID
	f.gotAnswers = answers
	return f.partialResponse, f.partialErr
}

func (f *fakePreferenceService) GetUserPreferences(_ context.Context, userID string) ([]preference_model.UserPreference, error) {
	f.calls++
	f.gotUserID = userID
	return f.preferences, f.preferencesErr
}

func newMux(s preference_service.Service) *http.ServeMux {
	h := NewPreferenceHandler(s, nil)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
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
			name:       "missing X-User-ID header returns 401",
			wantStatus: http.StatusUnauthorized,
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
			name:       "missing X-User-ID header returns 401",
			body:       `{"answers":[{"event-id":1,"likes":true}]}`,
			wantStatus: http.StatusUnauthorized,
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
			name:       "missing header returns 401",
			wantStatus: http.StatusUnauthorized,
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
			rec, ok := getAndCheck(t, "/preferences", tt.userHeader, tt.service, tt.wantStatus, tt.wantCalls, tt.wantError)
			if !ok {
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

// getAndCheck serves a GET request and verifies status, service calls and error
// message; ok is false when the case expected an error and nothing more must be checked.
func getAndCheck(t *testing.T, path, userHeader string, svc fakePreferenceService, wantStatus, wantCalls int, wantError string) (rec *httptest.ResponseRecorder, ok bool) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if userHeader != "" {
		req.Header.Set(requester.UserIDHeader, userHeader)
	}
	rec = serve(&svc, req)
	if rec.Code != wantStatus {
		t.Fatalf("status = %d, want %d (body %s)", rec.Code, wantStatus, rec.Body.String())
	}
	if svc.calls != wantCalls {
		t.Errorf("calls = %d, want %d", svc.calls, wantCalls)
	}
	if wantError != "" {
		if got := errorMessage(t, rec); got != wantError {
			t.Errorf("error = %q, want %q", got, wantError)
		}
		return rec, false
	}
	return rec, true
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
func assertTranslatesFirebaseUID(t *testing.T, path string, call func(*PreferenceHandler, http.ResponseWriter, *http.Request)) {
	t.Helper()
	s := &fakePreferenceService{}
	rec := httptest.NewRecorder()
	call(NewPreferenceHandler(s, fakeResolver{id: validUserID}), rec, withFirebaseUser(httptest.NewRequest(http.MethodGet, path, nil)))
	if rec.Code != http.StatusOK || s.gotUserID != validUserID {
		t.Fatalf("status = %d, user passed to service = %q; want 200 and %q", rec.Code, s.gotUserID, validUserID)
	}

	rec = httptest.NewRecorder()
	call(NewPreferenceHandler(s, fakeResolver{err: auth_service.ErrUserNotRegistered}), rec, withFirebaseUser(httptest.NewRequest(http.MethodGet, path, nil)))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("not registered: status = %d, want 401", rec.Code)
	}
}

func TestGetUserPreferences_TranslatesFirebaseUID(t *testing.T) {
	assertTranslatesFirebaseUID(t, "/preferences", (*PreferenceHandler).GetUserPreferences)
}

func TestGetTestStatus_TranslatesFirebaseUID(t *testing.T) {
	assertTranslatesFirebaseUID(t, "/preferences/test/status", (*PreferenceHandler).GetTestStatus)
}

func TestGetTestStatus(t *testing.T) {
	fixedTime := time.Date(2026, 10, 24, 18, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		userHeader string
		service    fakePreferenceService
		wantStatus int
		wantCalls  int
		wantError  string
		wantJSON   map[string]any
	}{
		{
			name:       "completed test returns completed true, prompted true and RFC3339 date-taken",
			userHeader: validUserID,
			service: fakePreferenceService{
				status: preference_model.TestStatus{
					Completed:     true,
					Prompted:      true,
					InProgress:    false,
					AnsweredCount: 0,
					DateTaken:     &fixedTime,
				},
			},
			wantStatus: http.StatusOK,
			wantCalls:  1,
			wantJSON: map[string]any{
				"completed":      true,
				"prompted":       true,
				"in-progress":    false,
				"answered-count": float64(0),
				"date-taken":     "2026-10-24T18:00:00Z",
			},
		},
		{
			name:       "uncompleted new user returns completed false, prompted false and explicit null date-taken",
			userHeader: validUserID,
			service: fakePreferenceService{
				status: preference_model.TestStatus{
					Completed:     false,
					Prompted:      false,
					InProgress:    false,
					AnsweredCount: 0,
					DateTaken:     nil,
				},
			},
			wantStatus: http.StatusOK,
			wantCalls:  1,
			wantJSON: map[string]any{
				"completed":      false,
				"prompted":       false,
				"in-progress":    false,
				"answered-count": float64(0),
				"date-taken":     nil,
			},
		},
		{
			name:       "paused test returns in-progress true and answered-count",
			userHeader: validUserID,
			service: fakePreferenceService{
				status: preference_model.TestStatus{
					Completed:     false,
					Prompted:      true,
					InProgress:    true,
					AnsweredCount: 6,
					DateTaken:     nil,
				},
			},
			wantStatus: http.StatusOK,
			wantCalls:  1,
			wantJSON: map[string]any{
				"completed":      false,
				"prompted":       true,
				"in-progress":    true,
				"answered-count": float64(6),
				"date-taken":     nil,
			},
		},
		{
			name:       "missing X-User-ID header returns 401",
			wantStatus: http.StatusUnauthorized,
			wantCalls:  0,
			wantError:  "missing X-User-ID header",
		},
		{
			name:       "invalid user id returns 400",
			userHeader: "bad-uuid",
			service:    fakePreferenceService{statusErr: preference_service.ErrInvalidUserID},
			wantStatus: http.StatusBadRequest,
			wantCalls:  1,
			wantError:  "invalid user id",
		},
		{
			name:       "service error returns 500",
			userHeader: validUserID,
			service:    fakePreferenceService{statusErr: errors.New("db error")},
			wantStatus: http.StatusInternalServerError,
			wantCalls:  1,
			wantError:  "internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec, ok := getAndCheck(t, "/preferences/test/status", tt.userHeader, tt.service, tt.wantStatus, tt.wantCalls, tt.wantError)
			if !ok {
				return
			}

			var raw map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
				t.Fatalf("unmarshal response %q: %v", rec.Body.String(), err)
			}
			if len(raw) != len(tt.wantJSON) {
				t.Fatalf("response keys = %v, want %v", raw, tt.wantJSON)
			}
			for k, wantVal := range tt.wantJSON {
				gotVal, exists := raw[k]
				if !exists {
					t.Errorf("missing JSON key %q in %s", k, rec.Body.String())
					continue
				}
				if gotVal != wantVal {
					t.Errorf("key %q = %#v, want %#v", k, gotVal, wantVal)
				}
			}
		})
	}
}

func TestSubmitTest_Partial(t *testing.T) {
	fixedTime := time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)
	s := fakePreferenceService{
		partialResponse: preference_model.SubmitTestResponse{
			UserID:        validUserID,
			DateTaken:     fixedTime,
			Preferences:   []preference_model.UserPreference{},
			Partial:       true,
			AnsweredCount: 2,
		},
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/preferences/test",
		bytes.NewBufferString(`{"answers":[{"event-id":1,"likes":true},{"event-id":2,"likes":false}],"partial":true}`),
	)
	req.Header.Set(requester.UserIDHeader, validUserID)

	rec := serve(&s, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
	}
	if s.partialCalls != 1 {
		t.Fatalf("partialCalls = %d, want 1", s.partialCalls)
	}
	if len(s.gotAnswers) != 2 || s.gotAnswers[0].EventID != 1 || s.gotAnswers[1].EventID != 2 {
		t.Fatalf("gotAnswers = %+v, want 2 answers", s.gotAnswers)
	}

	var raw map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if raw["partial"] != true || raw["answered-count"] != float64(2) {
		t.Errorf("response = %v, want partial=true answered-count=2", raw)
	}
}

func TestMarkTestPrompted(t *testing.T) {
	tests := []struct {
		name       string
		userHeader string
		service    fakePreferenceService
		wantStatus int
		wantCalls  int
	}{
		{
			name:       "success returns 204",
			userHeader: validUserID,
			wantStatus: http.StatusNoContent,
			wantCalls:  1,
		},
		{
			name:       "missing header returns 401",
			wantStatus: http.StatusUnauthorized,
			wantCalls:  0,
		},
		{
			name:       "invalid user id returns 400",
			userHeader: "bad-uuid",
			service:    fakePreferenceService{markPromptedErr: preference_service.ErrInvalidUserID},
			wantStatus: http.StatusBadRequest,
			wantCalls:  1,
		},
		{
			name:       "service error returns 500",
			userHeader: validUserID,
			service:    fakePreferenceService{markPromptedErr: errors.New("db error")},
			wantStatus: http.StatusInternalServerError,
			wantCalls:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.service
			req := httptest.NewRequest(http.MethodPost, "/preferences/test/prompted", nil)
			if tt.userHeader != "" {
				req.Header.Set(requester.UserIDHeader, tt.userHeader)
			}
			rec := serve(&s, req)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if s.promptCalls != tt.wantCalls {
				t.Errorf("promptCalls = %d, want %d", s.promptCalls, tt.wantCalls)
			}
		})
	}
}

func TestMarkTestPrompted_TranslatesFirebaseUID(t *testing.T) {
	s := &fakePreferenceService{}
	rec := httptest.NewRecorder()
	req := withFirebaseUser(httptest.NewRequest(http.MethodPost, "/preferences/test/prompted", nil))
	NewPreferenceHandler(s, fakeResolver{id: validUserID}).MarkTestPrompted(rec, req)
	if rec.Code != http.StatusNoContent || s.gotUserID != validUserID {
		t.Fatalf("status = %d, user = %q; want 204 and %q", rec.Code, s.gotUserID, validUserID)
	}
}

type fakeTranslator struct {
	userLang string
	langErr  error
	calls    int
	gotLang  string
}

func (f *fakeTranslator) UserLanguage(_ context.Context, _ string) (string, error) {
	return f.userLang, f.langErr
}

func (f *fakeTranslator) TranslateEvent(_ context.Context, _ string, e event_model.Event) event_model.Event {
	return e
}

func (f *fakeTranslator) TranslateEvents(_ context.Context, _ string, events []event_model.Event) []event_model.Event {
	return events
}

func (f *fakeTranslator) TranslateMapEvents(_ context.Context, _ string, events []event_model.MapEvent) []event_model.MapEvent {
	return events
}

func (f *fakeTranslator) TranslateTestQuestions(_ context.Context, lang string, questions []preference_model.TestQuestion) []preference_model.TestQuestion {
	f.calls++
	f.gotLang = lang
	out := make([]preference_model.TestQuestion, len(questions))
	for i, q := range questions {
		out[i] = q
		if lang != "ca" {
			out[i].Title = "[" + lang + "] " + q.Title
			if q.Description != "" {
				out[i].Description = "[" + lang + "] " + q.Description
			}
		}
	}
	return out
}

func TestGetQuestions_Language(t *testing.T) {
	baseQuestions := []preference_model.TestQuestion{
		{EventID: 1, Title: "Concert", Description: "Música en directe"},
	}

	tests := []struct {
		name       string
		target     string
		translator *fakeTranslator
		wantCode   int
		wantCalls  int
		wantTitle  string
		wantDesc   string
	}{
		{
			name:       "lang query parameter translates title and description",
			target:     "/preferences/test?lang=es",
			translator: &fakeTranslator{},
			wantCode:   http.StatusOK,
			wantCalls:  1,
			wantTitle:  "[es] Concert",
			wantDesc:   "[es] Música en directe",
		},
		{
			name:       "lang query parameter wins over user language",
			target:     "/preferences/test?lang=en",
			translator: &fakeTranslator{userLang: "es"},
			wantCode:   http.StatusOK,
			wantCalls:  1,
			wantTitle:  "[en] Concert",
			wantDesc:   "[en] Música en directe",
		},
		{
			name:       "without lang parameter uses user settings language",
			target:     "/preferences/test",
			translator: &fakeTranslator{userLang: "es"},
			wantCode:   http.StatusOK,
			wantCalls:  1,
			wantTitle:  "[es] Concert",
			wantDesc:   "[es] Música en directe",
		},
		{
			name:       "catalan keeps original text",
			target:     "/preferences/test?lang=ca",
			translator: &fakeTranslator{userLang: "en"},
			wantCode:   http.StatusOK,
			wantCalls:  1,
			wantTitle:  "Concert",
			wantDesc:   "Música en directe",
		},
		{
			name:       "user language lookup error falls back to catalan",
			target:     "/preferences/test",
			translator: &fakeTranslator{langErr: errors.New("db down")},
			wantCode:   http.StatusOK,
			wantCalls:  1,
			wantTitle:  "Concert",
			wantDesc:   "Música en directe",
		},
		{
			name:       "invalid lang returns 400 without calling service",
			target:     "/preferences/test?lang=fr",
			translator: &fakeTranslator{},
			wantCode:   http.StatusBadRequest,
			wantCalls:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakePreferenceService{questions: baseQuestions}
			mux := http.NewServeMux()
			NewPreferenceHandler(svc, fakeResolver{id: validUserID}).
				WithTranslator(tt.translator).
				RegisterRoutes(mux)

			req := withFirebaseUser(httptest.NewRequest(http.MethodGet, tt.target, nil))
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code != tt.wantCode {
				t.Fatalf("status = %d, want %d (body %s)", rec.Code, tt.wantCode, rec.Body.String())
			}
			if svc.calls != tt.wantCalls {
				t.Errorf("service calls = %d, want %d", svc.calls, tt.wantCalls)
			}
			if tt.wantCode != http.StatusOK {
				return
			}
			var got []preference_model.TestQuestion
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if len(got) != 1 || got[0].Title != tt.wantTitle || got[0].Description != tt.wantDesc {
				t.Errorf("got = %+v, want title=%q desc=%q", got, tt.wantTitle, tt.wantDesc)
			}
		})
	}
}
