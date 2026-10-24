package event_handler

import (
	"errors"
	"log"
	"net/http"

	"api-aprop/internal/handlers/response"
	"api-aprop/internal/services/event_service"
)

// writeServiceError maps event service errors to HTTP responses.
func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, event_service.ErrEventNotFound):
		response.WriteError(w, http.StatusNotFound, "event not found")
	case errors.Is(err, event_service.ErrInvalidEvent):
		response.WriteError(w, http.StatusUnprocessableEntity, "event is invalid")
	case errors.Is(err, event_service.ErrInvalidBoundingBox):
		response.WriteError(w, http.StatusBadRequest, "invalid bounding box")
	case errors.Is(err, event_service.ErrInvalidUserID):
		response.WriteError(w, http.StatusBadRequest, "invalid user id")
	default:
		log.Printf("internal error: %v", err)
		response.WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}

