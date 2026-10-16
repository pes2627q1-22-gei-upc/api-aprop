package event_handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"api-aprop/internal/handlers/response"
	"api-aprop/internal/models/event_model"
	"api-aprop/internal/services/event_service"
)

const maxBodyBytes = 1 << 20

type EventHandler struct {
	eventService event_service.Service
}

func NewEventHandler(eventService event_service.Service) *EventHandler {
	return &EventHandler{eventService: eventService}
}

// ListEvents handles GET /events
func (h *EventHandler) ListEvents(w http.ResponseWriter, r *http.Request) {
	list, err := h.eventService.ListEvents()
	if err != nil {
		writeServiceError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, list)
}

// GetEvent handles GET /events/{id}
func (h *EventHandler) GetEvent(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid event id")
		return
	}

	event, err := h.eventService.GetEvent(id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, event)
}

// CreateEvent handles POST /events
func (h *EventHandler) CreateEvent(w http.ResponseWriter, r *http.Request) {
	var e event_model.Event
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes)).Decode(&e); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	created, err := h.eventService.CreateEvent(e)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusCreated, created)
}
