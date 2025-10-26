package processor

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"
	"ulak/internal/config"
	"ulak/internal/models"
	"ulak/internal/store/message"
	testutil "ulak/internal/testing"
	"ulak/pkg/notification"

	"github.com/google/uuid"
)

func TestNewMessageProcessor(t *testing.T) {
	mockKVStore := &testutil.MockKeyValueStore{}
	mockMsgStore := &testutil.MockMessageStore{}
	mockNotificationService := &testutil.MockNotificationService{}

	cfg := config.MessageWorker{
		RateLimitPerMinute: 60,
		RateLimitBurst:     10,
	}

	processor := NewMessageProcessor(&cfg, mockKVStore, mockMsgStore, mockNotificationService)

	if processor == nil {
		t.Fatal("Expected non-nil processor")
	}

	if processor.cfg.RateLimitPerMinute != 60 {
		t.Errorf("Expected rate limit 60, got %d", processor.cfg.RateLimitPerMinute)
	}
}

func TestMessageProcessorWorker_GetMetrics(t *testing.T) {
	processor := &MessageProcessorWorker{}

	// Initial metrics should be 0
	processed, failed := processor.GetMetrics()
	if processed != 0 || failed != 0 {
		t.Errorf("Expected initial metrics (0, 0), got (%d, %d)", processed, failed)
	}

	// Increment metrics
	processor.processedCount.Add(10)
	processor.failedCount.Add(3)

	processed, failed = processor.GetMetrics()
	if processed != 10 || failed != 3 {
		t.Errorf("Expected metrics (10, 3), got (%d, %d)", processed, failed)
	}
}

func TestMessageProcessorWorker_ProcessMessage_Success(t *testing.T) {
	testMsg := testutil.NewTestMessage(1)
	testMsg.Status = models.StatusPending

	mockTx := &testutil.MockTransaction{}
	mockTx.UpdateFunc = func(ctx context.Context, message *models.Message) error {
		if message.Status != models.StatusSent {
			t.Errorf("Expected status SENT, got %s", message.Status)
		}
		return nil
	}

	mockMsgStore := &testutil.MockMessageStore{
		BeginTxFunc: func(ctx context.Context) (message.Transaction, error) {
			return mockTx, nil
		},
	}

	mockKVStore := &testutil.MockKeyValueStore{
		SetMessageDeliveredFunc: func(ctx context.Context, messageId, value string, ttl int) error {
			// Verify JSON structure
			var data map[string]interface{}
			if err := json.Unmarshal([]byte(value), &data); err != nil {
				t.Errorf("Failed to unmarshal Redis data: %v", err)
			}
			return nil
		},
	}

	mockNotificationService := &testutil.MockNotificationService{
		SendNotificationFunc: func(ctx context.Context, input notification.Input) (notification.AcknowledgeResponse, error) {
			return notification.AcknowledgeResponse{
				Message:   "Success",
				MessageId: uuid.New(),
			}, nil
		},
	}

	cfg := config.MessageWorker{
		RedisMessageTTLSeconds: 86400,
	}

	processor := &MessageProcessorWorker{
		cfg:                 &cfg,
		kvStore:             mockKVStore,
		msgStore:            mockMsgStore,
		notificationService: mockNotificationService,
	}

	err := processor.processMessage(context.Background(), testMsg)
	if err != nil {
		t.Errorf("Expected successful processing, got error: %v", err)
	}

	// Verify transaction was committed
	if !mockTx.IsCommitted() {
		t.Error("Expected transaction to be committed")
	}
}

func TestMessageProcessorWorker_ProcessMessage_NotificationFailure(t *testing.T) {
	testMsg := testutil.NewTestMessage(1)

	rolledBack := false
	mockTx := &testutil.MockTransaction{
		RollbackFunc: func() error {
			rolledBack = true
			return nil
		},
	}

	mockMsgStore := &testutil.MockMessageStore{
		BeginTxFunc: func(ctx context.Context) (message.Transaction, error) {
			return mockTx, nil
		},
		UpdateFunc: func(ctx context.Context, message *models.Message) error {
			// Should update to FAILED status
			if message.Status != models.StatusFailed {
				t.Errorf("Expected status FAILED, got %s", message.Status)
			}
			return nil
		},
	}

	mockKVStore := &testutil.MockKeyValueStore{}

	mockNotificationService := &testutil.MockNotificationService{
		SendNotificationFunc: func(ctx context.Context, input notification.Input) (notification.AcknowledgeResponse, error) {
			return notification.AcknowledgeResponse{}, errors.New("notification service unavailable")
		},
	}

	cfg := config.MessageWorker{}
	processor := &MessageProcessorWorker{
		cfg:                 &cfg,
		kvStore:             mockKVStore,
		msgStore:            mockMsgStore,
		notificationService: mockNotificationService,
	}

	err := processor.processMessage(context.Background(), testMsg)
	if err == nil {
		t.Error("Expected error when notification fails")
	}

	if !rolledBack {
		t.Error("Expected transaction to be rolled back on notification failure")
	}
}

func TestMessageProcessorWorker_ProcessMessage_TransactionBeginFailure(t *testing.T) {
	testMsg := testutil.NewTestMessage(1)

	mockMsgStore := &testutil.MockMessageStore{
		BeginTxFunc: func(ctx context.Context) (message.Transaction, error) {
			return nil, errors.New("failed to begin transaction")
		},
	}

	mockKVStore := &testutil.MockKeyValueStore{}
	mockNotificationService := &testutil.MockNotificationService{}

	processor := &MessageProcessorWorker{
		kvStore:             mockKVStore,
		msgStore:            mockMsgStore,
		notificationService: mockNotificationService,
	}

	err := processor.processMessage(context.Background(), testMsg)
	if err == nil {
		t.Error("Expected error when transaction begin fails")
	}
}

func TestMessageProcessorWorker_ProcessMessage_UpdateFailure(t *testing.T) {
	testMsg := testutil.NewTestMessage(1)

	rolledBack := false
	mockTx := &testutil.MockTransaction{
		UpdateFunc: func(ctx context.Context, message *models.Message) error {
			return errors.New("update failed")
		},
		RollbackFunc: func() error {
			rolledBack = true
			return nil
		},
	}

	mockMsgStore := &testutil.MockMessageStore{
		BeginTxFunc: func(ctx context.Context) (message.Transaction, error) {
			return mockTx, nil
		},
	}

	mockKVStore := &testutil.MockKeyValueStore{}

	mockNotificationService := &testutil.MockNotificationService{
		SendNotificationFunc: func(ctx context.Context, input notification.Input) (notification.AcknowledgeResponse, error) {
			return notification.AcknowledgeResponse{Message: "OK", MessageId: uuid.New()}, nil
		},
	}

	processor := &MessageProcessorWorker{
		kvStore:             mockKVStore,
		msgStore:            mockMsgStore,
		notificationService: mockNotificationService,
	}

	err := processor.processMessage(context.Background(), testMsg)
	if err == nil {
		t.Error("Expected error when update fails")
	}

	if !rolledBack {
		t.Error("Expected transaction to be rolled back on update failure")
	}
}

func TestMessageProcessorWorker_ProcessMessage_RedisFailure(t *testing.T) {
	testMsg := testutil.NewTestMessage(1)

	rolledBack := false
	mockTx := &testutil.MockTransaction{
		UpdateFunc: func(ctx context.Context, message *models.Message) error {
			return nil
		},
		RollbackFunc: func() error {
			rolledBack = true
			return nil
		},
	}

	mockMsgStore := &testutil.MockMessageStore{
		BeginTxFunc: func(ctx context.Context) (message.Transaction, error) {
			return mockTx, nil
		},
		UpdateFunc: func(ctx context.Context, message *models.Message) error {
			if message.Status != models.StatusFailed {
				t.Errorf("Expected status FAILED, got %s", message.Status)
			}
			return nil
		},
	}

	mockKVStore := &testutil.MockKeyValueStore{
		SetMessageDeliveredFunc: func(ctx context.Context, messageId, value string, ttl int) error {
			return errors.New("redis connection error")
		},
	}

	mockNotificationService := &testutil.MockNotificationService{
		SendNotificationFunc: func(ctx context.Context, input notification.Input) (notification.AcknowledgeResponse, error) {
			return notification.AcknowledgeResponse{Message: "OK", MessageId: uuid.New()}, nil
		},
	}

	cfg := config.MessageWorker{
		RedisMessageTTLSeconds: 86400,
	}

	processor := &MessageProcessorWorker{
		cfg:                 &cfg,
		kvStore:             mockKVStore,
		msgStore:            mockMsgStore,
		notificationService: mockNotificationService,
	}

	err := processor.processMessage(context.Background(), testMsg)
	if err == nil {
		t.Error("Expected error when Redis fails")
	}

	if !rolledBack {
		t.Error("Expected transaction to be rolled back on Redis failure")
	}
}

func TestMessageProcessorWorker_ProcessMessage_CommitFailure(t *testing.T) {
	testMsg := testutil.NewTestMessage(1)

	mockTx := &testutil.MockTransaction{
		UpdateFunc: func(ctx context.Context, message *models.Message) error {
			return nil
		},
		CommitFunc: func() error {
			return errors.New("commit failed")
		},
	}

	mockMsgStore := &testutil.MockMessageStore{
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
			return notification.AcknowledgeResponse{Message: "OK", MessageId: uuid.New()}, nil
		},
	}

	cfg := config.MessageWorker{
		RedisMessageTTLSeconds: 86400,
	}

	processor := &MessageProcessorWorker{
		cfg:                 &cfg,
		kvStore:             mockKVStore,
		msgStore:            mockMsgStore,
		notificationService: mockNotificationService,
	}

	err := processor.processMessage(context.Background(), testMsg)
	if err == nil {
		t.Error("Expected error when commit fails")
	}
}

func TestMessageProcessorWorker_Process_ChannelClosed(t *testing.T) {
	mockMsgStore := &testutil.MockMessageStore{}
	mockKVStore := &testutil.MockKeyValueStore{}
	mockNotificationService := &testutil.MockNotificationService{}

	cfg := config.MessageWorker{
		RateLimitPerMinute: 60,
		RateLimitBurst:     10,
	}

	processor := &MessageProcessorWorker{
		cfg:                 &cfg,
		kvStore:             mockKVStore,
		msgStore:            mockMsgStore,
		notificationService: mockNotificationService,
	}

	messagesCh := make(chan *models.Message)
	close(messagesCh) // Close immediately

	ctx := context.Background()
	err := processor.Process(ctx, messagesCh)

	// Should exit gracefully when channel is closed
	if err != nil {
		t.Errorf("Expected nil error on closed channel, got: %v", err)
	}
}

func TestMessageProcessorWorker_StartStop(t *testing.T) {
	mockMsgStore := &testutil.MockMessageStore{
		BeginTxFunc: func(ctx context.Context) (message.Transaction, error) {
			return &testutil.MockTransaction{}, nil
		},
	}
	mockKVStore := &testutil.MockKeyValueStore{}
	mockNotificationService := &testutil.MockNotificationService{
		SendNotificationFunc: func(ctx context.Context, input notification.Input) (notification.AcknowledgeResponse, error) {
			return notification.AcknowledgeResponse{Message: "OK", MessageId: uuid.New()}, nil
		},
	}

	cfg := config.MessageWorker{
		RateLimitPerMinute: 60,
		RateLimitBurst:     10,
	}

	processor := NewMessageProcessor(&cfg, mockKVStore, mockMsgStore, mockNotificationService)
	messagesCh := make(chan *models.Message, 10)

	ctx := context.Background()
	wg := &sync.WaitGroup{}
	wg.Add(1)

	// Test Start
	err := processor.Start(ctx, messagesCh, wg)
	if err != nil {
		t.Fatalf("Failed to start processor: %v", err)
	}

	// Give processor time to start
	time.Sleep(50 * time.Millisecond)

	// Verify running
	if !processor.IsRunning() {
		t.Error("Expected processor to be running")
	}

	// Test double start should fail
	err = processor.Start(ctx, messagesCh, wg)
	if err == nil {
		t.Error("Expected error when starting already running worker")
	}

	// Test Stop
	err = processor.Stop(2 * time.Second)
	if err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("Failed to stop processor: %v", err)
	}

	// Verify stopped
	if processor.IsRunning() {
		t.Error("Expected processor to be stopped")
	}

	close(messagesCh)
}
