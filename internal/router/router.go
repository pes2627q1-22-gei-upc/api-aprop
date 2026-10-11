package router

import (
	. "api-aprop/internal/handlers"
	"net/http"
	"fmt"
)

func MakeRouter() (*http.ServeMux){
	mux := http.NewServeMux()

	// Method + literal path.
	mux.HandleFunc("GET /events", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "list all events")
	})

	// Method + wildcard path segment. {id} is captured by name.
	mux.HandleFunc("GET /events/{id}", HandleGetEvent)

	mux.HandleFunc("POST /events", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "create an event")
	})

	return mux
}