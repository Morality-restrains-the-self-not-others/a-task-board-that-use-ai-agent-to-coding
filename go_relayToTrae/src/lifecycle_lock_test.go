package main

import (
	"os"
	"testing"
	"time"
)

func TestTryAcquireLifecycleLockTimesOut(t *testing.T) {
	t.Setenv("RELAY_LIFECYCLE_LOCK_WAIT_SEC", "0")
	// wait=0 means block forever acquire path; use short wait for timeout case.
	_ = os.Unsetenv("RELAY_LIFECYCLE_LOCK_WAIT_SEC")
	t.Setenv("RELAY_LIFECYCLE_LOCK_WAIT_SEC", "1")

	lifecycleMu.Lock()
	defer lifecycleMu.Unlock()

	start := time.Now()
	ok := tryAcquireLifecycleLock()
	elapsed := time.Since(start)
	if ok {
		lifecycleMu.Unlock()
		t.Fatal("expected tryAcquireLifecycleLock to fail while lock held")
	}
	if elapsed < 900*time.Millisecond {
		t.Fatalf("expected ~1s wait, got %s", elapsed)
	}
}

func TestTryAcquireLifecycleLockSucceeds(t *testing.T) {
	t.Setenv("RELAY_LIFECYCLE_LOCK_WAIT_SEC", "2")
	if !tryAcquireLifecycleLock() {
		t.Fatal("expected acquire")
	}
	lifecycleMu.Unlock()
}
