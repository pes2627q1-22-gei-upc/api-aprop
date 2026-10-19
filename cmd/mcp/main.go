// Command mcp serves the events catalogue as a Model Context Protocol server (Streamable HTTP).
// The API uses it as the tool provider of the chatbot.
package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"
	_ "time/tzdata" // the runtime image has no time zone database

	"github.com/joho/godotenv"

	"api-aprop/internal/drivers/postgres_driver"
	"api-aprop/internal/mcp_server"
	postgres_repository "api-aprop/internal/repositories/postgres"
)

const (
	appName        = "mcp-aprop"
	defaultSslMode = "disable"
	timeZone       = "Europe/Madrid"
)

func main() {
	if err := godotenv.Load(".env.example"); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Fatalf("load .env: %v", err)
	}

	port := getEnv("MCP_PORT")
	token := getEnv("MCP_TOKEN")
	location, err := time.LoadLocation(timeZone)
	if err != nil {
		log.Fatalf("load time zone %s: %v", timeZone, err)
	}

	sslMode := os.Getenv("DB_SSLMODE")
	if sslMode == "" {
		sslMode = defaultSslMode
	}
	dbConfig := postgres_driver.NewConfig(
		getEnv("DB_HOST"), getEnv("DB_PORT"), getEnv("DB_USERNAME"), getEnv("DB_PASSWORD"), getEnv("DB_DATABASE"),
		map[string]string{"application_name": appName, "sslmode": sslMode},
	)
	postgresDriver := postgres_driver.New()
	if err := postgresDriver.Connect(dbConfig.GetConnectionString(), context.Background()); err != nil {
		slog.Error("Error connecting to database", "error", err)
		os.Exit(1)
	}
	defer postgresDriver.Close()
	slog.Info("Successfully connected to database")

	catalogueRepository := postgres_repository.NewCatalogueRepository(postgresDriver.Pool())
	server := mcp_server.NewServer(catalogueRepository, func() time.Time { return time.Now().In(location) })

	mux := http.NewServeMux()
	mux.Handle("/mcp", mcp_server.NewHandler(server, token))

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Printf("MCP server listening on %s/mcp", srv.Addr)
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
