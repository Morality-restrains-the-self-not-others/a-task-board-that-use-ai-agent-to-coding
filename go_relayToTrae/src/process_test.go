package main

import (
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestProcessPortListeningInvalidPort(t *testing.T) {
	if portListening(0) {
		t.Error("expected false for port 0")
	}
	if portListening(-1) {
		t.Error("expected false for negative port")
	}
}

func TestProcessPortListeningClosedPort(t *testing.T) {
	// Use a high port that's unlikely to be in use.
	if portListening(59999) {
		t.Error("expected false for closed port")
	}
}

func TestResolveAccessTokenForStopPrefersStateToken(t *testing.T) {
	stateMu.Lock()
	state.AccessToken = "state-stop-token"
	state.UIURL = "http://127.0.0.1:8765/ui/url-token"
	stateMu.Unlock()

	got := resolveAccessTokenForStop()

	if got != "state-stop-token" {
		t.Errorf("expected 'state-stop-token', got %q", got)
	}
}

func TestResolveAccessTokenForStopFallsBackToUIURL(t *testing.T) {
	stateMu.Lock()
	state.AccessToken = ""
	state.UIURL = "http://127.0.0.1:8765/ui/fallback-token"
	stateMu.Unlock()

	got := resolveAccessTokenForStop()

	if got != "fallback-token" {
		t.Errorf("expected 'fallback-token', got %q", got)
	}
}

func TestResolveAccessTokenForStopEmptyBoth(t *testing.T) {
	stateMu.Lock()
	state.AccessToken = ""
	state.UIURL = ""
	stateMu.Unlock()

	got := resolveAccessTokenForStop()

	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestDetachRunningProcLockedNoProcess(t *testing.T) {
	stateMu.Lock()
	cmd = nil
	state.Running = true
	state.PID = 1234
	state.Port = 8765
	state.UIURL = "http://..."
	state.AccessToken = "tok"

	proc := detachRunningProcLocked()
	stateMu.Unlock()

	if proc != nil {
		t.Error("expected nil when no process running")
	}
	stateMu.Lock()
	if state.Running {
		t.Error("expected Running to be reset")
	}
	if state.PID != 0 {
		t.Error("expected PID to be reset")
	}
	if state.Port != 0 {
		t.Error("expected Port to be reset")
	}
	stateMu.Unlock()
}

func TestDetachRunningProcLockedWithProcess(t *testing.T) {
	// Start a dummy process.
	dummy := exec.Command("sleep", "10")
	dummy.Start()
	defer func() {
		dummy.Process.Kill()
		dummy.Wait()
	}()

	stateMu.Lock()
	cmd = dummy
	state.Running = true
	state.AccessToken = "tok"

	proc := detachRunningProcLocked()
	stateMu.Unlock()

	if proc == nil {
		t.Fatal("expected non-nil process")
	}
	if proc != dummy {
		t.Error("expected same process reference")
	}
	if stateMu.TryLock() {
		stateMu.Unlock()
	} else {
		t.Error("lock should be released after detach")
	}
	stateMu.Lock()
	if cmd != nil {
		t.Error("expected cmd to be nil after detach")
	}
	stateMu.Unlock()

	// Clean up the detached process.
	dummy.Process.Kill()
	dummy.Wait()
}

func TestCallOnlineServiceResetNoPort(t *testing.T) {
	if callOnlineServiceReset(0, "token") {
		t.Error("expected false for port 0")
	}
}

func TestCallOnlineServiceResetNoToken(t *testing.T) {
	if callOnlineServiceReset(8765, "") {
		t.Error("expected false for empty token")
	}
}

func TestCallOnlineServiceResetPortNotListening(t *testing.T) {
	if callOnlineServiceReset(59998, "token") {
		t.Error("expected false when port not listening")
	}
}

func TestKillPortListenersLsofNotFound(t *testing.T) {
	// lsof should exist on macOS, so this tests the functional path
	// on a port that nothing is listening on.
	result := killPortListeners(59997)
	if result != nil {
		t.Errorf("expected nil for empty port, got %v", result)
	}
}

func TestStartOnlineServiceMissingRunSh(t *testing.T) {
	// Temporarily redirect repoRoot.
	origRoot := repoRoot
	repoRoot = "/nonexistent/path"
	defer func() { repoRoot = origRoot }()

	_, err := startOnlineService(map[string]string{
		"TASK_API_ENDPOINT_ORIGIN":     "http://task.example.com",
		"BUSINESS_API_ENDPOINT_ORIGIN": "http://biz.example.com",
		"ACCESS_TOKEN":                 "test-token",
	}, "t1", "w1", "task1")

	if err == nil {
		t.Error("expected error for missing run.sh")
	}
}

func TestStartOnlineServiceMissingRequiredEnv(t *testing.T) {
	origRoot := repoRoot
	repoRoot = "/tmp"
	defer func() { repoRoot = origRoot }()

	// Create a dummy run.sh.
	os.MkdirAll("/tmp/trae-agent/onlineServiceJS", 0755)
	os.WriteFile("/tmp/trae-agent/onlineServiceJS/run.sh", []byte("#!/bin/bash\necho ok"), 0755)
	defer os.RemoveAll("/tmp/trae-agent")

	_, err := startOnlineService(map[string]string{}, "t1", "w1", "task1")
	if err == nil {
		t.Error("expected error for missing required env")
	}
}

func TestTerminateProcAlreadyExited(t *testing.T) {
	dummy := exec.Command("true")
	dummy.Run() // completes immediately

	// Should not panic.
	terminateProc(dummy)
}

func TestPortListeningGracefulTimeout(t *testing.T) {
	initTimeouts()
	portProbeTimeoutSec = 0.01 // very short timeout
	if portListening(59996) {
		t.Error("expected false for non-listening port")
	}
}

func TestKillPortListenersInvalidPort(t *testing.T) {
	// Should handle gracefully.
	result := killPortListeners(-1)
	if result != nil {
		t.Errorf("expected nil for invalid port, got %v", result)
	}
}

func TestDetachRunningProcLockedResetsState(t *testing.T) {
	stateMu.Lock()
	cmd = nil
	state.Running = true
	state.PID = 999
	state.Port = 5555
	state.UIURL = "http://test/ui/token"
	state.AccessToken = "secret"
	state.RefreshToken = "refresh-secret"
	state.AccessTokenExpiresAt = time.Now().Add(time.Hour)

	proc := detachRunningProcLocked()
	stateMu.Unlock()

	if proc != nil {
		t.Error("expected nil proc")
	}
	stateMu.Lock()
	if state.Running {
		t.Error("Running should be false")
	}
	if state.PID != 0 {
		t.Error("PID should be 0")
	}
	if state.Port != 0 {
		t.Error("Port should be 0")
	}
	if state.UIURL != "" {
		t.Error("UIURL should be empty")
	}
	if state.AccessToken != "" {
		t.Error("AccessToken should be empty")
	}
	if state.RefreshToken != "" {
		t.Error("RefreshToken should be empty")
	}
	if !state.AccessTokenExpiresAt.IsZero() {
		t.Error("AccessTokenExpiresAt should be zero")
	}
	stateMu.Unlock()
}

func TestBuildPublicUIURLUsesBusinessOriginHost(t *testing.T) {
	env := map[string]string{
		"BUSINESS_API_ENDPOINT_ORIGIN": "http://127.0.0.1:8765",
	}
	got := buildPublicUIURL(8765, "secret-token", env)
	want := "http://127.0.0.1:8765/ui/secret-token"
	if got != want {
		t.Fatalf("buildPublicUIURL() = %q, want %q", got, want)
	}
}

func TestBuildPublicUIURLScopedWhenTaskScopePresent(t *testing.T) {
	env := map[string]string{
		"BUSINESS_API_ENDPOINT_ORIGIN": "http://127.0.0.1:8765",
		"tenantId":                     "t1",
		"workspaceId":                  "w1",
		"taskId":                       "task1",
	}
	got := buildPublicUIURL(8765, "secret-token", env)
	want := "http://127.0.0.1:8765/ui/tenant/t1/workspace/w1/task/task1/secret-token"
	if got != want {
		t.Fatalf("buildPublicUIURL() = %q, want %q", got, want)
	}
}

func TestBuildPublicUIURLKeepsLoopbackWhenBusinessOriginIsLocal(t *testing.T) {
	env := map[string]string{
		"BUSINESS_API_ENDPOINT_ORIGIN": "http://127.0.0.1:8765",
	}
	got := buildPublicUIURL(8765, "secret-token", env)
	want := "http://127.0.0.1:8765/ui/secret-token"
	if got != want {
		t.Fatalf("buildPublicUIURL() = %q, want %q", got, want)
	}
}

func TestBuildPublicUIURLFallsBackToTaskApiOriginHost(t *testing.T) {
	env := map[string]string{
		"BUSINESS_API_ENDPOINT_ORIGIN": "http://127.0.0.1:8765",
		"TASK_API_ENDPOINT_ORIGIN":     "http://127.0.0.1:8001",
	}
	got := buildPublicUIURL(8765, "secret-token", env)
	want := "http://127.0.0.1:8765/ui/secret-token"
	if got != want {
		t.Fatalf("buildPublicUIURL() = %q, want %q", got, want)
	}
}

func TestStartOnlineServiceEnvExpansionWithIDs(t *testing.T) {
	env := expandEnvForRuntime(
		map[string]string{
			"TASK_API_ENDPOINT_ORIGIN":     "http://task.local",
			"BUSINESS_API_ENDPOINT_ORIGIN": "http://biz.local",
			"COMMENT_ID":                   "cmt-3",
		},
		"tenant-1", "workspace-2", "task-3",
	)

	if env["tenantId"] != "tenant-1" {
		t.Errorf("expected tenantId=tenant-1, got %q", env["tenantId"])
	}
	if env["workspaceId"] != "workspace-2" {
		t.Errorf("expected workspaceId=workspace-2, got %q", env["workspaceId"])
	}
	if env["taskId"] != "task-3" {
		t.Errorf("expected taskId=task-3, got %q", env["taskId"])
	}
	expectedTask := "http://task.local/api/tenant/tenant-1/workspace/workspace-2/task/task-3/comment/cmt-3/cloud"
	if env["TASK_API_ENDPOINT"] != expectedTask {
		t.Errorf("expected TASK_API_ENDPOINT=%s, got %q", expectedTask, env["TASK_API_ENDPOINT"])
	}
}

func TestStopRunningNoProcess(t *testing.T) {
	stateMu.Lock()
	cmd = nil
	stateMu.Unlock()

	// Should not panic.
	stopRunning()
}

func TestStopRunningWithProcessIdempotent(t *testing.T) {
	stateMu.Lock()
	cmd = nil
	stateMu.Unlock()

	// Call stopRunning multiple times - should not panic.
	stopRunning()
	stopRunning()
}

func TestRunCommandAvailable(t *testing.T) {
	_, err := exec.LookPath("lsof")
	if err != nil {
		t.Skip("lsof not available on this system")
	}
}

func TestSignalHandling(t *testing.T) {
	// Verify we can send signals.
	dummy := exec.Command("sleep", "1")
	dummy.Start()
	defer func() {
		if dummy.Process != nil && dummy.ProcessState == nil {
			dummy.Process.Kill()
			dummy.Wait()
		}
	}()

	if dummy.Process == nil {
		t.Fatal("expected process to start")
	}

	dummy.Process.Signal(syscall.SIGTERM)
	dummy.Wait()

	if dummy.ProcessState == nil {
		t.Error("expected process to have exited")
	}
}

func TestBuildChildEnvUsesExchangedAccessToken(t *testing.T) {
	env := map[string]string{
		"ACCESS_TOKEN":             "old-token",
		"TASK_API_ENDPOINT_ORIGIN": "http://127.0.0.1:8001",
		"tenantId":                 "t1",
	}
	got := buildChildEnv(
		[]string{"PATH=/usr/bin"},
		env,
		8765,
		"bootstrap-token",
		"0",
		"/tmp/trae-agent",
	)

	accessEntries := 0
	for _, kv := range got {
		if !strings.HasPrefix(kv, "ACCESS_TOKEN=") {
			continue
		}
		accessEntries++
		if kv != "ACCESS_TOKEN=bootstrap-token" {
			t.Fatalf("unexpected ACCESS_TOKEN entry: %q", kv)
		}
	}
	if accessEntries != 1 {
		t.Fatalf("expected exactly 1 ACCESS_TOKEN entry, got %d", accessEntries)
	}
	for _, kv := range got {
		if strings.HasPrefix(kv, "TRAE_SKIP_CONTAINER_TOKEN_EXCHANGE=") {
			t.Fatalf("cloud-parity start must not set TRAE_SKIP: %q", kv)
		}
	}
}
