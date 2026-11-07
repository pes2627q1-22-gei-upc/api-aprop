package notification_handler

import (
	"errors"
	"log"
	"net/http"

	"api-aprop/internal/handlers/response"
	"api-aprop/internal/services/notification_service"
)

// writeServiceError maps event service errors to HTTP responses.
func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, notification_service.ErrNotificationNotFound):
		response.WriteError(w, http.StatusNotFound, "notification not found")
	case errors.Is(err, notification_service.ErrUserNotFound):
		response.WriteError(w, http.StatusNotFound, "user not found")
	default:
		log.Printf("internal error: %v", err)
		response.WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}
