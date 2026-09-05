package main

import (
	"testing"
)

func TestExtractJobIDsFromResponseSingle(t *testing.T) {
	raw := []byte(`{"id":"job-abc","status":"running"}`)
	ids := extractJobIDsFromResponse(raw)
	if len(ids) != 1 || ids[0] != "job-abc" {
		t.Fatalf("ids=%v", ids)
	}
}

func TestExtractJobIDsFromResponseBatch(t *testing.T) {
	raw := []byte(`{"ok":true,"jobs":[{"model":"m1","job":{"id":"j1"}},{"model":"m2","job":{"id":"j2"}}]}`)
	ids := extractJobIDsFromResponse(raw)
	if len(ids) != 2 || ids[0] != "j1" || ids[1] != "j2" {
		t.Fatalf("ids=%v", ids)
	}
}

func TestExtractJobIDsFromResponseEmpty(t *testing.T) {
	if len(extractJobIDsFromResponse([]byte(`{}`))) != 0 {
		t.Fatal("expected empty")
	}
}

func TestJobStreamPollDisabledByDefault(t *testing.T) {
	t.Setenv("CONTAINER_JOB_STREAM_POLL_ENABLED", "")
	if jobStreamPollEnabled() {
		t.Fatal("poll must be off by default; container PUSH is the live path")
	}
	t.Setenv("CONTAINER_JOB_STREAM_POLL_ENABLED", "true")
	if !jobStreamPollEnabled() {
		t.Fatal("explicit true should enable poll escape hatch")
	}
}

func TestIsTerminalJobEventPhase(t *testing.T) {
	for _, p := range []string{"completed", "failed", "interrupted", "COMPLETED", " Failed "} {
		if !isTerminalJobEventPhase(p) {
			t.Fatalf("expected terminal: %q", p)
		}
	}
	for _, p := range []string{"", "running", "pending", "chunk", "step", "start", "done", "error"} {
		if isTerminalJobEventPhase(p) {
			t.Fatalf("expected non-terminal: %q", p)
		}
	}
}

func TestJobIDFromMap(t *testing.T) {
	m := map[string]any{"id": "x1"}
	if jobIDFromMap(m) != "x1" {
		t.Fatalf("got %q", jobIDFromMap(m))
	}
}

func TestDedupeStrings(t *testing.T) {
	out := dedupeStrings([]string{"a", "a", "b"})
	if len(out) != 2 {
		t.Fatalf("out=%v", out)
	}
}

func TestExtractJobIDsInvalidJSON(t *testing.T) {
	if extractJobIDsFromResponse([]byte("not-json")) != nil {
		t.Fatal("expected nil")
	}
}
