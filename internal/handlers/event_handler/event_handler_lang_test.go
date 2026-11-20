package event_handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"api-aprop/internal/clients/firebase_client"
	"api-aprop/internal/middleware/auth_middleware"
	"api-aprop/internal/models/event_model"
)

// fakeTranslator marks the title with the language so the tests can see what was asked for.
type fakeTranslator struct {
	userLang string
	langErr  error
	calls    int
	gotLang  string
}

func (f *fakeTranslator) UserLanguage(_ context.Context, _ string) (string, error) {
	return f.userLang, f.langErr
}

func (f *fakeTranslator) TranslateEvent(_ context.Context, lang string, e event_model.Event) event_model.Event {
	f.calls++
	f.gotLang = lang
	if lang != "ca" {
		e.Title = "[" + lang + "] " + e.Title
	}
	return e
}

func (f *fakeTranslator) TranslateEvents(_ context.Context, lang string, events []event_model.Event) []event_model.Event {
	f.calls++
	f.gotLang = lang
	out := make([]event_model.Event, len(events))
	for i, e := range events {
		out[i] = e
		if lang != "ca" {
			out[i].Title = "[" + lang + "] " + e.Title
		}
	}
	return out
}

func (f *fakeTranslator) TranslateMapEvents(_ context.Context, lang string, events []event_model.MapEvent) []event_model.MapEvent {
	f.calls++
	f.gotLang = lang
	out := make([]event_model.MapEvent, len(events))
	for i, e := range events {
		out[i] = e
		if lang != "ca" {
			out[i].Title = "[" + lang + "] " + e.Title
		}
	}
	return out
}

func serveWithTranslator(s *fakeEventService, tr *fakeTranslator, req *http.Request) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	NewEventHandler(s, fakeResolver{}).WithTranslator(tr).RegisterRoutes(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestGetEventLanguage(t *testing.T) {
	signedIn := func(target string) *http.Request {
		req := httptest.NewRequest(http.MethodGet, target, nil)
		return req.WithContext(auth_middleware.ContextWithUser(req.Context(), firebase_client.User{UID: firebaseUID}))
	}
	tests := []struct {
		name       string
		req        *http.Request
		translator *fakeTranslator
		wantTitle  string
		wantCode   int
	}{
		{"lang parameter", httptest.NewRequest(http.MethodGet, "/events/1?lang=es", nil), &fakeTranslator{}, "[es] Festa", http.StatusOK},
		{"lang parameter wins over the user language", signedIn("/events/1?lang=en"), &fakeTranslator{userLang: "es"}, "[en] Festa", http.StatusOK},
		{"user language", signedIn("/events/1"), &fakeTranslator{userLang: "es"}, "[es] Festa", http.StatusOK},
		{"user language lookup failure keeps the original", signedIn("/events/1"), &fakeTranslator{langErr: context.DeadlineExceeded}, "Festa", http.StatusOK},
		{"anonymous without lang is the original", httptest.NewRequest(http.MethodGet, "/events/1", nil), &fakeTranslator{userLang: "es"}, "Festa", http.StatusOK},
		{"catalan", httptest.NewRequest(http.MethodGet, "/events/1?lang=ca", nil), &fakeTranslator{}, "Festa", http.StatusOK},
		{"invalid lang", httptest.NewRequest(http.MethodGet, "/events/1?lang=fr", nil), &fakeTranslator{}, "", http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeEventService{event: event_model.Event{ID: 1, Title: "Festa"}}
			rec := serveWithTranslator(svc, tt.translator, tt.req)
			if rec.Code != tt.wantCode {
				t.Fatalf("status = %d, want %d: %s", rec.Code, tt.wantCode, rec.Body.String())
			}
			if tt.wantCode != http.StatusOK {
				return
			}
			var got event_model.Event
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			if got.Title != tt.wantTitle {
				t.Errorf("title = %q, want %q", got.Title, tt.wantTitle)
			}
		})
	}
}

func TestInvalidLangIsRejectedByEveryEventList(t *testing.T) {
	for _, target := range []string{
		"/events?lang=xx",
		"/events/search?q=a&lang=xx",
		"/events/map?lang=xx",
		"/event/map?lang=xx",
		"/events/popular?pageSize=1&pageNum=0&lang=xx",
		"/events/recommended?pageSize=1&pageNum=0&lang=xx",
	} {
		t.Run(target, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, target, nil)
			req.Header.Set("X-User-ID", validUserID)
			rec := serveWithTranslator(&fakeEventService{}, &fakeTranslator{}, req)
			if rec.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want 400", rec.Code)
			}
		})
	}
}

func TestEventListsAreTranslated(t *testing.T) {
	svc := &fakeEventService{
		mapEvents:         []event_model.MapEvent{{Event: event_model.Event{ID: 1, Title: "Mapa"}}},
		recommendedEvents: []event_model.Event{{ID: 2, Title: "Recomanat"}},
	}
	tests := []struct {
		target string
		want   string
	}{
		{"/events/search?q=jazz&lang=en", `[en] Concert de Jazz`},
		{"/events/map?lang=en", `[en] Mapa`},
		{"/events/recommended?pageSize=1&pageNum=0&lang=en", `[en] Recomanat`},
	}
	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.target, nil)
			req.Header.Set("X-User-ID", validUserID)
			rec := serveWithTranslator(svc, &fakeTranslator{}, req)
			if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), tt.want) {
				t.Errorf("status = %d, body = %s, want to contain %q", rec.Code, rec.Body.String(), tt.want)
			}
		})
	}
}

func TestWithoutTranslatorLangParameterIsIgnored(t *testing.T) {
	svc := &fakeEventService{event: event_model.Event{ID: 1, Title: "Festa"}}
	rec := serve(svc, httptest.NewRequest(http.MethodGet, "/events/1?lang=fr", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}
