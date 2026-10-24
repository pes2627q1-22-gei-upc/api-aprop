package user_handler

import (
	"errors"
	"net/http"

	"api-aprop/internal/handlers/response"
	"api-aprop/internal/services/user_service"
)

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, user_service.ErrUserNotFound):
		response.WriteError(w, http.StatusNotFound, "user not found")
	case errors.Is(err, user_service.ErrUnauthorized):
		response.WriteError(w, http.StatusUnauthorized, "unauthorized")
	case errors.Is(err, user_service.ErrForbidden):
		response.WriteError(w, http.StatusForbidden, "forbidden: profile plans are private")
	case errors.Is(err, user_service.ErrUsernameTaken):
		response.WriteError(w, http.StatusConflict, "username already taken")
	case errors.Is(err, user_service.ErrInvalidInput), errors.Is(err, user_service.ErrInvalidUserID):
		response.WriteError(w, http.StatusBadRequest, err.Error())
	default:
		response.WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}
