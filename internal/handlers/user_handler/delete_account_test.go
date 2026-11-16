package user_handler_test

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"api-aprop/internal/clients/firebase_client"
	"api-aprop/internal/middleware/auth_middleware"
	"api-aprop/internal/services/user_service"
)

func TestDeleteMyAccount(t *testing.T) {
	tests := []struct {
		name        string
		firebaseUID string // identity set by the auth middleware
		header      string // identity without middleware
		svcErr      error
		wantStatus  int
		wantDeleted bool
	}{
		{name: "success with firebase token", firebaseUID: aliceFirebaseUID, wantStatus: http.StatusNoContent, wantDeleted: true},
		{name: "success with header", header: aliceID, wantStatus: http.StatusNoContent, wantDeleted: true},
		{name: "no identity", wantStatus: http.StatusUnauthorized},
		{name: "not registered", firebaseUID: "unknown-uid", wantStatus: http.StatusUnauthorized},
		{name: "user not found", header: aliceID, svcErr: user_service.ErrUserNotFound, wantStatus: http.StatusNotFound, wantDeleted: true},
		{name: "invalid id", header: "bad", svcErr: user_service.ErrInvalidUserID, wantStatus: http.StatusBadRequest, wantDeleted: true},
		{name: "firebase error", header: aliceID, svcErr: fmt.Errorf("%w: boom", user_service.ErrAuthProvider), wantStatus: http.StatusBadGateway, wantDeleted: true},
		{name: "internal error", header: aliceID, svcErr: errors.New("db down"), wantStatus: http.StatusInternalServerError, wantDeleted: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeUserService{deleteErr: tt.svcErr}
			req := httptest.NewRequest(http.MethodDelete, "/users/me", nil)
			if tt.firebaseUID != "" {
				req = req.WithContext(auth_middleware.ContextWithUser(req.Context(), firebase_client.User{UID: tt.firebaseUID}))
			}
			if tt.header != "" {
				req.Header.Set("X-User-ID", tt.header)
			}
			w := httptest.NewRecorder()

			setupMux(svc).ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (%s)", w.Code, tt.wantStatus, w.Body.String())
			}
			if tt.wantStatus == http.StatusNoContent && w.Body.Len() != 0 {
				t.Errorf("204 with body %q", w.Body.String())
			}
			if got := len(svc.deletedIDs) == 1; got != tt.wantDeleted {
				t.Fatalf("service called = %v, want %v", got, tt.wantDeleted)
			}
			if tt.wantDeleted && tt.header == "" && svc.deletedIDs[0] != aliceID {
				t.Errorf("deleted %q, want the resolved id %q", svc.deletedIDs[0], aliceID)
			}
		})
	}
}
