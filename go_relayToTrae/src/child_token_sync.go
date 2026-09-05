package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// persistedContainerTokens mirrors onlineServiceJS container_refresh_token.json
// after child bootstrap exchange-refresh + refresh-access.
type persistedContainerTokens struct {
	TaskID       string `json:"task_id"`
	RefreshToken string `json:"refresh_token"`
	AccessToken  string `json:"access_token"`
	ExpiresAt    string `json:"expires_at"`
	UpdatedAt    string `json:"updated_at"`
}

const childTokenSyncPollInterval = 100 * time.Millisecond

// childTokenSyncTimeout is how long go_relay waits for onlineServiceJS to finish
// first exchange (cloud-parity path). Defaults to 2× tokenExchangeTimeout.
func childTokenSyncTimeout() time.Duration {
	return 2 * tokenExchangeTimeout
}

func containerRefreshTokenStorePath(traeAgentRoot string, env map[string]string) string {
	if root := strings.TrimSpace(env["ONLINE_PROJECT_STATE_ROOT"]); root != "" {
		return filepath.Join(root, "runtime", "container_refresh_token.json")
	}
	return filepath.Join(traeAgentRoot, "onlineProject_state", "runtime", "container_refresh_token.json")
}

func readPersistedContainerTokens(path, taskID string) (persistedContainerTokens, error) {
	var out persistedContainerTokens
	raw, err := os.ReadFile(path)
	if err != nil {
		return out, err
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return out, fmt.Errorf("parse container_refresh_token.json: %w", err)
	}
	wantTask := strings.TrimSpace(taskID)
	gotTask := strings.TrimSpace(out.TaskID)
	if wantTask != "" && gotTask != "" && wantTask != gotTask {
		return out, fmt.Errorf("persisted token task_id mismatch: want %s got %s", wantTask, gotTask)
	}
	if strings.TrimSpace(out.RefreshToken) == "" {
		return out, fmt.Errorf("persisted refresh_token empty")
	}
	if strings.TrimSpace(out.AccessToken) == "" {
		return out, fmt.Errorf("persisted access_token empty (child exchange incomplete)")
	}
	return out, nil
}

func clearPersistedContainerTokens(path string) {
	_ = os.Remove(path)
}

// logsContainTokenPersistFailure detects onlineServiceJS FAIL_PERSIST / TOKEN_PERSIST_FAILED
// lines so relay can surface a distinct message (not confused with TOKEN_ACCESS_INVALID).
func logsContainTokenPersistFailure(logs []string) bool {
	for _, line := range logs {
		if strings.Contains(line, "FAIL_PERSIST") ||
			strings.Contains(line, "TOKEN_PERSIST_FAILED") ||
			strings.Contains(line, "token-persist: FAIL") {
			return true
		}
	}
	return false
}

// formatTokenSyncFailureLog returns the relay log line for child token-sync failure.
// When child already logged a persist failure, emit FAIL_PERSIST with a user-facing hint.
func formatTokenSyncFailureLog(syncErr error, logs []string) string {
	if logsContainTokenPersistFailure(logs) {
		return "[relayToTrae] token-sync: FAIL_PERSIST error_code=TOKEN_PERSIST_FAILED 子进程换票成功但 refresh_token 落盘失败，请检查 ONLINE_PROJECT_STATE_ROOT 磁盘权限"
	}
	errText := ""
	if syncErr != nil {
		errText = syncErr.Error()
	}
	return "[relayToTrae] token-sync: FAIL " + errText
}

// waitForChildContainerTokens polls until onlineServiceJS has written post-exchange tokens.
func waitForChildContainerTokens(path, taskID string, timeout time.Duration) (persistedContainerTokens, error) {
	if timeout <= 0 {
		timeout = childTokenSyncTimeout()
	}
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		tok, err := readPersistedContainerTokens(path, taskID)
		if err == nil {
			return tok, nil
		}
		lastErr = err
		time.Sleep(childTokenSyncPollInterval)
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no persisted tokens")
	}
	return persistedContainerTokens{}, fmt.Errorf(
		"timeout waiting for child token exchange at %s: %w", path, lastErr,
	)
}
