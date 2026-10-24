package event_handler

import (
	"encoding/json"
	"math"
	"net/http"
	"strconv"

	"api-aprop/internal/handlers/response"
	"api-aprop/internal/models/event_model"
	"api-aprop/internal/services/event_service"
)

const maxBodyBytes = 1 << 20

// UserIDHeader carries the user identity until authentication exists.
const UserIDHeader = "X-User-ID"


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
	mux.HandleFunc("GET /events/search", h.SearchEvents)
	mux.HandleFunc("GET /event/map", h.ListMapEvents)
	mux.HandleFunc("GET /events/map", h.ListMapEvents)
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
	tmp := r.URL.Query().Get("filterCategory")
	if tmp!=""{
		params["filterCategory"], _ = strconv.Atoi(tmp)
	} else {
		params["filterCategory"] = 0
	}
	params["filterRegion"]= r.URL.Query().Get("filterRegion")
	params["startBefore"]= r.URL.Query().Get("startBefore")
	params["startAfter"]= r.URL.Query().Get("startAfter")
	params["endBefore"]= r.URL.Query().Get("endBefore")
	params["endAfter"]= r.URL.Query().Get("endAfter")
	params["filterPermanent"]= r.URL.Query().Get("filterPermanent")
	params["filterFree"]= r.URL.Query().Get("filterFree")
	params["filterOfficial"]= r.URL.Query().Get("filterOfficial")
	params["filterModality"]= r.URL.Query().Get("filterModality")
	
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

// ListMapEvents handles GET /event/map?minLat=...&maxLat=...&minLng=...&maxLng=...
func (h *EventHandler) ListMapEvents(w http.ResponseWriter, r *http.Request) {
	filter, ok := parseMapFilter(w, r)
	if !ok {
		return
	}

	events, err := h.eventService.ListMapEvents(r.Context(), filter)
	if err != nil {
		writeServiceError(w, err)
		return
	}
  response.WriteJSON(w, http.StatusOK, events)
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

func parseMapFilter(w http.ResponseWriter, r *http.Request) (event_model.MapFilter, bool) {
	q := r.URL.Query()
	minLatStr := q.Get("minLat")
	maxLatStr := q.Get("maxLat")
	minLngStr := q.Get("minLng")
	if minLngStr == "" {
		minLngStr = q.Get("minLon")
	}
	maxLngStr := q.Get("maxLng")
	if maxLngStr == "" {
		maxLngStr = q.Get("maxLon")
	}

	userID := r.Header.Get(UserIDHeader)
	if userID == "" {
		userID = q.Get("userId")
	}

	if minLatStr == "" && maxLatStr == "" && minLngStr == "" && maxLngStr == "" {
		return event_model.MapFilter{
			MinLat: -90,
			MaxLat: 90,
			MinLng: -180,
			MaxLng: 180,
			UserID: userID,
		}, true
	}

	if minLatStr == "" || maxLatStr == "" || minLngStr == "" || maxLngStr == "" {
		response.WriteError(w, http.StatusBadRequest, "missing bounding box parameters, expected minLat, maxLat, minLng, maxLng")
		return event_model.MapFilter{}, false
	}

	minLat, ok := parseCoordParam(w, "minLat", minLatStr)
	if !ok {
		return event_model.MapFilter{}, false
	}
	maxLat, ok := parseCoordParam(w, "maxLat", maxLatStr)
	if !ok {
		return event_model.MapFilter{}, false
	}
	minLng, ok := parseCoordParam(w, "minLng", minLngStr)
	if !ok {
		return event_model.MapFilter{}, false
	}
	maxLng, ok := parseCoordParam(w, "maxLng", maxLngStr)
	if !ok {
		return event_model.MapFilter{}, false
	}

	return event_model.MapFilter{
		MinLat: minLat,
		MaxLat: maxLat,
		MinLng: minLng,
		MaxLng: maxLng,
		UserID: userID,
	}, true
}

func parseCoordParam(w http.ResponseWriter, name, value string) (float32, bool) {
	f, err := strconv.ParseFloat(value, 32)
	if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
		response.WriteError(w, http.StatusBadRequest, "invalid "+name+" coordinate")
		return 0, false
	}
	return float32(f), true
}

// SearchEvents handles GET /events/search?q={query}&pageNum={n}&pageSize={m}
func (h *EventHandler) SearchEvents(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		q = r.URL.Query().Get("search")
	}
	pageNum, _ := strconv.Atoi(r.URL.Query().Get("pageNum"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
	if pageSize <= 0 {
		pageSize = 20
	}

	result, err := h.eventService.SearchEvents(r.Context(), q, pageNum, pageSize)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, result)
}
