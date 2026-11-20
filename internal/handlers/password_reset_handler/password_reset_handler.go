package password_reset_handler

import (
	"encoding/json"
	"net/http"

	"api-aprop/internal/handlers/response"
	"api-aprop/internal/services/password_reset_service"
)

// ResetPattern is a public route: whoever forgot the password has no session. It is registered
// outside the router so it does not go through the Firebase middleware (see cmd/app/main.go).
const ResetPattern = "POST /auth/password-reset"

const maxBodyBytes = 1 << 10

type PasswordResetHandler struct {
	svc password_reset_service.Service
}

func NewPasswordResetHandler(svc password_reset_service.Service) *PasswordResetHandler {
	return &PasswordResetHandler{svc: svc}
}

// Reset handles POST /auth/password-reset. It answers 204 whether the account exists or not.
func (h *PasswordResetHandler) Reset(w http.ResponseWriter, r *http.Request) {
	var req resetRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes)).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.svc.RequestReset(r.Context(), req.Identifier); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
