package service

import (
	"context"
	"ulak/internal/models"
)

func (s *service) SetMessageAutoSend(ctx context.Context, req models.SetMessageAutoSendRequest) models.SetMessageAutoSendResponse {
	// TODO IMPLEMENT ME
	return models.SetMessageAutoSendResponse{
		Data: &models.SetMessageAutoSendData{Status: "active",
			Enabled: false},
		Result: nil,
	}
}

func (s *service) GetSentMessages(ctx context.Context, req models.GetMessagesRequest) models.GetMessagesResponse {
	// TODO IMPLEMENT ME
	return models.GetMessagesResponse{
		Result: nil,
		Data:   nil,
	}
}
