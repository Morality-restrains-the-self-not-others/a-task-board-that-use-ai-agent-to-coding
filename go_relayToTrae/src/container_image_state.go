package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// relayOnlineStateBaseDir is the host directory that holds per-task bind-mount
// roots (…/<taskId>/). Override with RELAY_ONLINE_STATE_BASE.
func relayOnlineStateBaseDir() string {
	base := strings.TrimSpace(os.Getenv("RELAY_ONLINE_STATE_BASE"))
	if base == "" && strings.TrimSpace(repoRoot) != "" {
		// Prefer monorepo path over /tmp so host can clean leftovers when possible.
		base = filepath.Join(repoRoot, "go_relayToTrae", ".relay_online_state")
	}
	if base == "" {
		base = filepath.Join(os.TempDir(), "relay_online_state")
	}
	return base
}

// selectedImageHostStateRoot is the host path bind-mounted to the container's
// ONLINE_PROJECT_STATE_ROOT so go_relay can read container_refresh_token.json
// after the container's exchange-refresh (cloud-parity A1 for selected_image).
func selectedImageHostStateRoot(taskID string) string {
	tid := trimTaskID(taskID)
	if tid == "" {
		tid = "unknown"
	}
	return filepath.Join(relayOnlineStateBaseDir(), tid)
}

const selectedImageContainerStateRoot = "/app/onlineProject_state"

// removeAllHostPath is os.RemoveAll; swapped in tests to force docker fallback.
var removeAllHostPath = os.RemoveAll

// isSafeRelayStateWipeTarget refuses to wipe paths that are not under the
// configured relay online-state base (defense against bad task_id / env).
func isSafeRelayStateWipeTarget(hostPath string) bool {
	clean := filepath.Clean(strings.TrimSpace(hostPath))
	if clean == "" || clean == "." || clean == "/" {
		return false
	}
	base := filepath.Clean(relayOnlineStateBaseDir())
	if base == "" || base == "." || base == "/" {
		return false
	}
	rel, err := filepath.Rel(base, clean)
	if err != nil {
		return false
	}
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return false
	}
	// Only wipe direct children of the base (per-task dirs), never the base itself.
	if strings.Contains(rel, string(os.PathSeparator)) {
		return false
	}
	return true
}

// wipeHostStatePathViaDocker deletes hostPath as root inside a one-shot container
// (needed when prior containers left root-owned files on the bind mount).
func wipeHostStatePathViaDocker(hostPath, wipeImage string) error {
	img := strings.TrimSpace(wipeImage)
	if img == "" {
		return fmt.Errorf("wipe image required for docker state cleanup")
	}
	parent := filepath.Dir(hostPath)
	name := filepath.Base(hostPath)
	if name == "" || name == "." || name == ".." || name == "/" {
		return fmt.Errorf("unsafe wipe basename %q", name)
	}
	out, errText, code, err := runDockerCLI(
		"run", "--rm",
		"--entrypoint", "rm",
		"-v", parent+":/relay_state_wipe",
		img,
		"-rf", "/relay_state_wipe/"+name,
	)
	if err != nil {
		return fmt.Errorf("docker wipe state %s: %w", hostPath, err)
	}
	if code != 0 {
		msg := strings.TrimSpace(errText)
		if msg == "" {
			msg = strings.TrimSpace(out)
		}
		return fmt.Errorf("docker wipe state %s failed (code=%d): %s", hostPath, code, msg)
	}
	if _, err := os.Stat(hostPath); err == nil {
		return fmt.Errorf("docker wipe state %s: path still exists", hostPath)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("docker wipe state %s: stat after wipe: %w", hostPath, err)
	}
	return nil
}

// wipeHostStatePath removes a per-task state directory completely.
// Prefer host RemoveAll; if that fails (root-owned leftovers), use docker rm as root.
func wipeHostStatePath(hostPath, wipeImage string) error {
	hostPath = filepath.Clean(strings.TrimSpace(hostPath))
	if !isSafeRelayStateWipeTarget(hostPath) {
		return fmt.Errorf("refusing to wipe unsafe relay state path %q", hostPath)
	}
	if _, err := os.Stat(hostPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat relay state path %s: %w", hostPath, err)
	}
	if err := removeAllHostPath(hostPath); err == nil {
		if _, err2 := os.Stat(hostPath); os.IsNotExist(err2) {
			return nil
		}
	}
	if err := wipeHostStatePathViaDocker(hostPath, wipeImage); err != nil {
		return fmt.Errorf("clear residual state %s: %w", hostPath, err)
	}
	return nil
}

// clearRelayOnlineStateResiduals wipes the current task dir and any sibling leftovers
// under the state base so each selected_image start begins from an empty mount.
func clearRelayOnlineStateResiduals(taskID, wipeImage string) error {
	hostRoot := selectedImageHostStateRoot(taskID)
	base := filepath.Dir(hostRoot)
	entries, err := os.ReadDir(base)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read relay state base %s: %w", base, err)
	}
	for _, e := range entries {
		p := filepath.Join(base, e.Name())
		if err := wipeHostStatePath(p, wipeImage); err != nil {
			return err
		}
		appendLog(fmt.Sprintf("[relayToTrae] cleared residual state: %s", p))
	}
	return nil
}

func ensureSelectedImageHostStateRoot(taskID, wipeImage string) (hostRoot, refreshStorePath string, err error) {
	if err := clearRelayOnlineStateResiduals(taskID, wipeImage); err != nil {
		return "", "", err
	}
	hostRoot = selectedImageHostStateRoot(taskID)
	runtimeDir := filepath.Join(hostRoot, "runtime")
	if err := os.MkdirAll(runtimeDir, 0o777); err != nil {
		return "", "", fmt.Errorf("create selected-image state root: %w", err)
	}
	// Bind-mount 目录须对容器内 root 可写，且宿主机可读容器写出的 token 文件。
	_ = os.Chmod(hostRoot, 0o777)
	_ = os.Chmod(runtimeDir, 0o777)
	refreshStorePath = filepath.Join(runtimeDir, "container_refresh_token.json")
	clearPersistedContainerTokens(refreshStorePath)
	return hostRoot, refreshStorePath, nil
}

// readPersistedContainerTokensViaDockerExec reads the token file inside the container
// when the host bind-mount is not readable (e.g. root-owned 0600 from older images).
func readPersistedContainerTokensViaDockerExec(containerID, taskID string) (persistedContainerTokens, error) {
	id := strings.TrimSpace(containerID)
	if id == "" {
		return persistedContainerTokens{}, fmt.Errorf("empty container id")
	}
	out, errText, code, err := runDockerCLI(
		"exec", id, "cat", selectedImageContainerStateRoot+"/runtime/container_refresh_token.json",
	)
	if err != nil {
		return persistedContainerTokens{}, err
	}
	if code != 0 {
		msg := strings.TrimSpace(errText)
		if msg == "" {
			msg = strings.TrimSpace(out)
		}
		return persistedContainerTokens{}, fmt.Errorf("docker exec cat token file failed (code=%d): %s", code, msg)
	}
	tmp, err := os.CreateTemp("", "relay-container-token-*.json")
	if err != nil {
		return persistedContainerTokens{}, err
	}
	path := tmp.Name()
	_, _ = tmp.WriteString(out)
	_ = tmp.Close()
	defer os.Remove(path)
	return readPersistedContainerTokens(path, taskID)
}

func waitForSelectedImageContainerTokens(refreshStorePath, containerID, taskID string, timeout time.Duration) (persistedContainerTokens, error) {
	if timeout <= 0 {
		timeout = childTokenSyncTimeout()
	}
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		tok, err := readPersistedContainerTokens(refreshStorePath, taskID)
		if err == nil {
			return tok, nil
		}
		lastErr = err
		// Host bind-mount 不可读时回退 docker exec（兼容旧镜像 0600 落盘）。
		if containerID != "" && (os.IsPermission(err) || strings.Contains(err.Error(), "permission denied")) {
			tok2, err2 := readPersistedContainerTokensViaDockerExec(containerID, taskID)
			if err2 == nil {
				return tok2, nil
			}
			lastErr = err2
		}
		time.Sleep(childTokenSyncPollInterval)
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no persisted tokens")
	}
	return persistedContainerTokens{}, fmt.Errorf(
		"timeout waiting for selected-image token exchange at %s: %w", refreshStorePath, lastErr,
	)
}
