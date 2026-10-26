// Package requester identifies the user that makes a request.
package requester

import (
	"context"
	"errors"
	"log"
	"net/http"

	"api-aprop/internal/handlers/response"
	"api-aprop/internal/middleware/auth_middleware"
	"api-aprop/internal/services/event_service"
)

// UserIDHeader carries the user identity (a UUID) when the API runs without the auth middleware.
const UserIDHeader = "X-User-ID"

// Resolver translates the Firebase UID of a user into the database ID (UUID).
type Resolver interface {
	ResolveUserID(ctx context.Context, firebaseUID string) (string, error)
}

// Present reports whether the request carries an identity, from the auth middleware or the header.
func Present(r *http.Request) bool {
	if u, ok := auth_middleware.UserFromContext(r.Context()); ok && u.UID != "" {
		return true
	}
	return r.Header.Get(UserIDHeader) != ""
}

// ID returns the database ID (UUID) of the requester. The Firebase UID verified by the auth middleware is
// always translated; the header is only read when there is no middleware and the service validates it.
// It writes the error response and returns false when the requester cannot be identified.
func ID(w http.ResponseWriter, r *http.Request, resolver Resolver) (string, bool) {
	if u, ok := auth_middleware.UserFromContext(r.Context()); ok && u.UID != "" {
		id, err := resolver.ResolveUserID(r.Context(), u.UID)
		switch {
		case errors.Is(err, event_service.ErrUserNotRegistered):
			response.WriteError(w, http.StatusUnauthorized, "user not registered")
			return "", false
		case err != nil:
			log.Printf("internal error: %v", err)
			response.WriteError(w, http.StatusInternalServerError, "internal server error")
			return "", false
		}
		return id, true
	}

	id := r.Header.Get(UserIDHeader)
	if id == "" {
		response.WriteError(w, http.StatusBadRequest, "missing "+UserIDHeader+" header")
		return "", false
	}
	return id, true
}
