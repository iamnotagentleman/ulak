package notification

import "context"

type NotificationService interface {
	SendNotification(ctx context.Context, input Input) (AcknowledgeResponse, error)
}
