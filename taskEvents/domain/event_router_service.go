package domain

// EventRoute maps event_type to internal API path for a domain.
type EventRoute struct {
	EventType string
	Path      string
}

// DomainEventRouter resolves routes for a consumer domain.
type DomainEventRouter struct {
	Domain string
	Routes map[string]EventRoute
}

// CommandFor builds a DomainCommand or ok=false if unrouted.
func (r *DomainEventRouter) CommandFor(env EventEnvelope) (DomainCommand, bool) {
	route, ok := r.Routes[env.EventType]
	if !ok {
		return DomainCommand{}, false
	}
	return DomainCommand{
		Domain:    r.Domain,
		EventType: env.EventType,
		Path:      route.Path,
		Envelope:  env,
	}, true
}
