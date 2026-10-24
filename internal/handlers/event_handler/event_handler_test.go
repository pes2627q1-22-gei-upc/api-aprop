package event_handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"api-aprop/internal/models/event_model"
	"api-aprop/internal/services/event_service"
)

const validUserID = "3f2504e0-4f89-11d3-9a0c-0305e82c3301"

type fakeEventService struct {
	mapEvents []event_model.MapEvent
	mapErr    error

	mapCalls  int
	gotFilter event_model.MapFilter
}

func (f *fakeEventService) GetEvent(_ int) (event_model.Event, error) {
	return event_model.Event{}, nil
}

func (f *fakeEventService) ListEvents(_ map[string]any) ([]event_model.Event, error) {
	return nil, nil
}

func (f *fakeEventService) ListMapEvents(_ context.Context, filter event_model.MapFilter) ([]event_model.MapEvent, error) {
	f.mapCalls++
	f.gotFilter = filter
	return f.mapEvents, f.mapErr
}

func (f *fakeEventService) CreateEvent(e event_model.Event) (event_model.Event, error) {
	return e, nil
}

func (f *fakeEventService) LoadOfficialEvents(_ context.Context, _ int) error {
	return nil
}

func (f *fakeEventService) GetCategories() ([]event_model.Category, error) {
	return nil, nil
}

func (f *fakeEventService) GetCategory(_ int) (event_model.Category, error) {
	return event_model.Category{}, nil
}

func (f *fakeEventService) SearchEvents(_ context.Context, query string, pageNum, pageSize int) (event_model.SearchResult, error) {
	return event_model.SearchResult{
		Query:    query,
		Total:    1,
		PageNum:  pageNum,
		PageSize: pageSize,
		Events:   []event_model.Event{{ID: 1, Title: "Concert de Jazz"}},
	}, nil
}

func newMux(s event_service.Service) *http.ServeMux {
	h := NewEventHandler(s)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	return mux
}

func serve(s event_service.Service, req *http.Request) *httptest.ResponseRecorder {
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

func TestListMapEvents(t *testing.T) {
	sample := []event_model.MapEvent{
		{
			Event:      event_model.Event{ID: 10, Title: "Festa Major", Latitude: 41.38, Longitude: 2.17},
			IsOfficial: true,
			HasFriends: true,
		},
		{
			Event:      event_model.Event{ID: 20, Title: "Quedada", Latitude: 41.39, Longitude: 2.18},
			IsOfficial: false,
			HasFriends: false,
		},
	}

	tests := []struct {
		name       string
		path       string
		userHeader string
		service    fakeEventService
		wantStatus int
		wantCalls  int
		wantFilter event_model.MapFilter
		wantIDs    []int
		wantError  string
	}{
		{
			name:       "without bounding box defaults to world bounds",
			path:       "/event/map",
			service:    fakeEventService{mapEvents: sample},
			wantStatus: http.StatusOK,
			wantCalls:  1,
			wantFilter: event_model.MapFilter{MinLat: -90, MaxLat: 90, MinLng: -180, MaxLng: 180},
			wantIDs:    []int{10, 20},
		},
		{
			name:       "with bounding box and X-User-ID header",
			path:       "/event/map?minLat=41.3&maxLat=41.5&minLng=2.1&maxLng=2.3",
			userHeader: validUserID,
			service:    fakeEventService{mapEvents: sample},
			wantStatus: http.StatusOK,
			wantCalls:  1,
			wantFilter: event_model.MapFilter{MinLat: 41.3, MaxLat: 41.5, MinLng: 2.1, MaxLng: 2.3, UserID: validUserID},
			wantIDs:    []int{10, 20},
		},
		{
			name:       "events/map alias and minLon/maxLon aliases",
			path:       "/events/map?minLat=41.3&maxLat=41.5&minLon=2.1&maxLon=2.3&userId=" + validUserID,
			service:    fakeEventService{mapEvents: sample},
			wantStatus: http.StatusOK,
			wantCalls:  1,
			wantFilter: event_model.MapFilter{MinLat: 41.3, MaxLat: 41.5, MinLng: 2.1, MaxLng: 2.3, UserID: validUserID},
			wantIDs:    []int{10, 20},
		},
		{
			name:       "empty list returns empty JSON array",
			path:       "/event/map",
			service:    fakeEventService{mapEvents: []event_model.MapEvent{}},
			wantStatus: http.StatusOK,
			wantCalls:  1,
			wantFilter: event_model.MapFilter{MinLat: -90, MaxLat: 90, MinLng: -180, MaxLng: 180},
			wantIDs:    []int{},
		},
		{
			name:       "partial bounding box parameters returns 400",
			path:       "/event/map?minLat=41.3&maxLat=41.5",
			wantStatus: http.StatusBadRequest,
			wantError:  "missing bounding box parameters, expected minLat, maxLat, minLng, maxLng",
		},
		{
			name:       "invalid minLat coordinate returns 400",
			path:       "/event/map?minLat=abc&maxLat=41.5&minLng=2.1&maxLng=2.3",
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid minLat coordinate",
		},
		{
			name:       "invalid maxLng coordinate returns 400",
			path:       "/event/map?minLat=41.3&maxLat=41.5&minLng=2.1&maxLng=NaN",
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid maxLng coordinate",
		},
		{
			name:       "invalid bounding box from service returns 400",
			path:       "/event/map?minLat=42.0&maxLat=41.0&minLng=2.1&maxLng=2.3",
			service:    fakeEventService{mapErr: event_service.ErrInvalidBoundingBox},
			wantStatus: http.StatusBadRequest,
			wantCalls:  1,
			wantError:  "invalid bounding box",
		},
		{
			name:       "invalid user id from service returns 400",
			path:       "/event/map",
			userHeader: "not-a-uuid",
			service:    fakeEventService{mapErr: event_service.ErrInvalidUserID},
			wantStatus: http.StatusBadRequest,
			wantCalls:  1,
			wantError:  "invalid user id",
		},
		{
			name:       "internal repository error returns 500",
			path:       "/event/map",
			service:    fakeEventService{mapErr: errors.New("db down")},
			wantStatus: http.StatusInternalServerError,
			wantCalls:  1,
			wantError:  "internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.service
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			if tt.userHeader != "" {
				req.Header.Set(UserIDHeader, tt.userHeader)
			}

			rec := serve(&s, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if s.mapCalls != tt.wantCalls {
				t.Errorf("service calls = %d, want %d", s.mapCalls, tt.wantCalls)
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
			if s.gotFilter != tt.wantFilter {
				t.Errorf("filter = %+v, want %+v", s.gotFilter, tt.wantFilter)
			}
			if len(tt.wantIDs) == 0 && strings.TrimSpace(rec.Body.String()) != "[]" {
				t.Errorf("body = %q, want []", rec.Body.String())
			}
			var events []event_model.MapEvent
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
				if events[i].IsOfficial != sample[i].IsOfficial {
					t.Errorf("event[%d].IsOfficial = %v, want %v", i, events[i].IsOfficial, sample[i].IsOfficial)
				}
				if events[i].HasFriends != sample[i].HasFriends {
					t.Errorf("event[%d].HasFriends = %v, want %v", i, events[i].HasFriends, sample[i].HasFriends)
				}
			}
		})
	}
}

func TestSearchEvents(t *testing.T) {
	s := &fakeEventService{}
	mux := newMux(s)

	req := httptest.NewRequest(http.MethodGet, "/events/search?q=jazz", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var res event_model.SearchResult
	if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode json: %v", err)
	}
	if res.Total != 1 || res.Query != "jazz" || len(res.Events) != 1 {
		t.Errorf("unexpected search result: %+v", res)
	}
}
