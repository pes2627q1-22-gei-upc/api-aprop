package router

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"api-aprop/internal/handlers/assistant_handler"
	"api-aprop/internal/handlers/event_handler"
	"api-aprop/internal/handlers/signup_handler"
	"api-aprop/internal/models/event_model"
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

func (s *stubEventService) CreateEvent(e event_model.Event) (event_model.Event, error) {
	return e, nil
}

func (s *stubEventService) LoadOfficialEvents(_ context.Context, _ int) error {
	return nil
}

func TestRouter_MapRoutesAndEventDetailRoute(t *testing.T) {
	for _, path := range []string{"/event/map", "/events/map"} {
		t.Run("GET "+path, func(t *testing.T) {
			svc := &stubEventService{}
			mux := NewRouter(
				event_handler.NewEventHandler(svc),
				assistant_handler.NewAssistantHandler(nil),
				signup_handler.NewSignUpHandler(nil),
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
			event_handler.NewEventHandler(svc),
			assistant_handler.NewAssistantHandler(nil),
			signup_handler.NewSignUpHandler(nil),
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
