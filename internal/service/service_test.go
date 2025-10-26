package service

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
	"ulak/internal/config"
	"ulak/internal/manager"
	"ulak/internal/models"
	"ulak/internal/store/message"
	testutil "ulak/internal/testing"
	"ulak/internal/worker/populator"
	"ulak/internal/worker/processor"
)

// Helper to create a test worker manager
func createTestWorkerManager(t *testing.T) *manager.WorkerManager {
	mockMsgStore := &testutil.MockMessageStore{
		ListByOffsetFunc: func(ctx context.Context, offset int64, status models.MessageSendingStatus, limit int) ([]*models.Message, error) {
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
		WorkerStopTimeout: 1 * time.Second,
	}

	messagesCh := make(chan *models.Message, 10)
	ticker := time.NewTicker(100 * time.Millisecond)
	t.Cleanup(func() {
		ticker.Stop()
		close(messagesCh)
	})

	populatorWorker := populator.GetMessagePopulator(&cfg, mockMsgStore)
	processorWorker := processor.NewMessageProcessor(&cfg, mockKVStore, mockMsgStore, mockNotificationService)

	ctx := context.Background()
	return manager.NewWorkerManager(ctx, populatorWorker, processorWorker, messagesCh, ticker, &serverCfg)
}

func TestNewService(t *testing.T) {
	mockKVStore := &testutil.MockKeyValueStore{}
	mockMsgStore := &testutil.MockMessageStore{}
	mockManager := createTestWorkerManager(t)

	svc := NewService(mockKVStore, mockMsgStore, mockManager)
	if svc == nil {
		t.Fatal("Expected non-nil service")
	}
}

func TestService_SetMessageAutoSend_Start(t *testing.T) {
	tests := []struct {
		name                string
		initialStatus       manager.WorkerStatusResponse
		expectedError       bool
		expectedPopulatorOn bool
		expectedProcessorOn bool
		expectedHTTPCode    int
	}{
		{
			name: "start when workers are stopped",
			initialStatus: manager.WorkerStatusResponse{
				IsPopulatorActive: false,
				IsProcessorActive: false,
			},
			expectedError:       false,
			expectedPopulatorOn: true,
			expectedProcessorOn: true,
			expectedHTTPCode:    0,
		},
		{
			name: "start when workers are already running",
			initialStatus: manager.WorkerStatusResponse{
				IsPopulatorActive: true,
				IsProcessorActive: true,
			},
			expectedError:       true,
			expectedPopulatorOn: true,
			expectedProcessorOn: true,
			expectedHTTPCode:    http.StatusExpectationFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockKVStore := &testutil.MockKeyValueStore{}
			mockMsgStore := &testutil.MockMessageStore{}
			mockManager := createTestWorkerManager(t)

			// Set initial state
			if tt.initialStatus.IsPopulatorActive && tt.initialStatus.IsProcessorActive {
				mockManager.Start()
			}
			t.Cleanup(func() {
				mockManager.Stop()
			})

			svc := NewService(mockKVStore, mockMsgStore, mockManager)

			req := models.SetMessageAutoSendRequest{
				Action: models.AutoSendStart,
			}

			resp := svc.SetMessageAutoSend(context.Background(), req)

			if tt.expectedError {
				if resp.Result == nil {
					t.Error("Expected error result but got nil")
				} else if resp.Result.Code != tt.expectedHTTPCode {
					t.Errorf("Expected HTTP code %d, got %d", tt.expectedHTTPCode, resp.Result.Code)
				}
			} else {
				if resp.Result != nil {
					t.Errorf("Expected nil error but got: %v", resp.Result)
				}
				if resp.Data == nil {
					t.Fatal("Expected data but got nil")
				}
				if resp.Data.IsPopulatorEnabled != tt.expectedPopulatorOn {
					t.Errorf("Expected populator=%v, got %v", tt.expectedPopulatorOn, resp.Data.IsPopulatorEnabled)
				}
				if resp.Data.IsProcessorEnabled != tt.expectedProcessorOn {
					t.Errorf("Expected processor=%v, got %v", tt.expectedProcessorOn, resp.Data.IsProcessorEnabled)
				}
			}
		})
	}
}

func TestService_SetMessageAutoSend_Stop(t *testing.T) {
	tests := []struct {
		name                string
		initialStatus       manager.WorkerStatusResponse
		expectedError       bool
		expectedPopulatorOn bool
		expectedProcessorOn bool
		expectedHTTPCode    int
	}{
		{
			name: "stop when workers are running",
			initialStatus: manager.WorkerStatusResponse{
				IsPopulatorActive: true,
				IsProcessorActive: true,
			},
			expectedError:       false,
			expectedPopulatorOn: false,
			expectedProcessorOn: false,
			expectedHTTPCode:    0,
		},
		{
			name: "stop when workers are already stopped",
			initialStatus: manager.WorkerStatusResponse{
				IsPopulatorActive: false,
				IsProcessorActive: false,
			},
			expectedError:       true,
			expectedPopulatorOn: false,
			expectedProcessorOn: false,
			expectedHTTPCode:    http.StatusExpectationFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockKVStore := &testutil.MockKeyValueStore{}
			mockMsgStore := &testutil.MockMessageStore{}
			mockManager := createTestWorkerManager(t)

			// Set initial state
			if tt.initialStatus.IsPopulatorActive && tt.initialStatus.IsProcessorActive {
				mockManager.Start()
			}
			t.Cleanup(func() {
				mockManager.Stop()
			})

			svc := NewService(mockKVStore, mockMsgStore, mockManager)

			req := models.SetMessageAutoSendRequest{
				Action: models.AutoSendStop,
			}

			resp := svc.SetMessageAutoSend(context.Background(), req)

			if tt.expectedError {
				if resp.Result == nil {
					t.Error("Expected error result but got nil")
				} else if resp.Result.Code != tt.expectedHTTPCode {
					t.Errorf("Expected HTTP code %d, got %d", tt.expectedHTTPCode, resp.Result.Code)
				}
			} else {
				if resp.Result != nil {
					t.Errorf("Expected nil error but got: %v", resp.Result)
				}
				if resp.Data == nil {
					t.Fatal("Expected data but got nil")
				}
				if resp.Data.IsPopulatorEnabled != tt.expectedPopulatorOn {
					t.Errorf("Expected populator=%v, got %v", tt.expectedPopulatorOn, resp.Data.IsPopulatorEnabled)
				}
				if resp.Data.IsProcessorEnabled != tt.expectedProcessorOn {
					t.Errorf("Expected processor=%v, got %v", tt.expectedProcessorOn, resp.Data.IsProcessorEnabled)
				}
			}
		})
	}
}

func TestService_SetMessageAutoSend_InvalidAction(t *testing.T) {
	mockKVStore := &testutil.MockKeyValueStore{}
	mockMsgStore := &testutil.MockMessageStore{}
	mockManager := createTestWorkerManager(t)
	t.Cleanup(func() {
		mockManager.Stop()
	})

	svc := NewService(mockKVStore, mockMsgStore, mockManager)

	req := models.SetMessageAutoSendRequest{
		Action: "invalid_action",
	}

	resp := svc.SetMessageAutoSend(context.Background(), req)

	if resp.Result == nil {
		t.Fatal("Expected error result but got nil")
	}

	if resp.Result.Code != http.StatusForbidden {
		t.Errorf("Expected HTTP 403 for invalid action, got %d", resp.Result.Code)
	}

	if resp.Data != nil {
		t.Error("Expected nil data for invalid action")
	}
}

func TestService_GetSentMessages_Success(t *testing.T) {
	tests := []struct {
		name           string
		requestLimit   int
		requestOffset  int
		expectedLimit  int
		expectedOffset int
	}{
		{
			name:           "default pagination",
			requestLimit:   0,
			requestOffset:  0,
			expectedLimit:  10,
			expectedOffset: 0,
		},
		{
			name:           "custom pagination",
			requestLimit:   20,
			requestOffset:  5,
			expectedLimit:  20,
			expectedOffset: 5,
		},
		{
			name:           "negative offset should be clamped to 0",
			requestLimit:   10,
			requestOffset:  -5,
			expectedLimit:  10,
			expectedOffset: 0,
		},
		{
			name:           "negative limit should use default",
			requestLimit:   -10,
			requestOffset:  0,
			expectedLimit:  10,
			expectedOffset: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			messages := testutil.NewTestMessages(5, 100)

			mockMsgStore := &testutil.MockMessageStore{
				ListFunc: func(ctx context.Context, limit, skip int, status models.MessageSendingStatus) ([]*models.Message, error) {
					if limit != tt.expectedLimit {
						t.Errorf("Expected limit %d, got %d", tt.expectedLimit, limit)
					}
					if skip != tt.expectedOffset {
						t.Errorf("Expected offset %d, got %d", tt.expectedOffset, skip)
					}
					if status != models.StatusSent {
						t.Errorf("Expected status SENT, got %s", status)
					}
					return messages, nil
				},
				GetTotalCountFunc: func(ctx context.Context) (int64, error) {
					return 100, nil
				},
			}

			mockKVStore := &testutil.MockKeyValueStore{}
			mockManager := createTestWorkerManager(t)
			t.Cleanup(func() {
				mockManager.Stop()
			})

			svc := NewService(mockKVStore, mockMsgStore, mockManager)

			req := models.GetMessagesRequest{
				Limit:  tt.requestLimit,
				Offset: tt.requestOffset,
			}

			resp := svc.GetSentMessages(context.Background(), req)

			if resp.Result != nil {
				t.Errorf("Expected nil error but got: %v", resp.Result)
			}

			if resp.Data == nil {
				t.Fatal("Expected data but got nil")
			}

			if len(resp.Data.Messages) != 5 {
				t.Errorf("Expected 5 messages, got %d", len(resp.Data.Messages))
			}

			if resp.Data.TotalCount != 100 {
				t.Errorf("Expected total count 100, got %d", resp.Data.TotalCount)
			}

			if resp.Data.Limit != tt.expectedLimit {
				t.Errorf("Expected limit %d, got %d", tt.expectedLimit, resp.Data.Limit)
			}

			if resp.Data.Offset != tt.expectedOffset {
				t.Errorf("Expected offset %d, got %d", tt.expectedOffset, resp.Data.Offset)
			}
		})
	}
}

func TestService_GetSentMessages_DatabaseError(t *testing.T) {
	mockMsgStore := &testutil.MockMessageStore{
		ListFunc: func(ctx context.Context, limit, skip int, status models.MessageSendingStatus) ([]*models.Message, error) {
			return nil, errors.New("database connection error")
		},
	}

	mockKVStore := &testutil.MockKeyValueStore{}
	mockManager := createTestWorkerManager(t)
	t.Cleanup(func() {
		mockManager.Stop()
	})

	svc := NewService(mockKVStore, mockMsgStore, mockManager)

	req := models.GetMessagesRequest{
		Limit:  10,
		Offset: 0,
	}

	resp := svc.GetSentMessages(context.Background(), req)

	if resp.Result == nil {
		t.Fatal("Expected error result but got nil")
	}

	if resp.Result.Code != http.StatusInternalServerError {
		t.Errorf("Expected HTTP 500, got %d", resp.Result.Code)
	}

	if resp.Data != nil {
		t.Error("Expected nil data on error")
	}
}

func TestService_GetSentMessages_TotalCountError(t *testing.T) {
	messages := testutil.NewTestMessages(5, 100)

	mockMsgStore := &testutil.MockMessageStore{
		ListFunc: func(ctx context.Context, limit, skip int, status models.MessageSendingStatus) ([]*models.Message, error) {
			return messages, nil
		},
		GetTotalCountFunc: func(ctx context.Context) (int64, error) {
			return 0, errors.New("count query failed")
		},
	}

	mockKVStore := &testutil.MockKeyValueStore{}
	mockManager := createTestWorkerManager(t)
	t.Cleanup(func() {
		mockManager.Stop()
	})

	svc := NewService(mockKVStore, mockMsgStore, mockManager)

	req := models.GetMessagesRequest{
		Limit:  10,
		Offset: 0,
	}

	resp := svc.GetSentMessages(context.Background(), req)

	if resp.Result != nil {
		t.Errorf("Expected nil error but got: %v", resp.Result)
	}

	if resp.Data == nil {
		t.Fatal("Expected data but got nil")
	}

	if resp.Data.TotalCount != 0 {
		t.Errorf("Expected total count 0 on error, got %d", resp.Data.TotalCount)
	}

	if len(resp.Data.Messages) != 5 {
		t.Errorf("Expected 5 messages, got %d", len(resp.Data.Messages))
	}
}
