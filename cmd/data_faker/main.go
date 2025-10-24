package main

import (
	"context"
	"log"
	"time"
	"ulak/internal/config"
	"ulak/internal/enums"
	"ulak/internal/models"
	"ulak/internal/store/postgres"

	"github.com/google/uuid"
)

func main() {
	log.Println("Starting database data_faker...")

	cfg, err := config.LoadEnvVars()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	// Wait for database to be ready
	time.Sleep(5 * time.Second)

	// Initialize message store
	msgStore, err := postgres.NewPostgresStore(cfg.Postgres)
	if err != nil {
		log.Fatal("Failed to initialize message store:", err)
	}

	log.Println("Connected to database successfully")

	ctx := context.Background()

	// Check if data already exists
	count, err := msgStore.GetTotalCount(ctx)
	if err != nil {
		log.Fatal("Failed to get message count:", err)
	}

	if count > 0 {
		log.Printf("Database already populated with %d messages. Skipping seed data.\n", count)
		return
	}

	// Seed initial messages
	messages := []*models.Message{
		{
			ID:       uuid.New(),
			To:       "insider@example.com",
			Message:  "Welcome to Ulak! This is your first test message. We're always welcome to insiderians :>",
			Channel:  enums.ChannelEmail,
			Status:   enums.StatusSent,
			IsActive: true,
		},
		{
			ID:       uuid.New(),
			To:       "+1234567890",
			Message:  "This is a pending SMS notification.",
			Channel:  enums.ChannelSMS,
			Status:   enums.StatusPending,
			IsActive: true,
		},
		{
			ID:       uuid.New(),
			To:       "device-token-123",
			Message:  "This is a pending PUSH notification.",
			Channel:  enums.ChannelPush,
			Status:   enums.StatusSent,
			IsActive: true,
		},
		{
			ID:       uuid.New(),
			To:       "failed@example.com",
			Message:  "Failed email case.",
			Channel:  enums.ChannelEmail,
			Status:   enums.StatusFailed,
			IsActive: true,
		},
		{
			ID:       uuid.New(),
			To:       "https://webhook.example.com/notify",
			Message:  "webhook pending.",
			Channel:  enums.ChannelWebhook,
			Status:   enums.StatusPending,
			IsActive: true,
		},
	}

	log.Printf("Seeding %d initial messages...\n", len(messages))

	for i, msg := range messages {
		if err := msgStore.Create(ctx, msg); err != nil {
			log.Printf("Failed to create message %d: %v\n", i+1, err)
			continue
		}
		log.Printf("Created message %d: %s\n", i+1, msg.Message)
	}

	log.Println("Database population completed successfully!")
}
