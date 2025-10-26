package populator

import (
	"context"
	"sync"
	"testing"
	"time"
	"ulak/internal/config"
	"ulak/internal/models"
	testutil "ulak/internal/testing"

	"github.com/google/uuid"
)

func resetSingleton() {
	populatorInstance = nil
	once = sync.Once{}
}

func TestGetMessagePopulator_Singleton(t *testing.T) {
	resetSingleton()
	defer resetSingleton()

	mockStore := &testutil.MockMessageStore{}
	cfg := config.MessageWorker{
		MaxRecordsPerRead:         10,
		MessageChannelSendTimeout: 5,
	}

	// Get first instance
	instance1 := GetMessagePopulator(cfg, mockStore)
	if instance1 == nil {
		t.Fatal("Expected non-nil populator instance")
	}

	// Get second instance - should be same
	instance2 := GetMessagePopulator(cfg, mockStore)
	if instance1 != instance2 {
		t.Error("Expected singleton pattern to return same instance")
	}
}

func TestMessagePopulatorWorker_GetOffset(t *testing.T) {
	resetSingleton()
	defer resetSingleton()

	mockStore := &testutil.MockMessageStore{}
	cfg := config.MessageWorker{MaxRecordsPerRead: 10}

	populator := GetMessagePopulator(cfg, mockStore)

	// Initial offset should be 0
	if offset := populator.GetOffset(); offset != 0 {
		t.Errorf("Expected initial offset 0, got %d", offset)
	}

	// Manually set offset
	populator.lastProcessedOffset.Store(42)
	if offset := populator.GetOffset(); offset != 42 {
		t.Errorf("Expected offset 42, got %d", offset)
	}
}

func TestMessagePopulatorWorker_SendMessage(t *testing.T) {
	resetSingleton()
	defer resetSingleton()

	tests := []struct {
		name        string
		channelSize int
		timeout     int
		preFillCh   bool
		expectError bool
	}{
		{
			name:        "successful send to available channel",
			channelSize: 10,
			timeout:     5,
			preFillCh:   false,
			expectError: false,
		},
		{
			name:        "timeout when channel is full",
			channelSize: 1,
			timeout:     1,
			preFillCh:   true,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.MessageWorker{
				MessageChannelSendTimeout: tt.timeout,
			}
			populator := &MessagePopulatorWorker{cfg: cfg}

			messagesCh := make(chan *models.Message, tt.channelSize)
			if tt.preFillCh {
				// Fill the channel
				for i := 0; i < tt.channelSize; i++ {
					messagesCh <- testutil.NewTestMessage(int64(i))
				}
			}

			ctx := context.Background()
			testMsg := testutil.NewTestMessage(999)

			err := populator.sendMessage(ctx, messagesCh, testMsg)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			// Cleanup
			close(messagesCh)
		})
	}
}

func TestMessagePopulatorWorker_Populate_OffsetTracking(t *testing.T) {
	resetSingleton()
	defer resetSingleton()

	messages := []*models.Message{
		{ID: uuid.New(), Offset: 10, Message: "msg1"},
		{ID: uuid.New(), Offset: 11, Message: "msg2"},
		{ID: uuid.New(), Offset: 12, Message: "msg3"},
	}

	mockStore := &testutil.MockMessageStore{
		ListByOffsetFunc: func(ctx context.Context, offset int64, status models.MessageSendingStatus, limit int) ([]*models.Message, error) {
			return messages, nil
		},
	}

	cfg := config.MessageWorker{
		MaxRecordsPerRead:         10,
		MessageChannelSendTimeout: 5,
	}

	populator := &MessagePopulatorWorker{
		cfg:   cfg,
		store: mockStore,
	}

	messagesCh := make(chan *models.Message, 100)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	go populator.populate(ctx, ticker, messagesCh)

	// Wait for processing
	time.Sleep(30 * time.Millisecond)

	// Check offset was updated to last message's offset
	finalOffset := populator.GetOffset()
	expectedOffset := int64(12) // Last message offset

	if finalOffset != expectedOffset {
		t.Errorf("Expected offset %d, got %d", expectedOffset, finalOffset)
	}

	close(messagesCh)
}

func TestMessagePopulatorWorker_StartStop(t *testing.T) {
	resetSingleton()
	defer resetSingleton()

	mockStore := &testutil.MockMessageStore{
		ListByOffsetFunc: func(ctx context.Context, offset int64, status models.MessageSendingStatus, limit int) ([]*models.Message, error) {
			return []*models.Message{}, nil
		},
	}

	cfg := config.MessageWorker{
		MaxRecordsPerRead:         10,
		MessageChannelSendTimeout: 5,
	}

	populator := GetMessagePopulator(cfg, mockStore)
	messagesCh := make(chan *models.Message, 10)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	ctx := context.Background()

	// Test Start
	err := populator.Start(ctx, ticker, messagesCh)
	if err != nil {
		t.Fatalf("Failed to start populator: %v", err)
	}

	// Verify running
	if !populator.IsRunning() {
		t.Error("Expected populator to be running")
	}

	// Test double start should fail
	err = populator.Start(ctx, ticker, messagesCh)
	if err == nil {
		t.Error("Expected error when starting already running worker")
	}

	// Test Stop
	err = populator.Stop(2 * time.Second)
	if err != nil {
		t.Fatalf("Failed to stop populator: %v", err)
	}

	// Verify stopped
	if populator.IsRunning() {
		t.Error("Expected populator to be stopped")
	}

	close(messagesCh)
}
