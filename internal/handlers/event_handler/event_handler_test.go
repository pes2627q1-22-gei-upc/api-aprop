package event_handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"api-aprop/internal/clients/firebase_client"
	"api-aprop/internal/handlers/requester"
	"api-aprop/internal/middleware/auth_middleware"
	"api-aprop/internal/models/event_model"
	"api-aprop/internal/services/auth_service"
	"api-aprop/internal/services/event_service"
)

const (
	validUserID = "3f2504e0-4f89-11d3-9a0c-0305e82c3301"
	otherUserID = "9b2e6c1a-1d3f-4c5e-8a7b-6f0e2d4c8a19"
	// firebaseUID is the Firebase UID of the user validUserID
	firebaseUID = "Vw3kU1pZ8rQ2hXyT6mN9aB4cD7e1"
)

// unofficialEvent returns an event created by the given user.
func unofficialEvent(t *testing.T, id int, creator string) event_model.Event {
	t.Helper()
	var uuid pgtype.UUID
	if err := uuid.Scan(creator); err != nil {
		t.Fatal(err)
	}
	return event_model.Event{ID: id, Title: "Taller de Teatre", Creator: uuid, DateCreated: time.Now().UTC()}
}

// officialEvent returns an event loaded from the Generalitat open data (it has a code and no creator).
func officialEvent(id int) event_model.Event {
	return event_model.Event{ID: id, Title: "Festa Major", Code: 20261026001, IsOfficial: true}
}

type fakeEventService struct {
	event                event_model.Event
	getErr               error
	updateErr            error
	deleteErr            error
	cancelErr            error
	cancelCalls          int
	notified             int64
	mapEvents            []event_model.MapEvent
	mapErr               error
	createdUnofficial    event_model.Event
	createUnofficialErr  error
	uploadedImageURL     string
	uploadImageErr       error
	listErr              error
	popularErr           error
	categoryErr          error
	gotListParams        map[string]any
	recommendedEvents    []event_model.Event
	recommendedErr       error
	recommendedCalls     int
	gotRecommendedUserID string
	gotRecommendedSize   int
	gotRecommendedNum    int

	mapCalls        int
	unofficialCalls int
	updateCalls     int
	deleteCalls     int
	uploadCalls     int
	gotFilter       event_model.MapFilter
}

func (f *fakeEventService) GetEvent(_ int) (event_model.Event, error) {
	return f.event, f.getErr
}

func (f *fakeEventService) CancelUnofficialEvent(_ context.Context, id int) (event_model.Cancellation, error) {
	f.cancelCalls++
	if f.cancelErr != nil {
		return event_model.Cancellation{}, f.cancelErr
	}
	return event_model.Cancellation{EventID: id, State: event_model.UnofficialStateCancelled, NotifiedUsers: f.notified}, nil
}

func (f *fakeEventService) DeleteUnofficialEvent(_ context.Context, _ int) error {
	f.deleteCalls++
	return f.deleteErr
}

func (f *fakeEventService) UpdateUnofficialEvent(_ context.Context, id int, e event_model.Event) (event_model.Event, error) {
	f.updateCalls++
	e.ID = id
	return e, f.updateErr
}

func (f *fakeEventService) ListEvents(params map[string]any) ([]event_model.Event, error) {
	f.gotListParams = params
	return nil, f.listErr
}

func (f *fakeEventService) GetRecommendedEvents(_ context.Context, userID string, pageSize int, pageNum int) ([]event_model.Event, error) {
	f.recommendedCalls++
	f.gotRecommendedUserID = userID
	f.gotRecommendedSize = pageSize
	f.gotRecommendedNum = pageNum
	return f.recommendedEvents, f.recommendedErr
}

func (f *fakeEventService) GetPopularEvents(_ int, _ int) ([]event_model.Event, error) {
	return nil, f.popularErr
}

func (f *fakeEventService) ListMapEvents(_ context.Context, filter event_model.MapFilter) ([]event_model.MapEvent, error) {
	f.mapCalls++
	f.gotFilter = filter
	return f.mapEvents, f.mapErr
}

// fakeResolver translates the Firebase UID of the test user and accepts the test UUIDs as they are,
// like auth_service does.
type fakeResolver struct{}

func (fakeResolver) ResolveUserID(_ context.Context, identifier string) (string, error) {
	if identifier == "" || identifier == "unregistered-firebase-uid" {
		return "", auth_service.ErrUserNotRegistered
	}
	if identifier == firebaseUID {
		return validUserID, nil
	}
	return identifier, nil
}

func (f *fakeEventService) LoadOfficialEvents(_ context.Context, _ int) error {
	return nil
}

func (f *fakeEventService) GetCategories() ([]event_model.Category, error) {
	return nil, nil
}

func (f *fakeEventService) GetCategory(_ int) (event_model.Category, error) {
	return event_model.Category{}, f.categoryErr
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

func (f *fakeEventService) CreateUnofficialEvent(_ context.Context, _ string, e event_model.Event) (event_model.Event, error) {
	f.unofficialCalls++
	f.createdUnofficial = e
	return e, f.createUnofficialErr
}

func (f *fakeEventService) UploadEventImage(_ context.Context, _ int, _ string, _ string, _ io.Reader) (string, error) {
	f.uploadCalls++
	return f.uploadedImageURL, f.uploadImageErr
}

func (f *fakeEventService) GetReceivedRatings(id int) ([]event_model.ReceivedRating, error) {
	return []event_model.ReceivedRating{}, nil
}

func newMux(s event_service.Service) *http.ServeMux {
	h := NewEventHandler(s, fakeResolver{})
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
			path:       "/events/map?minLat=41.3&maxLat=41.5&minLon=2.1&maxLon=2.3",
			userHeader: validUserID,
			service:    fakeEventService{mapEvents: sample},
			wantStatus: http.StatusOK,
			wantCalls:  1,
			wantFilter: event_model.MapFilter{MinLat: 41.3, MaxLat: 41.5, MinLng: 2.1, MaxLng: 2.3, UserID: validUserID},
			wantIDs:    []int{10, 20},
		},
		{
			name:       "userId query parameter is ignored",
			path:       "/event/map?userId=" + otherUserID,
			service:    fakeEventService{mapEvents: sample},
			wantStatus: http.StatusOK,
			wantCalls:  1,
			wantFilter: event_model.MapFilter{MinLat: -90, MaxLat: 90, MinLng: -180, MaxLng: 180},
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
				req.Header.Set(requester.UserIDHeader, tt.userHeader)
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

func TestCreateUnofficialEventHandler(t *testing.T) {
	futureStart := time.Now().UTC().Add(24 * time.Hour)
	eventJSON := `{"title":"Taller de Teatre","description":"Taller obert per a joves","categories":["Teatre"],"start-date":"` + futureStart.Format(time.RFC3339) + `"}`

	tests := []struct {
		name       string
		body       string
		userID     string
		service    fakeEventService
		wantStatus int
		wantCalls  int
	}{
		{
			name:       "valid unofficial event creation",
			body:       eventJSON,
			userID:     validUserID,
			service:    fakeEventService{},
			wantStatus: http.StatusCreated,
			wantCalls:  1,
		},
		{
			name:       "validation failure title required",
			body:       `{"title":"","description":"Desc","categories":["Teatre"]}`,
			userID:     validUserID,
			service:    fakeEventService{createUnofficialErr: event_service.ErrInvalidEventTitle},
			wantStatus: http.StatusBadRequest,
			wantCalls:  1,
		},
		{
			name:       "invalid json body",
			body:       `{not-valid-json`,
			userID:     validUserID,
			wantStatus: http.StatusBadRequest,
			wantCalls:  0,
		},
		{
			name:       "missing user",
			body:       eventJSON,
			wantStatus: http.StatusUnauthorized,
			wantCalls:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.service
			req := httptest.NewRequest(http.MethodPost, "/events", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			if tt.userID != "" {
				req.Header.Set(requester.UserIDHeader, tt.userID)
			}

			rec := serve(&s, req)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if s.unofficialCalls != tt.wantCalls {
				t.Errorf("service calls = %d, want %d", s.unofficialCalls, tt.wantCalls)
			}
		})
	}
}

func TestUploadEventImageHandler(t *testing.T) {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("image", "poster.jpg")
	_, _ = part.Write([]byte("fake-jpeg-content"))
	writer.Close()

	tests := []struct {
		name       string
		path       string
		userID     string
		service    fakeEventService
		wantStatus int
		wantCalls  int
	}{
		{
			name:       "successful image upload",
			path:       "/events/15/image",
			userID:     validUserID,
			service:    fakeEventService{event: unofficialEvent(t, 15, validUserID), uploadedImageURL: "https://s3/events/15/poster.jpg"},
			wantStatus: http.StatusCreated,
			wantCalls:  1,
		},
		{
			name:       "event not found",
			path:       "/events/99/image",
			userID:     validUserID,
			service:    fakeEventService{getErr: event_service.ErrEventNotFound},
			wantStatus: http.StatusNotFound,
			wantCalls:  0,
		},
		{
			name:       "another user cannot change the image",
			path:       "/events/15/image",
			userID:     otherUserID,
			service:    fakeEventService{event: unofficialEvent(t, 15, validUserID)},
			wantStatus: http.StatusForbidden,
			wantCalls:  0,
		},
		{
			name:       "official event image cannot be changed",
			path:       "/events/15/image",
			userID:     validUserID,
			service:    fakeEventService{event: officialEvent(15)},
			wantStatus: http.StatusForbidden,
			wantCalls:  0,
		},
		{
			name:       "missing user",
			path:       "/events/15/image",
			service:    fakeEventService{event: unofficialEvent(t, 15, validUserID)},
			wantStatus: http.StatusUnauthorized,
			wantCalls:  0,
		},
		{
			name:       "invalid event id",
			path:       "/events/abc/image",
			userID:     validUserID,
			wantStatus: http.StatusBadRequest,
			wantCalls:  0,
		},
		{
			name:       "storage error returns 502",
			path:       "/events/15/image",
			userID:     validUserID,
			service:    fakeEventService{event: unofficialEvent(t, 15, validUserID), uploadImageErr: event_service.ErrImageUploadFailed},
			wantStatus: http.StatusBadGateway,
			wantCalls:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.service
			req := httptest.NewRequest(http.MethodPost, tt.path, bytes.NewReader(body.Bytes()))
			req.Header.Set("Content-Type", writer.FormDataContentType())
			if tt.userID != "" {
				req.Header.Set(requester.UserIDHeader, tt.userID)
			}

			rec := serve(&s, req)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if s.uploadCalls != tt.wantCalls {
				t.Errorf("service upload calls = %d, want %d", s.uploadCalls, tt.wantCalls)
			}
		})
	}
}

func TestUpdateEventHandler(t *testing.T) {
	futureStart := time.Now().UTC().Add(24 * time.Hour)
	eventJSON := `{"title":"Taller de Teatre (nova data)","description":"Taller obert per a joves","categories":["Teatre"],"start-date":"` + futureStart.Format(time.RFC3339) + `"}`

	tests := []struct {
		name       string
		path       string
		body       string
		userID     string
		service    fakeEventService
		wantStatus int
		wantCalls  int
	}{
		{
			name:       "creator updates the event",
			path:       "/events/7",
			body:       eventJSON,
			userID:     validUserID,
			service:    fakeEventService{event: unofficialEvent(t, 7, validUserID)},
			wantStatus: http.StatusOK,
			wantCalls:  1,
		},
		{
			name:       "another user cannot update the event",
			path:       "/events/7",
			body:       eventJSON,
			userID:     otherUserID,
			service:    fakeEventService{event: unofficialEvent(t, 7, validUserID)},
			wantStatus: http.StatusForbidden,
			wantCalls:  0,
		},
		{
			name:       "official events cannot be updated",
			path:       "/events/7",
			body:       eventJSON,
			userID:     validUserID,
			service:    fakeEventService{event: officialEvent(7)},
			wantStatus: http.StatusForbidden,
			wantCalls:  0,
		},
		{
			name:       "missing user",
			path:       "/events/7",
			body:       eventJSON,
			service:    fakeEventService{event: unofficialEvent(t, 7, validUserID)},
			wantStatus: http.StatusUnauthorized,
			wantCalls:  0,
		},
		{
			name:       "event not found",
			path:       "/events/99",
			body:       eventJSON,
			userID:     validUserID,
			service:    fakeEventService{getErr: event_service.ErrEventNotFound},
			wantStatus: http.StatusNotFound,
			wantCalls:  0,
		},
		{
			name:       "validation error from the service",
			path:       "/events/7",
			body:       `{"title":"","description":"Desc","categories":["Teatre"]}`,
			userID:     validUserID,
			service:    fakeEventService{event: unofficialEvent(t, 7, validUserID), updateErr: event_service.ErrInvalidEventTitle},
			wantStatus: http.StatusBadRequest,
			wantCalls:  1,
		},
		{
			name:       "withdrawn event is not active",
			path:       "/events/7",
			body:       eventJSON,
			userID:     validUserID,
			service:    fakeEventService{event: unofficialEvent(t, 7, validUserID), updateErr: fmt.Errorf("update event 7: %w", event_service.ErrEventNotActive)},
			wantStatus: http.StatusConflict,
			wantCalls:  1,
		},
		{
			name:       "invalid json body",
			path:       "/events/7",
			body:       `{not-valid-json`,
			userID:     validUserID,
			service:    fakeEventService{event: unofficialEvent(t, 7, validUserID)},
			wantStatus: http.StatusBadRequest,
			wantCalls:  0,
		},
		{
			name:       "invalid event id",
			path:       "/events/abc",
			body:       eventJSON,
			userID:     validUserID,
			wantStatus: http.StatusBadRequest,
			wantCalls:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.service
			req := httptest.NewRequest(http.MethodPut, tt.path, strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			if tt.userID != "" {
				req.Header.Set(requester.UserIDHeader, tt.userID)
			}

			rec := serve(&s, req)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if s.updateCalls != tt.wantCalls {
				t.Errorf("service update calls = %d, want %d", s.updateCalls, tt.wantCalls)
			}
		})
	}
}

func TestDeleteEventHandler(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		userID     string
		service    fakeEventService
		wantStatus int
		wantCalls  int
	}{
		{
			name:       "creator deletes the event",
			path:       "/events/7",
			userID:     validUserID,
			service:    fakeEventService{event: unofficialEvent(t, 7, validUserID)},
			wantStatus: http.StatusNoContent,
			wantCalls:  1,
		},
		{
			name:       "another user cannot delete the event",
			path:       "/events/7",
			userID:     otherUserID,
			service:    fakeEventService{event: unofficialEvent(t, 7, validUserID)},
			wantStatus: http.StatusForbidden,
			wantCalls:  0,
		},
		{
			name:       "official events cannot be deleted",
			path:       "/events/7",
			userID:     validUserID,
			service:    fakeEventService{event: officialEvent(7)},
			wantStatus: http.StatusForbidden,
			wantCalls:  0,
		},
		{
			name:       "missing user",
			path:       "/events/7",
			service:    fakeEventService{event: unofficialEvent(t, 7, validUserID)},
			wantStatus: http.StatusUnauthorized,
			wantCalls:  0,
		},
		{
			name:       "event not found",
			path:       "/events/99",
			userID:     validUserID,
			service:    fakeEventService{getErr: event_service.ErrEventNotFound},
			wantStatus: http.StatusNotFound,
			wantCalls:  0,
		},
		{
			name:       "invalid event id",
			path:       "/events/abc",
			userID:     validUserID,
			wantStatus: http.StatusBadRequest,
			wantCalls:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.service
			req := httptest.NewRequest(http.MethodDelete, tt.path, nil)
			if tt.userID != "" {
				req.Header.Set(requester.UserIDHeader, tt.userID)
			}

			rec := serve(&s, req)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if s.deleteCalls != tt.wantCalls {
				t.Errorf("service delete calls = %d, want %d", s.deleteCalls, tt.wantCalls)
			}
		})
	}
}

func TestListEvents(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		listErr    error
		wantStatus int
		wantParams map[string]any
	}{
		{
			name:       "default page",
			path:       "/events",
			wantStatus: http.StatusOK,
			wantParams: map[string]any{"pageSize": 20, "pageNum": 0, "filterCategory": 0},
		},
		{
			name:       "page size zero uses the default",
			path:       "/events?pageSize=0&pageNum=2&filterCategory=5",
			wantStatus: http.StatusOK,
			wantParams: map[string]any{"pageSize": 20, "pageNum": 2, "filterCategory": 5},
		},
		{name: "page size not a number", path: "/events?pageSize=abc", wantStatus: http.StatusBadRequest},
		{name: "page number not a number", path: "/events?pageNum=abc", wantStatus: http.StatusBadRequest},
		{name: "category not a number", path: "/events?filterCategory=music", wantStatus: http.StatusBadRequest},
		{
			name:       "invalid filter",
			path:       "/events?filterFree=maybe",
			listErr:    fmt.Errorf("%w: filterFree", event_service.ErrInvalidFilter),
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := fakeEventService{listErr: tt.listErr}
			rec := serve(&s, httptest.NewRequest(http.MethodGet, tt.path, nil))
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d: %s", rec.Code, tt.wantStatus, rec.Body.String())
			}
			for k, want := range tt.wantParams {
				if s.gotListParams[k] != want {
					t.Errorf("param %s = %v, want %v", k, s.gotListParams[k], want)
				}
			}
		})
	}
}

func TestGetPopularEvents_NegativePage(t *testing.T) {
	s := fakeEventService{popularErr: fmt.Errorf("%w: negative page", event_service.ErrInvalidFilter)}
	rec := serve(&s, httptest.NewRequest(http.MethodGet, "/events/popular?pageSize=10&pageNum=-1", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestGetCategory_NotFound(t *testing.T) {
	s := fakeEventService{categoryErr: fmt.Errorf("%w: id 99", event_service.ErrCategoryNotFound)}
	rec := serve(&s, httptest.NewRequest(http.MethodGet, "/categories/99", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestListMapEvents_SignedInUser(t *testing.T) {
	s := fakeEventService{mapEvents: []event_model.MapEvent{}}
	req := httptest.NewRequest(http.MethodGet, "/events/map", nil)
	// A signed in user cannot ask for the friends of another user with the header
	req.Header.Set(requester.UserIDHeader, otherUserID)
	req = req.WithContext(auth_middleware.ContextWithUser(req.Context(), firebase_client.User{UID: firebaseUID}))

	rec := serve(&s, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if s.gotFilter.UserID != validUserID {
		t.Errorf("user id = %q, want %q", s.gotFilter.UserID, validUserID)
	}
}

func TestUploadEventImageHandler_TooLarge(t *testing.T) {
	tests := []struct {
		name string
		size int
	}{
		{"image just over the limit", maxImageBytes + 1},
		{"request body over the limit", maxImageBytes + 1<<20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := new(bytes.Buffer)
			writer := multipart.NewWriter(body)
			part, _ := writer.CreateFormFile("image", "poster.jpg")
			_, _ = part.Write(bytes.Repeat([]byte("x"), tt.size))
			writer.Close()

			s := fakeEventService{event: unofficialEvent(t, 15, validUserID)}
			req := httptest.NewRequest(http.MethodPost, "/events/15/image", body)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			req.Header.Set(requester.UserIDHeader, validUserID)

			rec := serve(&s, req)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
			}
			if s.uploadCalls != 0 {
				t.Errorf("upload calls = %d, want 0", s.uploadCalls)
			}
		})
	}
}

func TestGetRecommendedEvents(t *testing.T) {
	sample := []event_model.Event{
		{
			ID:          10,
			Title:       "Concert de Jazz",
			IsOfficial:  true,
			Categories:  []string{"Musica"},
			DateCreated: time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC),
		},
		unofficialEvent(t, 20, validUserID),
	}

	tests := []struct {
		name         string
		path         string
		userHeader   string
		firebaseUser string
		service      fakeEventService
		wantStatus   int
		wantCalls    int
		wantUserID   string
		wantPageSize int
		wantPageNum  int
		wantIDs      []int
		wantError    string
	}{
		{
			name:         "valid request with X-User-ID",
			path:         "/events/recommended?pageSize=2&pageNum=1",
			userHeader:   validUserID,
			service:      fakeEventService{recommendedEvents: sample},
			wantStatus:   http.StatusOK,
			wantCalls:    1,
			wantUserID:   validUserID,
			wantPageSize: 2,
			wantPageNum:  1,
			wantIDs:      []int{10, 20},
		},
		{
			name:         "valid request with Firebase UID ignores X-User-ID header",
			path:         "/events/recommended?pageSize=5&pageNum=0",
			userHeader:   otherUserID,
			firebaseUser: firebaseUID,
			service:      fakeEventService{recommendedEvents: sample[:1]},
			wantStatus:   http.StatusOK,
			wantCalls:    1,
			wantUserID:   validUserID,
			wantPageSize: 5,
			wantPageNum:  0,
			wantIDs:      []int{10},
		},
		{
			name:         "empty list returns empty JSON array",
			path:         "/events/recommended?pageSize=10&pageNum=0",
			userHeader:   validUserID,
			service:      fakeEventService{recommendedEvents: []event_model.Event{}},
			wantStatus:   http.StatusOK,
			wantCalls:    1,
			wantUserID:   validUserID,
			wantPageSize: 10,
			wantPageNum:  0,
			wantIDs:      []int{},
		},
		{
			name:       "missing pageSize returns 400",
			path:       "/events/recommended?pageNum=0",
			userHeader: validUserID,
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid page size",
		},
		{
			name:       "pageSize zero returns 400",
			path:       "/events/recommended?pageSize=0&pageNum=0",
			userHeader: validUserID,
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid page size",
		},
		{
			name:       "negative pageSize returns 400",
			path:       "/events/recommended?pageSize=-3&pageNum=0",
			userHeader: validUserID,
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid page size",
		},
		{
			name:       "non-numeric pageSize returns 400",
			path:       "/events/recommended?pageSize=abc&pageNum=0",
			userHeader: validUserID,
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid page size",
		},
		{
			name:       "missing pageNum returns 400",
			path:       "/events/recommended?pageSize=10",
			userHeader: validUserID,
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid page num",
		},
		{
			name:       "negative pageNum returns 400",
			path:       "/events/recommended?pageSize=10&pageNum=-1",
			userHeader: validUserID,
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid page num",
		},
		{
			name:       "non-numeric pageNum returns 400",
			path:       "/events/recommended?pageSize=10&pageNum=xyz",
			userHeader: validUserID,
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid page num",
		},
		{
			name:       "missing user returns 401",
			path:       "/events/recommended?pageSize=10&pageNum=0",
			wantStatus: http.StatusUnauthorized,
			wantError:  "missing X-User-ID header",
		},
		{
			name:         "unregistered Firebase user returns 401",
			path:         "/events/recommended?pageSize=10&pageNum=0",
			firebaseUser: "unregistered-firebase-uid",
			wantStatus:   http.StatusUnauthorized,
			wantError:    "user not registered",
		},
		{
			name:       "service ErrInvalidFilter returns 400",
			path:       "/events/recommended?pageSize=10&pageNum=0",
			userHeader: validUserID,
			service:    fakeEventService{recommendedErr: event_service.ErrInvalidFilter},
			wantStatus: http.StatusBadRequest,
			wantCalls:  1,
			wantError:  "invalid filter or page parameters",
		},
		{
			name:       "service ErrInvalidUserID returns 400",
			path:       "/events/recommended?pageSize=10&pageNum=0",
			userHeader: "not-a-uuid",
			service:    fakeEventService{recommendedErr: event_service.ErrInvalidUserID},
			wantStatus: http.StatusBadRequest,
			wantCalls:  1,
			wantError:  "invalid user id",
		},
		{
			name:       "unexpected service error returns 500",
			path:       "/events/recommended?pageSize=10&pageNum=0",
			userHeader: validUserID,
			service:    fakeEventService{recommendedErr: errors.New("db unavailable")},
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
				req.Header.Set(requester.UserIDHeader, tt.userHeader)
			}
			if tt.firebaseUser != "" {
				req = req.WithContext(auth_middleware.ContextWithUser(req.Context(), firebase_client.User{UID: tt.firebaseUser}))
			}

			rec := serve(&s, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if s.recommendedCalls != tt.wantCalls {
				t.Errorf("service calls = %d, want %d", s.recommendedCalls, tt.wantCalls)
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

			if s.gotRecommendedUserID != tt.wantUserID {
				t.Errorf("userID = %q, want %q", s.gotRecommendedUserID, tt.wantUserID)
			}
			if s.gotRecommendedSize != tt.wantPageSize {
				t.Errorf("pageSize = %d, want %d", s.gotRecommendedSize, tt.wantPageSize)
			}
			if s.gotRecommendedNum != tt.wantPageNum {
				t.Errorf("pageNum = %d, want %d", s.gotRecommendedNum, tt.wantPageNum)
			}

			if len(tt.wantIDs) == 0 {
				if strings.TrimSpace(rec.Body.String()) != "[]" {
					t.Errorf("body = %q, want []", rec.Body.String())
				}
				return
			}

			raw := rec.Body.String()
			if !strings.Contains(raw, `"is-official"`) || !strings.Contains(raw, `"date-created"`) {
				t.Errorf("expected kebab-case JSON keys in body %s", raw)
			}

			var got []event_model.Event
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if len(got) != len(tt.wantIDs) {
				t.Fatalf("got %d events, want %d", len(got), len(tt.wantIDs))
			}
			for i, wantID := range tt.wantIDs {
				if got[i].ID != wantID {
					t.Errorf("event[%d].ID = %d, want %d", i, got[i].ID, wantID)
				}
			}
		})
	}
}

func TestGetEventHandler(t *testing.T) {
	name := "Marc Vila"
	rel := float32(4.5)

	withRating := unofficialEvent(t, 2, validUserID)
	withRating.CreatorName = &name
	withRating.CreatorReliability = &rel

	noRatings := unofficialEvent(t, 3, validUserID)
	noRatings.CreatorName = &name
	noRatings.CreatorReliability = nil

	tests := []struct {
		name            string
		path            string
		service         fakeEventService
		wantStatus      int
		wantName        any
		wantReliability any
	}{
		{
			name:            "official event has null creator-name and creator-reliability",
			path:            "/events/1",
			service:         fakeEventService{event: officialEvent(1)},
			wantStatus:      http.StatusOK,
			wantName:        nil,
			wantReliability: nil,
		},
		{
			name:            "community event with rating includes creator-name and creator-reliability",
			path:            "/events/2",
			service:         fakeEventService{event: withRating},
			wantStatus:      http.StatusOK,
			wantName:        name,
			wantReliability: float64(rel),
		},
		{
			name:            "community event with no ratings has null creator-reliability",
			path:            "/events/3",
			service:         fakeEventService{event: noRatings},
			wantStatus:      http.StatusOK,
			wantName:        name,
			wantReliability: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.service
			rec := serve(&s, httptest.NewRequest(http.MethodGet, tt.path, nil))
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d: %s", rec.Code, tt.wantStatus, rec.Body.String())
			}
			var raw map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
				t.Fatalf("decode json: %v", err)
			}
			cname, hasName := raw["creator-name"]
			if !hasName {
				t.Errorf("response missing key creator-name")
			}
			if cname != tt.wantName {
				t.Errorf("creator-name = %v, want %v", cname, tt.wantName)
			}
			crel, hasRel := raw["creator-reliability"]
			if !hasRel {
				t.Errorf("response missing key creator-reliability")
			}
			if crel != tt.wantReliability {
				t.Errorf("creator-reliability = %v, want %v", crel, tt.wantReliability)
			}
		})
	}
}

func TestCancelEventHandler(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		userID     string
		service    fakeEventService
		wantStatus int
		wantCalls  int
	}{
		{
			name:       "creator cancels the event",
			path:       "/events/7/cancel",
			userID:     validUserID,
			service:    fakeEventService{event: unofficialEvent(t, 7, validUserID), notified: 3},
			wantStatus: http.StatusOK,
			wantCalls:  1,
		},
		{
			name:       "another user cannot cancel the event",
			path:       "/events/7/cancel",
			userID:     otherUserID,
			service:    fakeEventService{event: unofficialEvent(t, 7, validUserID)},
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "official events cannot be cancelled",
			path:       "/events/7/cancel",
			userID:     validUserID,
			service:    fakeEventService{event: officialEvent(7)},
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "missing user",
			path:       "/events/7/cancel",
			service:    fakeEventService{event: unofficialEvent(t, 7, validUserID)},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "event not found",
			path:       "/events/99/cancel",
			userID:     validUserID,
			service:    fakeEventService{getErr: event_service.ErrEventNotFound},
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "invalid event id",
			path:       "/events/abc/cancel",
			userID:     validUserID,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "event already cancelled",
			path:       "/events/7/cancel",
			userID:     validUserID,
			service:    fakeEventService{event: unofficialEvent(t, 7, validUserID), cancelErr: fmt.Errorf("x: %w", event_service.ErrEventNotActive)},
			wantStatus: http.StatusConflict,
			wantCalls:  1,
		},
		{
			name:       "event already finished",
			path:       "/events/7/cancel",
			userID:     validUserID,
			service:    fakeEventService{event: unofficialEvent(t, 7, validUserID), cancelErr: fmt.Errorf("x: %w", event_service.ErrEventFinished)},
			wantStatus: http.StatusConflict,
			wantCalls:  1,
		},
		{
			name:       "database error",
			path:       "/events/7/cancel",
			userID:     validUserID,
			service:    fakeEventService{event: unofficialEvent(t, 7, validUserID), cancelErr: errors.New("db down")},
			wantStatus: http.StatusInternalServerError,
			wantCalls:  1,
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
			if s.cancelCalls != tt.wantCalls {
				t.Errorf("service cancel calls = %d, want %d", s.cancelCalls, tt.wantCalls)
			}
		})
	}
}

func TestCancelEventHandler_Body(t *testing.T) {
	s := fakeEventService{event: unofficialEvent(t, 7, validUserID), notified: 3}
	req := httptest.NewRequest(http.MethodPost, "/events/7/cancel", nil)
	req.Header.Set(requester.UserIDHeader, validUserID)
	rec := serve(&s, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (body %s)", rec.Code, rec.Body.String())
	}
	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["event-id"] != float64(7) || got["state"] != "Cancelled" || got["notified-users"] != float64(3) {
		t.Errorf("body = %v", got)
	}
}
