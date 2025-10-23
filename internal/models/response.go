package models

import "ulak/internal/apierror"

type SetMessageAutoSendData struct {
	Status  string `json:"status"`
	Enabled bool   `json:"enabled"`
}

type GetMessagesData struct {
	Status  string `json:"status"`
	Enabled bool   `json:"enabled"`
}

type SetMessageAutoSendResponse struct {
	Result *apierror.APIError      `json:"result"`
	Data   *SetMessageAutoSendData `json:"data"`
}

type GetMessagesResponse struct {
	Result *apierror.APIError `json:"result"`
	Data   *GetMessagesData
}
