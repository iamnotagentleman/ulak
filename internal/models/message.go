package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Message struct {
	ID        uuid.UUID            `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	To        string               `json:"to" gorm:"type:varchar(255);not null"`
	Message   string               `json:"message" gorm:"type:text;not null" validate:"max=256"`
	Offset    int64                `json:"offset" gorm:"column:record_offset;type:serial;autoIncrement;uniqueIndex"`
	Channel   MessageChannel       `json:"channel" gorm:"type:varchar(50);not null"`
	Status    MessageSendingStatus `json:"status" gorm:"type:varchar(50);not null;default:'PENDING';index"`
	CreatedAt time.Time            `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time            `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt       `json:"deleted_at,omitempty" gorm:"index" swaggertype:"string"`
	IsActive  bool                 `json:"is_active" gorm:"default:true"`
}

type MessageSendingStatus string

const (
	StatusPending MessageSendingStatus = "PENDING"
	StatusSent    MessageSendingStatus = "SENT"
	StatusFailed  MessageSendingStatus = "FAILED"
)

type MessageChannel string

const (
	ChannelSMS     MessageChannel = "SMS"
	ChannelEmail   MessageChannel = "EMAIL"
	ChannelPush    MessageChannel = "PUSH"
	ChannelWebhook MessageChannel = "WEBHOOK"
)

type AutoSendAction string

const (
	AutoSendStart AutoSendAction = "start"
	AutoSendStop  AutoSendAction = "stop"
)
