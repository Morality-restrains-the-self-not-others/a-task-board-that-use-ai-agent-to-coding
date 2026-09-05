package main

import (
	"fmt"
	"syscall"
	"testing"
)

func TestIsRetryableTerminationWaitError_EINTR(t *testing.T) {
	if !isRetryableTerminationWaitError(syscall.EINTR) {
		t.Fatal("syscall.EINTR must be retryable so rt_sigtimedwait interruptions do not exit runAll")
	}
	wrapped := fmt.Errorf("waitTerminationSignal: %w", syscall.EINTR)
	if !isRetryableTerminationWaitError(wrapped) {
		t.Fatalf("wrapped EINTR must be retryable, got %v", wrapped)
	}
	if isRetryableTerminationWaitError(syscall.EINVAL) {
		t.Fatal("EINVAL must not be treated as a retryable wait interruption")
	}
	if isRetryableTerminationWaitError(nil) {
		t.Fatal("nil error is not retryable")
	}
}
