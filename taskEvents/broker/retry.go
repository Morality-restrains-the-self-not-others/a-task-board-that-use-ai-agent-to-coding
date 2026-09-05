package broker

import (
	"context"
	"log"
	"sync"
	"time"

	"taskEvents/domain"
)

// Backoff implements exponential backoff with reset.
type Backoff struct {
	Initial time.Duration
	Max     time.Duration
	current time.Duration
}

// BackoffFromPolicy maps a domain BackoffPolicy to a mutable broker Backoff.
func BackoffFromPolicy(p domain.BackoffPolicy) Backoff {
	return Backoff{Initial: p.Initial, Max: p.Max}
}

// DefaultRuntimeBackoff is used when the broker loses connectivity during consume.
func DefaultRuntimeBackoff() Backoff {
	return BackoffFromPolicy(domain.DefaultRuntimeReconnectPolicy())
}

// DefaultStartupBackoff is used for startup Ping retries.
func DefaultStartupBackoff() Backoff {
	return BackoffFromPolicy(domain.DefaultStartupPingPolicy())
}

// DelayForAttempt returns the exponential wait for a 1-based attempt count.
// Attempt 1 is Initial; each subsequent attempt doubles until Max.
// This is durable across Ack+republish (which constructs a fresh Backoff).
func DelayForAttempt(initial, max time.Duration, attempt int) time.Duration {
	if initial <= 0 {
		initial = time.Second
	}
	if max < initial {
		max = initial
	}
	if attempt < 1 {
		attempt = 1
	}
	d := initial
	for i := 1; i < attempt; i++ {
		if d >= max {
			return max
		}
		next := d * 2
		if next > max || next < d {
			return max
		}
		d = next
	}
	return d
}

// WaitAttempt sleeps DelayForAttempt(Initial, Max, attempt). Unlike Wait, it
// does not depend on in-memory current, so retry-republish keeps growing delay.
func (b *Backoff) WaitAttempt(ctx context.Context, attempt int) error {
	wait := DelayForAttempt(b.Initial, b.Max, attempt)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(wait):
		return nil
	}
}

// Wait sleeps for the current backoff interval and doubles it up to Max.
func (b *Backoff) Wait(ctx context.Context) error {
	if b.current == 0 {
		b.current = b.Initial
	}
	wait := b.current
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(wait):
	}
	next := time.Duration(float64(b.current) * 2)
	if next > b.Max {
		next = b.Max
	}
	b.current = next
	return nil
}

// Reset clears accumulated backoff after a successful operation.
func (b *Backoff) Reset() {
	b.current = 0
}

// RateLimitedLogger emits at most one log line per interval.
type RateLimitedLogger struct {
	interval time.Duration
	mu       sync.Mutex
	last     time.Time
}

// NewRateLimitedLogger creates a logger with the given minimum interval.
func NewRateLimitedLogger(interval time.Duration) *RateLimitedLogger {
	return &RateLimitedLogger{interval: interval}
}

// Warnf logs when the interval has elapsed since the previous message.
func (l *RateLimitedLogger) Warnf(format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	if !l.last.IsZero() && now.Sub(l.last) < l.interval {
		return
	}
	log.Printf(format, args...)
	l.last = now
}
