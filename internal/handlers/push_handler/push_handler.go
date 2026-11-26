package push_handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"api-aprop/internal/handlers/requester"
	"api-aprop/internal/handlers/response"
	"api-aprop/internal/services/push_service"
)

// PushHandler registers the devices that receive the push notifications of the requester (Taiga #275).
type PushHandler struct {
	service  push_service.Service
	resolver requester.Resolver
}

func NewPushHandler(service push_service.Service, resolver requester.Resolver) *PushHandler {
	return &PushHandler{service: service, resolver: resolver}
}

// RegisterRoutes registers the device endpoints.
func (h *PushHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("PUT /users/me/devices", h.RegisterDevice)
	mux.HandleFunc("DELETE /users/me/devices/{token}", h.UnregisterDevice)
}

// RegisterDevice handles PUT /users/me/devices with body {"token": "...", "platform": "android|ios|web"}.
// The app calls it after signing in and whenever FCM refreshes the token. Responds 204.
func (h *PushHandler) RegisterDevice(w http.ResponseWriter, r *http.Request) {
	userID, ok := requester.ID(w, r, h.resolver)
	if !ok {
		return
	}
	var body struct {
		Token    *string `json:"token"`
		Platform *string `json:"platform"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Token == nil || body.Platform == nil {
		response.WriteError(w, http.StatusBadRequest, `invalid body, expected {"token": "...", "platform": "android|ios|web"}`)
		return
	}

	if err := h.service.RegisterDevice(r.Context(), userID, *body.Token, *body.Platform); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// UnregisterDevice handles DELETE /users/me/devices/{token}. The app calls it when signing out. It responds 204
// even when the device was not registered, so the call can be repeated.
func (h *PushHandler) UnregisterDevice(w http.ResponseWriter, r *http.Request) {
	userID, ok := requester.ID(w, r, h.resolver)
	if !ok {
		return
	}
	if err := h.service.UnregisterDevice(r.Context(), userID, r.PathValue("token")); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// writeServiceError maps push service errors to HTTP responses.
func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, push_service.ErrInvalidToken), errors.Is(err, push_service.ErrInvalidPlatform):
		response.WriteError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, push_service.ErrInvalidUserID):
		response.WriteError(w, http.StatusBadRequest, "invalid user id")
	default:
		log.Printf("internal error: %v", err)
		response.WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}
