package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"

	"api-aprop/internal/clients/firebase_client"
	"api-aprop/internal/clients/gencat_client"
	"api-aprop/internal/drivers/postgres_driver"
	"api-aprop/internal/handlers/event_handler"
	"api-aprop/internal/middleware/auth_middleware"
	postgres_repository "api-aprop/internal/repositories/postgres"
	"api-aprop/internal/router"
	"api-aprop/internal/services/event_service"
	"api-aprop/migrations"
)

const (
	appName = "api-aprop"

	// Database environment variables
	dbEnvHost      = "DB_HOST"
	dbEnvPort      = "DB_PORT"
	dbEnvUsername  = "DB_USERNAME"
	dbEnvPassword  = "DB_PASSWORD"
	dbEnvDatabase  = "DB_DATABASE"
	dbEnvSslMode   = "DB_SSLMODE"   // optional, defaults to defaultSslMode
	dbEnvPoolSize  = "DB_POOL_SIZE" // optional, defaults to the pgxpool default
	defaultSslMode = "disable"
)

func main() {
	// Configuration
	if err := godotenv.Load(".env"); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Fatalf("load .env: %v", err)
	}

	port := getEnv("PORT")
	appToken := getEnv("APP_TOKEN")
	// Path to the Firebase service account JSON (Firebase console > Project settings > Service accounts)
	firebaseCredentialsFile := getEnv("FIREBASE_CREDENTIALS_FILE")
	gencatEventsURL := getEnv("GENCAT_EVENTS_URL")
	gencatPageSize, err := strconv.Atoi(getEnv("GENCAT_PAGE_SIZE"))
	if err != nil {
		log.Fatalf("invalid GENCAT_PAGE_SIZE: %v", err)
	}

	// Database
	dbHost := getEnv(dbEnvHost)
	dbPort := getEnv(dbEnvPort)
	dbUsername := getEnv(dbEnvUsername)
	dbPassword := getEnv(dbEnvPassword)
	dbDatabase := getEnv(dbEnvDatabase)
	dbSslMode := os.Getenv(dbEnvSslMode)
	if dbSslMode == "" {
		dbSslMode = defaultSslMode
	}

	dbOptions := map[string]string{
		"application_name": appName,
		"sslmode":          dbSslMode,
	}
	dbPoolSize := os.Getenv(dbEnvPoolSize)
	if dbPoolSize != "" {
		poolSize, err := strconv.Atoi(dbPoolSize)
		if err != nil || poolSize <= 0 {
			slog.Error(fmt.Sprintf("Please set the %s environment variable to a positive integer", dbEnvPoolSize))
			os.Exit(1)
		}
		dbOptions["pool_max_conns"] = dbPoolSize
	}

	dbConfig := postgres_driver.NewConfig(
		dbHost, dbPort, dbUsername, dbPassword, dbDatabase,
		dbOptions,
	)

	postgresDriver := postgres_driver.New()
	if err := postgresDriver.Connect(dbConfig.GetConnectionString(), context.Background()); err != nil {
		slog.Error("Error connecting to database", "error", err)
		os.Exit(1)
	}
	slog.Info("Successfully connected to database")
	defer postgresDriver.Close()

	// Migrate database
	if err := postgresDriver.MigrateFS(migrations.EmbedFS, dbConfig.GetConnectionString()); err != nil {
		slog.Error("Error migrating database", "error", err)
		os.Exit(1)
	}
	slog.Info("Database schema is up to date.")

	// Clients
	httpClient := &http.Client{Timeout: 15 * time.Second}
	gencatClient := gencat_client.NewGencatClient(httpClient, gencatEventsURL, appToken)
	firebaseClient, err := firebase_client.NewFirebaseClient(context.Background(), firebaseCredentialsFile)
	if err != nil {
		slog.Error("Error initializing Firebase", "error", err)
		os.Exit(1)
	}

	// Repositories
	eventRepository := postgres_repository.NewEventRepository(postgresDriver.Pool())

	// Services
	eventService := event_service.NewEventService(eventRepository, gencatClient)

	// Handlers
	eventHandler := event_handler.NewEventHandler(eventService)

	// Middlewares
	authMiddleware := auth_middleware.NewAuthMiddleware(firebaseClient)

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
		Handler:           router.NewRouter(eventHandler, authMiddleware),
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
