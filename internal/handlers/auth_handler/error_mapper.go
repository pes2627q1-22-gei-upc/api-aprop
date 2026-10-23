package auth_handler

import (
	"errors"
	"log"
	"net/http"

	"api-aprop/internal/handlers/response"
	"api-aprop/internal/services/auth_service"
)

// writeServiceError maps auth service errors to HTTP responses.
func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, auth_service.ErrInvalidUsername):
		response.WriteError(w, http.StatusUnprocessableEntity, err.Error())
	case errors.Is(err, auth_service.ErrUsernameTaken):
		response.WriteError(w, http.StatusConflict, "username already taken")
	default:
		log.Printf("internal error: %v", err)
		response.WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}
