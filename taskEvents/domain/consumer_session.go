package domain

// ConsumerSession is the aggregate root for one domain-scoped consumer process.
type ConsumerSession struct {
	DomainName       string
	Transport        TransportKind
	GroupID          string
	SubscribedEvents []string
}

// AllowsEvent returns whether this session may handle the event type.
func (s *ConsumerSession) AllowsEvent(eventType string) bool {
	for _, e := range s.SubscribedEvents {
		if e == eventType {
			return true
		}
	}
	return false
}
