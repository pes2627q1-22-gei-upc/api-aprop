package apikey_middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name       string
		key        string
		header     string
		wantStatus int
		wantNext   bool
	}{
		{"valid key", "secret", "secret", http.StatusOK, true},
		{"valid key with spaces", "secret", "  secret ", http.StatusOK, true},
		{"missing key", "secret", "", http.StatusUnauthorized, false},
		{"wrong key", "secret", "other", http.StatusUnauthorized, false},
		{"prefix of the key", "secret", "secre", http.StatusUnauthorized, false},
		{"not configured", "", "anything", http.StatusServiceUnavailable, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				called = true
				w.WriteHeader(http.StatusOK)
			})
			req := httptest.NewRequest(http.MethodPost, "/api/v1/congestion/estimate", nil)
			if tt.header != "" {
				req.Header.Set(Header, tt.header)
			}
			rec := httptest.NewRecorder()

			New(tt.key)(next).ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if called != tt.wantNext {
				t.Errorf("next called = %v, want %v", called, tt.wantNext)
			}
		})
	}
}
