package congestion_handler

import (
	"errors"
	"log"
	"net/http"

	"api-aprop/internal/handlers/response"
	"api-aprop/internal/services/congestion_service"
)

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, congestion_service.ErrNoLocations),
		errors.Is(err, congestion_service.ErrTooManyLocations),
		errors.Is(err, congestion_service.ErrInvalidLocation),
		errors.Is(err, congestion_service.ErrInvalidTimeframe):
		response.WriteError(w, http.StatusBadRequest, err.Error())
	default:
		log.Printf("congestion estimate: %v", err)
		response.WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}
