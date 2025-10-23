package service

import (
	"context"
	"ulak/internal/models"
	"ulak/internal/store/keyval"
)

// compile-time proofs of service interface implementation
var _ Service = (*service)(nil)

type service struct {
	store keyval.KeyValueStore
}

type Service interface {
	SetMessageAutoSend(ctx context.Context, req models.SetMessageAutoSendRequest) models.SetMessageAutoSendResponse
	GetSentMessages(ctx context.Context, req models.GetMessagesRequest) models.GetMessagesResponse
}

func NewService(store keyval.KeyValueStore) Service {
	return &service{
		store: store,
	}
}
