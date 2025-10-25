package manager

import (
	"context"
	"sync"
	"testing"
	"time"
	"ulak/internal/config"
	"ulak/internal/enums"
	"ulak/internal/models"
	"ulak/internal/store/message"
	testutil "ulak/internal/testing"
	"ulak/internal/worker/populator"
	"ulak/internal/worker/processor"
	"ulak/pkg/notification"

	"github.com/google/uuid"
)

func TestNewWorkerManager(t *testing.T) {
	mockMsgStore := &testutil.MockMessageStore{}
	mockKVStore := &testutil.MockKeyValueStore{}
	mockNotificationService := &testutil.MockNotificationService{}

	cfg := config.MessageWorker{
		MaxRecordsPerRead: 10,
	}

	serverCfg := config.Server{
		WorkerStopTimeout: 5 * time.Second,
	}

	messagesCh := make(chan *models.Message, 100)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	populatorWorker := populator.GetMessagePopulator(cfg, mockMsgStore)
	processorWorker := processor.NewMessageProcessor(cfg, mockKVStore, mockMsgStore, mockNotificationService)

	ctx := context.Background()
	manager := NewWorkerManager(ctx, populatorWorker, processorWorker, messagesCh, ticker, serverCfg)

	if manager == nil {
		t.Fatal("Expected non-nil manager")
	}

	if manager.populator == nil {
		t.Error("Expected populator to be set")
	}

	if manager.processor == nil {
		t.Error("Expected processor to be set")
	}

	if manager.messagesCh == nil {
		t.Error("Expected messagesCh to be set")
	}

	if manager.ticker == nil {
		t.Error("Expected ticker to be set")
	}

	close(messagesCh)
}

func TestWorkerManager_Start(t *testing.T) {
	mockMsgStore := &testutil.MockMessageStore{
		ListByOffsetFunc: func(ctx context.Context, offset int64, status enums.MessageSendingStatus, limit int) ([]*models.Message, error) {
			return []*models.Message{}, nil
		},
		BeginTxFunc: func(ctx context.Context) (message.Transaction, error) {
			return &testutil.MockTransaction{}, nil
		},
	}
	mockKVStore := &testutil.MockKeyValueStore{}
	mockNotificationService := &testutil.MockNotificationService{}

	cfg := config.MessageWorker{
		MaxRecordsPerRead:  10,
		RateLimitPerMinute: 60,
		RateLimitBurst:     10,
	}

	serverCfg := config.Server{
		WorkerStopTimeout: 5 * time.Second,
	}

	messagesCh := make(chan *models.Message, 100)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	populatorWorker := populator.GetMessagePopulator(cfg, mockMsgStore)
	processorWorker := processor.NewMessageProcessor(cfg, mockKVStore, mockMsgStore, mockNotificationService)

	ctx := context.Background()
	manager := NewWorkerManager(ctx, populatorWorker, processorWorker, messagesCh, ticker, serverCfg)

	err := manager.Start()
	if err != nil {
		t.Fatalf("Failed to start manager: %v", err)
	}

	// Verify both workers are running
	status := manager.GetStatus()
	if !status.IsPopulatorActive {
		t.Error("Expected populator to be active")
	}
	if !status.IsProcessorActive {
		t.Error("Expected processor to be active")
	}

	// Clean up
	manager.Stop()
	close(messagesCh)
}

func TestWorkerManager_Stop(t *testing.T) {
	mockMsgStore := &testutil.MockMessageStore{
		ListByOffsetFunc: func(ctx context.Context, offset int64, status enums.MessageSendingStatus, limit int) ([]*models.Message, error) {
			return []*models.Message{}, nil
		},
		BeginTxFunc: func(ctx context.Context) (message.Transaction, error) {
			return &testutil.MockTransaction{}, nil
		},
	}
	mockKVStore := &testutil.MockKeyValueStore{}
	mockNotificationService := &testutil.MockNotificationService{}

	cfg := config.MessageWorker{
		MaxRecordsPerRead:  10,
		RateLimitPerMinute: 60,
		RateLimitBurst:     10,
	}

	serverCfg := config.Server{
		WorkerStopTimeout: 5 * time.Second,
	}

	messagesCh := make(chan *models.Message, 100)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	populatorWorker := populator.GetMessagePopulator(cfg, mockMsgStore)
	processorWorker := processor.NewMessageProcessor(cfg, mockKVStore, mockMsgStore, mockNotificationService)

	ctx := context.Background()
	manager := NewWorkerManager(ctx, populatorWorker, processorWorker, messagesCh, ticker, serverCfg)

	// Start workers
	err := manager.Start()
	if err != nil {
		t.Fatalf("Failed to start manager: %v", err)
	}

	// Stop workers
	manager.Stop()

	// Verify both workers are stopped
	status := manager.GetStatus()
	if status.IsPopulatorActive {
		t.Error("Expected populator to be stopped")
	}
	if status.IsProcessorActive {
		t.Error("Expected processor to be stopped")
	}

	close(messagesCh)
}

func TestWorkerManager_GetStatus(t *testing.T) {
	mockMsgStore := &testutil.MockMessageStore{
		ListByOffsetFunc: func(ctx context.Context, offset int64, status enums.MessageSendingStatus, limit int) ([]*models.Message, error) {
			return []*models.Message{}, nil
		},
		BeginTxFunc: func(ctx context.Context) (message.Transaction, error) {
			return &testutil.MockTransaction{}, nil
		},
	}
	mockKVStore := &testutil.MockKeyValueStore{}
	mockNotificationService := &testutil.MockNotificationService{}

	cfg := config.MessageWorker{
		MaxRecordsPerRead:  10,
		RateLimitPerMinute: 60,
		RateLimitBurst:     10,
	}

	serverCfg := config.Server{
		WorkerStopTimeout: 5 * time.Second,
	}

	messagesCh := make(chan *models.Message, 100)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	populatorWorker := populator.GetMessagePopulator(cfg, mockMsgStore)
	processorWorker := processor.NewMessageProcessor(cfg, mockKVStore, mockMsgStore, mockNotificationService)

	ctx := context.Background()
	manager := NewWorkerManager(ctx, populatorWorker, processorWorker, messagesCh, ticker, serverCfg)

	// Initial status - both stopped
	status := manager.GetStatus()
	if status.IsPopulatorActive || status.IsProcessorActive {
		t.Error("Expected both workers to be inactive initially")
	}

	// Start and check status
	manager.Start()
	status = manager.GetStatus()
	if !status.IsPopulatorActive || !status.IsProcessorActive {
		t.Error("Expected both workers to be active after start")
	}

	// Stop and check status
	manager.Stop()
	status = manager.GetStatus()
	if status.IsPopulatorActive || status.IsProcessorActive {
		t.Error("Expected both workers to be inactive after stop")
	}

	close(messagesCh)
}

func TestWorkerManager_GracefulShutdown(t *testing.T) {
	// Test that manager properly coordinates graceful shutdown
	processedMessages := 0
	var mu sync.Mutex

	mockTx := &testutil.MockTransaction{}

	mockMsgStore := &testutil.MockMessageStore{
		ListByOffsetFunc: func(ctx context.Context, offset int64, status enums.MessageSendingStatus, limit int) ([]*models.Message, error) {
			// Return some messages
			return testutil.NewTestMessages(2, offset), nil
		},
		BeginTxFunc: func(ctx context.Context) (message.Transaction, error) {
			return mockTx, nil
		},
	}

	mockKVStore := &testutil.MockKeyValueStore{
		SetMessageDeliveredFunc: func(ctx context.Context, messageId, value string, ttl int) error {
			return nil
		},
	}

	mockNotificationService := &testutil.MockNotificationService{
		SendNotificationFunc: func(ctx context.Context, input notification.Input) (notification.AcknowledgeResponse, error) {
			mu.Lock()
			processedMessages++
			mu.Unlock()
			time.Sleep(10 * time.Millisecond) // Simulate work
			return notification.AcknowledgeResponse{Message: "OK", MessageId: uuid.New()}, nil
		},
	}

	cfg := config.MessageWorker{
		MaxRecordsPerRead:         10,
		RateLimitPerMinute:        600,
		RateLimitBurst:            100,
		MessageChannelSendTimeout: 5,
		RedisMessageTTLSeconds:    86400,
	}

	serverCfg := config.Server{
		WorkerStopTimeout: 2 * time.Second,
	}

	messagesCh := make(chan *models.Message, 100)
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	populatorWorker := populator.GetMessagePopulator(cfg, mockMsgStore)
	processorWorker := processor.NewMessageProcessor(cfg, mockKVStore, mockMsgStore, mockNotificationService)

	ctx := context.Background()
	manager := NewWorkerManager(ctx, populatorWorker, processorWorker, messagesCh, ticker, serverCfg)

	// Start workers
	err := manager.Start()
	if err != nil {
		t.Fatalf("Failed to start manager: %v", err)
	}

	// Let them run for a bit
	time.Sleep(200 * time.Millisecond)

	// Stop workers
	manager.Stop()

	// Verify workers are stopped
	status := manager.GetStatus()
	if status.IsPopulatorActive {
		t.Error("Expected populator to be stopped after graceful shutdown")
	}
	if status.IsProcessorActive {
		t.Error("Expected processor to be stopped after graceful shutdown")
	}

	// Verify some messages were processed
	mu.Lock()
	count := processedMessages
	mu.Unlock()

	if count == 0 {
		t.Error("Expected at least some messages to be processed before shutdown")
	}

	t.Logf("Processed %d messages before shutdown", count)

	close(messagesCh)
}

func TestWorkerManager_StartAlreadyRunning(t *testing.T) {
	mockMsgStore := &testutil.MockMessageStore{
		ListByOffsetFunc: func(ctx context.Context, offset int64, status enums.MessageSendingStatus, limit int) ([]*models.Message, error) {
			return []*models.Message{}, nil
		},
		BeginTxFunc: func(ctx context.Context) (message.Transaction, error) {
			return &testutil.MockTransaction{}, nil
		},
	}
	mockKVStore := &testutil.MockKeyValueStore{}
	mockNotificationService := &testutil.MockNotificationService{}

	cfg := config.MessageWorker{
		MaxRecordsPerRead:  10,
		RateLimitPerMinute: 60,
		RateLimitBurst:     10,
	}

	serverCfg := config.Server{
		WorkerStopTimeout: 5 * time.Second,
	}

	messagesCh := make(chan *models.Message, 100)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	populatorWorker := populator.GetMessagePopulator(cfg, mockMsgStore)
	processorWorker := processor.NewMessageProcessor(cfg, mockKVStore, mockMsgStore, mockNotificationService)

	ctx := context.Background()
	manager := NewWorkerManager(ctx, populatorWorker, processorWorker, messagesCh, ticker, serverCfg)

	// First start should succeed
	err := manager.Start()
	if err != nil {
		t.Fatalf("First start failed: %v", err)
	}

	// Second start should fail
	err = manager.Start()
	if err == nil {
		t.Error("Expected error when starting already running workers")
	}

	// Clean up
	manager.Stop()
	close(messagesCh)
}
