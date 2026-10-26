package cors_middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

const webOrigin = "https://admin.aprop.cat"

// serve runs a request through the middleware and reports whether it reached the next handler.
func serve(t *testing.T, allowedOrigins []string, method, origin string, preflight bool) (*httptest.ResponseRecorder, bool) {
	t.Helper()
	reached := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(method, "/events", nil)
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	if preflight {
		req.Header.Set("Access-Control-Request-Method", http.MethodPost)
		req.Header.Set("Access-Control-Request-Headers", "authorization, content-type")
	}
	rec := httptest.NewRecorder()
	NewCORSMiddleware(allowedOrigins)(next).ServeHTTP(rec, req)
	return rec, reached
}

func TestCORSMiddleware_AnyOrigin(t *testing.T) {
	t.Run("preflight is answered without reaching the next handler", func(t *testing.T) {
		rec, reached := serve(t, []string{AnyOrigin}, http.MethodOptions, webOrigin, true)

		if reached {
			t.Fatal("preflight reached the next handler")
		}
		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", rec.Code)
		}
		want := map[string]string{
			"Access-Control-Allow-Origin":  AnyOrigin,
			"Access-Control-Allow-Methods": allowedMethods,
			"Access-Control-Allow-Headers": allowedHeaders,
			"Access-Control-Max-Age":       maxAge,
		}
		for header, value := range want {
			if got := rec.Header().Get(header); got != value {
				t.Errorf("%s = %q, want %q", header, got, value)
			}
		}
	})

	t.Run("request from a browser reaches the next handler with the CORS header", func(t *testing.T) {
		rec, reached := serve(t, []string{AnyOrigin}, http.MethodGet, webOrigin, false)

		if !reached {
			t.Fatal("request did not reach the next handler")
		}
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != AnyOrigin {
			t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, AnyOrigin)
		}
		if got := rec.Header().Get("Access-Control-Allow-Methods"); got != "" {
			t.Errorf("Access-Control-Allow-Methods = %q, want it only on preflight", got)
		}
	})

	t.Run("request without Origin is not changed", func(t *testing.T) {
		rec, reached := serve(t, []string{AnyOrigin}, http.MethodGet, "", false)

		if !reached {
			t.Fatal("request did not reach the next handler")
		}
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Errorf("Access-Control-Allow-Origin = %q, want none", got)
		}
	})

	t.Run("OPTIONS that is not a preflight reaches the next handler", func(t *testing.T) {
		_, reached := serve(t, []string{AnyOrigin}, http.MethodOptions, webOrigin, false)

		if !reached {
			t.Fatal("OPTIONS without Access-Control-Request-Method did not reach the next handler")
		}
	})
}

func TestCORSMiddleware_AllowedOrigins(t *testing.T) {
	allowed := []string{" http://localhost:3000 ", webOrigin, ""}

	t.Run("allowed origin gets its own origin back", func(t *testing.T) {
		rec, reached := serve(t, allowed, http.MethodOptions, webOrigin, true)

		if reached || rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d, reached = %v, want 204 without reaching the next handler", rec.Code, reached)
		}
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != webOrigin {
			t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, webOrigin)
		}
		if got := rec.Header().Get("Vary"); got != "Origin" {
			t.Errorf("Vary = %q, want Origin", got)
		}
	})

	t.Run("origins are trimmed", func(t *testing.T) {
		rec, _ := serve(t, allowed, http.MethodGet, "http://localhost:3000", false)

		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
			t.Errorf("Access-Control-Allow-Origin = %q, want http://localhost:3000", got)
		}
	})

	t.Run("other origin gets no CORS headers", func(t *testing.T) {
		rec, reached := serve(t, allowed, http.MethodOptions, "https://evil.example", true)

		if !reached {
			t.Fatal("request from a not allowed origin should reach the next handler unchanged")
		}
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Errorf("Access-Control-Allow-Origin = %q, want none", got)
		}
	})
}
