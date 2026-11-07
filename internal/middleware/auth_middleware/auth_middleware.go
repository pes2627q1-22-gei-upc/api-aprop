package auth_middleware

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"api-aprop/internal/clients/firebase_client"
	"api-aprop/internal/handlers/response"
)

const bearerPrefix = "Bearer "

// contextKey is a private type so no other package can overwrite the user stored in the context.
type contextKey struct{}

var userContextKey = contextKey{}

// NewAuthMiddleware only lets through requests with a valid Firebase ID token in the header
// "Authorization: Bearer <token>". The authenticated user is stored in the request context.
func NewAuthMiddleware(firebaseClient firebase_client.Client) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			idToken, ok := strings.CutPrefix(header, bearerPrefix)
			idToken = strings.TrimSpace(idToken)
			if !ok || idToken == "" {
				response.WriteError(w, http.StatusUnauthorized, "missing bearer token")
				return
			}

			user, err := firebaseClient.VerifyUser(r.Context(), idToken)
			if err != nil {
				writeAuthError(w, err)
				return
			}

			next.ServeHTTP(w, r.WithContext(ContextWithUser(r.Context(), user)))
		})
	}
}

// NewDevAuthMiddleware handles requests in local development when Firebase Admin credentials are not provided.
// If an "Authorization: Bearer <token>" header is present, it extracts the claims (UID and Email) from the JWT payload
// without validating the cryptographic signature, allowing frontend development with real Firebase tokens.
// If no Bearer token is present, it passes the request through unauthenticated so that requester.UserIDHeader can be used.
func NewDevAuthMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if idToken, ok := strings.CutPrefix(header, bearerPrefix); ok {
				idToken = strings.TrimSpace(idToken)
				if user, err := parseUnverifiedJWT(idToken); err == nil && user.UID != "" {
					next.ServeHTTP(w, r.WithContext(ContextWithUser(r.Context(), user)))
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func parseUnverifiedJWT(tokenString string) (firebase_client.User, error) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return firebase_client.User{}, errors.New("invalid jwt format")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		payload, err = base64.URLEncoding.DecodeString(parts[1])
		if err != nil {
			return firebase_client.User{}, err
		}
	}
	var claims struct {
		UserID string `json:"user_id"`
		Sub    string `json:"sub"`
		Email  string `json:"email"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return firebase_client.User{}, err
	}
	uid := claims.UserID
	if uid == "" {
		uid = claims.Sub
	}
	if uid == "" {
		return firebase_client.User{}, errors.New("missing uid in jwt claims")
	}
	return firebase_client.User{UID: uid, Email: claims.Email}, nil
}

// ContextWithUser returns a copy of ctx that carries the authenticated user. Handler tests use it
// to simulate a request that already went through the middleware.
func ContextWithUser(ctx context.Context, user firebase_client.User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

// UserFromContext returns the user authenticated by the middleware.
func UserFromContext(ctx context.Context) (firebase_client.User, bool) {
	user, ok := ctx.Value(userContextKey).(firebase_client.User)
	return user, ok
}

// writeAuthError maps firebase client errors to HTTP responses.
func writeAuthError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, firebase_client.ErrInvalidToken):
		response.WriteError(w, http.StatusUnauthorized, "invalid token")
	case errors.Is(err, firebase_client.ErrTokenRevoked):
		response.WriteError(w, http.StatusUnauthorized, "token revoked")
	case errors.Is(err, firebase_client.ErrUserNotFound):
		response.WriteError(w, http.StatusUnauthorized, "user not found")
	case errors.Is(err, firebase_client.ErrUserDisabled):
		response.WriteError(w, http.StatusUnauthorized, "user disabled")
	default:
		log.Printf("auth error: %v", err)
		response.WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}
