package domain

import "context"

// DomainCommand describes an HTTP call to Django internal API.
type DomainCommand struct {
	Domain    string
	EventType string
	Path      string
	Envelope  EventEnvelope
}

// DispatchOutcome tells the dispatch service whether to Ack.
type DispatchOutcome int

const (
	DispatchSuccess DispatchOutcome = iota
	DispatchRetryable
	DispatchPermanent
)

// DomainCommandPort invokes business handling in saas-backend.
type DomainCommandPort interface {
	Dispatch(ctx context.Context, cmd DomainCommand) (DispatchOutcome, error)
}
