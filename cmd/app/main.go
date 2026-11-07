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
	"strings"
	"time"
	_ "time/tzdata" // the runtime image has no time zone database

	"github.com/joho/godotenv"
	"github.com/rs/cors"

	"api-aprop/internal/clients/firebase_client"
	"api-aprop/internal/clients/gencat_client"
	"api-aprop/internal/clients/groq_client"
	"api-aprop/internal/clients/s3_client"
	"api-aprop/internal/drivers/postgres_driver"
	"api-aprop/internal/handlers/assistant_handler"
	"api-aprop/internal/handlers/auth_handler"
	"api-aprop/internal/handlers/event_handler"
	"api-aprop/internal/handlers/preference_handler"
	"api-aprop/internal/handlers/requester"
	"api-aprop/internal/handlers/signup_handler"
	"api-aprop/internal/handlers/user_handler"
	"api-aprop/internal/handlers/notification_handler"
	"api-aprop/internal/middleware/auth_middleware"
	postgres_repository "api-aprop/internal/repositories/postgres"
	"api-aprop/internal/router"
	"api-aprop/internal/services/assistant_service"
	"api-aprop/internal/services/auth_service"
	"api-aprop/internal/services/event_service"
	"api-aprop/internal/services/preference_service"
	"api-aprop/internal/services/signup_service"
	"api-aprop/internal/services/user_service"
	"api-aprop/internal/services/notification_service"
	"api-aprop/migrations"
)

const (
	appName = "api-aprop"

	// Relative dates in the chatbot queries ("demà", "aquest cap de setmana") are resolved in this time zone
	timeZone = "Europe/Madrid"

	// Database environment variables
	dbEnvHost      = "DB_HOST"
	dbEnvPort      = "DB_PORT"
	dbEnvUsername  = "DB_USERNAME"
	dbEnvPassword  = "DB_PASSWORD"
	dbEnvDatabase  = "DB_DATABASE"
	dbEnvSslMode   = "DB_SSLMODE"   // optional, defaults to defaultSslMode
	dbEnvPoolSize  = "DB_POOL_SIZE" // optional, defaults to the pgxpool default
	defaultSslMode = "disable"

	// CORS environment variable: comma separated origins that can call the API from a browser
	corsEnvAllowedOrigins = "CORS_ALLOWED_ORIGINS" // optional, defaults to defaultCORSOrigins
	defaultCORSOrigins    = "*"                    // any origin
)

func runDaily(task func()) {
	go func() {
		for {
			now := time.Now()

			next := time.Date(
				now.Year(), now.Month(), now.Day(),
				0, 0, 0, 0,
				now.Location(),
			)

			// If today's time has passed, schedule tomorrow.
			if !next.After(now) {
				next = next.AddDate(0, 0, 1)
			}

			log.Printf("next load: %s", next.String())
			timer := time.NewTimer(time.Until(next))
			<-timer.C
			log.Printf("loading official events")
			task()
		}
	}()
}

func checkForReminders(minute_interval int, handle_notifs func()) {
	go func() {
		handle_notifs()
		for {
			now := time.Now()
			next := now.Add(time.Minute * time.Duration(minute_interval))
			log.Printf("next reminder load: %s", next.String())
			timer := time.NewTimer(time.Until(next))
			<-timer.C
			log.Printf("loading reminders")
			handle_notifs()
		}
	}()
}

func main() {
	// Configuration
	if err := godotenv.Load(".env", ".env.example"); err != nil && !errors.Is(err, os.ErrNotExist) {
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
	groqAPIKey := os.Getenv("GROQ_API_KEYS")
	if groqAPIKey == "" {
		groqAPIKey = getEnv("GROQ_API_KEY")
	}
	groqModel := getEnv("GROQ_MODEL")
	location, err := time.LoadLocation(timeZone)
	if err != nil {
		log.Fatalf("load time zone %s: %v", timeZone, err)
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
	var chatClient groq_client.Client = groq_client.NewGroqClient(httpClient, groq_client.DefaultBaseURL, groqAPIKey, groqModel)
	// Gemini is the backup when every Groq key fails.
	if geminiKey := os.Getenv("GEMINI_API_KEY"); geminiKey != "" {
		geminiModel := os.Getenv("GEMINI_MODEL")
		if geminiModel == "" {
			geminiModel = "gemini-3.8-flash" // TODO(DP-027): gemini-2.5-flash returns 404; this one is the model Google suggests, not verified with a successful call
		}
		chatClient = groq_client.NewFallbackClient(chatClient, groq_client.NewGroqClient(httpClient, groq_client.GeminiBaseURL, geminiKey, geminiModel))
	}
	s3Client := s3_client.NewS3Client(httpClient, s3_client.S3Config{
		Bucket:    os.Getenv("S3_BUCKET"),
		Region:    os.Getenv("S3_REGION"),
		Endpoint:  os.Getenv("S3_ENDPOINT"),
		PublicURL: os.Getenv("S3_PUBLIC_URL"),
	})

	// CORS: it wraps the auth middleware (see router.NewRouter) because browsers send the preflight without the token.
	// Requests without an Origin header (mobile app, curl) are not affected.
	corsOrigins := os.Getenv(corsEnvAllowedOrigins)
	if corsOrigins == "" {
		corsOrigins = defaultCORSOrigins
	}
	allowedOrigins := strings.Split(corsOrigins, ",")
	for i, origin := range allowedOrigins {
		allowedOrigins[i] = strings.TrimSpace(origin)
	}
	corsMiddleware := cors.New(cors.Options{
		AllowedOrigins: allowedOrigins,
		AllowedMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete},
		AllowedHeaders: []string{"Authorization", "Content-Type", requester.UserIDHeader},
		MaxAge:         600, // seconds the browser caches the preflight response
	}).Handler

	var authMiddleware func(http.Handler) http.Handler
	if _, err := os.Stat(firebaseCredentialsFile); err == nil {
		firebaseClient, err := firebase_client.NewFirebaseClient(context.Background(), firebaseCredentialsFile)
		if err != nil {
			slog.Error("Error initializing Firebase", "error", err)
			os.Exit(1)
		}
		authMiddleware = auth_middleware.NewAuthMiddleware(firebaseClient)
	} else {
		slog.Warn("Firebase credentials file not found; running with dev auth middleware for local development", "file", firebaseCredentialsFile)
		authMiddleware = auth_middleware.NewDevAuthMiddleware()
	}

	// Repositories
	eventRepository := postgres_repository.NewEventRepository(postgresDriver.Pool())
	chatbotRepository := postgres_repository.NewChatbotRepository(postgresDriver.Pool())
	catalogueRepository := postgres_repository.NewCatalogueRepository(postgresDriver.Pool())
	signUpRepository := postgres_repository.NewSignUpRepository(postgresDriver.Pool())
	authRepository := postgres_repository.NewAuthRepository(postgresDriver.Pool())
	userRepository := postgres_repository.NewUserRepository(postgresDriver.Pool())
	notificationRepository := postgres_repository.NewNotificationRepository(postgresDriver.Pool())
	preferenceRepository := postgres_repository.NewPreferenceRepository(postgresDriver.Pool())

	// Services
	eventService := event_service.NewEventService(eventRepository, authRepository, preferenceRepository, gencatClient, s3Client)
	assistantService := assistant_service.NewAssistantService(
		chatbotRepository, catalogueRepository, chatClient,
		func() time.Time { return time.Now().In(location) }, assistant_service.ReplyTimeout,
	).WithPreferences(preferenceRepository)
	signUpService := signup_service.NewSignUpService(signUpRepository, eventRepository)
	authService := auth_service.NewAuthService(authRepository)
	userService := user_service.NewUserService(userRepository, s3Client)
	notificationService := notification_service.NewNotificationService(notificationRepository)
	preferenceService := preference_service.NewPreferenceService(preferenceRepository)

	// Handlers
	eventHandler := event_handler.NewEventHandler(eventService)
	assistantHandler := assistant_handler.NewAssistantHandler(assistantService, eventService)
	signUpHandler := signup_handler.NewSignUpHandler(signUpService, eventService)
	authHandler := auth_handler.NewAuthHandler(authService)
	userHandler := user_handler.NewUserHandler(userService)
	notificationHandler := notification_handler.NewNotificationHandler(notificationService)
	preferenceHandler := preference_handler.NewPreferenceHandler(preferenceService, eventService)

	// Initial data
	go func() {
		if err := eventService.LoadOfficialEvents(context.Background(), gencatPageSize); err != nil {
			log.Printf("could not load official events: %v", err)
		}
	}()
	// Start goroutines
	runDaily(func() {
		if err := eventService.LoadOfficialEvents(context.Background(), gencatPageSize); err != nil {
			log.Printf("could not load official events: %v", err)
		}
	})
	checkForReminders(5, func(){
		if nots, err := notificationService.CreateReminders(); err!=nil{
			log.Printf("could not load reminders: %v", err)
		} else {
			log.Printf("loaded %d new reminders", len(nots))
		}
	})

	mux := http.NewServeMux()
	mux.Handle("/", router.NewRouter(corsMiddleware, authMiddleware, eventHandler, assistantHandler, signUpHandler, authHandler, notificationHandler, userHandler, preferenceHandler))

	// Server: HTTP/1.1 and HTTP/2 without TLS (h2c)
	protocols := new(http.Protocols)
	protocols.SetHTTP1(true)
	protocols.SetUnencryptedHTTP2(true)

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
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
