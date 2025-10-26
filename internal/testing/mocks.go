package testing

import (
	"context"
	"sync"
	"ulak/internal/models"
	"ulak/internal/store/message"
	"ulak/pkg/notification"
)

// MockTransaction is a mock implementation of message.Transaction
type MockTransaction struct {
	mu           sync.Mutex
	UpdateFunc   func(ctx context.Context, message *models.Message) error
	CommitFunc   func() error
	RollbackFunc func() error
	committed    bool
	rolledBack   bool
}

func (t *MockTransaction) Update(ctx context.Context, message *models.Message) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.UpdateFunc != nil {
		return t.UpdateFunc(ctx, message)
	}
	return nil
}

func (t *MockTransaction) Commit() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.CommitFunc != nil {
		err := t.CommitFunc()
		if err == nil {
			t.committed = true
		}
		return err
	}
	t.committed = true
	return nil
}

func (t *MockTransaction) Rollback() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.RollbackFunc != nil {
		err := t.RollbackFunc()
		if err == nil {
			t.rolledBack = true
		}
		return err
	}
	t.rolledBack = true
	return nil
}

func (t *MockTransaction) IsCommitted() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.committed
}

func (t *MockTransaction) IsRolledBack() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.rolledBack
}

// MockMessageStore is a mock implementation of message.MessageStore
type MockMessageStore struct {
	mu                sync.RWMutex
	CreateFunc        func(ctx context.Context, message *models.Message) error
	GetByIDFunc       func(ctx context.Context, id string) (*models.Message, error)
	GetByOffsetFunc   func(ctx context.Context, offset int64) (*models.Message, error)
	ListByOffsetFunc  func(ctx context.Context, offset int64, status models.MessageSendingStatus, limit int) ([]*models.Message, error)
	ListFunc          func(ctx context.Context, limit, skip int, status models.MessageSendingStatus) ([]*models.Message, error)
	UpdateFunc        func(ctx context.Context, message *models.Message) error
	DeleteFunc        func(ctx context.Context, id string) error
	GetTotalCountFunc func(ctx context.Context) (int64, error)
	BeginTxFunc       func(ctx context.Context) (message.Transaction, error)
}

func (m *MockMessageStore) Create(ctx context.Context, message *models.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, message)
	}
	return nil
}

func (m *MockMessageStore) GetByID(ctx context.Context, id string) (*models.Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockMessageStore) GetByOffset(ctx context.Context, offset int64) (*models.Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.GetByOffsetFunc != nil {
		return m.GetByOffsetFunc(ctx, offset)
	}
	return nil, nil
}

func (m *MockMessageStore) ListByOffset(ctx context.Context, offset int64, status models.MessageSendingStatus, limit int) ([]*models.Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.ListByOffsetFunc != nil {
		return m.ListByOffsetFunc(ctx, offset, status, limit)
	}
	return []*models.Message{}, nil
}

func (m *MockMessageStore) List(ctx context.Context, limit, skip int, status models.MessageSendingStatus) ([]*models.Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.ListFunc != nil {
		return m.ListFunc(ctx, limit, skip, status)
	}
	return []*models.Message{}, nil
}

func (m *MockMessageStore) Update(ctx context.Context, message *models.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, message)
	}
	return nil
}

func (m *MockMessageStore) Delete(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}

func (m *MockMessageStore) GetTotalCount(ctx context.Context) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.GetTotalCountFunc != nil {
		return m.GetTotalCountFunc(ctx)
	}
	return 0, nil
}

func (m *MockMessageStore) BeginTx(ctx context.Context) (message.Transaction, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.BeginTxFunc != nil {
		return m.BeginTxFunc(ctx)
	}
	return &MockTransaction{}, nil
}

// MockKeyValueStore is a mock implementation of keyval.KeyValueStore
type MockKeyValueStore struct {
	mu                      sync.RWMutex
	GetMessageDeliveredFunc func(ctx context.Context, messageId string) (string, error)
	SetMessageDeliveredFunc func(ctx context.Context, messageId, value string, ttl int) error
}

func (m *MockKeyValueStore) GetMessageDelivered(ctx context.Context, messageId string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.GetMessageDeliveredFunc != nil {
		return m.GetMessageDeliveredFunc(ctx, messageId)
	}
	return "", nil
}

func (m *MockKeyValueStore) SetMessageDelivered(ctx context.Context, messageId, value string, ttl int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.SetMessageDeliveredFunc != nil {
		return m.SetMessageDeliveredFunc(ctx, messageId, value, ttl)
	}
	return nil
}

// MockNotificationService is a mock implementation of notification.NotificationService
type MockNotificationService struct {
	mu                   sync.RWMutex
	SendNotificationFunc func(ctx context.Context, input notification.Input) (notification.AcknowledgeResponse, error)
}

func (m *MockNotificationService) SendNotification(ctx context.Context, input notification.Input) (notification.AcknowledgeResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.SendNotificationFunc != nil {
		return m.SendNotificationFunc(ctx, input)
	}
	return notification.AcknowledgeResponse{}, nil
}
