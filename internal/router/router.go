package router

import (
	"net/http"

	"api-aprop/internal/handlers/event_handler"
	"api-aprop/internal/handlers/signup_handler"
)

// NewRouter registers the endpoints of every handler. Each handler has its own chunk of endpoints.
func NewRouter(eventHandler *event_handler.EventHandler, signUpHandler *signup_handler.SignUpHandler) *http.ServeMux {
	mux := http.NewServeMux()

	// Event handler: events from the Generalitat open data and the ones created by users
	mux.HandleFunc("GET /events", eventHandler.ListEvents)
	mux.HandleFunc("GET /events/{id}", eventHandler.GetEvent)
	mux.HandleFunc("POST /events", eventHandler.CreateEvent)

	// Sign up handler: users signed up to events (calendar)
	mux.HandleFunc("POST /events/{id}/signup", signUpHandler.SignUp)
	mux.HandleFunc("GET /users/{userId}/events", signUpHandler.ListUserEvents)

	return mux
}
