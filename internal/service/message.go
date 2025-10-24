package service

import (
	"context"
	"ulak/internal/apierror"
	"ulak/internal/enums"
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
	limit := req.Limit
	if limit <= 0 {
		limit = 10 // default page size
	}

	offset := req.Offset
	if offset < 0 {
		offset = 0
	}

	statusFilter := enums.StatusSent

	// Fetch messages from the store
	messages, err := s.messageStore.List(ctx, limit, offset, statusFilter)
	if err != nil {
		return models.GetMessagesResponse{
			Result: &apierror.APIError{
				Code:    500,
				Message: "Failed to fetch messages: " + err.Error(),
			},
			Data: nil,
		}
	}

	// Get total count
	totalCount, err := s.messageStore.GetTotalCount(ctx)
	if err != nil {
		totalCount = 0

	}

	return models.GetMessagesResponse{
		Result: nil,
		Data: &models.GetMessagesData{
			Messages:   messages,
			TotalCount: totalCount,
			Limit:      limit,
			Offset:     offset,
		},
	}
}
