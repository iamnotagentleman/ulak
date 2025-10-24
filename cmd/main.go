package main

import (
	"log"
	"net/http"
	"ulak/internal/config"
	"ulak/internal/handlers"
	"ulak/internal/models"
	"ulak/internal/service"
	"ulak/internal/store/postgres"
	"ulak/internal/store/redis"

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

	// Initialize service with both stores
	s := service.NewService(kvStore, msgStore)
	handler := handlers.NewHandler(s)

	mux := http.NewServeMux()

	mux.Handle("POST /messages/auto-send", handlers.WithHTTPIn(handler.SetMessageAutoSend, models.SetMessageAutoSendRequest{}))
	mux.Handle("GET /messages/sent", handlers.WithHTTPIn(handler.GetSentMessages, models.GetMessagesRequest{}))

	// Swagger UI endpoint
	mux.Handle("GET /swagger/", httpSwagger.WrapHandler)

	log.Println("Listening on port 8080")
	log.Println("Swagger UI available at http://localhost:8080/swagger/")
	http.ListenAndServe(":8080", mux)
}
