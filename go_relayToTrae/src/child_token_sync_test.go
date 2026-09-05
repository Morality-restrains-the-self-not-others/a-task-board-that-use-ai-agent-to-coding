package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestContainerRefreshTokenStorePathDefault(t *testing.T) {
	got := containerRefreshTokenStorePath("/repo/trae-agent", map[string]string{})
	want := filepath.Join("/repo/trae-agent", "onlineProject_state", "runtime", "container_refresh_token.json")
	if got != want {
		t.Fatalf("path=%q want %q", got, want)
	}
}

func TestContainerRefreshTokenStorePathHonorsStateRoot(t *testing.T) {
	got := containerRefreshTokenStorePath("/repo/trae-agent", map[string]string{
		"ONLINE_PROJECT_STATE_ROOT": "/tmp/state-root",
	})
	want := filepath.Join("/tmp/state-root", "runtime", "container_refresh_token.json")
	if got != want {
		t.Fatalf("path=%q want %q", got, want)
	}
}

func TestWaitForChildContainerTokensOK(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "container_refresh_token.json")
	go func() {
		time.Sleep(50 * time.Millisecond)
		_ = os.WriteFile(path, []byte(`{"task_id":"task-1","refresh_token":"rt-abc","access_token":"at-xyz","expires_at":"2099-01-01 00:00:00"}`+"\n"), 0o600)
	}()
	got, err := waitForChildContainerTokens(path, "task-1", 2*time.Second)
	if err != nil {
		t.Fatalf("wait: %v", err)
	}
	if got.AccessToken != "at-xyz" || got.RefreshToken != "rt-abc" {
		t.Fatalf("got=%+v", got)
	}
}

func TestWaitForChildContainerTokensTimeout(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "missing.json")
	_, err := waitForChildContainerTokens(path, "task-1", 200*time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "timeout") {
		t.Fatalf("err=%v", err)
	}
}

func TestReadPersistedContainerTokensTaskMismatch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "container_refresh_token.json")
	_ = os.WriteFile(path, []byte(`{"task_id":"other","refresh_token":"rt","access_token":"at"}`), 0o600)
	_, err := readPersistedContainerTokens(path, "task-1")
	if err == nil || !strings.Contains(err.Error(), "mismatch") {
		t.Fatalf("err=%v", err)
	}
}

func TestBuildChildEnvOmitsSkipTokenExchange(t *testing.T) {
	got := buildChildEnv(
		[]string{"PATH=/usr/bin"},
		map[string]string{
			"ACCESS_TOKEN":                       "should-not-win",
			"TRAE_SKIP_CONTAINER_TOKEN_EXCHANGE": "1",
			"tenantId":                           "t1",
		},
		8765,
		"bootstrap-tok",
		"0",
		"/tmp/trae-agent",
	)
	for _, kv := range got {
		if strings.HasPrefix(kv, "TRAE_SKIP_CONTAINER_TOKEN_EXCHANGE=") {
			t.Fatalf("must not inject skip flag: %q", kv)
		}
		if kv == "ACCESS_TOKEN=should-not-win" {
			t.Fatal("payload ACCESS_TOKEN must not override bootstrap arg")
		}
	}
	found := false
	for _, kv := range got {
		if kv == "ACCESS_TOKEN=bootstrap-tok" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected ACCESS_TOKEN=bootstrap-tok")
	}
}

func TestLogsContainTokenPersistFailure(t *testing.T) {
	if logsContainTokenPersistFailure([]string{"[onlineServiceJS] token-exchange: done"}) {
		t.Fatal("expected false for success logs")
	}
	if !logsContainTokenPersistFailure([]string{
		"[onlineServiceJS] token-exchange: FAIL_PERSIST error_code=TOKEN_PERSIST_FAILED token-persist: FAIL write /tmp/x: EACCES",
	}) {
		t.Fatal("expected true for FAIL_PERSIST")
	}
	if !logsContainTokenPersistFailure([]string{"token-persist: FAIL write /x"}) {
		t.Fatal("expected true for token-persist: FAIL")
	}
}

func TestFormatTokenSyncFailureLogPersistVsGeneric(t *testing.T) {
	persistLine := formatTokenSyncFailureLog(fmt.Errorf("timeout"), []string{
		"[onlineServiceJS] token-exchange: FAIL_PERSIST error_code=TOKEN_PERSIST_FAILED boom",
	})
	if !strings.Contains(persistLine, "FAIL_PERSIST") || !strings.Contains(persistLine, "TOKEN_PERSIST_FAILED") {
		t.Fatalf("persist line=%q", persistLine)
	}
	if !strings.Contains(persistLine, "落盘失败") {
		t.Fatalf("expected 落盘失败 hint, got %q", persistLine)
	}
	generic := formatTokenSyncFailureLog(fmt.Errorf("timeout waiting"), []string{
		"[onlineServiceJS] token-exchange: FAIL HTTP 401 TOKEN_ACCESS_INVALID",
	})
	if strings.Contains(generic, "FAIL_PERSIST") {
		t.Fatalf("must not tag access-invalid as persist: %q", generic)
	}
	if !strings.Contains(generic, "token-sync: FAIL timeout waiting") {
		t.Fatalf("generic line=%q", generic)
	}
}
