package user_handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"api-aprop/internal/clients/firebase_client"
	"api-aprop/internal/handlers/user_handler"
	"api-aprop/internal/middleware/auth_middleware"
	"api-aprop/internal/models/event_model"
	"api-aprop/internal/models/user_model"
	"api-aprop/internal/services/user_service"
)

type fakeUserService struct {
	profiles      map[string]user_model.UserProfile
	createdEvents map[string][]event_model.Event
	failUsername  bool
}

func (f *fakeUserService) GetUserProfile(_ context.Context, targetIdentifier, requesterIdentifier string) (user_model.UserProfile, error) {
	if targetIdentifier == "me" {
		if requesterIdentifier == "" {
			return user_model.UserProfile{}, user_service.ErrUnauthorized
		}
		p, ok := f.profiles[requesterIdentifier]
		if !ok {
			return user_model.UserProfile{}, user_service.ErrUserNotFound
		}
		return p, nil
	}
	p, ok := f.profiles[targetIdentifier]
	if !ok {
		return user_model.UserProfile{}, user_service.ErrUserNotFound
	}
	return p, nil
}

func (f *fakeUserService) ListCreatedEvents(_ context.Context, targetIdentifier string) ([]event_model.Event, error) {
	events, ok := f.createdEvents[targetIdentifier]
	if !ok {
		return nil, user_service.ErrUserNotFound
	}
	return events, nil
}

func (f *fakeUserService) UpdateProfile(_ context.Context, requesterIdentifier string, input user_service.UpdateProfileInput) (user_model.UserProfile, error) {
	if requesterIdentifier == "" {
		return user_model.UserProfile{}, user_service.ErrUnauthorized
	}
	if f.failUsername {
		return user_model.UserProfile{}, user_service.ErrUsernameTaken
	}
	p, ok := f.profiles[requesterIdentifier]
	if !ok {
		return user_model.UserProfile{}, user_service.ErrUserNotFound
	}
	if input.Name != nil {
		p.Name = *input.Name
	}
	if input.Username != nil {
		p.Username = *input.Username
	}
	if input.Biography != nil {
		p.Biography = *input.Biography
	}
	if input.ShowPlans != nil {
		p.ShowPlans = *input.ShowPlans
	}
	f.profiles[requesterIdentifier] = p
	return p, nil
}

func setupMux(svc user_service.Service) *http.ServeMux {
	h := user_handler.NewUserHandler(svc)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	return mux
}

func TestGetMyProfile_Success(t *testing.T) {
	svc := &fakeUserService{
		profiles: map[string]user_model.UserProfile{
			"alice-uid": {
				ID:             1,
				Username:       "alice",
				Name:           "Alice Smith",
				Email:          "alice@example.com",
				PlansCount:     3,
				FriendsCount:   10,
				DateRegistered: time.Now(),
				PlansVisible:   true,
			},
		},
	}
	mux := setupMux(svc)

	req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	req.Header.Set("X-User-ID", "alice-uid")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var res user_model.UserProfile
	if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode json: %v", err)
	}

	if res.Username != "alice" || res.Email != "alice@example.com" {
		t.Errorf("unexpected profile: %+v", res)
	}
}

func TestGetMyProfile_WithAuthContext(t *testing.T) {
	svc := &fakeUserService{
		profiles: map[string]user_model.UserProfile{
			"alice-uid": {
				ID:             1,
				Username:       "alice",
				Email:          "alice@example.com",
				PlansVisible:   true,
				DateRegistered: time.Now(),
			},
		},
	}
	mux := setupMux(svc)

	req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	ctx := auth_middleware.ContextWithUser(req.Context(), firebase_client.User{
		UID:   "alice-uid",
		Email: "alice@example.com",
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetMyProfile_Unauthorized(t *testing.T) {
	svc := &fakeUserService{profiles: make(map[string]user_model.UserProfile)}
	mux := setupMux(svc)

	req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestGetUserProfile_Success(t *testing.T) {
	svc := &fakeUserService{
		profiles: map[string]user_model.UserProfile{
			"bob": {
				ID:             2,
				Username:       "bob",
				Name:           "Bob Builder",
				PlansVisible:   false,
				FriendsCount:   5,
				DateRegistered: time.Now(),
			},
		},
	}
	mux := setupMux(svc)

	req := httptest.NewRequest(http.MethodGet, "/users/bob", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var res user_model.UserProfile
	if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode json: %v", err)
	}
	if res.Username != "bob" {
		t.Errorf("expected bob, got %s", res.Username)
	}
}

func TestGetUserProfile_NotFound(t *testing.T) {
	svc := &fakeUserService{profiles: make(map[string]user_model.UserProfile)}
	mux := setupMux(svc)

	req := httptest.NewRequest(http.MethodGet, "/users/unknown", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestUpdateMyProfile_Success(t *testing.T) {
	svc := &fakeUserService{
		profiles: map[string]user_model.UserProfile{
			"alice-uid": {
				ID:             1,
				Username:       "alice",
				Name:           "Alice Smith",
				ShowPlans:      true,
				DateRegistered: time.Now(),
			},
		},
	}
	mux := setupMux(svc)

	body := []byte(`{"name":"Alice Updated","biography":"New bio","show-plans":false}`)
	req := httptest.NewRequest(http.MethodPatch, "/users/me", bytes.NewReader(body))
	req.Header.Set("X-User-ID", "alice-uid")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var res user_model.UserProfile
	if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode json: %v", err)
	}
	if res.Name != "Alice Updated" || res.ShowPlans != false {
		t.Errorf("unexpected updated profile: %+v", res)
	}
}

func TestUpdateMyProfile_UsernameConflict(t *testing.T) {
	svc := &fakeUserService{
		failUsername: true,
		profiles: map[string]user_model.UserProfile{
			"alice-uid": {ID: 1, Username: "alice", DateRegistered: time.Now()},
		},
	}
	mux := setupMux(svc)

	body := []byte(`{"username":"taken_name"}`)
	req := httptest.NewRequest(http.MethodPatch, "/users/me", bytes.NewReader(body))
	req.Header.Set("X-User-ID", "alice-uid")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", w.Code)
	}
}

func TestListCreatedEvents_Success(t *testing.T) {
	svc := &fakeUserService{
		createdEvents: map[string][]event_model.Event{
			"alice": {
				{ID: 10, Title: "Exposicio d'Art"},
			},
		},
	}
	mux := setupMux(svc)

	req := httptest.NewRequest(http.MethodGet, "/users/alice/created-events", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var events []event_model.Event
	if err := json.NewDecoder(w.Body).Decode(&events); err != nil {
		t.Fatalf("failed to decode json: %v", err)
	}
	if len(events) != 1 || events[0].Title != "Exposicio d'Art" {
		t.Errorf("unexpected events: %+v", events)
	}
}
