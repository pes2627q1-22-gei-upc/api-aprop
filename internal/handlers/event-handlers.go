package handlers

import (
	. "api-aprop/internal/services"
	. "api-aprop/internal/repositories"
	"encoding/json"
	"net/http"
	"log"
	"errors"
	"strconv"
)

type ApiError struct {
	Error string `json:"error"`
}

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("encode response: %v", err)
	}
}

func WriteError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrEventNotFound):
		WriteJSON(w, http.StatusNotFound, ApiError{"event not found"})
	case errors.Is(err, ErrInvalidEvent):
		WriteJSON(w, http.StatusUnprocessableEntity, ApiError{"event is invalid"})
	default:
		log.Printf("internal error: %v", err)
		WriteJSON(w, http.StatusInternalServerError, ApiError{"internal server error"})
	}
}

func HandleGetEvent(w http.ResponseWriter, r *http.Request){
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	event, err := ServeGetEvent(id)
	if err==nil{
		WriteJSON(w, 200, event)
	} else {
		WriteError(w, err)
	}
}