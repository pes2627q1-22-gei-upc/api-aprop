package main

import (
	"fmt"
	. "api-aprop/internal/handlers"
	"log"
	"net/http"
	// Debug temporal
	. "api-aprop/internal/services"
)

func main() {
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

	// Debug temporal
	Events.Create("Event1", "Description1")
	Events.Create("Event2", "Description2")

	log.Fatal(http.ListenAndServe(":8080", mux))
}