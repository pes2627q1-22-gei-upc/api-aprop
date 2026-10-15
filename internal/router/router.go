package router

import (
	"net/http"

	"api-aprop/internal/handlers/event_handler"
)

// NewRouter registers the endpoints of every handler. Each handler has its own chunk of endpoints.
func NewRouter(eventHandler *event_handler.EventHandler) *http.ServeMux {
	mux := http.NewServeMux()

	// Event handler: events from the Generalitat open data and the ones created by users
	mux.HandleFunc("GET /events", eventHandler.ListEvents)
	mux.HandleFunc("GET /events/{id}", eventHandler.GetEvent)
	mux.HandleFunc("POST /events", eventHandler.CreateEvent)

	return mux
}
