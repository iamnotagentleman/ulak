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
	"ulak/internal/models"
	"ulak/internal/service"
	"ulak/internal/store/postgres"
	"ulak/internal/store/redis"
	"ulak/internal/worker/populator"
	"ulak/internal/worker/processor"
	"ulak/pkg/notification"

	_ "ulak/docs"

	httpSwagger "github.com/swaggo/http-swagger"
)

// @title           Ulak API
// @version         1.0
// @description     API for managing messages and auto-send functionality

// @host      localhost:8080
// @BasePath  /

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
	messagesCh := make(chan *models.Message, 100)

	// Initialize webhook notification service
	webhookService := notification.NewWebhookNotificationService(cfg.Notification)

	// Initialize workers
	populatorWorker := populator.GetMessagePopulator(cfg.MsgWorker, msgStore)
	processorWorker := processor.NewMessageProcessor(cfg.MsgWorker, kvStore, msgStore, &webhookService)

	// Start populator worker
	populatorTicker := time.NewTicker(5 * time.Second)
	if err := populatorWorker.Start(ctx, populatorTicker, messagesCh); err != nil {
		log.Fatalf("Failed to start populator worker: %v", err)
	}
	log.Println("Populator worker started")

	// Start processor worker
	if err := processorWorker.Start(ctx, messagesCh, nil); err != nil {
		log.Fatalf("Failed to start processor worker: %v", err)
	}
	log.Println("Processor worker started")

	s := service.NewService(kvStore, msgStore)
	handler := handlers.NewHandler(s)

	// Setup HTTP server
	mux := http.NewServeMux()

	mux.Handle("POST /messages/auto-send", handlers.WithHTTPIn(handler.SetMessageAutoSend, models.SetMessageAutoSendRequest{}))
	mux.Handle("GET /messages/sent", handlers.WithHTTPIn(handler.GetSentMessages, models.GetMessagesRequest{}))

	// Swagger UI endpoint
	mux.Handle("GET /swagger/", httpSwagger.WrapHandler)

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	// Start HTTP server in a goroutine
	go func() {
		log.Println("Listening on port 8080")
		log.Println("Swagger UI available at http://localhost:8080/swagger/")
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
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Cancel worker context
	cancel()

	// Stop populator worker first (stops fetching new messages)
	if err := populatorWorker.Stop(10 * time.Second); err != nil {
		log.Printf("Error stopping populator worker: %v", err)
	} else {
		log.Println("Populator worker stopped successfully")
	}

	// Stop processor worker
	if err := processorWorker.Stop(10 * time.Second); err != nil {
		log.Printf("Error stopping processor worker: %v", err)
	} else {
		log.Println("Processor worker stopped successfully")
	}

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
