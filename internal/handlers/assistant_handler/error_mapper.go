package assistant_handler

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"api-aprop/internal/handlers/response"
	"api-aprop/internal/services/assistant_service"
)

// writeServiceError maps assistant service errors to HTTP responses.
// Chatbot failures only affect this response, so the client can show an error and retry.
func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, assistant_service.ErrInvalidUserID):
		response.WriteError(w, http.StatusBadRequest, "invalid user id")
	case errors.Is(err, assistant_service.ErrInvalidMessage):
		response.WriteError(w, http.StatusBadRequest,
			fmt.Sprintf("invalid message: the text must have between 1 and %d characters", assistant_service.MaxMessageRunes))
	case errors.Is(err, assistant_service.ErrTooManyMessages):
		response.WriteError(w, http.StatusTooManyRequests, "too many messages, wait a minute and try again")
	case errors.Is(err, assistant_service.ErrMessageNotFound):
		response.WriteError(w, http.StatusNotFound, "assistant message not found")
	case errors.Is(err, assistant_service.ErrConversationNotFound):
		response.WriteError(w, http.StatusNotFound, "conversation not found")
	case errors.Is(err, assistant_service.ErrAssistantTimeout):
		log.Printf("assistant timeout: %v", err)
		response.WriteError(w, http.StatusGatewayTimeout, "the assistant took too long to answer, try again")
	case errors.Is(err, assistant_service.ErrAssistantUnavailable):
		log.Printf("assistant unavailable: %v", err)
		response.WriteError(w, http.StatusBadGateway, "the assistant is not available right now, try again")
	default:
		log.Printf("internal error: %v", err)
		response.WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}
