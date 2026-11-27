package password_reset_handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"api-aprop/internal/services/password_reset_service"
)

type fakeService struct {
	err   error
	calls int
	got   string
}

func (f *fakeService) RequestReset(_ context.Context, identifier string) error {
	f.calls++
	f.got = identifier
	return f.err
}

// do sends the request without an Authorization header: the route is public.
func do(t *testing.T, svc password_reset_service.Service, body string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc(ResetPattern, NewPasswordResetHandler(svc).Reset)
	req := httptest.NewRequest(http.MethodPost, "/auth/password-reset", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestResetAccepted(t *testing.T) {
	svc := &fakeService{}
	rec := do(t, svc, `{"identifier":"laia@aprop.cat"}`)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204; body %s", rec.Code, rec.Body)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want empty", rec.Body)
	}
	if svc.got != "laia@aprop.cat" {
		t.Errorf("identifier = %q", svc.got)
	}
}

func TestResetInvalidBody(t *testing.T) {
	for _, body := range []string{``, `{`, `[]`, `{"identifier":` + `"` + strings.Repeat("a", maxBodyBytes) + `"}`} {
		svc := &fakeService{}
		rec := do(t, svc, body)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("body %.20q: status = %d, want 400", body, rec.Code)
		}
		if svc.calls != 0 {
			t.Errorf("body %.20q: the service was called", body)
		}
	}
}

func TestResetServiceErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{"invalid identifier", password_reset_service.ErrInvalidIdentifier, http.StatusBadRequest},
		{"rate limited", &password_reset_service.RateLimitedError{RetryAfter: 1500 * time.Millisecond}, http.StatusTooManyRequests},
		{"not configured", password_reset_service.ErrNotConfigured, http.StatusServiceUnavailable},
		{"provider", fmt.Errorf("%w: status 500", password_reset_service.ErrProvider), http.StatusBadGateway},
		{"unexpected", errors.New("db down"), http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := do(t, &fakeService{err: tt.err}, `{"identifier":"laia@aprop.cat"}`)
			if rec.Code != tt.want {
				t.Errorf("status = %d, want %d", rec.Code, tt.want)
			}
		})
	}
}

func TestResetRetryAfter(t *testing.T) {
	rec := do(t, &fakeService{err: &password_reset_service.RateLimitedError{RetryAfter: 1500 * time.Millisecond}}, `{"identifier":"laiasoler"}`)
	if got := rec.Header().Get("Retry-After"); got != "2" {
		t.Errorf("Retry-After = %q, want 2 (rounded up)", got)
	}
}
