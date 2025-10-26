package models

type SetMessageAutoSendRequest struct {
	Action AutoSendAction `json:"action" enums:"start,stop"`
}

type GetMessagesRequest struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}
