package notification

import "github.com/google/uuid"

type Input struct {
	To      string `json:"to"`
	Content string `json:"content"`
}

type AcknowledgeResponse struct {
	Message   string    `json:"message"`
	MessageId uuid.UUID `json:"messageId"`
}
