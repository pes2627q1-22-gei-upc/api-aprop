package signup_handler

import (
	"errors"
	"log"
	"net/http"

	"api-aprop/internal/handlers/response"
	"api-aprop/internal/services/signup_service"
)

// writeServiceError maps sign up service errors to HTTP responses.
func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, signup_service.ErrEventNotFound):
		response.WriteError(w, http.StatusNotFound, "event not found")
	case errors.Is(err, signup_service.ErrNotEnrolled):
		response.WriteError(w, http.StatusNotFound, "user is not enrolled in the event")
	case errors.Is(err, signup_service.ErrEventNotStarted):
		// TODO(DP-029): 409 chosen over 422 (state conflict, R8); no frontend contract yet.
		response.WriteError(w, http.StatusConflict, "attendance can only be set once the event has started")
	case errors.Is(err, signup_service.ErrInvalidUserID):
		response.WriteError(w, http.StatusBadRequest, "invalid user id")
	case errors.Is(err, signup_service.ErrInvalidEventID):
		response.WriteError(w, http.StatusBadRequest, "invalid event id")
	case errors.Is(err, signup_service.ErrInvalidDateRange):
		response.WriteError(w, http.StatusBadRequest, "invalid date range: from must not be after to")
	default:
		log.Printf("internal error: %v", err)
		response.WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}
