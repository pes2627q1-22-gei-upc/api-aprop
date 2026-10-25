package event_handler

import (
	"encoding/json"
	"io"
	"math"
	"net/http"
	"strconv"

	"api-aprop/internal/handlers/response"
	"api-aprop/internal/middleware/auth_middleware"
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
	mux.HandleFunc("PUT /events/{id}", h.UpdateEvent)
	mux.HandleFunc("POST /events/{id}/image", h.UploadEventImage)
	mux.HandleFunc("GET /categories", h.GetCategories)
	mux.HandleFunc("GET /categories/{id}", h.GetCategory)
}

func getUserID(r *http.Request) string {
	if u, ok := auth_middleware.UserFromContext(r.Context()); ok && u.UID != "" {
		return u.UID
	}
	return r.Header.Get(UserIDHeader)
}

// requesterID returns the database ID (UUID) of the user making the request.
func (h *EventHandler) requesterID(r *http.Request) (string, error) {
	return h.eventService.ResolveUserID(r.Context(), getUserID(r))
}

// authorizeCreator checks that the requester is the creator of the event. Official events have no creator,
// so nobody can modify or delete them. It writes the error response and returns false when not allowed.
func (h *EventHandler) authorizeCreator(w http.ResponseWriter, r *http.Request, id int) bool {
	requester, err := h.requesterID(r)
	if err != nil {
		writeServiceError(w, err)
		return false
	}

	event, err := h.eventService.GetEvent(id)
	if err != nil {
		writeServiceError(w, err)
		return false
	}

	if event.IsOfficial || !event.Creator.Valid || event.Creator.String() != requester {
		response.WriteError(w, http.StatusForbidden, "only the creator can modify this event")
		return false
	}
	return true
}

// ListEvents handles GET /events
func (h *EventHandler) ListEvents(w http.ResponseWriter, r *http.Request) {
	params := make(map[string]any)
	params["pageSize"], _ = strconv.Atoi(r.URL.Query().Get("pageSize"))
	params["pageNum"], _ = strconv.Atoi(r.URL.Query().Get("pageNum"))
	tmp := r.URL.Query().Get("filterCategory")
	if tmp != "" {
		params["filterCategory"], _ = strconv.Atoi(tmp)
	} else {
		params["filterCategory"] = 0
	}
	params["filterRegion"] = r.URL.Query().Get("filterRegion")
	params["startBefore"] = r.URL.Query().Get("startBefore")
	params["startAfter"] = r.URL.Query().Get("startAfter")
	params["endBefore"] = r.URL.Query().Get("endBefore")
	params["endAfter"] = r.URL.Query().Get("endAfter")
	params["filterPermanent"] = r.URL.Query().Get("filterPermanent")
	params["filterFree"] = r.URL.Query().Get("filterFree")
	params["filterOfficial"] = r.URL.Query().Get("filterOfficial")
	params["filterModality"] = r.URL.Query().Get("filterModality")

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

	// Users can only create unofficial events, so the creator is required
	requester, err := h.requesterID(r)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	created, err := h.eventService.CreateUnofficialEvent(r.Context(), requester, e)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusCreated, created)
}

// UpdateEvent handles PUT /events/{id}. Only the creator of an unofficial event can modify it.
func (h *EventHandler) UpdateEvent(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid event id")
		return
	}

	var e event_model.Event
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes)).Decode(&e); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if !h.authorizeCreator(w, r, id) {
		return
	}

	updated, err := h.eventService.UpdateUnofficialEvent(r.Context(), id, e)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, updated)
}

// UploadEventImage handles POST /events/{id}/image. Only the creator of an unofficial event can change its image.
func (h *EventHandler) UploadEventImage(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid event id")
		return
	}

	if !h.authorizeCreator(w, r, id) {
		return
	}

	if err := r.ParseMultipartForm(5 << 20); err != nil {
		response.WriteError(w, http.StatusBadRequest, "file too large or invalid multipart form")
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		file, header, err = r.FormFile("file")
		if err != nil {
			response.WriteError(w, http.StatusBadRequest, "missing image file in multipart form")
			return
		}
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	if contentType == "" || contentType == "application/octet-stream" {
		buf := make([]byte, 512)
		n, _ := file.Read(buf)
		contentType = http.DetectContentType(buf[:n])
		if seeker, ok := file.(io.Seeker); ok {
			_, _ = seeker.Seek(0, io.SeekStart)
		}
	}

	imageURL, err := h.eventService.UploadEventImage(r.Context(), id, header.Filename, contentType, file)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusCreated, map[string]any{
		"event-id":  id,
		"image-url": imageURL,
	})
}

// GetCategories handles GET /categories
func (h *EventHandler) GetCategories(w http.ResponseWriter, r *http.Request) {
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

// GetCategory handles GET /categories/{id}
func (h *EventHandler) GetCategory(w http.ResponseWriter, r *http.Request) {
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
