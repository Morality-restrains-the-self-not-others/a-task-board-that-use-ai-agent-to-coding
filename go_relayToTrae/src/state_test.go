package main

import (
	"strings"
	"testing"

	"tracelog"
)

func TestAppendLogLockedAddsLine(t *testing.T) {
	// Reset state.
	stateMu.Lock()
	state.Logs = nil
	stateMu.Unlock()

	appendLog("line 1")
	appendLog("line 2")

	stateMu.Lock()
	defer stateMu.Unlock()
	if len(state.Logs) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(state.Logs))
	}
	if state.Logs[0] != "line 1" {
		t.Errorf("expected 'line 1', got %q", state.Logs[0])
	}
	if state.Logs[1] != "line 2" {
		t.Errorf("expected 'line 2', got %q", state.Logs[1])
	}
}

func TestAppendLogSkipsEmptyString(t *testing.T) {
	stateMu.Lock()
	state.Logs = nil
	stateMu.Unlock()

	appendLog("")

	stateMu.Lock()
	defer stateMu.Unlock()
	if len(state.Logs) != 0 {
		t.Errorf("expected 0 lines for empty input, got %d", len(state.Logs))
	}
}

func TestAppendLogTruncatesAt4000Lines(t *testing.T) {
	stateMu.Lock()
	state.Logs = nil
	stateMu.Unlock()

	// Add 4100 lines.
	for i := 0; i < 4100; i++ {
		appendLog("x")
	}

	stateMu.Lock()
	defer stateMu.Unlock()
	if len(state.Logs) != 4000 {
		t.Errorf("expected 4000 lines after truncation, got %d", len(state.Logs))
	}
}

func TestAppendLogLockedPreservesLast4000(t *testing.T) {
	stateMu.Lock()
	state.Logs = nil
	// Directly add 4000 lines + 1 more via appendLogLocked.
	for i := 0; i < 4000; i++ {
		state.Logs = append(state.Logs, "old")
	}
	appendLogLocked("new")
	stateMu.Unlock()

	stateMu.Lock()
	defer stateMu.Unlock()
	if len(state.Logs) != 4000 {
		t.Fatalf("expected 4000 lines, got %d", len(state.Logs))
	}
	if state.Logs[0] != "old" {
		t.Errorf("expected first line 'old', got %q", state.Logs[0])
	}
	if state.Logs[3999] != "new" {
		t.Errorf("expected last line 'new', got %q", state.Logs[3999])
	}
}

func TestRegisterTaskLockedRequiresTaskID(t *testing.T) {
	stateMu.Lock()
	before := len(registeredTasks)
	registerTaskLocked("t1", "w1", "", "http://origin/api", "token", "")
	after := len(registeredTasks)
	stateMu.Unlock()

	if after != before {
		t.Errorf("expected no task registered without task_id, but count changed from %d to %d", before, after)
	}
}

func TestRegisterTaskLockedCreatesEntry(t *testing.T) {
	stateMu.Lock()
	state.AccessToken = "" // reset to avoid pollution from other tests
	delete(registeredTasks, "task-1")
	registerTaskLocked("tenant-a", "ws-a", "task-1", "http://origin.example.com/api", "secret-token", "")
	reg, ok := registeredTasks["task-1"]
	stateMu.Unlock()

	if !ok {
		t.Fatal("expected task-1 to be registered")
	}
	if reg.TenantID != "tenant-a" {
		t.Errorf("expected tenant-a, got %s", reg.TenantID)
	}
	if reg.TaskAPIOrigin != "http://origin.example.com" {
		t.Errorf("expected origin extracted, got %s", reg.TaskAPIOrigin)
	}
	if reg.AccessToken != "secret-token" {
		t.Errorf("expected secret-token, got %s", reg.AccessToken)
	}
}

func TestUnregisterTaskLockedRemovesEntry(t *testing.T) {
	stateMu.Lock()
	registeredTasks["task-rm"] = &RegisteredTask{TaskID: "task-rm"}
	unregisterTaskLocked("task-rm")
	_, ok := registeredTasks["task-rm"]
	stateMu.Unlock()

	if ok {
		t.Error("expected task-rm to be unregistered")
	}
}

func TestUnregisterTaskLockedSilentForMissing(t *testing.T) {
	stateMu.Lock()
	delete(registeredTasks, "nonexistent")
	unregisterTaskLocked("nonexistent")
	stateMu.Unlock()
	// Should not panic.
}

func TestResolveAccessTokenForRegisterLockedPrefersStateToken(t *testing.T) {
	stateMu.Lock()
	state.AccessToken = "state-token"
	token := resolveAccessTokenForRegisterLocked("request-token")
	state.AccessToken = ""
	stateMu.Unlock()

	if token != "state-token" {
		t.Errorf("expected state-token, got %s", token)
	}
}

func TestResolveAccessTokenForRegisterLockedFallsBackToInput(t *testing.T) {
	stateMu.Lock()
	state.AccessToken = ""
	token := resolveAccessTokenForRegisterLocked("request-token")
	stateMu.Unlock()

	if token != "request-token" {
		t.Errorf("expected request-token, got %s", token)
	}
}

func TestSyncRegisteredAccessTokenFromStateLocked(t *testing.T) {
	stateMu.Lock()
	state.AccessToken = "new-state-token"
	reg := &RegisteredTask{AccessToken: "old-token"}
	syncRegisteredAccessTokenFromStateLocked(reg)
	state.AccessToken = ""
	stateMu.Unlock()

	if reg.AccessToken != "new-state-token" {
		t.Errorf("expected new-state-token, got %s", reg.AccessToken)
	}
}

func TestSyncRegisteredAccessTokenSkipsWhenStateEmpty(t *testing.T) {
	stateMu.Lock()
	state.AccessToken = ""
	reg := &RegisteredTask{AccessToken: "keep-me"}
	syncRegisteredAccessTokenFromStateLocked(reg)
	stateMu.Unlock()

	if reg.AccessToken != "keep-me" {
		t.Errorf("expected keep-me, got %s", reg.AccessToken)
	}
}

func TestAppendLogHandlesNilLogs(t *testing.T) {
	stateMu.Lock()
	state.Logs = nil
	stateMu.Unlock()

	appendLog("recovery test")

	stateMu.Lock()
	defer stateMu.Unlock()
	if len(state.Logs) != 1 {
		t.Fatalf("expected 1 line after nil recovery, got %d", len(state.Logs))
	}
	if state.Logs[0] != "recovery test" {
		t.Errorf("expected 'recovery test', got %q", state.Logs[0])
	}
}

func TestRequiredEnvKeys(t *testing.T) {
	expected := []string{
		"TASK_API_ENDPOINT_ORIGIN",
		"BUSINESS_API_ENDPOINT_ORIGIN",
		"ACCESS_TOKEN",
	}
	if len(requiredEnvKeys) != len(expected) {
		t.Fatalf("expected %d keys, got %d", len(expected), len(requiredEnvKeys))
	}
	for i, k := range expected {
		if requiredEnvKeys[i] != k {
			t.Errorf("key %d: expected %s, got %s", i, k, requiredEnvKeys[i])
		}
	}
}

func TestStateDefaults(t *testing.T) {
	s := relayState{}
	if s.Running {
		t.Error("expected Running to default to false")
	}
	if s.PID != 0 {
		t.Error("expected PID to default to 0")
	}
	if s.Port != 0 {
		t.Error("expected Port to default to 0")
	}
}

func TestRegisteredTaskDefaults(t *testing.T) {
	reg := RegisteredTask{}
	if reg.LogCursor != 0 {
		t.Error("expected LogCursor to default to 0")
	}
}

func TestActiveTraceIDUsesActiveCorrelationCtx(t *testing.T) {
	stateMu.Lock()
	state.ActiveTraceID = "relay-trace-test123456"
	state.ActiveSpanID = "a1b2c3d4e5f67890"
	stateMu.Unlock()
	defer func() {
		stateMu.Lock()
		state.ActiveTraceID = ""
		state.ActiveSpanID = ""
		stateMu.Unlock()
	}()

	if got := activeTraceID(); got != "relay-trace-test123456" {
		t.Fatalf("activeTraceID() = %q", got)
	}
	corr := tracelog.CorrelationFromContext(activeCorrelationCtx())
	if corr.TraceID != "relay-trace-test123456" {
		t.Fatalf("context trace = %q", corr.TraceID)
	}
	if corr.SpanID != "a1b2c3d4e5f67890" {
		t.Fatalf("context span = %q", corr.SpanID)
	}
}

func TestAppendLogLargeVolume(t *testing.T) {
	stateMu.Lock()
	state.Logs = nil
	stateMu.Unlock()

	// Rapidly append many lines.
	for i := 0; i < 100; i++ {
		appendLog(strings.Repeat("a", 100))
	}

	stateMu.Lock()
	defer stateMu.Unlock()
	if len(state.Logs) != 100 {
		t.Errorf("expected 100 lines, got %d", len(state.Logs))
	}
}
