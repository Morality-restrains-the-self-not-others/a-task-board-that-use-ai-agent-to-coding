package filter

import (
	"context"
	"fmt"

	"taskEvents/domain"
)

// SingleEvent wraps a DomainCommandPort to accept only one event_type.
type SingleEvent struct {
	EventType string
	Inner     domain.DomainCommandPort
}

func (s *SingleEvent) Dispatch(ctx context.Context, cmd domain.DomainCommand) (domain.DispatchOutcome, error) {
	if cmd.EventType != s.EventType {
		return domain.DispatchPermanent, fmt.Errorf("unexpected event %s want %s", cmd.EventType, s.EventType)
	}
	return s.Inner.Dispatch(ctx, cmd)
}
