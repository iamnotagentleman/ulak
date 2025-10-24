package models

import "ulak/internal/apierror"

type SetMessageAutoSendData struct {
	IsPopulatorEnabled bool `json:"isPopulatorEnabled"`
	IsProcessorEnabled bool `json:"isProcessorEnabled"`
}

type GetMessagesData struct {
	Messages   []*Message `json:"messages"`
	TotalCount int64      `json:"totalCount"`
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
