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

const defaultGencatEventsURL = "https://analisi.transparenciacatalunya.cat/api/v3/views/rhpv-yr4f/query.json"

func main() {
	// Configuration
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Fatalf("load .env: %v", err)
	}

	port := getEnv("PORT", "8080")
	appToken := os.Getenv("APP_TOKEN")
	gencatEventsURL := getEnv("GENCAT_EVENTS_URL", defaultGencatEventsURL)
	gencatPageSize, err := strconv.Atoi(getEnv("GENCAT_PAGE_SIZE", "10"))
	if err != nil {
		log.Fatalf("invalid GENCAT_PAGE_SIZE: %v", err)
	}
	tlsCertFile := os.Getenv("TLS_CERT_FILE")
	tlsKeyFile := os.Getenv("TLS_KEY_FILE")

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

	// Server: HTTP/1.1, HTTP/2 over TLS and HTTP/2 without TLS (h2c)
	protocols := new(http.Protocols)
	protocols.SetHTTP1(true)
	protocols.SetHTTP2(true)
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

	if tlsCertFile != "" && tlsKeyFile != "" {
		log.Printf("listening on %s (TLS)", srv.Addr)
		err = srv.ListenAndServeTLS(tlsCertFile, tlsKeyFile)
	} else {
		log.Printf("listening on %s", srv.Addr)
		err = srv.ListenAndServe()
	}
	if err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func getEnv(key string, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
