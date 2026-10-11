package main

import (
	"log"
	"net/http"
	. "api-aprop/internal/router"
	// Debug temporal
	. "api-aprop/internal/services"
)

func main() {
	// Debug temporal
	Events.Create("Event1", "Description1")
	Events.Create("Event2", "Description2")

	log.Fatal(http.ListenAndServe(":8080", MakeRouter()))
}