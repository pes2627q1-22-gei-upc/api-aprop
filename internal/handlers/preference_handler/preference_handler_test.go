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
	"api-aprop/internal/services/auth_service"
	"api-aprop/internal/services/preference_service"
	"api-aprop/internal/services/translation_service"
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
func getAndCheck(t *testing.T, path, userHeader string, svc fakePreferenceService, wantStatus, wantCalls int, wantError string) (*httptest.ResponseRecorder, bool) {
	t.Helper()
	rec, _, ok := serveAndCheck(t, http.MethodGet, path, "", userHeader, svc, wantStatus, wantCalls, wantError)
	return rec, ok
}

func serveAndCheck(
	t *testing.T,
	method, path, body, userHeader string,
	svc fakePreferenceService,
	wantStatus, wantCalls int,
	wantError string,
) (*httptest.ResponseRecorder, *fakePreferenceService, bool) {
	t.Helper()
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, bytes.NewBufferString(body))
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	if userHeader != "" {
		req.Header.Set(requester.UserIDHeader, userHeader)
	}
	rec := serve(&svc, req)
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
		return rec, &svc, false
	}
	return rec, &svc, true
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
func assertTranslatesFirebaseUID(t *testing.T, method, path string, wantStatus int, call func(*PreferenceHandler, http.ResponseWriter, *http.Request)) {
	t.Helper()
	s := &fakePreferenceService{}
	rec := httptest.NewRecorder()
	call(NewPreferenceHandler(s, fakeResolver{id: validUserID}), rec, withFirebaseUser(httptest.NewRequest(method, path, nil)))
	if rec.Code != wantStatus || s.gotUserID != validUserID {
		t.Fatalf("status = %d, user passed to service = %q; want %d and %q", rec.Code, s.gotUserID, wantStatus, validUserID)
	}

	rec = httptest.NewRecorder()
	call(NewPreferenceHandler(s, fakeResolver{err: auth_service.ErrUserNotRegistered}), rec, withFirebaseUser(httptest.NewRequest(method, path, nil)))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("not registered: status = %d, want 401", rec.Code)
	}
}

func TestGetUserPreferences_TranslatesFirebaseUID(t *testing.T) {
	assertTranslatesFirebaseUID(t, http.MethodGet, "/preferences", http.StatusOK, (*PreferenceHandler).GetUserPreferences)
}

func TestGetTestStatus_TranslatesFirebaseUID(t *testing.T) {
	assertTranslatesFirebaseUID(t, http.MethodGet, "/preferences/test/status", http.StatusOK, (*PreferenceHandler).GetTestStatus)
}

func TestGetTestStatus(t *testing.T) {
	fixedTime := time.Date(2026, 10, 24, 18, 0, 0, 0, time.UTC)

	type statusCase struct {
		name       string
		userHeader string
		service    fakePreferenceService
		wantStatus int
		wantCalls  int
		wantError  string
		wantJSON   map[string]any
	}
	okCase := func(name string, st preference_model.TestStatus, wantDate any) statusCase {
		return statusCase{
			name:       name,
			userHeader: validUserID,
			service:    fakePreferenceService{status: st},
			wantStatus: http.StatusOK,
			wantCalls:  1,
			wantJSON: map[string]any{
				"completed":      st.Completed,
				"prompted":       st.Prompted,
				"in-progress":    st.InProgress,
				"answered-count": float64(st.AnsweredCount),
				"date-taken":     wantDate,
			},
		}
	}

	tests := []statusCase{
		okCase(
			"completed test returns completed true, prompted true and RFC3339 date-taken",
			preference_model.TestStatus{Completed: true, Prompted: true, DateTaken: &fixedTime},
			"2026-10-24T18:00:00Z",
		),
		okCase(
			"uncompleted new user returns completed false, prompted false and explicit null date-taken",
			preference_model.TestStatus{},
			nil,
		),
		okCase(
			"paused test returns in-progress true and answered-count",
			preference_model.TestStatus{Prompted: true, InProgress: true, AnsweredCount: 6},
			nil,
		),
		{name: "missing X-User-ID header returns 401", wantStatus: http.StatusUnauthorized, wantError: "missing X-User-ID header"},
		{name: "invalid user id returns 400", userHeader: "bad-uuid", service: fakePreferenceService{statusErr: preference_service.ErrInvalidUserID}, wantStatus: http.StatusBadRequest, wantCalls: 1, wantError: "invalid user id"},
		{name: "service error returns 500", userHeader: validUserID, service: fakePreferenceService{statusErr: errors.New("db error")}, wantStatus: http.StatusInternalServerError, wantCalls: 1, wantError: "internal server error"},
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
	svc := fakePreferenceService{
		partialResponse: preference_model.SubmitTestResponse{
			UserID:        validUserID,
			DateTaken:     fixedTime,
			Preferences:   []preference_model.UserPreference{},
			Partial:       true,
			AnsweredCount: 2,
		},
	}

	rec, s, _ := serveAndCheck(
		t,
		http.MethodPost,
		"/preferences/test",
		`{"answers":[{"event-id":1,"likes":true},{"event-id":2,"likes":false}],"partial":true}`,
		validUserID,
		svc,
		http.StatusOK,
		1,
		"",
	)
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
		{"success returns 204", validUserID, fakePreferenceService{}, http.StatusNoContent, 1},
		{"missing header returns 401", "", fakePreferenceService{}, http.StatusUnauthorized, 0},
		{"invalid user id returns 400", "bad-uuid", fakePreferenceService{markPromptedErr: preference_service.ErrInvalidUserID}, http.StatusBadRequest, 1},
		{"service error returns 500", validUserID, fakePreferenceService{markPromptedErr: errors.New("db error")}, http.StatusInternalServerError, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, s, _ := serveAndCheck(t, http.MethodPost, "/preferences/test/prompted", "", tt.userHeader, tt.service, tt.wantStatus, tt.wantCalls, "")
			if s.promptCalls != tt.wantCalls {
				t.Errorf("promptCalls = %d, want %d", s.promptCalls, tt.wantCalls)
			}
		})
	}
}

func TestMarkTestPrompted_TranslatesFirebaseUID(t *testing.T) {
	assertTranslatesFirebaseUID(t, http.MethodPost, "/preferences/test/prompted", http.StatusNoContent, (*PreferenceHandler).MarkTestPrompted)
}

type fakeTranslator struct {
	translation_service.Service
	userLang string
	langErr  error
	calls    int
	gotLang  string
}

func (f *fakeTranslator) UserLanguage(_ context.Context, _ string) (string, error) {
	return f.userLang, f.langErr
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
		name      string
		target    string
		userLang  string
		langErr   error
		wantCode  int
		wantCalls int
		wantTitle string
		wantDesc  string
	}{
		{"lang query parameter translates title and description", "/preferences/test?lang=es", "", nil, http.StatusOK, 1, "[es] Concert", "[es] Música en directe"},
		{"lang query parameter wins over user language", "/preferences/test?lang=en", "es", nil, http.StatusOK, 1, "[en] Concert", "[en] Música en directe"},
		{"without lang parameter uses user settings language", "/preferences/test", "es", nil, http.StatusOK, 1, "[es] Concert", "[es] Música en directe"},
		{"catalan keeps original text", "/preferences/test?lang=ca", "en", nil, http.StatusOK, 1, "Concert", "Música en directe"},
		{"user language lookup error falls back to catalan", "/preferences/test", "", errors.New("db down"), http.StatusOK, 1, "Concert", "Música en directe"},
		{"invalid lang returns 400 without calling service", "/preferences/test?lang=fr", "", nil, http.StatusBadRequest, 0, "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakePreferenceService{questions: baseQuestions}
			mux := http.NewServeMux()
			NewPreferenceHandler(svc, fakeResolver{id: validUserID}).
				WithTranslator(&fakeTranslator{userLang: tt.userLang, langErr: tt.langErr}).
				RegisterRoutes(mux)

			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, withFirebaseUser(httptest.NewRequest(http.MethodGet, tt.target, nil)))

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
