package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// runDockerCLI executes `docker <args...>` and returns stdout/stderr/exit.
// Swapped in tests.
var runDockerCLI = defaultRunDockerCLI

var lookPathDocker = exec.LookPath

func defaultRunDockerCLI(args ...string) (stdout, stderr string, code int, err error) {
	exe, lookErr := lookPathDocker("docker")
	if lookErr != nil {
		return "", "", -1, fmt.Errorf("docker not found: %w", lookErr)
	}
	cmd := exec.Command(exe, args...)
	out, runErr := cmd.CombinedOutput()
	text := string(out)
	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			return "", text, exitErr.ExitCode(), nil
		}
		return "", text, -1, runErr
	}
	return text, "", 0, nil
}

func relayContainerNameForTask(taskID string) (string, error) {
	tid := trimTaskID(taskID)
	if tid == "" {
		return "", fmt.Errorf("task_id required for container name")
	}
	for _, r := range tid {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' || r == '.' {
			continue
		}
		return "", fmt.Errorf("invalid task_id for container name")
	}
	return "relay_taskId_" + tid, nil
}

func startSelectedImageContainer(envPayload map[string]string, tenantID, workspaceID, taskID, image string) (map[string]interface{}, error) {
	image = strings.TrimSpace(image)
	if image == "" {
		return nil, fmt.Errorf("image required")
	}
	if _, err := lookPathDocker("docker"); err != nil {
		return nil, fmt.Errorf("docker not found: %w", err)
	}

	env := expandEnvForRuntime(envPayload, tenantID, workspaceID, taskID)
	for _, key := range requiredEnvKeys {
		if strings.TrimSpace(env[key]) == "" {
			return nil, fmt.Errorf("missing required env: %s", key)
		}
	}
	accessToken := strings.TrimSpace(env["ACCESS_TOKEN"])
	if accessToken == "" {
		return nil, fmt.Errorf("missing required env: ACCESS_TOKEN")
	}
	env["ACCESS_TOKEN"] = accessToken

	port := pickPort()
	if strings.TrimSpace(env["PORT"]) == "" {
		env["PORT"] = strconv.Itoa(port)
	} else if p, err := strconv.Atoi(strings.TrimSpace(env["PORT"])); err == nil && p > 0 {
		port = p
	}
	// Align reachability host-mapped HTTP port with listen port when unset.
	if strings.TrimSpace(env["TRAE_HOST_HTTP_PORT"]) == "" {
		env["TRAE_HOST_HTTP_PORT"] = strconv.Itoa(port)
	}
	// Host-network containers must not advertise 127.0.0.1 as server_url to browsers.
	rewriteLoopbackBusinessAPIForReachability(env, port)
	// taskCloudService 与 host-network 容器同机：register-reachability 的 server_url 须用 127.0.0.1。
	env["TRAE_SAAS_COLOCATED"] = "1"

	containerName, err := relayContainerNameForTask(taskID)
	if err != nil {
		return nil, err
	}

	env["ONLINE_PROJECT_STATE_ROOT"] = selectedImageContainerStateRoot

	// 清掉旧注册，避免换票窗口内继续用 bootstrap 做 status-push。
	stateMu.Lock()
	unregisterTaskLocked(taskID)
	state.TokenSyncPending = true
	stateMu.Unlock()

	appendLog(fmt.Sprintf("[relayToTrae] selected-image start: pull %s", image))
	pullOut, pullErrText, pullCode, pullErr := runDockerCLI("pull", image)
	if pullErr != nil {
		return nil, fmt.Errorf("docker pull failed: %w", pullErr)
	}
	if pullCode != 0 {
		msg := strings.TrimSpace(pullErrText)
		if msg == "" {
			msg = strings.TrimSpace(pullOut)
		}
		appendLog(fmt.Sprintf("[relayToTrae] docker pull failed (code=%d): %s", pullCode, msg))
		return nil, fmt.Errorf("docker pull failed (code=%d): %s", pullCode, msg)
	}
	ident, identErr := inspectImageIdentity(image)
	if identErr != nil {
		appendLog(fmt.Sprintf("[relayToTrae] docker pull ok: %s (inspect failed: %v)", image, identErr))
		ident = imageIdentity{}
	} else {
		appendLog(formatImageIdentityLog(image, ident))
	}

	// Host-network: any leftover relay_taskId_* container keeps PORT bound on the
	// host. Non-root lsof often cannot see that listener, so killPortListeners alone
	// leaves EADDRINUSE for the new container's onlineServiceJS.
	stopOrphanRelayContainers()
	// Also remove by exact name (covers race if filter miss / rename).
	_, _, _, _ = runDockerCLI("rm", "-f", containerName)
	ensureListenPortFree(port)
	if portListening(port) {
		return nil, fmt.Errorf(
			"port %d still in use before docker run (host network); stop the occupying process or relay_taskId_* container and retry",
			port,
		)
	}

	// Wipe prior bind-mount leftovers (layers/logs/job_logs) before docker run.
	// Uses the just-pulled image as wipe tool so root-owned files can be removed
	// without sudo / without pulling an extra alpine image.
	hostStateRoot, refreshStorePath, err := ensureSelectedImageHostStateRoot(taskID, image)
	if err != nil {
		return nil, err
	}

	runArgs := []string{
		"run", "-d", "--rm", "--network", "host",
		"--name", containerName,
		"-v", hostStateRoot + ":" + selectedImageContainerStateRoot,
	}
	// Overlay whole monorepo onlineServiceJS/src (includes new modules like
	// layerFileContent.mjs). Set RELAY_OVERLAY_ONLINE_SERVICE_SRC=0 for image-only.
	runArgs = appendOnlineServiceSrcOverlayMounts(runArgs, repoRoot)
	for k, v := range env {
		if strings.TrimSpace(v) == "" {
			continue
		}
		runArgs = append(runArgs, "-e", fmt.Sprintf("%s=%s", k, v))
	}
	runArgs = append(runArgs, image)

	runLog := fmt.Sprintf("[relayToTrae] docker run image=%s name=%s port=%d state_mount=%s", image, containerName, port, hostStateRoot)
	if ident.ID != "" {
		runLog += fmt.Sprintf(" image_id=%s", ident.ID)
	}
	if ident.Digest != "" {
		runLog += fmt.Sprintf(" image_digest=%s", ident.Digest)
	}
	appendLog(runLog)
	runOut, runErrText, runCode, runErr := runDockerCLI(runArgs...)
	if runErr != nil {
		return nil, fmt.Errorf("docker run failed: %w", runErr)
	}
	if runCode != 0 {
		msg := strings.TrimSpace(runErrText)
		if msg == "" {
			msg = strings.TrimSpace(runOut)
		}
		appendLog(fmt.Sprintf("[relayToTrae] docker run failed (code=%d): %s", runCode, msg))
		return nil, fmt.Errorf("docker run failed (code=%d): %s", runCode, msg)
	}
	containerID := strings.TrimSpace(runOut)
	if containerID == "" {
		return nil, fmt.Errorf("docker run returned empty container id")
	}
	appendLog(fmt.Sprintf("[relayToTrae] container started id=%s name=%s", containerID, containerName))

	// Do NOT fabricate /ui/{token} as the console URL for selected images.
	// Browser must open register-reachability server_url / container_page_url instead.
	appendLog("[relayToTrae] selected-image: ui_url left empty until container registers server_url")

	// Provisional state while container exchanges (do NOT keep bootstrap ACCESS_TOKEN for push).
	stateMu.Lock()
	state.Running = true
	state.PID = 0
	state.Port = port
	state.UIURL = ""
	state.AccessToken = ""
	state.RefreshToken = ""
	state.AccessTokenExpiresAt = time.Time{}
	state.Error = ""
	state.StartedAt = float64(time.Now().UnixNano()) / 1e9
	state.ContainerID = containerID
	state.ContainerName = containerName
	state.Image = image
	state.TokenSyncPending = true
	stateMu.Unlock()

	go followDockerLogs(containerID)

	appendLog("[relayToTrae] token-sync: waiting for selected-image container exchange-refresh (cloud parity)")
	persisted, syncErr := waitForSelectedImageContainerTokens(refreshStorePath, containerID, taskID, childTokenSyncTimeout())
	if syncErr != nil {
		stateMu.Lock()
		logsSnapshot := append([]string(nil), state.Logs...)
		sameContainer := state.ContainerID == containerID
		state.TokenSyncPending = false
		stateMu.Unlock()
		failLine := formatTokenSyncFailureLog(syncErr, logsSnapshot)
		appendLog(failLine)
		if sameContainer {
			_, _, _, _ = runDockerCLI("stop", containerName)
			stateMu.Lock()
			if state.ContainerID == containerID {
				state.Running = false
				state.ContainerID = ""
				state.ContainerName = ""
				state.Image = ""
				if logsContainTokenPersistFailure(logsSnapshot) {
					state.Error = "TOKEN_PERSIST_FAILED: child token persist failed"
				} else {
					state.Error = syncErr.Error()
				}
			}
			stateMu.Unlock()
		}
		return nil, fmt.Errorf("selected-image token exchange sync failed: %w", syncErr)
	}

	syncedAccess := strings.TrimSpace(persisted.AccessToken)
	syncedRefresh := strings.TrimSpace(persisted.RefreshToken)
	syncedExpires := parseAccessTokenExpiresAt(persisted.ExpiresAt)

	stateMu.Lock()
	if state.ContainerID == containerID {
		state.AccessToken = syncedAccess
		state.RefreshToken = syncedRefresh
		state.AccessTokenExpiresAt = syncedExpires
		state.TokenSyncPending = false
	}
	stateMu.Unlock()

	appendLog(fmt.Sprintf(
		"[relayToTrae] token-sync: OK access_token len=%d refresh_token len=%d expires_at_set=%v",
		len(syncedAccess), len(syncedRefresh), !syncedExpires.IsZero(),
	))

	out := map[string]interface{}{
		"running":        true,
		"container_id":   containerID,
		"container_name": containerName,
		"image":          image,
		"mode":           "selected_image",
		"port":           port,
		"ui_url":         "",
	}
	if ident.ID != "" {
		out["image_id"] = ident.ID
	}
	if ident.Digest != "" {
		out["image_digest"] = ident.Digest
	}
	return out, nil
}

func followDockerLogs(containerID string) {
	id := strings.TrimSpace(containerID)
	if id == "" {
		return
	}
	exe, err := lookPathDocker("docker")
	if err != nil {
		return
	}
	cmd := exec.Command(exe, "logs", "-f", id)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return
	}
	cmd.Stderr = cmd.Stdout
	if err := cmd.Start(); err != nil {
		return
	}
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		appendLog(scanner.Text())
	}
	_ = cmd.Wait()
	appendLog(fmt.Sprintf("[relayToTrae] docker logs ended for %s", id))
}

// writeSelectedImageTokenFixture writes a container_refresh_token.json for tests.
func writeSelectedImageTokenFixture(taskID, access, refresh string) error {
	root := selectedImageHostStateRoot(taskID)
	if err := os.MkdirAll(filepath.Join(root, "runtime"), 0o755); err != nil {
		return err
	}
	path := filepath.Join(root, "runtime", "container_refresh_token.json")
	payload := persistedContainerTokens{
		TaskID:       trimTaskID(taskID),
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresAt:    time.Now().Add(time.Hour).UTC().Format(time.RFC3339),
		UpdatedAt:    time.Now().UTC().Format(time.RFC3339),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o600)
}
