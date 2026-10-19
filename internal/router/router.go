package router

import (
	"net/http"

	"api-aprop/internal/handlers/assistant_handler"
	"api-aprop/internal/handlers/event_handler"
)

// NewRouter registers the endpoints of every handler. Each handler has its own chunk of endpoints.
func NewRouter(eventHandler *event_handler.EventHandler, assistantHandler *assistant_handler.AssistantHandler) *http.ServeMux {
	mux := http.NewServeMux()

	// Event handler: events from the Generalitat open data and the ones created by users
	mux.HandleFunc("GET /events", eventHandler.ListEvents)
	mux.HandleFunc("GET /events/{id}", eventHandler.GetEvent)
	mux.HandleFunc("POST /events", eventHandler.CreateEvent)

	// Assistant handler: chatbot that recommends events of the catalogue (Gemini + MCP)
	mux.HandleFunc("POST /assistant/messages", assistantHandler.SendMessage)
	mux.HandleFunc("GET /assistant/conversation", assistantHandler.GetConversation)

	return mux
}
