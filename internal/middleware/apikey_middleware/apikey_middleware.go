// Package apikey_middleware protects the endpoints served to other PES teams (Mobilicat) with a
// static API key instead of a Firebase token.
package apikey_middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"api-aprop/internal/handlers/response"
)

// Header is the request header that carries the API key.
const Header = "X-API-Key"

// New only lets through requests whose X-API-Key header matches key. With an empty key the
// service is not configured and every request gets 503, so it can never be called without a key.
func New(key string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if key == "" {
				response.WriteError(w, http.StatusServiceUnavailable, "service not configured")
				return
			}
			got := strings.TrimSpace(r.Header.Get(Header))
			if got == "" {
				response.WriteError(w, http.StatusUnauthorized, "missing api key")
				return
			}
			if subtle.ConstantTimeCompare([]byte(got), []byte(key)) != 1 {
				response.WriteError(w, http.StatusUnauthorized, "invalid api key")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
