package models

import "ulak/internal/apierror"

type SetMessageAutoSendData struct {
	Status  string `json:"status"`
	Enabled bool   `json:"enabled"`
}

type GetMessagesData struct {
	Messages   []*Message `json:"messages"`
	TotalCount int64      `json:"total_count"`
	Limit      int        `json:"limit"`
	Offset     int        `json:"offset"`
}

type SetMessageAutoSendResponse struct {
	Result *apierror.APIError      `json:"result"`
	Data   *SetMessageAutoSendData `json:"data"`
}

type GetMessagesResponse struct {
	Result *apierror.APIError `json:"result"`
	Data   *GetMessagesData   `json:"data"`
}
