package firebase_client

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const testAPIKey = "test-api-key"

func newResetServer(t *testing.T, status int, body string, check func(*http.Request)) *IdentityToolkitClient {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if check != nil {
			check(r)
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return NewIdentityToolkitClient(testAPIKey, srv.URL, srv.Client())
}

func TestSendPasswordResetEmailRequest(t *testing.T) {
	var got sendOobCodeRequest
	client := newResetServer(t, http.StatusOK, `{"email":"laia@aprop.cat"}`, func(r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/accounts:sendOobCode" {
			t.Errorf("request = %s %s, want POST /v1/accounts:sendOobCode", r.Method, r.URL.Path)
		}
		if key := r.URL.Query().Get("key"); key != testAPIKey {
			t.Errorf("key = %q, want %q", key, testAPIKey)
		}
		if locale := r.Header.Get("X-Firebase-Locale"); locale != "ca" {
			t.Errorf("X-Firebase-Locale = %q, want ca", locale)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode body: %v", err)
		}
	})

	if err := client.SendPasswordResetEmail(context.Background(), "laia@aprop.cat", "ca"); err != nil {
		t.Fatalf("SendPasswordResetEmail: %v", err)
	}
	if got.RequestType != "PASSWORD_RESET" || got.Email != "laia@aprop.cat" {
		t.Errorf("body = %+v", got)
	}
}

func TestSendPasswordResetEmailErrors(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		message string
		want    error
	}{
		{"email not found", http.StatusBadRequest, "EMAIL_NOT_FOUND", ErrEmailNotFound},
		{"invalid email", http.StatusBadRequest, "INVALID_EMAIL", ErrInvalidEmail},
		{"too many attempts with details", http.StatusBadRequest, "TOO_MANY_ATTEMPTS_TRY_LATER : Try again later.", ErrTooManyAttempts},
		{"reset limit", http.StatusBadRequest, "RESET_PASSWORD_EXCEED_LIMIT", ErrTooManyAttempts},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newResetServer(t, tt.status, `{"error":{"code":400,"message":"`+tt.message+`"}}`, nil)
			err := client.SendPasswordResetEmail(context.Background(), "a@b.cat", "ca")
			if !errors.Is(err, tt.want) {
				t.Errorf("err = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestSendPasswordResetEmailUnknownErrorHidesAPIKey(t *testing.T) {
	client := newResetServer(t, http.StatusInternalServerError, `{"error":{"message":"INTERNAL"}}`, nil)
	err := client.SendPasswordResetEmail(context.Background(), "a@b.cat", "ca")
	if err == nil {
		t.Fatal("want an error")
	}
	for _, known := range []error{ErrEmailNotFound, ErrInvalidEmail, ErrTooManyAttempts} {
		if errors.Is(err, known) {
			t.Errorf("err = %v, want an unknown error", err)
		}
	}
	if strings.Contains(err.Error(), testAPIKey) {
		t.Errorf("the error shows the API key: %v", err)
	}
}

func TestSendPasswordResetEmailRequestFailureHidesAPIKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	srv.Close() // the connection is refused
	client := NewIdentityToolkitClient(testAPIKey, srv.URL, nil)

	err := client.SendPasswordResetEmail(context.Background(), "a@b.cat", "ca")
	if err == nil {
		t.Fatal("want an error")
	}
	if strings.Contains(err.Error(), testAPIKey) {
		t.Errorf("the error shows the API key: %v", err)
	}
}

func TestSendPasswordResetEmailCanceledContext(t *testing.T) {
	client := newResetServer(t, http.StatusOK, `{}`, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := client.SendPasswordResetEmail(ctx, "a@b.cat", "ca")
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
}
