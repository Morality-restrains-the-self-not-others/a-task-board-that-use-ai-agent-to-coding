package publish

import "context"

// EventPublisher publishes outbound domain events.
type EventPublisher interface {
	PublishEvent(ctx context.Context, eventType string, data map[string]interface{}, key string) error
}
