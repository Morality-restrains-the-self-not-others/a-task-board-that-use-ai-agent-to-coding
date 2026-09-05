package main

import (
	"strings"
	"testing"
)

// OPT-20260813-007: a silent graceful shutdown (periodic SIGTERM, not shutdown-self)
// must be attributable from logs. These tests pin the exit-reason recorder and the
// lifecycle snapshot that main.go logs right before process exit.

func TestRunAllExitReason_FirstWins(t *testing.T) {
	r := newRunAllExitReason()
	r.record("signal", "terminated")
	// A shutdown-self during the cascade must not overwrite the initiating signal.
	r.record("shutdown-self", "/api/shutdown-self")

	source, detail := r.get()
	if source != "signal" || detail != "terminated" {
		t.Fatalf("get() = %q/%q, want signal/terminated", source, detail)
	}
}

func TestRunAllExitReason_ShutdownSelfAttributable(t *testing.T) {
	r := newRunAllExitReason()
	r.record("shutdown-self", "/api/shutdown-self")

	source, detail := r.get()
	if source != "shutdown-self" || detail != "/api/shutdown-self" {
		t.Fatalf("get() = %q/%q, want shutdown-self//api/shutdown-self", source, detail)
	}
}

func TestRunAllExitReason_NilReceiverSafe(t *testing.T) {
	var r *runAllExitReason
	r.record("signal", "terminated") // must not panic
	if source, detail := r.get(); source != "" || detail != "" {
		t.Fatalf("nil receiver get() = %q/%q, want empty", source, detail)
	}
}

func TestRunAllExitReason_UnknownWhenUnset(t *testing.T) {
	r := newRunAllExitReason()
	source, detail := r.get()
	if source != "" || detail != "" {
		t.Fatalf("unset get() = %q/%q, want empty", source, detail)
	}
}

func TestSummarizeServiceLifecycle_CountsAndNonHealthy(t *testing.T) {
	store := NewStatusStore()
	store.Init([]string{"svc-a", "svc-b", "svc-c", "svc-d"})
	store.Update("svc-a", StatusHealthy, "")
	store.Update("svc-b", StatusFailed, "boom")
	store.Update("svc-c", StatusStopped, "")
	// svc-d stays StatusPending (in-flight when the run loop ended).

	out := summarizeServiceLifecycle(store)
	for _, want := range []string{"healthy=1", "failed=1", "stopped=1", "pending=1", "svc-b:failed"} {
		if !strings.Contains(out, want) {
			t.Fatalf("summarizeServiceLifecycle = %q, want substring %q", out, want)
		}
	}
	if strings.Contains(out, "svc-a:") || strings.Contains(out, "svc-c:") {
		t.Fatalf("summarizeServiceLifecycle must not list healthy/stopped in non_healthy: %q", out)
	}
}

func TestSummarizeServiceLifecycle_NilStore(t *testing.T) {
	if out := summarizeServiceLifecycle(nil); out != "store=nil" {
		t.Fatalf("nil store = %q, want store=nil", out)
	}
}
