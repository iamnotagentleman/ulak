package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"
	"ulak/internal/config"
	"ulak/internal/models"
	"ulak/internal/store/postgres"

	"github.com/google/uuid"
)

const (
	TotalRecords = 100000
	BatchSize    = 1000
)

var (
	channels = []models.MessageChannel{
		models.ChannelEmail,
		models.ChannelSMS,
		models.ChannelPush,
		models.ChannelWebhook,
	}

	statuses = []models.MessageSendingStatus{
		models.StatusPending,
		models.StatusSent,
		models.StatusFailed,
	}

	emailTemplates = []string{
		"Welcome to Ulak! Your account has been created successfully.",
		"Password reset requested for your account.",
		"Your verification code is: %d",
		"Important security update for your account.",
		"Monthly newsletter - Stay updated with latest features.",
		"Payment confirmation for order #%d",
		"Shipping notification - Your order is on the way.",
		"Account activity detected from new device.",
		"Subscription renewal reminder.",
		"Thank you for your feedback!",
	}

	smsTemplates = []string{
		"Your verification code is: %d",
		"Login attempt detected from new device.",
		"Your order #%d has been shipped.",
		"Payment of $%.2f received. Thank you!",
		"Appointment reminder for tomorrow at 2PM.",
		"Your package will arrive today.",
		"Security alert: Password changed successfully.",
		"Welcome! Your account is now active.",
	}

	pushTemplates = []string{
		"New message from support team",
		"Your order has been delivered!",
		"Flash sale: 50%% off for next 2 hours",
		"Breaking news update",
		"Friend request from %s",
		"Comment on your post",
		"Weekly summary is ready",
		"Don't forget to check your daily rewards",
	}

	webhookTemplates = []string{
		"Order created: %s",
		"Payment processed: %s",
		"User registered: %s",
		"Shipment updated: %s",
		"Inventory alert: %s",
		"System health check: %s",
		"Backup completed: %s",
		"Security scan finished: %s",
	}
)

func main() {
	log.Println("Starting database data_faker for 100K records...")

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

	pgStore, ok := msgStore.(*postgres.PostgresStore)
	if !ok {
		log.Fatal("Expected PostgresStore implementation")
	}

	rand.Seed(time.Now().UnixNano())

	startTime := time.Now()
	totalBatches := TotalRecords / BatchSize

	log.Printf("Generating %d records in %d batches of %d...\n", TotalRecords, totalBatches, BatchSize)

	for batch := 0; batch < totalBatches; batch++ {
		messages := make([]*models.Message, BatchSize)

		for i := 0; i < BatchSize; i++ {
			messages[i] = generateRandomMessage()
		}

		if err := pgStore.GetDB().WithContext(ctx).CreateInBatches(messages, BatchSize).Error; err != nil {
			log.Printf("Failed to create batch %d: %v\n", batch+1, err)
			continue
		}

		if (batch+1)%10 == 0 {
			elapsed := time.Since(startTime)
			recordsCreated := (batch + 1) * BatchSize
			rate := float64(recordsCreated) / elapsed.Seconds()
			log.Printf("Progress: %d/%d batches (%d records) - %.0f records/sec\n",
				batch+1, totalBatches, recordsCreated, rate)
		}
	}

	elapsed := time.Since(startTime)
	rate := float64(TotalRecords) / elapsed.Seconds()

	log.Printf("\nDatabase population completed successfully!")
	log.Printf("Total records: %d", TotalRecords)
	log.Printf("Total time: %s", elapsed)
	log.Printf("Average rate: %.0f records/sec", rate)
}

func generateRandomMessage() *models.Message {
	channel := channels[rand.Intn(len(channels))]
	status := statuses[rand.Intn(len(statuses))]

	msg := &models.Message{
		ID:       uuid.New(),
		Channel:  channel,
		Status:   status,
		IsActive: true,
	}

	// Generate appropriate To and Message based on channel
	switch channel {
	case models.ChannelEmail:
		msg.To = fmt.Sprintf("user%d@example.com", rand.Intn(10000))
		template := emailTemplates[rand.Intn(len(emailTemplates))]
		msg.Message = fmt.Sprintf(template, rand.Intn(999999))

	case models.ChannelSMS:
		msg.To = fmt.Sprintf("+1%d", 2000000000+rand.Intn(899999999))
		template := smsTemplates[rand.Intn(len(smsTemplates))]
		msg.Message = fmt.Sprintf(template, rand.Intn(999999))

	case models.ChannelPush:
		msg.To = fmt.Sprintf("device-token-%s", uuid.New().String()[:8])
		template := pushTemplates[rand.Intn(len(pushTemplates))]
		msg.Message = fmt.Sprintf(template, fmt.Sprintf("User%d", rand.Intn(1000)))

	case models.ChannelWebhook:
		webhookID := rand.Intn(100)
		msg.To = fmt.Sprintf("https://webhook.example.com/notify/%d", webhookID)
		template := webhookTemplates[rand.Intn(len(webhookTemplates))]
		msg.Message = fmt.Sprintf(template, uuid.New().String()[:8])
	}

	return msg
}
