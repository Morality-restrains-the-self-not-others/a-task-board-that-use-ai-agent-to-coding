package domain

import "time"

// DomainEventReceived is raised when a message is pulled from the broker.
type DomainEventReceived struct {
	EventType  string
	OccurredAt time.Time
}

// DomainEventDispatched is raised after successful DomainCommandPort.Dispatch.
type DomainEventDispatched struct {
	EventType  string
	IdempotencyKey IdempotencyKey
	OccurredAt time.Time
}

// DomainEventProcessingFailed is raised on permanent failure or retry exhaustion.
type DomainEventProcessingFailed struct {
	EventType  string
	Reason     string
	OccurredAt time.Time
}
