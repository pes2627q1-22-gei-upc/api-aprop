package user_handler

import (
	"net/http"

	"api-aprop/internal/handlers/requester"
	"api-aprop/internal/handlers/response"
)

// ListBlockedUsers handles GET /users/me/blocked: the users blocked by the requester, the most recent first.
func (h *UserHandler) ListBlockedUsers(w http.ResponseWriter, r *http.Request) {
	requesterID, ok := requester.ID(w, r, h.resolver)
	if !ok {
		return
	}
	users, err := h.userService.ListBlockedUsers(r.Context(), requesterID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, users)
}

// BlockUser handles POST /users/{id}/block. It also removes the friendship with the user, if any.
func (h *UserHandler) BlockUser(w http.ResponseWriter, r *http.Request) {
	targetID := r.PathValue("id")
	requesterID, ok := requester.ID(w, r, h.resolver)
	if !ok {
		return
	}
	blocked, err := h.userService.BlockUser(r.Context(), requesterID, targetID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusCreated, blocked)
}

// UnblockUser handles DELETE /users/{id}/block.
func (h *UserHandler) UnblockUser(w http.ResponseWriter, r *http.Request) {
	targetID := r.PathValue("id")
	requesterID, ok := requester.ID(w, r, h.resolver)
	if !ok {
		return
	}
	if err := h.userService.UnblockUser(r.Context(), requesterID, targetID); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
