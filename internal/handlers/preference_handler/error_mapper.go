package preference_handler

import (
	"errors"
	"net/http"

	"api-aprop/internal/handlers/response"
	"api-aprop/internal/services/preference_service"
)

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, preference_service.ErrInvalidUserID),
		errors.Is(err, preference_service.ErrEmptyAnswers),
		errors.Is(err, preference_service.ErrTooManyAnswers),
		errors.Is(err, preference_service.ErrInvalidEventID):
		response.WriteError(w, http.StatusBadRequest, err.Error())
	default:
		response.WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}
