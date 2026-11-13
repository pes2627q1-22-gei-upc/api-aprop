package requester

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"api-aprop/internal/clients/firebase_client"
	"api-aprop/internal/middleware/auth_middleware"
	"api-aprop/internal/services/auth_service"
)

const (
	firebaseUID = "Vw3kU1pZ8rQ2hXyT6mN9aB4cD7e1"
	userUUID    = "3f2504e0-4f89-11d3-9a0c-0305e82c3301"
	otherUUID   = "9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d"
)

type fakeResolver struct {
	id     string
	err    error
	gotUID string
}

func (f *fakeResolver) ResolveUserID(_ context.Context, uid string) (string, error) {
	f.gotUID = uid
	return f.id, f.err
}

func request(header string, authenticated bool) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if header != "" {
		req.Header.Set(UserIDHeader, header)
	}
	if authenticated {
		req = req.WithContext(auth_middleware.ContextWithUser(req.Context(), firebase_client.User{UID: firebaseUID}))
	}
	return req
}

func TestID(t *testing.T) {
	tests := []struct {
		name       string
		req        *http.Request
		resolver   *fakeResolver
		wantID     string
		wantStatus int
		wantUID    string
	}{
		{
			name:     "firebase user is translated to the database UUID",
			req:      request("", true),
			resolver: &fakeResolver{id: userUUID},
			wantID:   userUUID, wantStatus: http.StatusOK, wantUID: firebaseUID,
		},
		{
			// A client must not impersonate another user by sending the header next to a valid token
			name:     "header is ignored when the request is authenticated",
			req:      request(otherUUID, true),
			resolver: &fakeResolver{id: userUUID},
			wantID:   userUUID, wantStatus: http.StatusOK, wantUID: firebaseUID,
		},
		{
			name:       "user not synced returns 401",
			req:        request("", true),
			resolver:   &fakeResolver{err: fmt.Errorf("%w: not found", auth_service.ErrUserNotRegistered)},
			wantStatus: http.StatusUnauthorized, wantUID: firebaseUID,
		},
		{
			name:       "resolver failure returns 500",
			req:        request("", true),
			resolver:   &fakeResolver{err: errors.New("db down")},
			wantStatus: http.StatusInternalServerError, wantUID: firebaseUID,
		},
		{
			name:     "header is used without the auth middleware",
			req:      request(userUUID, false),
			resolver: &fakeResolver{},
			wantID:   userUUID, wantStatus: http.StatusOK,
		},
		{
			name:       "no identity returns 401",
			req:        request("", false),
			resolver:   &fakeResolver{},
			wantStatus: http.StatusUnauthorized,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			id, ok := ID(rec, tt.req, tt.resolver)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if ok != (tt.wantStatus == http.StatusOK) || id != tt.wantID {
				t.Errorf("ID() = (%q, %v), want (%q, %v)", id, ok, tt.wantID, tt.wantStatus == http.StatusOK)
			}
			if tt.resolver.gotUID != tt.wantUID {
				t.Errorf("resolved uid = %q, want %q", tt.resolver.gotUID, tt.wantUID)
			}
		})
	}
}

func TestPresent(t *testing.T) {
	if Present(request("", false)) {
		t.Error("Present() = true without identity")
	}
	if !Present(request(userUUID, false)) || !Present(request("", true)) {
		t.Error("Present() = false with an identity")
	}
}
