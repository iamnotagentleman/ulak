package testing

import (
	"time"
	"ulak/internal/models"

	"github.com/google/uuid"
)

// NewTestMessage creates a test message with default values
func NewTestMessage(offset int64) *models.Message {
	return &models.Message{
		ID:        uuid.New(),
		To:        "test@example.com",
		Message:   "Test message content",
		Offset:    offset,
		Channel:   models.ChannelEmail,
		Status:    models.StatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		IsActive:  true,
	}
}

// NewTestMessages creates multiple test messages with sequential offsets
func NewTestMessages(count int, startOffset int64) []*models.Message {
	messages := make([]*models.Message, count)
	for i := 0; i < count; i++ {
		messages[i] = NewTestMessage(startOffset + int64(i))
	}
	return messages
}
