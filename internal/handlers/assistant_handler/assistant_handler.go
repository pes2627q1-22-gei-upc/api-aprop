package assistant_handler

import (
	"encoding/json"
	"net/http"

	"api-aprop/internal/handlers/requester"
	"api-aprop/internal/handlers/response"
	"api-aprop/internal/services/assistant_service"
)

const maxBodyBytes = 16 << 10

type AssistantHandler struct {
	assistantService assistant_service.Service
	resolver         requester.Resolver
}

func NewAssistantHandler(assistantService assistant_service.Service, resolver requester.Resolver) *AssistantHandler {
	return &AssistantHandler{assistantService: assistantService, resolver: resolver}
}

// RegisterRoutes registers the assistant endpoints.
func (h *AssistantHandler) RegisterRoutes(mux *http.ServeMux) {
	// Assistant handler: chatbot that recommends events of the catalogue (Groq with Gemini as backup, tool calling)
	mux.HandleFunc("POST /assistant/messages", h.SendMessage)
	mux.HandleFunc("GET /assistant/conversation", h.GetConversation)
	mux.HandleFunc("GET /assistant/suggestions", h.GetSuggestions)
	mux.HandleFunc("POST /assistant/feedback", h.SendFeedback)
}

// SendMessage handles POST /assistant/messages
func (h *AssistantHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	userID, ok := requester.ID(w, r, h.resolver)
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
	userID, ok := requester.ID(w, r, h.resolver)
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

// SendFeedback handles POST /assistant/feedback
func (h *AssistantHandler) SendFeedback(w http.ResponseWriter, r *http.Request) {
	userID, ok := requester.ID(w, r, h.resolver)
	if !ok {
		return
	}
	var req feedbackRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes)).Decode(&req); err != nil ||
		req.ConversationStartedAt.IsZero() || req.SentAt.IsZero() || req.Helpful == nil {
		response.WriteError(w, http.StatusBadRequest, "invalid request body: conversation-started-at, sent-at and helpful are required")
		return
	}
	if err := h.assistantService.SendFeedback(r.Context(), userID, req.ConversationStartedAt, req.SentAt, *req.Helpful); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
