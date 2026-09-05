// Package domain holds event-delivery core types (no broker SDK imports).
package domain

import "encoding/json"

// EventEnvelope is the transport-agnostic domain event payload.
type EventEnvelope struct {
	EventType string
	Data      json.RawMessage
	Key       string
}

// IdempotencyKey deduplicates at-least-once delivery.
type IdempotencyKey string

// ForEvent builds a stable key from event type and business id.
func IdempotencyKeyForEvent(eventType, businessID string) IdempotencyKey {
	if businessID == "" {
		return IdempotencyKey(eventType)
	}
	return IdempotencyKey(eventType + ":" + businessID)
}
