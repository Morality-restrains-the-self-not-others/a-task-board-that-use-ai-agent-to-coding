package domain

import "time"

// BackoffPolicy configures exponential retry delays for broker connectivity.
type BackoffPolicy struct {
	Initial  time.Duration
	Max      time.Duration
	Attempts int
}

// DefaultStartupPingPolicy is used before the consumer enters the subscribe loop.
// Uses 10 attempts with exponential backoff up to 30s max (total ~181s) to tolerate
// slow-starting Kafka brokers without immediately crashing.
func DefaultStartupPingPolicy() BackoffPolicy {
	return BackoffPolicy{
		Initial:  time.Second,
		Max:      30 * time.Second,
		Attempts: 10,
	}
}

// DefaultRuntimeReconnectPolicy is used when the broker loses connectivity during consume.
func DefaultRuntimeReconnectPolicy() BackoffPolicy {
	return BackoffPolicy{
		Initial: 100 * time.Millisecond,
		Max:     30 * time.Second,
	}
}

// ConnectionHealth summarizes broker reachability for readiness probes.
type ConnectionHealth struct {
	Connected bool
	Transport TransportKind
}
