package models

import "ulak/internal/enums"

type SetMessageAutoSendRequest struct {
	Action enums.AutoSendAction `json:"action" enums:"start,stop"`
}

type GetMessagesRequest struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}
