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
	case errors.Is(err, event_service.ErrCategoryNotFound):
		response.WriteError(w, http.StatusNotFound, "category not found")
	case errors.Is(err, event_service.ErrInvalidFilter):
		response.WriteError(w, http.StatusBadRequest, "invalid filter or page parameters")
	case errors.Is(err, event_service.ErrInvalidEventTitle):
		response.WriteError(w, http.StatusBadRequest, "title is required")
	case errors.Is(err, event_service.ErrInvalidEventDescription):
		response.WriteError(w, http.StatusBadRequest, "description is required")
	case errors.Is(err, event_service.ErrInvalidEventCategory):
		response.WriteError(w, http.StatusBadRequest, "at least one category is required")
	case errors.Is(err, event_service.ErrInvalidEventDates):
		response.WriteError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, event_service.ErrInvalidCoordinates):
		response.WriteError(w, http.StatusBadRequest, "invalid coordinates")
	case errors.Is(err, event_service.ErrInvalidContentType):
		response.WriteError(w, http.StatusBadRequest, "invalid content type")
	case errors.Is(err, event_service.ErrInvalidEvent):
		response.WriteError(w, http.StatusUnprocessableEntity, "event is invalid")
	case errors.Is(err, event_service.ErrInvalidBoundingBox):
		response.WriteError(w, http.StatusBadRequest, "invalid bounding box")
	case errors.Is(err, event_service.ErrInvalidUserID):
		response.WriteError(w, http.StatusBadRequest, "invalid user id")
	case errors.Is(err, event_service.ErrUserNotRegistered):
		response.WriteError(w, http.StatusUnauthorized, "user not registered")
	default:
		log.Printf("internal error: %v", err)
		response.WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}
