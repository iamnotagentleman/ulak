package models

type SetMessageAutoSendRequest struct {
	Action string `json:"action"`
}

type GetMessagesRequest struct {
	Limit       int  `json:"limit"`
	Offset      int  `json:"offset"`
	IsDelivered bool `json:"isDelivered"`
}
