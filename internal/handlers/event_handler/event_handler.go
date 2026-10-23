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

// RegisterRoutes registers the event endpoints.
func (h *EventHandler) RegisterRoutes(mux *http.ServeMux) {
	// Event handler: events from the Generalitat open data and the ones created by users
	mux.HandleFunc("GET /events", h.ListEvents)
	mux.HandleFunc("GET /events/{id}", h.GetEvent)
	mux.HandleFunc("POST /events", h.CreateEvent)
	mux.HandleFunc("GET /categories", h.GetCategories)
	mux.HandleFunc("GET /categories/{id}", h.GetCategory)
}

// ListEvents handles GET /events
func (h *EventHandler) ListEvents(w http.ResponseWriter, r *http.Request) {
	params := make(map[string]any)
	params["pageSize"], _ = strconv.Atoi(r.URL.Query().Get("pageSize"))
	params["pageNum"], _ = strconv.Atoi(r.URL.Query().Get("pageNum"))
	
	list, err := h.eventService.ListEvents(params)
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

// GetCategories handles GET /categories
func (h *EventHandler) GetCategories(w http.ResponseWriter, r *http.Request){
	list, err := h.eventService.GetCategories()
	if err != nil {
		writeServiceError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, list)
}

// GetCategories handles GET /categories/{id}
func (h *EventHandler) GetCategory(w http.ResponseWriter, r *http.Request){
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid category id")
		return
	}
	
	cat, err := h.eventService.GetCategory(id)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, cat)
}