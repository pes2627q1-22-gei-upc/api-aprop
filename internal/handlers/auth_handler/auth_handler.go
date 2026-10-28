package auth_handler

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"

	"api-aprop/internal/handlers/response"
	"api-aprop/internal/middleware/auth_middleware"
	"api-aprop/internal/services/auth_service"
)

const maxBodyBytes = 1 << 20

type AuthHandler struct {
	authService auth_service.Service
}

func NewAuthHandler(authService auth_service.Service) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// RegisterRoutes registers the auth endpoints.
func (h *AuthHandler) RegisterRoutes(mux *http.ServeMux) {
	// Auth handler: keeps the database users in sync with the Firebase accounts
	mux.HandleFunc("POST /auth/sync", h.Sync)
}

// Sync handles POST /auth/sync. The frontend calls it after signing in with Firebase to get the
// database user, which is created the first time with the username of the body.
func (h *AuthHandler) Sync(w http.ResponseWriter, r *http.Request) {
	firebaseUser, ok := auth_middleware.UserFromContext(r.Context())
	if !ok {
		// Only happens if the route is registered without the auth middleware
		log.Printf("auth sync: no authenticated user in context")
		response.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	// The body is optional: users that already exist do not need to send it
	var req syncRequest
	err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes)).Decode(&req)
	if err != nil && !errors.Is(err, io.EOF) {
		response.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.authService.Sync(r.Context(), firebaseUser.UID, firebaseUser.Email, req.Username)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	status := http.StatusOK
	if result.Created {
		status = http.StatusCreated
	}
	response.WriteJSON(w, status, result)
}
