package service

import (
	"context"
	"ulak/internal/models"
	"ulak/internal/store/keyval"
	"ulak/internal/store/message"
)

// compile-time proofs of service interface implementation
var _ Service = (*service)(nil)

type service struct {
	kvStore      keyval.KeyValueStore
	messageStore message.MessageStore
}

type Service interface {
	SetMessageAutoSend(ctx context.Context, req models.SetMessageAutoSendRequest) models.SetMessageAutoSendResponse
	GetSentMessages(ctx context.Context, req models.GetMessagesRequest) models.GetMessagesResponse
}

func NewService(kvStore keyval.KeyValueStore, msgStore message.MessageStore) Service {
	return &service{
		kvStore:      kvStore,
		messageStore: msgStore,
	}
}
