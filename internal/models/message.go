package models

import (
	"time"
	"ulak/internal/enums"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Message struct {
	ID        uuid.UUID                  `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Message   string                     `json:"message" gorm:"type:text;not null" validate:"max=4096"`
	Offset    int64                      `json:"offset" gorm:"type:serial;autoIncrement;uniqueIndex"`
	Channel   enums.MessageChannel       `json:"channel" gorm:"type:varchar(50);not null"`
	Status    enums.MessageSendingStatus `json:"status" gorm:"type:varchar(50);not null;default:'PENDING'"`
	CreatedAt time.Time                  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time                  `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt             `json:"deleted_at,omitempty" gorm:"index"`
	IsActive  bool                       `json:"is_active" gorm:"default:true"`
}
