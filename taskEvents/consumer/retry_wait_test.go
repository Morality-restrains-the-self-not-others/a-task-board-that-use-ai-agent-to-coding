package consumer

import (
	"testing"
	"time"
)

func TestDispatchRetryWaitUsesDurableAttemptNotFreshBackoff(t *testing.T) {
	if got := dispatchRetryWait(1); got != time.Second {
		t.Fatalf("attempt 1: got %v want 1s", got)
	}
	if got := dispatchRetryWait(5); got != 16*time.Second {
		t.Fatalf("attempt 5: got %v want 16s", got)
	}
	if got := dispatchRetryWait(6); got != 30*time.Second {
		t.Fatalf("attempt 6: got %v want 30s (cap)", got)
	}
	// After republish the consumer constructs a new Backoff; Wait() would be 1s.
	if got := dispatchRetryWait(11); got != 30*time.Second {
		t.Fatalf("attempt 11 after republish: got %v want 30s", got)
	}
}
