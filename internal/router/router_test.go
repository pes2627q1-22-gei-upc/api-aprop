package router

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"api-aprop/internal/handlers/assistant_handler"
	"api-aprop/internal/handlers/event_handler"
	"api-aprop/internal/handlers/signup_handler"
	"api-aprop/internal/models/event_model"

	"github.com/rs/cors"
)

type stubEventService struct {
	mapCalled bool
	getCalled bool
	gotID     int
}

func (s *stubEventService) GetEvent(id int) (event_model.Event, error) {
	s.getCalled = true
	s.gotID = id
	return event_model.Event{ID: id, Title: "Detail Event"}, nil
}

func (s *stubEventService) ListEvents(_ map[string]any) ([]event_model.Event, error) {
	return []event_model.Event{}, nil
}

func (s *stubEventService) GetRecommendedEvents(_ context.Context, _ string, _ int, _ int) ([]event_model.Event, error) {
	return []event_model.Event{}, nil
}

func (s *stubEventService) GetPopularEvents(_ int, _ int) ([]event_model.Event, error) {
	return []event_model.Event{}, nil
}

func (s *stubEventService) ListMapEvents(_ context.Context, _ event_model.MapFilter) ([]event_model.MapEvent, error) {
	s.mapCalled = true
	return []event_model.MapEvent{
		{
			Event:      event_model.Event{ID: 1, Title: "Map Event", Latitude: 41.38, Longitude: 2.17},
			IsOfficial: true,
			HasFriends: true,
		},
	}, nil
}

func (s *stubEventService) UpdateUnofficialEvent(_ context.Context, id int, e event_model.Event) (event_model.Event, error) {
	e.ID = id
	return e, nil
}

func (s *stubEventService) CancelUnofficialEvent(_ context.Context, _ int) (event_model.Cancellation, error) {
	return event_model.Cancellation{}, nil
}

func (s *stubEventService) DeleteUnofficialEvent(_ context.Context, _ int) error {
	return nil
}

func (s *stubEventService) ResolveUserID(_ context.Context, identifier string) (string, error) {
	return identifier, nil
}

func (s *stubEventService) LoadOfficialEvents(_ context.Context, _ int) error {
	return nil
}

func (s *stubEventService) GetCategories() ([]event_model.Category, error) {
	return nil, nil
}

func (s *stubEventService) GetCategory(_ int) (event_model.Category, error) {
	return event_model.Category{}, nil
}

func (s *stubEventService) SearchEvents(_ context.Context, query string, pageNum, pageSize int) (event_model.SearchResult, error) {
	return event_model.SearchResult{}, nil
}

func (s *stubEventService) CreateUnofficialEvent(_ context.Context, _ string, e event_model.Event) (event_model.Event, error) {
	return e, nil
}

func (s *stubEventService) UploadEventImage(_ context.Context, _ int, _ string, _ string, _ io.Reader) (string, error) {
	return "https://s3/events/1/poster.jpg", nil
}

func (s *stubEventService) GetReceivedRatings(id int) ([]event_model.ReceivedRating, error) {
	return []event_model.ReceivedRating{}, nil
}

var noopAuth = func(next http.Handler) http.Handler { return next }

var noopCORS = func(next http.Handler) http.Handler { return next }

func TestRouter_MapRoutesAndEventDetailRoute(t *testing.T) {
	for _, path := range []string{"/event/map", "/events/map"} {
		t.Run("GET "+path, func(t *testing.T) {
			svc := &stubEventService{}
			mux := NewRouter(
				noopCORS,
				noopAuth,
				event_handler.NewEventHandler(svc, svc),
				assistant_handler.NewAssistantHandler(nil, nil),
				signup_handler.NewSignUpHandler(nil, nil),
			)

			req := httptest.NewRequest(http.MethodGet, path+"?minLat=41.0&maxLat=42.0&minLng=2.0&maxLng=3.0", nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
			}
			if !svc.mapCalled || svc.getCalled {
				t.Fatalf("expected ListMapEvents to be called (mapCalled=%v, getCalled=%v)", svc.mapCalled, svc.getCalled)
			}
			var got []event_model.MapEvent
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if len(got) != 1 || got[0].ID != 1 || !got[0].IsOfficial || !got[0].HasFriends {
				t.Errorf("unexpected response: %+v", got)
			}
		})
	}

	t.Run("GET /events/{id} still routes to GetEvent", func(t *testing.T) {
		svc := &stubEventService{}
		mux := NewRouter(
			noopCORS,
			noopAuth,
			event_handler.NewEventHandler(svc, svc),
			assistant_handler.NewAssistantHandler(nil, nil),
			signup_handler.NewSignUpHandler(nil, nil),
		)

		req := httptest.NewRequest(http.MethodGet, "/events/75", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		if !svc.getCalled || svc.gotID != 75 || svc.mapCalled {
			t.Fatalf("expected GetEvent(75) to be called, got getCalled=%v gotID=%d mapCalled=%v", svc.getCalled, svc.gotID, svc.mapCalled)
		}
	})
}

// The CORS preflight has no bearer token, so it must be answered before the auth middleware rejects it.
func TestRouter_CORSPreflightSkipsAuth(t *testing.T) {
	rejectAuth := func(http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		})
	}
	mux := NewRouter(
		cors.AllowAll().Handler,
		rejectAuth,
		event_handler.NewEventHandler(&stubEventService{}, &stubEventService{}),
	)

	req := httptest.NewRequest(http.MethodOptions, "/events", nil)
	req.Header.Set("Origin", "https://admin.aprop.cat")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("preflight status = %d, want 204", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, "*")
	}

	// Real requests still go through the auth middleware, with the CORS header so the browser can read the 401
	req = httptest.NewRequest(http.MethodGet, "/events", nil)
	req.Header.Set("Origin", "https://admin.aprop.cat")
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("request status = %d, want 401 from the auth middleware", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Access-Control-Allow-Origin = %q on the 401, want %q", got, "*")
	}
}
