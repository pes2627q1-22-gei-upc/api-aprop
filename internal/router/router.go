package router

import (
	"net/http"
)

// Handler is implemented by every handler of the API. Each handler registers its own chunk of endpoints,
// so adding a new one only needs passing it to NewRouter.
type Handler interface {
	RegisterRoutes(mux *http.ServeMux)
}

// NewRouter registers the endpoints of every handler.
// Every endpoint goes through authMiddleware, so only signed in users can use the API.
func NewRouter(authMiddleware func(http.Handler) http.Handler, handlers ...Handler) http.Handler {
	mux := http.NewServeMux()
	for _, handler := range handlers {
		handler.RegisterRoutes(mux)
	}
	return authMiddleware(mux)
}
