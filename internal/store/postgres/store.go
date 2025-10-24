package postgres

import (
	"context"
	"fmt"
	"ulak/internal/config"
	"ulak/internal/enums"
	"ulak/internal/models"
	"ulak/internal/store/message"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// compile-time proof of interface implementation
var _ message.MessageStore = (*postgresStore)(nil)

type postgresStore struct {
	db *gorm.DB
}

// NewPostgresStore creates a new PostgreSQL store instance
func NewPostgresStore(cfg config.Postgres) (message.MessageStore, error) {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s",
		cfg.Host, cfg.User, cfg.Password, cfg.Database, cfg.Port, cfg.SSLMode)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Auto-migrate the Message model
	if err := db.AutoMigrate(&models.Message{}); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return &postgresStore{db: db}, nil
}

func (ps *postgresStore) Create(ctx context.Context, msg *models.Message) error {
	if msg.ID == uuid.Nil {
		msg.ID = uuid.New()
	}

	if msg.Status == "" {
		msg.Status = enums.StatusPending
	}

	if !msg.IsActive {
		msg.IsActive = true
	}

	result := ps.db.WithContext(ctx).Create(msg)
	if result.Error != nil {
		return fmt.Errorf("failed to create message: %w", result.Error)
	}

	return nil
}

func (ps *postgresStore) GetByID(ctx context.Context, id string) (*models.Message, error) {
	msgID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid UUID: %w", err)
	}

	var msg models.Message
	result := ps.db.WithContext(ctx).Where("id = ?", msgID).First(&msg)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("message not found with id: %s", id)
		}
		return nil, fmt.Errorf("failed to get message: %w", result.Error)
	}

	return &msg, nil
}

func (ps *postgresStore) GetByOffset(ctx context.Context, offset int64) (*models.Message, error) {
	var msg models.Message
	result := ps.db.WithContext(ctx).Where("record_offset = ?", offset).First(&msg)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("message not found with offset: %d", offset)
		}
		return nil, fmt.Errorf("failed to get message: %w", result.Error)
	}

	return &msg, nil
}

func (ps *postgresStore) ListByOffset(ctx context.Context, offset int64, status enums.MessageSendingStatus, limit int) ([]*models.Message, error) {
	var messages []*models.Message

	query := ps.db.WithContext(ctx).Model(&models.Message{}).Where("record_offset > ?", offset)

	if status != "" {
		query = query.Where("status = ?", status)
	}

	if limit > 0 {
		query = query.Limit(limit)
	}

	query = query.Order("record_offset ASC")

	result := query.Find(&messages)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to list messages by offset greater than %d: %w", offset, result.Error)
	}

	return messages, nil
}

func (ps *postgresStore) List(ctx context.Context, limit, skip int, status enums.MessageSendingStatus) ([]*models.Message, error) {
	var messages []*models.Message

	query := ps.db.WithContext(ctx).Model(&models.Message{})

	if status != "" {
		query = query.Where("status = ?", status)
	}

	if limit > 0 {
		query = query.Limit(limit)
	}
	if skip > 0 {
		query = query.Offset(skip)
	}

	query = query.Order("created_at DESC")

	result := query.Find(&messages)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to list messages: %w", result.Error)
	}

	return messages, nil
}

func (ps *postgresStore) Update(ctx context.Context, msg *models.Message) error {
	result := ps.db.WithContext(ctx).Save(msg)
	if result.Error != nil {
		return fmt.Errorf("failed to update message: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("message not found with id: %s", msg.ID)
	}

	return nil
}

func (ps *postgresStore) Delete(ctx context.Context, id string) error {
	msgID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid UUID: %w", err)
	}

	result := ps.db.WithContext(ctx).Delete(&models.Message{}, msgID)
	if result.Error != nil {
		return fmt.Errorf("failed to delete message: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("message not found with id: %s", id)
	}

	return nil
}

func (ps *postgresStore) GetTotalCount(ctx context.Context) (int64, error) {
	var count int64
	result := ps.db.WithContext(ctx).Model(&models.Message{}).Count(&count)

	if result.Error != nil {
		return 0, fmt.Errorf("failed to count messages: %w", result.Error)
	}

	return count, nil
}

func (ps *postgresStore) GetDB() *gorm.DB {
	return ps.db
}
