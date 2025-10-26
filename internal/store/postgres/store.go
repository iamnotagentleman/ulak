package postgres

import (
	"context"
	"fmt"
	"ulak/internal/config"
	"ulak/internal/models"
	"ulak/internal/store/message"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// compile-time proof of interface implementation
var _ message.MessageStore = (*postgresStore)(nil)
var _ message.Transaction = (*postgresTransaction)(nil)

type postgresStore struct {
	db       *gorm.DB
	validate *validator.Validate
}

type PostgresStore = postgresStore

type postgresTransaction struct {
	tx *gorm.DB
}

// NewPostgresStore creates a new PostgreSQL store instance
func NewPostgresStore(cfg config.Postgres) (message.MessageStore, error) {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s",
		cfg.Host, cfg.User, cfg.Password, cfg.Database, cfg.Port, cfg.SSLMode)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// TODO change in production (golang-migrate)
	if err := db.AutoMigrate(&models.Message{}); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return &postgresStore{
		db:       db,
		validate: validator.New(),
	}, nil
}

func (ps *postgresStore) Create(ctx context.Context, msg *models.Message) error {
	// Validate message before creating
	if err := ps.validate.Struct(msg); err != nil {
		return fmt.Errorf("message validation failed: %w", err)
	}

	if msg.ID == uuid.Nil {
		msg.ID = uuid.New()
	}

	if msg.Status == "" {
		msg.Status = models.StatusPending
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

func (ps *postgresStore) ListByOffset(ctx context.Context, offset int64, status models.MessageSendingStatus, limit int) ([]*models.Message, error) {
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

func (ps *postgresStore) List(ctx context.Context, limit, skip int, status models.MessageSendingStatus) ([]*models.Message, error) {
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

func (ps *postgresStore) BeginTx(ctx context.Context) (message.Transaction, error) {
	tx := ps.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}

	return &postgresTransaction{tx: tx}, nil
}

func (pt *postgresTransaction) Update(ctx context.Context, msg *models.Message) error {
	result := pt.tx.WithContext(ctx).Save(msg)
	if result.Error != nil {
		return fmt.Errorf("failed to update message: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("message not found with id: %s", msg.ID)
	}

	return nil
}

func (pt *postgresTransaction) Commit() error {
	if err := pt.tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

func (pt *postgresTransaction) Rollback() error {
	if err := pt.tx.Rollback().Error; err != nil {
		return fmt.Errorf("failed to rollback transaction: %w", err)
	}
	return nil
}

// GetDB Do not use in production code, script only.
func (ps *postgresStore) GetDB() *gorm.DB {
	return ps.db
}
