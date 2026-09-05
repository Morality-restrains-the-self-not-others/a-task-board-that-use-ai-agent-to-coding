package broker

import (
	"sync/atomic"

	"taskEvents/domain"
)

var _ domain.MQConnectivity = (*ConnectionState)(nil)

// ConnectionState tracks whether the message broker is reachable.
type ConnectionState struct {
	connected atomic.Bool
}

// NewConnectionState creates a connection tracker with an initial value.
func NewConnectionState(connected bool) *ConnectionState {
	state := &ConnectionState{}
	state.connected.Store(connected)
	return state
}

// SetConnected updates the broker connectivity flag.
func (c *ConnectionState) SetConnected(v bool) {
	if c == nil {
		return
	}
	c.connected.Store(v)
}

// Connected reports whether the broker is currently reachable.
func (c *ConnectionState) Connected() bool {
	if c == nil {
		return false
	}
	return c.connected.Load()
}
