package router

import (
	"net/http"

	"api-aprop/internal/handlers/assistant_handler"
	"api-aprop/internal/handlers/event_handler"
	"api-aprop/internal/handlers/signup_handler"
)

// NewRouter registers the endpoints of every handler. Each handler has its own chunk of endpoints.
// Every endpoint goes through authMiddleware, so only signed in users can use the API.
func NewRouter(eventHandler *event_handler.EventHandler, assistantHandler *assistant_handler.AssistantHandler, signUpHandler *signup_handler.SignUpHandler, authMiddleware func(http.Handler) http.Handler) http.Handler {
	mux := http.NewServeMux()

	// Event handler: events from the Generalitat open data and the ones created by users
	mux.HandleFunc("GET /events", eventHandler.ListEvents)
	mux.HandleFunc("GET /events/{id}", eventHandler.GetEvent)
	mux.HandleFunc("POST /events", eventHandler.CreateEvent)

	// Assistant handler: chatbot that recommends events of the catalogue (Gemini + MCP)
	mux.HandleFunc("POST /assistant/messages", assistantHandler.SendMessage)
	mux.HandleFunc("GET /assistant/conversation", assistantHandler.GetConversation)

	// Sign up handler: users signed up to events (calendar)
	mux.HandleFunc("POST /events/{id}/signup", signUpHandler.SignUp)
	mux.HandleFunc("GET /users/{userId}/events", signUpHandler.ListUserEvents)

	return authMiddleware(mux)
}
