package domain

import "context"

// BrokerMessage wraps an envelope with opaque cursor for Ack.
type BrokerMessage struct {
	Envelope EventEnvelope
	Cursor   BrokerCursor
	// AckFunc commits the message when non-nil (set by infrastructure adapter).
	AckFunc func(ctx context.Context) error
}

// BrokerCursor is opaque to domain (offset, stream ID, etc.).
type BrokerCursor struct {
	Raw string
}

// EventBrokerPort subscribes and acknowledges without leaking Kafka/Redis types.
type EventBrokerPort interface {
	Subscribe(ctx context.Context, topics []string) (<-chan BrokerMessage, error)
	Ack(ctx context.Context, msg BrokerMessage) error
	Close() error
}

// AckMessage commits when AckFunc is set.
func AckMessage(ctx context.Context, msg BrokerMessage) error {
	if msg.AckFunc != nil {
		return msg.AckFunc(ctx)
	}
	return nil
}
