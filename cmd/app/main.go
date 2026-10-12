package main

import (
	"log"
	"net/http"

	"api-aprop/internal/router"
	"api-aprop/internal/repositories/event_repository"
)

func main() {
	event_repository.LoadOfficialEvents() 

	if err := http.ListenAndServe(":8080", router.MakeRouter()); err != nil {
		log.Fatalf("Error init server: %v", err)
	}
 
}