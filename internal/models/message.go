package models

import (
	"time"
	"ulak/internal/enums"

	"github.com/google/uuid"
)

type Message struct {
	ID        uuid.UUID                  `json:"id"`
	Message   string                     `json:"message" validate:"max=255"`
	Offset    int64                      `json:"offset"`
	Channel   enums.MessageChannel       `json:"channel"`
	Status    enums.MessageSendingStatus `json:"status"`
	CreatedAt time.Time                  `json:"created_at"`
	UpdatedAt time.Time                  `json:"updated_at"`
	DeletedAt *time.Time                 `json:"deleted_at,omitempty"`
	IsActive  bool                       `json:"is_active"`
}
