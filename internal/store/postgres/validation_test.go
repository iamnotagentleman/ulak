package postgres

import (
	"strings"
	"testing"
	"ulak/internal/models"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

func TestMessageValidation(t *testing.T) {
	// Create validator instance
	store := &postgresStore{
		db:       nil, // We don't need DB for validation testing
		validate: nil,
	}

	// Initialize validator like in NewPostgresStore
	store.validate = validator.New()

	t.Run("Valid message with 256 characters", func(t *testing.T) {
		msg := &models.Message{
			ID:      uuid.New(),
			To:      "test@example.com",
			Message: strings.Repeat("a", 256), // exactly 256 characters
			Channel: models.ChannelEmail,
			Status:  models.StatusPending,
		}

		// We only test validation, not DB insertion
		err := store.validate.Struct(msg)
		if err != nil {
			t.Errorf("Expected no validation error for 256 character message, got: %v", err)
		}
	})

	t.Run("Invalid message with 257 characters", func(t *testing.T) {
		msg := &models.Message{
			ID:      uuid.New(),
			To:      "test@example.com",
			Message: strings.Repeat("a", 257), // 257 characters - should fail
			Channel: models.ChannelEmail,
			Status:  models.StatusPending,
		}

		err := store.validate.Struct(msg)
		if err == nil {
			t.Error("Expected validation error for 257 character message, but got none")
		}
	})

	t.Run("Valid short message", func(t *testing.T) {
		msg := &models.Message{
			ID:      uuid.New(),
			To:      "test@example.com",
			Message: "Hello, World!",
			Channel: models.ChannelEmail,
			Status:  models.StatusPending,
		}

		err := store.validate.Struct(msg)
		if err != nil {
			t.Errorf("Expected no validation error for short message, got: %v", err)
		}
	})

	t.Run("Empty message should be valid", func(t *testing.T) {
		msg := &models.Message{
			ID:      uuid.New(),
			To:      "test@example.com",
			Message: "",
			Channel: models.ChannelEmail,
			Status:  models.StatusPending,
		}

		err := store.validate.Struct(msg)
		if err != nil {
			t.Errorf("Expected no validation error for empty message, got: %v", err)
		}
	})
}
