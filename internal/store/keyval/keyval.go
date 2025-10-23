package keyval

import (
	"context"
)

type KeyValueStore interface {
	GetMessageDelivered(ctx context.Context, messageId string) (string, error)
	SetMessageDelivered(ctx context.Context, messageId, value string, ttl int) error
}
