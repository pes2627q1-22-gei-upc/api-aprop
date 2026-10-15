package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"

	"api-aprop/internal/clients/gencat_client"
	"api-aprop/internal/handlers/event_handler"
	"api-aprop/internal/repositories/memory_event_repository"
	"api-aprop/internal/router"
	"api-aprop/internal/services/event_service"
)

func main() {
	// Configuration
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Fatalf("load .env: %v", err)
	}

	port := getEnv("PORT")
	appToken := getEnv("APP_TOKEN")
	gencatEventsURL := getEnv("GENCAT_EVENTS_URL")
	gencatPageSize, err := strconv.Atoi(getEnv("GENCAT_PAGE_SIZE"))
	if err != nil {
		log.Fatalf("invalid GENCAT_PAGE_SIZE: %v", err)
	}

	// Clients
	httpClient := &http.Client{Timeout: 15 * time.Second}
	gencatClient := gencat_client.NewGencatClient(httpClient, gencatEventsURL, appToken)

	// Repositories
	eventRepository := memory_event_repository.NewMemoryEventRepository()

	// Services
	eventService := event_service.NewEventService(eventRepository, gencatClient)

	// Handlers
	eventHandler := event_handler.NewEventHandler(eventService)

	// Initial data
	if err := eventService.LoadOfficialEvents(context.Background(), gencatPageSize); err != nil {
		log.Printf("could not load official events: %v", err)
	}

	// Server: HTTP/1.1 and HTTP/2 without TLS (h2c)
	protocols := new(http.Protocols)
	protocols.SetHTTP1(true)
	protocols.SetUnencryptedHTTP2(true)

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           router.NewRouter(eventHandler),
		Protocols:         protocols,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("listening on %s", srv.Addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

// getEnv returns the value of a required environment variable and exits if it is not set.
func getEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("missing environment variable %s", key)
	}
	return value
}
