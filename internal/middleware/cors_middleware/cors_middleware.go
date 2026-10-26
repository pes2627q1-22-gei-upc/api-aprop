package cors_middleware

import (
	"net/http"
	"slices"
	"strings"
)

const (
	// AnyOrigin allows every origin to call the API from a browser.
	AnyOrigin = "*"

	allowedMethods = "GET, POST, PUT, PATCH, DELETE, OPTIONS"
	allowedHeaders = "Authorization, Content-Type, X-User-ID"
	maxAge         = "600" // browsers cache the preflight response for 10 minutes
)

// NewCORSMiddleware lets the allowed origins call the API from a browser. AnyOrigin allows all of them.
// It must wrap the auth middleware: browsers send the preflight request without the bearer token,
// so it is answered here. Requests without an Origin header (mobile app, curl) pass through unchanged.
func NewCORSMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	origins := make([]string, 0, len(allowedOrigins))
	for _, o := range allowedOrigins {
		if o = strings.TrimSpace(o); o != "" {
			origins = append(origins, o)
		}
	}
	allowAll := slices.Contains(origins, AnyOrigin)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if !allowAll {
				// The response depends on the origin, so caches must not share it between origins
				w.Header().Add("Vary", "Origin")
			}

			switch {
			case origin == "":
				next.ServeHTTP(w, r)
				return
			case allowAll:
				w.Header().Set("Access-Control-Allow-Origin", AnyOrigin)
			case slices.Contains(origins, origin):
				w.Header().Set("Access-Control-Allow-Origin", origin)
			default:
				// Without the CORS headers the browser blocks the response
				next.ServeHTTP(w, r)
				return
			}

			if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
				w.Header().Set("Access-Control-Allow-Methods", allowedMethods)
				w.Header().Set("Access-Control-Allow-Headers", allowedHeaders)
				w.Header().Set("Access-Control-Max-Age", maxAge)
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
