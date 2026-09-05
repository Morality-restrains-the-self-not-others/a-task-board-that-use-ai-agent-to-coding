package domain

// MQConnectivity reports whether the configured message broker is reachable.
type MQConnectivity interface {
	Connected() bool
}

// NewConnectionHealth builds readiness state from live broker connectivity.
func NewConnectionHealth(mq MQConnectivity, transport TransportKind) ConnectionHealth {
	connected := false
	if mq != nil {
		connected = mq.Connected()
	}
	return ConnectionHealth{
		Connected: connected,
		Transport: transport,
	}
}

// ReadinessStatus returns "ok" when connected, otherwise "degraded".
func (h ConnectionHealth) ReadinessStatus() string {
	if h.Connected {
		return "ok"
	}
	return "degraded"
}
