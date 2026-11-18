package user_handler

import (
	"net/http"
	"strconv"

	"api-aprop/internal/handlers/requester"
	"api-aprop/internal/handlers/response"
)

// SearchUsers handles GET /users?q=<query>&pageNum=<n>&pageSize=<n>, with the same pagination as GET /events/search.
func (h *UserHandler) SearchUsers(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	pageNum, ok := intQueryParam(w, query.Get("pageNum"), "pageNum")
	if !ok {
		return
	}
	pageSize, ok := intQueryParam(w, query.Get("pageSize"), "pageSize")
	if !ok {
		return
	}

	requesterID, ok := requester.ID(w, r, h.resolver)
	if !ok {
		return
	}

	result, err := h.userService.SearchUsers(r.Context(), requesterID, query.Get("q"), pageNum, pageSize)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, result)
}

// intQueryParam reads an optional integer query parameter (0 when missing). It writes a 400 when it is not a number.
func intQueryParam(w http.ResponseWriter, value, name string) (int, bool) {
	if value == "" {
		return 0, true
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid "+name)
		return 0, false
	}
	return n, true
}
