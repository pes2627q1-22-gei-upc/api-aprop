package auth_handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"api-aprop/internal/clients/firebase_client"
	"api-aprop/internal/middleware/auth_middleware"
	"api-aprop/internal/models/user_model"
	"api-aprop/internal/services/auth_service"
)

var firebaseUser = firebase_client.User{UID: "Vw3kU1pZ8rQ2hXyT6mN9aB4cD7e1", Email: "anna@example.com"}

// testID returns a deterministic user UUID whose last byte is n.
func testID(n byte) pgtype.UUID {
	return pgtype.UUID{Bytes: [16]byte{15: n}, Valid: true}
}

type fakeService struct {
	result user_model.SyncResult
	err    error

	gotUID, gotEmail, gotUsername string
}

func (f *fakeService) Sync(_ context.Context, uid, email, username string) (user_model.SyncResult, error) {
	f.gotUID, f.gotEmail, f.gotUsername = uid, email, username
	return f.result, f.err
}

// sync calls the handler as if the request already went through the auth middleware.
func sync(t *testing.T, service *fakeService, body string, authenticated bool) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/auth/sync", strings.NewReader(body))
	if authenticated {
		req = req.WithContext(auth_middleware.ContextWithUser(req.Context(), firebaseUser))
	}
	rec := httptest.NewRecorder()
	NewAuthHandler(service).Sync(rec, req)
	return rec
}

func TestSync_Created(t *testing.T) {
	service := &fakeService{result: user_model.SyncResult{User: user_model.User{ID: testID(1), Username: "anna"}, Created: true}}

	rec := sync(t, service, `{"username": "anna"}`, true)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusCreated, rec.Body)
	}
	if service.gotUID != firebaseUser.UID || service.gotEmail != firebaseUser.Email || service.gotUsername != "anna" {
		t.Errorf("service got uid=%q email=%q username=%q", service.gotUID, service.gotEmail, service.gotUsername)
	}
	var got user_model.SyncResult
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !got.Created || got.User.Username != "anna" {
		t.Errorf("response = %+v", got)
	}
}

func TestSync_ExistingUserWithoutBody(t *testing.T) {
	service := &fakeService{result: user_model.SyncResult{User: user_model.User{ID: testID(1)}}}

	rec := sync(t, service, "", true)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d: %s", rec.Code, http.StatusOK, rec.Body)
	}
}

func TestSync_Errors(t *testing.T) {
	tests := []struct {
		name          string
		body          string
		authenticated bool
		serviceErr    error
		want          int
	}{
		{"invalid json", `{"username":`, true, nil, http.StatusBadRequest},
		{"invalid username", `{"username": "a"}`, true, auth_service.ErrInvalidUsername, http.StatusUnprocessableEntity},
		{"username taken", `{"username": "anna"}`, true, auth_service.ErrUsernameTaken, http.StatusConflict},
		{"not behind middleware", `{"username": "anna"}`, false, nil, http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := sync(t, &fakeService{err: tt.serviceErr}, tt.body, tt.authenticated)
			if rec.Code != tt.want {
				t.Errorf("status = %d, want %d: %s", rec.Code, tt.want, rec.Body)
			}
		})
	}
}
