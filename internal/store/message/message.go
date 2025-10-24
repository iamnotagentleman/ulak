package message

import (
	"context"
	"ulak/internal/enums"
	"ulak/internal/models"

	"gorm.io/gorm"
)

// MessageStore defines the interface for message persistence operations
type MessageStore interface {
	Create(ctx context.Context, message *models.Message) error
	GetByID(ctx context.Context, id string) (*models.Message, error)
	GetByOffset(ctx context.Context, offset int64) (*models.Message, error)
	ListByOffset(ctx context.Context, offset int64, status enums.MessageSendingStatus, limit int) ([]*models.Message, error)
	List(ctx context.Context, limit, skip int, status enums.MessageSendingStatus) ([]*models.Message, error)
	Update(ctx context.Context, message *models.Message) error
	Delete(ctx context.Context, id string) error
	GetTotalCount(ctx context.Context) (int64, error)
	GetDB() *gorm.DB
}
