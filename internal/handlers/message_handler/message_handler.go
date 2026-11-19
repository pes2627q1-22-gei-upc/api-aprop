package message_handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"api-aprop/internal/handlers/requester"
	"api-aprop/internal/handlers/response"
	"api-aprop/internal/services/message_service"
)

type MessageHandler struct {
	service  message_service.Service
	resolver requester.Resolver
}

func NewMessageHandler(service message_service.Service, resolver requester.Resolver) *MessageHandler {
	return &MessageHandler{service: service, resolver: resolver}
}

// RegisterRoutes registers the event chat endpoints.
func (h *MessageHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /events/{id}/messages", h.ListMessages)
	mux.HandleFunc("POST /events/{id}/messages", h.SendMessage)
}

// ListMessages handles GET /events/{id}/messages?pageNum=<n>&pageSize=<n>.
func (h *MessageHandler) ListMessages(w http.ResponseWriter, r *http.Request) {
	eventID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid event id")
		return
	}
	query := r.URL.Query()
	pageNum, ok := intQueryParam(w, query.Get("pageNum"), "pageNum")
	if !ok {
		return
	}
	pageSize, ok := intQueryParam(w, query.Get("pageSize"), "pageSize")
	if !ok {
		return
	}
	userID, ok := requester.ID(w, r, h.resolver)
	if !ok {
		return
	}

	page, err := h.service.ListMessages(r.Context(), userID, eventID, pageNum, pageSize)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, page)
}

// SendMessage handles POST /events/{id}/messages with body {"text": "..."}.
func (h *MessageHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	eventID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid event id")
		return
	}
	userID, ok := requester.ID(w, r, h.resolver)
	if !ok {
		return
	}
	var body struct {
		Text *string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Text == nil {
		response.WriteError(w, http.StatusBadRequest, `invalid body, expected {"text": "..."}`)
		return
	}

	message, err := h.service.SendMessage(r.Context(), userID, eventID, *body.Text)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusCreated, message)
}

// intQueryParam parses an optional integer query parameter; an empty value is 0.
func intQueryParam(w http.ResponseWriter, value, name string) (int, bool) {
	if value == "" {
		return 0, true
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid "+name)
		return 0, false
	}
	return n, true
}

// writeServiceError maps message service errors to HTTP responses.
func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, message_service.ErrEventNotFound):
		response.WriteError(w, http.StatusNotFound, "event not found")
	case errors.Is(err, message_service.ErrNotSignedUp):
		// TODO(DP-040): 403 for R11, like R10 in the ratings.
		response.WriteError(w, http.StatusForbidden, "only users signed up to the event can use its chat")
	case errors.Is(err, message_service.ErrDuplicateMessage):
		response.WriteError(w, http.StatusConflict, "message already sent at the same instant, try again")
	case errors.Is(err, message_service.ErrInvalidText):
		response.WriteError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, message_service.ErrInvalidPagination):
		response.WriteError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, message_service.ErrInvalidUserID):
		response.WriteError(w, http.StatusBadRequest, "invalid user id")
	case errors.Is(err, message_service.ErrInvalidEventID):
		response.WriteError(w, http.StatusBadRequest, "invalid event id")
	default:
		log.Printf("internal error: %v", err)
		response.WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}
