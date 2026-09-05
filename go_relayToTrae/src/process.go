package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var (
	// cmd is the currently running onlineServiceJS subprocess.
	// Protected by stateMu when being inspected/replaced.
	cmd *exec.Cmd
)

func portListening(port int) bool {
	if port <= 0 {
		return false
	}
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	timeout := time.Duration(portProbeTimeoutSec * float64(time.Second))
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// portListenerPIDs returns PIDs that are LISTENing on TCP port (not clients connected to it).
// Plain `lsof -ti :PORT` on macOS also matches inbound HTTP clients (e.g. Django calling
// onlineServiceJS reset), which caused task2app to hang after relay stop.
func portListenerPIDs(port int) []string {
	if port <= 0 {
		return nil
	}
	portTCP := fmt.Sprintf(":%d", port)
	out, err := exec.Command("lsof", "-tiTCP"+portTCP, "-sTCP:LISTEN").Output()
	if err != nil {
		return nil
	}
	return strings.Fields(string(out))
}

// listPortListenerPIDs is swapped in tests to simulate host-network Docker listeners
// that non-root lsof cannot see.
var listPortListenerPIDs = portListenerPIDs

func onlineServicePortForCleanup() int {
	port := pickPort()
	stateMu.Lock()
	if state.Port > 0 {
		port = state.Port
	}
	stateMu.Unlock()
	return port
}

// ensureOnlineServicePortFree kills orphan listeners after stop or before restart.
func ensureOnlineServicePortFree() {
	ensureListenPortFree(onlineServicePortForCleanup())
}

// ensureListenPortFree frees a TCP listen port used by host-network onlineServiceJS.
// Host-network Docker listeners are often invisible to non-root `lsof`, so after
// killing visible PIDs we also remove leftover `relay_taskId_*` containers.
func ensureListenPortFree(port int) {
	if port <= 0 {
		return
	}
	if !portListening(port) {
		return
	}
	appendLog(fmt.Sprintf("[relayToTrae] port %d occupied; cleaning host listeners and orphan relay containers", port))
	killPortListeners(port)
	if !portListening(port) {
		return
	}
	if _, err := lookPathDocker("docker"); err != nil {
		appendLog(fmt.Sprintf("[relayToTrae] port %d still occupied and docker unavailable: %v", port, err))
		return
	}
	stopOrphanRelayContainers()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if !portListening(port) {
			appendLog(fmt.Sprintf("[relayToTrae] port %d freed after orphan relay container cleanup", port))
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	appendLog(fmt.Sprintf("[relayToTrae] port %d still occupied after cleanup", port))
}

func killPortListeners(port int) []string {
	pids := listPortListenerPIDs(port)
	if len(pids) == 0 {
		return nil
	}
	appendLog(fmt.Sprintf("[relayToTrae] port %d is occupied, killing listener processes: %s", port, strings.Join(pids, ", ")))

	for _, pidStr := range pids {
		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			continue
		}
		if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
			appendLog(fmt.Sprintf("[relayToTrae] SIGTERM pid=%d failed: %v", pid, err))
		}
	}

	time.Sleep(500 * time.Millisecond)

	remaining := listPortListenerPIDs(port)
	for _, pidStr := range remaining {
		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			continue
		}
		if err := syscall.Kill(pid, syscall.SIGKILL); err != nil {
			appendLog(fmt.Sprintf("[relayToTrae] SIGKILL pid=%d failed: %v", pid, err))
		}
	}
	return pids
}

// stopOrphanRelayContainers force-removes all containers named relay_taskId_*.
// selected_image uses --network host; a leftover container keeps PORT bound on the
// host even when go_relay state no longer tracks it, and non-root lsof often cannot
// see that listener — causing onlineServiceJS EADDRINUSE on the next start.
func stopOrphanRelayContainers() []string {
	if _, err := lookPathDocker("docker"); err != nil {
		return nil
	}
	out, errText, code, err := runDockerCLI("ps", "-aq", "--filter", "name=relay_taskId_")
	if err != nil {
		appendLog(fmt.Sprintf("[relayToTrae] list orphan relay containers failed: %v", err))
		return nil
	}
	if code != 0 {
		msg := strings.TrimSpace(errText)
		if msg == "" {
			msg = strings.TrimSpace(out)
		}
		appendLog(fmt.Sprintf("[relayToTrae] list orphan relay containers failed (code=%d): %s", code, msg))
		return nil
	}
	ids := strings.Fields(strings.TrimSpace(out))
	if len(ids) == 0 {
		return nil
	}
	appendLog(fmt.Sprintf("[relayToTrae] removing orphan relay containers: %s", strings.Join(ids, ", ")))
	stopped := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		_, rmErrText, rmCode, rmErr := runDockerCLI("rm", "-f", id)
		if rmErr != nil {
			appendLog(fmt.Sprintf("[relayToTrae] docker rm -f %s error: %v", id, rmErr))
			continue
		}
		if rmCode != 0 {
			msg := strings.TrimSpace(rmErrText)
			appendLog(fmt.Sprintf("[relayToTrae] docker rm -f %s code=%d: %s", id, rmCode, msg))
			continue
		}
		stopped = append(stopped, id)
	}
	return stopped
}

func callOnlineServiceReset(port int, accessToken string) bool {
	token := strings.TrimSpace(accessToken)
	if port <= 0 || token == "" {
		return false
	}
	if !portListening(port) {
		return false
	}

	resetURL := fmt.Sprintf("http://127.0.0.1:%d/api/jobs/reset", port)
	req, err := http.NewRequest("POST", resetURL, bytes.NewReader([]byte("{}")))
	if err != nil {
		return false
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Access-Token", token)

	appendLog(fmt.Sprintf("[relayToTrae] requesting onlineServiceJS reset: %s", resetURL))

	client := &http.Client{
		Transport: &http.Transport{Proxy: nil},
		Timeout:   time.Duration(resetTimeoutSec * float64(time.Second)),
	}

	resp, err := client.Do(req)
	if err != nil {
		appendLog(fmt.Sprintf("[relayToTrae] onlineServiceJS reset failed: %v", err))
		return false
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		appendLog(fmt.Sprintf("[relayToTrae] onlineServiceJS reset HTTP failed (%d): %s", resp.StatusCode, string(respBody)))
		return false
	}

	var payload map[string]interface{}
	if json.Unmarshal(respBody, &payload) != nil {
		payload = nil
	}
	if payload != nil && len(payload) > 0 {
		appendLog(fmt.Sprintf("[relayToTrae] onlineServiceJS reset complete: %v", payload))
	} else {
		appendLog("[relayToTrae] onlineServiceJS reset complete")
	}
	return true
}

func resolveAccessTokenForStop() string {
	stateMu.Lock()
	defer stateMu.Unlock()

	token := strings.TrimSpace(state.AccessToken)
	if token != "" {
		return token
	}
	return extractAccessTokenFromUIURL(state.UIURL)
}

func extractAccessTokenFromUIURL(uiURL string) string {
	text := strings.TrimSpace(uiURL)
	if text == "" {
		return ""
	}
	// Parse the path from something like:
	//   http://127.0.0.1:8765/ui/{token}
	//   http://127.0.0.1:8765/ui/tenant/{t}/workspace/{w}/task/{task}/{token}
	idx := strings.Index(text, "://")
	if idx < 0 {
		return ""
	}
	rest := text[idx+3:]
	slashIdx := strings.IndexByte(rest, '/')
	if slashIdx < 0 {
		return ""
	}
	path := strings.Trim(rest[slashIdx:], "/")
	parts := strings.Split(path, "/")
	if len(parts) >= 8 &&
		parts[0] == "ui" &&
		parts[1] == "tenant" &&
		parts[3] == "workspace" &&
		parts[5] == "task" {
		return parts[7]
	}
	if len(parts) >= 2 && parts[0] == "ui" {
		return parts[1]
	}
	return ""
}

func isLoopbackHost(host string) bool {
	h := strings.ToLower(strings.TrimSpace(host))
	return h == "localhost" || h == "127.0.0.1" || h == "::1" || h == "0.0.0.0"
}

// hostFromHTTPOrigin extracts hostname from an http(s) origin/URL; empty if unusable.
func hostFromHTTPOrigin(raw string) string {
	origin := extractOrigin(raw)
	if origin == "" {
		return ""
	}
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Host == "" {
		return ""
	}
	host, _, splitErr := net.SplitHostPort(parsed.Host)
	if splitErr != nil {
		host = parsed.Host
	}
	return strings.TrimSpace(host)
}

// rewriteLoopbackBusinessAPIForReachability replaces 127.0.0.1/localhost BUSINESS_API*
// with a browser-reachable origin so register-reachability does not advertise loopback.
// Prefer RELAY_TO_TRAE_PUBLIC_ORIGIN / BUSINESS conf origin, else TRAE_PUBLIC_IP host,
// else non-loopback host from TASK_API_ENDPOINT_ORIGIN.
func rewriteLoopbackBusinessAPIForReachability(env map[string]string, listenPort int) {
	if env == nil {
		return
	}
	bizHost := hostFromHTTPOrigin(env["BUSINESS_API_ENDPOINT_ORIGIN"])
	if bizHost == "" {
		bizHost = hostFromHTTPOrigin(env["BUSINESS_API_ENDPOINT"])
	}
	if bizHost == "" || !isLoopbackHost(bizHost) {
		return
	}

	origin := strings.TrimSpace(os.Getenv("RELAY_TO_TRAE_PUBLIC_ORIGIN"))
	if origin == "" || isLoopbackHost(hostFromHTTPOrigin(origin)) {
		origin = strings.TrimSpace(os.Getenv("RELAY_TO_TRAE_BUSINESS_API_ORIGIN"))
	}
	if origin == "" || isLoopbackHost(hostFromHTTPOrigin(origin)) {
		pip := strings.TrimSpace(env["TRAE_PUBLIC_IP"])
		if pip == "" {
			pip = strings.TrimSpace(env["PUBLIC_IP"])
		}
		if pip == "" || isLoopbackHost(pip) {
			pip = strings.TrimSpace(os.Getenv("TRAE_PUBLIC_IP"))
			if pip == "" {
				pip = strings.TrimSpace(os.Getenv("PUBLIC_IP"))
			}
		}
		if pip == "" || isLoopbackHost(pip) {
			taskHost := hostFromHTTPOrigin(env["TASK_API_ENDPOINT_ORIGIN"])
			if taskHost == "" {
				taskHost = hostFromHTTPOrigin(env["TASK_API_ENDPOINT"])
			}
			if taskHost != "" && !isLoopbackHost(taskHost) {
				pip = taskHost
			}
		}
		if pip == "" || isLoopbackHost(pip) {
			appendLog("[relayToTrae] BUSINESS_API is loopback but no public host available; leave as-is")
			return
		}
		origin = publicReachableOriginFromHost(pip, listenPort)
	}
	origin = strings.TrimRight(strings.TrimSpace(origin), "/")
	if origin == "" {
		return
	}
	host := hostFromHTTPOrigin(origin)
	if host == "" {
		host = strings.TrimSpace(env["TRAE_PUBLIC_IP"])
	}
	api := origin + "/api"
	if host != "" {
		env["TRAE_PUBLIC_IP"] = host
	}
	env["BUSINESS_API_ENDPOINT_ORIGIN"] = origin
	env["BUSINESS_API_ENDPOINT"] = api
	env["BusinessApiEndPoint"] = api
	appendLog(fmt.Sprintf("[relayToTrae] rewrote loopback BUSINESS_API → %s (reachability)", origin))
}

// publicReachableOriginFromHost builds a public origin from a host or IP.
// Domain hostnames use https without port; IP literals keep http://ip:port for published maps.
func publicReachableOriginFromHost(host string, listenPort int) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return ""
	}
	if strings.Contains(host, "://") {
		return strings.TrimRight(host, "/")
	}
	if ip := net.ParseIP(host); ip != nil {
		if listenPort <= 0 {
			listenPort = 8765
		}
		return fmt.Sprintf("http://%s:%d", host, listenPort)
	}
	scheme := strings.TrimSpace(os.Getenv("PUBLIC_SCHEME"))
	if scheme == "" {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s", scheme, host)
}

func uiURLHostFromEnv(env map[string]string) string {
	for _, key := range []string{"BUSINESS_API_ENDPOINT_ORIGIN", "TASK_API_ENDPOINT_ORIGIN"} {
		bizOrigin := extractOrigin(env[key])
		if bizOrigin == "" {
			continue
		}
		parsed, err := url.Parse(bizOrigin)
		if err != nil || parsed.Host == "" {
			continue
		}
		host, _, splitErr := net.SplitHostPort(parsed.Host)
		if splitErr != nil {
			host = parsed.Host
		}
		if host != "" && !isLoopbackHost(host) {
			return host
		}
	}
	return "127.0.0.1"
}

func buildPublicUIURL(port int, accessToken string, env map[string]string) string {
	host := uiURLHostFromEnv(env)
	token := strings.TrimSpace(accessToken)
	tenant := strings.TrimSpace(env["tenantId"])
	if tenant == "" {
		tenant = strings.TrimSpace(env["TENANT_ID"])
	}
	workspace := strings.TrimSpace(env["workspaceId"])
	if workspace == "" {
		workspace = strings.TrimSpace(env["WORKSPACE_ID"])
	}
	task := strings.TrimSpace(env["taskId"])
	if task == "" {
		task = strings.TrimSpace(env["TASK_ID"])
	}
	if tenant != "" && workspace != "" && task != "" && token != "" {
		return fmt.Sprintf(
			"http://%s:%d/ui/tenant/%s/workspace/%s/task/%s/%s",
			host, port,
			url.PathEscape(tenant),
			url.PathEscape(workspace),
			url.PathEscape(task),
			url.PathEscape(token),
		)
	}
	return fmt.Sprintf("http://%s:%d/ui/%s", host, port, token)
}

func stopRunning() {
	stateMu.Lock()
	containerName := strings.TrimSpace(state.ContainerName)
	containerID := strings.TrimSpace(state.ContainerID)
	proc := detachRunningProcLocked()
	state.ContainerID = ""
	state.ContainerName = ""
	state.Image = ""
	stateMu.Unlock()

	target := containerName
	if target == "" {
		target = containerID
	}
	if target != "" {
		appendLog(fmt.Sprintf("[relayToTrae] docker stop %s", target))
		_, errText, code, err := runDockerCLI("stop", target)
		if err != nil {
			appendLog(fmt.Sprintf("[relayToTrae] docker stop error: %v", err))
		} else if code != 0 {
			appendLog(fmt.Sprintf("[relayToTrae] docker stop code=%d: %s", code, strings.TrimSpace(errText)))
		}
	}
	if proc != nil {
		terminateProc(proc)
	}
}

func detachRunningProcLocked() *exec.Cmd {
	proc := cmd
	cmd = nil

	state.Running = false
	state.PID = 0
	state.Port = 0
	state.UIURL = ""
	state.AccessToken = ""
	state.RefreshToken = ""
	state.AccessTokenExpiresAt = time.Time{}
	state.ActiveTaskID = ""
	state.ActiveTraceID = ""
	state.ActiveSpanID = ""

	if proc != nil && proc.Process != nil && proc.ProcessState == nil {
		return proc
	}
	return nil
}

func terminateProc(proc *exec.Cmd) {
	appendLog("[relayToTrae] stopping onlineServiceJS...")
	if proc.Process == nil {
		return
	}

	proc.Process.Signal(syscall.SIGTERM)

	done := make(chan error, 1)
	go func() {
		done <- proc.Wait()
	}()

	select {
	case <-done:
		return
	case <-time.After(time.Duration(procTermTimeoutSec * float64(time.Second))):
		proc.Process.Kill()
		select {
		case <-done:
		case <-time.After(time.Duration(procKillTimeoutSec * float64(time.Second))):
		}
	}
}

func buildChildEnv(baseEnv []string, env map[string]string, port int, accessToken, traeDocker, resolvedRepoRoot string) []string {
	childEnv := make([]string, 0, len(baseEnv)+8)
	for _, kv := range baseEnv {
		if strings.HasPrefix(strings.ToUpper(kv), "TRAE_SKIP_CONTAINER_TOKEN_EXCHANGE=") {
			continue
		}
		if strings.HasPrefix(strings.ToUpper(kv), "ACCESS_TOKEN=") {
			continue
		}
		childEnv = append(childEnv, kv)
	}
	childEnv = append(childEnv, "PORT="+strconv.Itoa(port))
	childEnv = append(childEnv, "ACCESS_TOKEN="+accessToken)
	childEnv = append(childEnv, "TRAE_ONLINE_JS_DOCKER="+traeDocker)
	// Cloud parity: do NOT set TRAE_SKIP_CONTAINER_TOKEN_EXCHANGE — onlineServiceJS
	// must perform exchange-refresh like UserData-started containers.
	childEnv = append(childEnv, "REPO_ROOT="+resolvedRepoRoot)

	for k, v := range env {
		key := strings.TrimSpace(k)
		if strings.EqualFold(key, "ACCESS_TOKEN") {
			continue
		}
		// Strip skip flag from payload so mock start cannot silently diverge from cloud.
		if strings.EqualFold(key, "TRAE_SKIP_CONTAINER_TOKEN_EXCHANGE") {
			continue
		}
		childEnv = append(childEnv, k+"="+v)
	}
	return childEnv
}

func startOnlineService(envPayload map[string]string, tenantID, workspaceID, taskID string) (map[string]interface{}, error) {
	runSh := onlineServiceRunSh(repoRoot)
	if _, err := os.Stat(runSh); err != nil {
		return nil, fmt.Errorf("%s not found", runSh)
	}

	port := pickPort()
	env := expandEnvForRuntime(envPayload, tenantID, workspaceID, taskID)

	for _, key := range requiredEnvKeys {
		if strings.TrimSpace(env[key]) == "" {
			return nil, fmt.Errorf("missing required env: %s", key)
		}
	}

	// Bootstrap ACCESS_TOKEN from CRED /v1/token/init (same as cloud UserData).
	// First exchange-refresh is performed by the child onlineServiceJS process.
	accessToken := strings.TrimSpace(env["ACCESS_TOKEN"])
	if accessToken == "" {
		return nil, fmt.Errorf("missing required env: ACCESS_TOKEN")
	}
	env["ACCESS_TOKEN"] = accessToken

	traeAgentRoot := filepath.Join(repoRoot, "trae-agent")
	refreshStorePath := containerRefreshTokenStorePath(traeAgentRoot, env)
	clearPersistedContainerTokens(refreshStorePath)

	// Build child environment.
	traeDocker := os.Getenv("TRAE_ONLINE_JS_DOCKER")
	if traeDocker == "" {
		traeDocker = "0"
	}
	childEnv := buildChildEnv(
		os.Environ(),
		env,
		port,
		accessToken,
		traeDocker,
		traeAgentRoot,
	)

	// Log important env keys (masking tokens).
	var envLogParts []string
	for _, k := range []string{"TASK_API_ENDPOINT_ORIGIN", "BUSINESS_API_ENDPOINT_ORIGIN", "ACCESS_TOKEN", "TASK_API_ENDPOINT", "BUSINESS_API_ENDPOINT", "tenantId", "workspaceId", "taskId"} {
		if v, ok := env[k]; ok {
			display := v
			if strings.Contains(strings.ToUpper(k), "TOKEN") {
				display = "***"
			}
			envLogParts = append(envLogParts, fmt.Sprintf("%s=%s", k, display))
		}
	}
	appendLog(fmt.Sprintf("[relayToTrae] start: bash %s", runSh))
	appendLog("[relayToTrae] env: " + strings.Join(envLogParts, ", "))
	appendLog("[relayToTrae] token-sync: waiting for child onlineServiceJS exchange-refresh (cloud parity)")

	proc := exec.Command("bash", runSh)
	proc.Dir = filepath.Dir(runSh)
	proc.Env = childEnv

	stdout, err := proc.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	proc.Stderr = proc.Stdout // merge stderr into stdout

	if err := proc.Start(); err != nil {
		return nil, fmt.Errorf("failed to start subprocess: %w", err)
	}

	// Provisional state while child exchanges (bootstrap token; UI URL updated after sync).
	uiURL := buildPublicUIURL(port, accessToken, env)
	stateMu.Lock()
	cmd = proc
	state.Running = true
	state.PID = proc.Process.Pid
	state.Port = port
	state.UIURL = uiURL
	state.AccessToken = accessToken
	state.RefreshToken = ""
	state.AccessTokenExpiresAt = time.Time{}
	state.Error = ""
	state.StartedAt = float64(time.Now().UnixNano()) / 1e9
	stateMu.Unlock()

	// 子进程启动（Running=true）事件驱动推送状态（OPT-20260816-032）。
	pushAllRegisteredTasks()

	// Read subprocess output in background.
	go func() {
		grouper := newSubprocessLogGrouper()
		scanner := bufio.NewScanner(stdout)
		// feature-params-env / init 快照等长 JSON 行可能超过默认 64KiB token 上限
		scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
		for scanner.Scan() {
			for _, block := range grouper.Feed(scanner.Text()) {
				appendSubprocessLog(block)
			}
		}
		for _, block := range grouper.Flush() {
			appendSubprocessLog(block)
		}
		proc.Wait()

		exitCode := 0
		if proc.ProcessState != nil {
			exitCode = proc.ProcessState.ExitCode()
		}

		stateMu.Lock()
		// Only update state if cmd still points to THIS process.
		// If a new process was started, the new startOnlineService call
		// already set state.Running=true for the new cmd.
		if cmd == proc {
			state.Running = false
			state.PID = 0
			if exitCode != 0 {
				state.Error = fmt.Sprintf("onlineServiceJS exited (code=%d)", exitCode)
			}
		}
		stateMu.Unlock()

		// 子进程退出（Running=false）事件驱动推送终态（OPT-20260816-032）。
		pushAllRegisteredTasks()

		appendLog(fmt.Sprintf("[relayToTrae] onlineServiceJS process ended, exit code %d", exitCode))
	}()

	persisted, syncErr := waitForChildContainerTokens(refreshStorePath, taskID, childTokenSyncTimeout())
	if syncErr != nil {
		stateMu.Lock()
		logsSnapshot := append([]string(nil), state.Logs...)
		sameProc := cmd == proc
		stateMu.Unlock()
		failLine := formatTokenSyncFailureLog(syncErr, logsSnapshot)
		appendLog(failLine)
		if sameProc {
			terminateProc(proc)
			stateMu.Lock()
			if cmd == proc {
				cmd = nil
				state.Running = false
				state.PID = 0
				if logsContainTokenPersistFailure(logsSnapshot) {
					state.Error = "TOKEN_PERSIST_FAILED: child token persist failed"
				} else {
					state.Error = syncErr.Error()
				}
			}
			stateMu.Unlock()
			// 换票失败导致子进程终止（Running=false）事件驱动推送错误终态（OPT-20260816-032）。
			pushAllRegisteredTasks()
		}
		return nil, fmt.Errorf("child token exchange sync failed: %w", syncErr)
	}

	syncedAccess := strings.TrimSpace(persisted.AccessToken)
	syncedRefresh := strings.TrimSpace(persisted.RefreshToken)
	syncedExpires := parseAccessTokenExpiresAt(persisted.ExpiresAt)
	uiURL = buildPublicUIURL(port, syncedAccess, env)
	env["ACCESS_TOKEN"] = syncedAccess

	stateMu.Lock()
	if cmd == proc {
		state.AccessToken = syncedAccess
		state.RefreshToken = syncedRefresh
		state.AccessTokenExpiresAt = syncedExpires
		state.UIURL = uiURL
	}
	stateMu.Unlock()

	appendLog(fmt.Sprintf(
		"[relayToTrae] token-sync: OK access_token len=%d refresh_token len=%d expires_at_set=%v",
		len(syncedAccess), len(syncedRefresh), !syncedExpires.IsZero(),
	))

	result := map[string]interface{}{
		"port":   float64(port),
		"ui_url": uiURL,
		"pid":    float64(proc.Process.Pid),
	}
	return result, nil
}
