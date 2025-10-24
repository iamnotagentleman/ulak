package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"ulak/internal/config"
	"ulak/internal/handlers"
	"ulak/internal/manager"
	"ulak/internal/models"
	"ulak/internal/service"
	"ulak/internal/store/postgres"
	"ulak/internal/store/redis"
	"ulak/internal/worker/populator"
	"ulak/internal/worker/processor"
	"ulak/middleware"
	"ulak/pkg/notification"

	"ulak/docs"

	httpSwagger "github.com/swaggo/http-swagger"
)

// @title           Ulak API
// @version         1.0
// @description     API for managing messages and auto-send functionality

// @host      localhost:8080
// @BasePath  /

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name x-ins-auth-key
// @description API key authentication.

func main() {
	cfg, err := config.LoadEnvVars()
	if err != nil {
		log.Fatal(err)
	}

	// Initialize key-value store (Redis)
	kvStore, err := redis.NewRedisStore(cfg.Redis)
	if err != nil {
		log.Fatalf("Failed to initialize key-value store: %v", err)
	}

	// Initialize message store (PostgreSQL)
	msgStore, err := postgres.NewPostgresStore(cfg.Postgres)
	if err != nil {
		log.Fatalf("Failed to initialize message store: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create shared message channel
	messagesCh := make(chan *models.Message, cfg.Common.MessageChannelSize)

	// Initialize webhook notification service
	webhookService := notification.NewWebhookNotificationService(cfg.Notification)

	// Initialize workers
	populatorWorker := populator.GetMessagePopulator(cfg.MsgWorker, msgStore)
	processorWorker := processor.NewMessageProcessor(cfg.MsgWorker, kvStore, msgStore, &webhookService)

	populatorTicker := time.NewTicker(cfg.MsgWorker.PopulatorInterval)
	defer populatorTicker.Stop()

	workerManager := manager.NewWorkerManager(ctx, populatorWorker, processorWorker, messagesCh, populatorTicker, cfg.Server)
	workerManager.Start()

	s := service.NewService(kvStore, msgStore, workerManager)
	handler := handlers.NewHandler(s)

	// Configure Swagger with dynamic host and basepath
	docs.SwaggerInfo.Host = cfg.Server.SwaggerHost
	docs.SwaggerInfo.BasePath = cfg.Server.SwaggerBasePath

	// Setup HTTP server
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	mux.Handle("POST /messages/auto-send", middleware.ApiKeyAuthMiddleware(cfg.Auth, middleware.SerializeRequestsMiddleware(http.HandlerFunc(handler.SetMessageAutoSend))))
	mux.Handle("GET /messages/sent", middleware.ApiKeyAuthMiddleware(cfg.Auth, http.HandlerFunc(handler.GetSentMessages)))

	// Swagger UI endpoint
	mux.Handle("GET /swagger/", httpSwagger.WrapHandler)

	server := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: mux,
	}

	// Start HTTP server in a goroutine
	go func() {
		log.Printf("Listening on port %s", cfg.Server.Port)
		log.Printf("Swagger UI available at http://%s/swagger/", cfg.Server.SwaggerHost)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Setup signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	// Wait for shutdown signal
	<-sigCh
	log.Println("Shutdown signal received, initiating graceful shutdown...")

	// Create shutdown context with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer shutdownCancel()

	// Cancel worker context
	cancel()

	workerManager.Stop()
	// Close message channel
	close(messagesCh)

	// Shutdown HTTP server
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	} else {
		log.Println("HTTP server stopped successfully")
	}

	log.Println("Application shutdown complete")
}
