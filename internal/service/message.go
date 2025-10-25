package service

import (
	"context"
	"net/http"
	"ulak/internal/apierror"
	"ulak/internal/enums"
	"ulak/internal/models"
)

func (s *service) SetMessageAutoSend(ctx context.Context, req models.SetMessageAutoSendRequest) models.SetMessageAutoSendResponse {
	status := s.workerManager.GetStatus()

	switch req.Action {
	case enums.AutoSendStart:
		if status.IsPopulatorActive && status.IsProcessorActive {
			return models.SetMessageAutoSendResponse{
				Result: &apierror.APIError{
					Code:    http.StatusExpectationFailed,
					Message: "workers already started",
				},
				Data: nil}
		}
		s.workerManager.Start()
	case enums.AutoSendStop:
		if !status.IsPopulatorActive && !status.IsProcessorActive {
			return models.SetMessageAutoSendResponse{
				Result: &apierror.APIError{
					Code:    http.StatusExpectationFailed,
					Message: "workers already stopped",
				},
				Data: nil}
		}
		s.workerManager.Stop()
	default:
		return models.SetMessageAutoSendResponse{
			Result: &apierror.APIError{
				Code:    http.StatusForbidden,
				Message: "invalid action",
			},
			Data: nil}
	}
	status = s.workerManager.GetStatus()

	return models.SetMessageAutoSendResponse{
		Data:   &models.SetMessageAutoSendData{IsPopulatorEnabled: status.IsPopulatorActive, IsProcessorEnabled: status.IsProcessorActive},
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
				Code:    http.StatusInternalServerError,
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
