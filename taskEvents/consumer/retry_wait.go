package consumer

import "time"

import "taskEvents/broker"

const (
	dispatchRetryInitial = time.Second
	dispatchRetryMax     = 30 * time.Second
)

// dispatchRetryWait is the pause before re-publishing a retryable event.
// It is derived from the durable `_retry_attempt` counter so Ack+republish
// (which starts a fresh in-memory Backoff) cannot collapse 11 retries into ~20s.
func dispatchRetryWait(attempt int) time.Duration {
	return broker.DelayForAttempt(dispatchRetryInitial, dispatchRetryMax, attempt)
}
