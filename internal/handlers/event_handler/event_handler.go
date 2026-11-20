package event_handler

import (
	"encoding/json"
	"io"
	"math"
	"net/http"
	"strconv"

	"api-aprop/internal/handlers/requester"
	"api-aprop/internal/handlers/response"
	"api-aprop/internal/models/event_model"
	"api-aprop/internal/services/event_service"
)

const maxBodyBytes = 1 << 20

// maxImageBytes is the largest event image. The request can be a bit larger because of the multipart form.
const maxImageBytes = 5 << 20

type EventHandler struct {
	eventService event_service.Service
	resolver     requester.Resolver
}

func NewEventHandler(eventService event_service.Service, resolver requester.Resolver) *EventHandler {
	return &EventHandler{eventService: eventService, resolver: resolver}
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
	mux.HandleFunc("DELETE /events/{id}", h.DeleteEvent)
	mux.HandleFunc("POST /events/{id}/image", h.UploadEventImage)
	mux.HandleFunc("GET /categories", h.GetCategories)
	mux.HandleFunc("GET /categories/{id}", h.GetCategory)
	mux.HandleFunc("GET /events/recommended", h.GetRecommendedEvents)
	mux.HandleFunc("GET /events/popular", h.GetPopularEvents)
	mux.HandleFunc("GET /events/{id}/ratings", h.GetReceivedRatings)
}

// authorizeCreator checks that the requester is the creator of the event. Official events have no creator,
// so nobody can modify or delete them. It writes the error response and returns false when not allowed.
func (h *EventHandler) authorizeCreator(w http.ResponseWriter, r *http.Request, id int) bool {
	userID, ok := requester.ID(w, r, h.resolver)
	if !ok {
		return false
	}

	event, err := h.eventService.GetEvent(id)
	if err != nil {
		writeServiceError(w, err)
		return false
	}

	if event.IsOfficial || !event.Creator.Valid || event.Creator.String() != userID {
		response.WriteError(w, http.StatusForbidden, "only the creator can modify this event")
		return false
	}
	return true
}

// defaultPageSize is used when a list request does not ask for a page size, like GET /events/search.
const defaultPageSize = 20

// intParam reads an optional integer query parameter, returning def when it is missing.
// It writes the error response and returns false when the value is not a number.
func intParam(w http.ResponseWriter, r *http.Request, name string, def int) (int, bool) {
	value := r.URL.Query().Get(name)
	if value == "" {
		return def, true
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid "+name)
		return 0, false
	}
	return n, true
}

func (h *EventHandler) GetReceivedRatings(w http.ResponseWriter, r *http.Request){
	id, err := strconv.Atoi(r.PathValue("id"))
	if err!=nil{
		response.WriteError(w, http.StatusBadRequest, "invalid event id")
		return
	}

	ratings, err := h.eventService.GetReceivedRatings(id)
	if err!=nil{
		writeServiceError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, ratings)
}

// ListEvents handles GET /events
func (h *EventHandler) ListEvents(w http.ResponseWriter, r *http.Request) {
	params := make(map[string]any)
	pageSize, ok := intParam(w, r, "pageSize", defaultPageSize)
	if !ok {
		return
	}
	if pageSize == 0 {
		pageSize = defaultPageSize
	}
	pageNum, ok := intParam(w, r, "pageNum", 0)
	if !ok {
		return
	}
	filterCategory, ok := intParam(w, r, "filterCategory", 0)
	if !ok {
		return
	}
	params["pageSize"] = pageSize
	params["pageNum"] = pageNum
	params["filterCategory"] = filterCategory
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

// GetRecommendedEvents handles GET /events/recommended?pageSize={m}&pageNum={n}
// TODO(DP-034): confirm whether pageSize/pageNum should be required or default like GET /events.
func (h *EventHandler) GetRecommendedEvents(w http.ResponseWriter, r *http.Request) {
	pageSize, err := strconv.Atoi(r.URL.Query().Get("pageSize"))
	if err != nil || pageSize <= 0 {
		response.WriteError(w, http.StatusBadRequest, "invalid page size")
		return
	}
	pageNum, err := strconv.Atoi(r.URL.Query().Get("pageNum"))
	if err != nil || pageNum < 0 {
		response.WriteError(w, http.StatusBadRequest, "invalid page num")
		return
	}
	userID, ok := requester.ID(w, r, h.resolver)
	if !ok {
		return
	}

	ev, err := h.eventService.GetRecommendedEvents(r.Context(), userID, pageSize, pageNum)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, ev)
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
	userID, ok := requester.ID(w, r, h.resolver)
	if !ok {
		return
	}

	created, err := h.eventService.CreateUnofficialEvent(r.Context(), userID, e)
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

// DeleteEvent handles DELETE /events/{id}. Only the creator of an unofficial event can delete it.
func (h *EventHandler) DeleteEvent(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid event id")
		return
	}

	if !h.authorizeCreator(w, r, id) {
		return
	}

	if err := h.eventService.DeleteUnofficialEvent(r.Context(), id); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
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

	// ParseMultipartForm only limits the memory it uses (the rest goes to disk), so the body is limited here
	r.Body = http.MaxBytesReader(w, r.Body, maxImageBytes+(64<<10))
	if err := r.ParseMultipartForm(maxImageBytes); err != nil {
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
	if header.Size > maxImageBytes {
		response.WriteError(w, http.StatusBadRequest, "file too large or invalid multipart form")
		return
	}

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
// The friends indicator is computed for the signed in user, so it cannot be asked for another user.
func (h *EventHandler) ListMapEvents(w http.ResponseWriter, r *http.Request) {
	filter, ok := parseMapFilter(w, r)
	if !ok {
		return
	}
	if requester.Present(r) {
		if filter.UserID, ok = requester.ID(w, r, h.resolver); !ok {
			return
		}
	}

	events, err := h.eventService.ListMapEvents(r.Context(), filter)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, events)
}

func (h *EventHandler) GetPopularEvents(w http.ResponseWriter, r *http.Request) {
	pageSize, err := strconv.Atoi(r.URL.Query().Get("pageSize"))
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid page size")
		return
	}
	pageNum, err := strconv.Atoi(r.URL.Query().Get("pageNum"))
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid page num")
		return
	}
	ev, err := h.eventService.GetPopularEvents(pageSize, pageNum)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, ev)
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

	if minLatStr == "" && maxLatStr == "" && minLngStr == "" && maxLngStr == "" {
		return event_model.MapFilter{
			MinLat: -90,
			MaxLat: 90,
			MinLng: -180,
			MaxLng: 180,
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
