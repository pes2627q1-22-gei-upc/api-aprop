package assistant_handler

import (
	"encoding/json"
	"net/http"

	"api-aprop/internal/handlers/response"
	"api-aprop/internal/services/assistant_service"
)

// UserIDHeader carries the user identity until authentication exists.
const UserIDHeader = "X-User-ID"

const maxBodyBytes = 16 << 10

type AssistantHandler struct {
	assistantService assistant_service.Service
}

func NewAssistantHandler(assistantService assistant_service.Service) *AssistantHandler {
	return &AssistantHandler{assistantService: assistantService}
}

// SendMessage handles POST /assistant/messages
func (h *AssistantHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromHeader(w, r)
	if !ok {
		return
	}
	var req sendMessageRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes)).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	exchange, err := h.assistantService.SendMessage(r.Context(), userID, req.Text, req.ByVoice)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusCreated, toExchangeResponse(exchange))
}

// GetConversation handles GET /assistant/conversation
func (h *AssistantHandler) GetConversation(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromHeader(w, r)
	if !ok {
		return
	}
	conversation, err := h.assistantService.GetConversation(r.Context(), userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, toConversationResponse(conversation))
}

// TODO(DP-011): replace X-User-ID with the authenticated Firebase identity.
func userIDFromHeader(w http.ResponseWriter, r *http.Request) (string, bool) {
	userID := r.Header.Get(UserIDHeader)
	if userID == "" {
		response.WriteError(w, http.StatusBadRequest, "missing "+UserIDHeader+" header")
		return "", false
	}
	return userID, true
}
