package password_reset_handler

import (
	"errors"
	"log"
	"math"
	"net/http"
	"strconv"

	"api-aprop/internal/handlers/response"
	"api-aprop/internal/services/password_reset_service"
)

func writeServiceError(w http.ResponseWriter, err error) {
	var limited *password_reset_service.RateLimitedError
	switch {
	case errors.Is(err, password_reset_service.ErrInvalidIdentifier):
		response.WriteError(w, http.StatusBadRequest, "identifier must be an email or a username")
	case errors.As(err, &limited):
		w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(limited.RetryAfter.Seconds()))))
		response.WriteError(w, http.StatusTooManyRequests, "too many password reset requests")
	case errors.Is(err, password_reset_service.ErrNotConfigured):
		response.WriteError(w, http.StatusServiceUnavailable, "password reset is not available")
	case errors.Is(err, password_reset_service.ErrProvider):
		log.Printf("password reset: %v", err)
		response.WriteError(w, http.StatusBadGateway, "could not send the password reset email")
	default:
		log.Printf("password reset: %v", err)
		response.WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}
