package user_handler_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"api-aprop/internal/models/user_model"
	"api-aprop/internal/services/user_service"
)

func TestBlockedUsersStatusCodes(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		url        string
		header     string
		svcErr     error
		wantStatus int
	}{
		{"block", http.MethodPost, "/users/bob/block", aliceID, nil, http.StatusCreated},
		{"unblock", http.MethodDelete, "/users/bob/block", aliceID, nil, http.StatusNoContent},
		{"list", http.MethodGet, "/users/me/blocked", aliceID, nil, http.StatusOK},
		{"block without identity", http.MethodPost, "/users/bob/block", "", nil, http.StatusUnauthorized},
		{"unblock without identity", http.MethodDelete, "/users/bob/block", "", nil, http.StatusUnauthorized},
		{"list without identity", http.MethodGet, "/users/me/blocked", "", nil, http.StatusUnauthorized},
		{"block yourself (R2)", http.MethodPost, "/users/alice/block", aliceID, user_service.ErrSameUser, http.StatusBadRequest},
		{"already blocked", http.MethodPost, "/users/bob/block", aliceID, fmt.Errorf("x: %w", user_service.ErrAlreadyBlocked), http.StatusConflict},
		{"user not found", http.MethodPost, "/users/nobody/block", aliceID, user_service.ErrUserNotFound, http.StatusNotFound},
		{"unblock a user that is not blocked", http.MethodDelete, "/users/bob/block", aliceID, user_service.ErrNotBlocked, http.StatusNotFound},
		{"database error", http.MethodGet, "/users/me/blocked", aliceID, errors.New("db down"), http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeUserService{blockErr: tt.svcErr}
			req := httptest.NewRequest(tt.method, tt.url, nil)
			if tt.header != "" {
				req.Header.Set("X-User-ID", tt.header)
			}
			w := httptest.NewRecorder()
			setupMux(svc).ServeHTTP(w, req)
			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (%s)", w.Code, tt.wantStatus, w.Body.String())
			}
			if tt.header != "" && svc.gotRequester != aliceID {
				t.Errorf("service got requester %q, want %q", svc.gotRequester, aliceID)
			}
		})
	}
}

func TestBlockUser_Body(t *testing.T) {
	svc := &fakeUserService{blocked: user_model.BlockedUser{
		UserSummary: user_model.UserSummary{ID: testID(2), Username: "bob", Name: "Bob Puig", ProfilePicture: "https://img/bob.png"},
		DateBlocked: time.Date(2026, 11, 19, 10, 0, 0, 0, time.UTC),
	}}
	req := httptest.NewRequest(http.MethodPost, "/users/bob/block", nil)
	req.Header.Set("X-User-ID", aliceID)
	w := httptest.NewRecorder()
	setupMux(svc).ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d (%s)", w.Code, w.Body.String())
	}
	if svc.gotTarget != "bob" || svc.blockCalls != 1 {
		t.Errorf("service got target %q, calls %d", svc.gotTarget, svc.blockCalls)
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["username"] != "bob" || body["profile-picture"] != "https://img/bob.png" || body["date-blocked"] != "2026-11-19T10:00:00Z" || body["id"] == nil {
		t.Errorf("body = %v, want the user summary flattened with date-blocked", body)
	}
}

func TestListBlockedUsers_Body(t *testing.T) {
	t.Run("users", func(t *testing.T) {
		svc := &fakeUserService{blockedList: []user_model.BlockedUser{{UserSummary: user_model.UserSummary{ID: testID(2), Username: "bob"}}}}
		req := httptest.NewRequest(http.MethodGet, "/users/me/blocked", nil)
		req.Header.Set("X-User-ID", aliceID)
		w := httptest.NewRecorder()
		setupMux(svc).ServeHTTP(w, req)
		var body []map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode %s: %v", w.Body.String(), err)
		}
		if len(body) != 1 || body[0]["username"] != "bob" || svc.listCalls != 1 {
			t.Errorf("body = %v, calls = %d", body, svc.listCalls)
		}
	})
	t.Run("empty list", func(t *testing.T) {
		svc := &fakeUserService{blockedList: []user_model.BlockedUser{}}
		req := httptest.NewRequest(http.MethodGet, "/users/me/blocked", nil)
		req.Header.Set("X-User-ID", aliceID)
		w := httptest.NewRecorder()
		setupMux(svc).ServeHTTP(w, req)
		if got := w.Body.String(); got != "[]\n" && got != "[]" {
			t.Errorf("body = %q, want []", got)
		}
	})
}

// GET /users/me/blocked must not be taken by GET /users/{id} and the other user routes.
func TestBlockedRoutesDoNotShadowProfileRoutes(t *testing.T) {
	svc := &fakeUserService{}
	req := httptest.NewRequest(http.MethodGet, "/users/me/blocked", nil)
	req.Header.Set("X-User-ID", aliceID)
	w := httptest.NewRecorder()
	setupMux(svc).ServeHTTP(w, req)
	if svc.listCalls != 1 {
		t.Fatalf("GET /users/me/blocked did not reach ListBlockedUsers (status %d)", w.Code)
	}
}
