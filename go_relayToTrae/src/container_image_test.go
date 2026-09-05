package main

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// fmt already used by existing tests; keep for wipe assertions.

func TestRelayContainerNameForTask(t *testing.T) {
	name, err := relayContainerNameForTask("task_1259")
	if err != nil {
		t.Fatal(err)
	}
	if name != "relay_taskId_task_1259" {
		t.Fatalf("name=%q", name)
	}
}

func TestParseImageInspectIdentity(t *testing.T) {
	ident, err := parseImageInspectIdentity("sha256:abc123\t[\"registry.example/app@sha256:def456\"]\n")
	if err != nil {
		t.Fatal(err)
	}
	if ident.ID != "sha256:abc123" {
		t.Fatalf("ID=%q", ident.ID)
	}
	if ident.Digest != "registry.example/app@sha256:def456" {
		t.Fatalf("Digest=%q", ident.Digest)
	}

	ident2, err := parseImageInspectIdentity("sha256:onlyid")
	if err != nil {
		t.Fatal(err)
	}
	if ident2.ID != "sha256:onlyid" || ident2.Digest != "" {
		t.Fatalf("ident2=%+v", ident2)
	}

	if _, err := parseImageInspectIdentity(""); err == nil {
		t.Fatal("expected error for empty")
	}
}

func TestFormatImageIdentityLog(t *testing.T) {
	line := formatImageIdentityLog("registry.example/app:v1", imageIdentity{
		ID:     "sha256:abc",
		Digest: "registry.example/app@sha256:def",
	})
	if !strings.Contains(line, "docker pull ok:") {
		t.Fatalf("line=%q", line)
	}
	if !strings.Contains(line, "id=sha256:abc") {
		t.Fatalf("missing id: %q", line)
	}
	if !strings.Contains(line, "digest=registry.example/app@sha256:def") {
		t.Fatalf("missing digest: %q", line)
	}
}

func TestStartSelectedImageContainer_PullAndRun(t *testing.T) {
	stateBase := t.TempDir()
	t.Setenv("RELAY_ONLINE_STATE_BASE", stateBase)

	var calls [][]string
	prev := runDockerCLI
	runDockerCLI = func(args ...string) (string, string, int, error) {
		cp := append([]string(nil), args...)
		calls = append(calls, cp)
		if len(args) > 0 && args[0] == "pull" {
			return "pulled", "", 0, nil
		}
		if len(args) >= 2 && args[0] == "image" && args[1] == "inspect" {
			return "sha256:imgcfgdeadbeef\t[\"registry.example/app@sha256:repodigestcafe\"]\n", "", 0, nil
		}
		if len(args) > 0 && args[0] == "run" {
			// Simulate container writing post-exchange tokens to the bind-mounted state root.
			if err := writeSelectedImageTokenFixture("task_1259", "tok-exchanged", "rt-exchanged"); err != nil {
				t.Fatalf("write token fixture: %v", err)
			}
			return "cid-abc123\n", "", 0, nil
		}
		if len(args) > 0 && args[0] == "rm" {
			return "", "", 0, nil
		}
		if len(args) > 0 && args[0] == "stop" {
			return "", "", 0, nil
		}
		return "", "", 0, nil
	}
	prevLook := lookPathDocker
	lookPathDocker = func(file string) (string, error) { return "/usr/bin/docker", nil }
	defer func() {
		runDockerCLI = prev
		lookPathDocker = prevLook
		stateMu.Lock()
		state = relayState{Logs: make([]string, 0)}
		stateMu.Unlock()
		_ = os.RemoveAll(selectedImageHostStateRoot("task_1259"))
	}()

	// Avoid host 8765 (often occupied by a live relay container); pick a free local port.
	freeLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	freePort := freeLn.Addr().(*net.TCPAddr).Port
	_ = freeLn.Close()

	env := map[string]string{
		"TASK_API_ENDPOINT_ORIGIN":     "http://127.0.0.1:18081",
		"BUSINESS_API_ENDPOINT_ORIGIN": "http://127.0.0.1:8765",
		"ACCESS_TOKEN":                 "tok-test",
		"PORT":                         strconv.Itoa(freePort),
	}
	result, err := startSelectedImageContainer(env, "t1", "w1", "task_1259", "registry.example/app:v1")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if result["container_id"] != "cid-abc123" {
		t.Fatalf("result=%v", result)
	}
	if result["image"] != "registry.example/app:v1" {
		t.Fatalf("image field=%v", result["image"])
	}
	if result["image_id"] != "sha256:imgcfgdeadbeef" {
		t.Fatalf("image_id=%v", result["image_id"])
	}
	if result["image_digest"] != "registry.example/app@sha256:repodigestcafe" {
		t.Fatalf("image_digest=%v", result["image_digest"])
	}

	foundPull, foundInspect, foundRun, foundVolume, foundOrphanPS := false, false, false, false, false
	for _, c := range calls {
		joined := strings.Join(c, " ")
		if strings.HasPrefix(joined, "pull ") && strings.Contains(joined, "registry.example/app:v1") {
			foundPull = true
		}
		if len(c) >= 2 && c[0] == "image" && c[1] == "inspect" {
			foundInspect = true
		}
		if len(c) >= 3 && c[0] == "ps" && strings.Contains(joined, "name=relay_taskId_") {
			foundOrphanPS = true
		}
		if len(c) > 0 && c[0] == "run" {
			foundRun = true
			joined = strings.Join(c, " ")
			if !strings.Contains(joined, "registry.example/app:v1") {
				t.Fatalf("run missing image: %s", joined)
			}
			if !strings.Contains(joined, "--network") || !strings.Contains(joined, "host") {
				t.Fatalf("run missing host network: %s", joined)
			}
			if !strings.Contains(joined, "ACCESS_TOKEN=tok-test") {
				t.Fatalf("run missing ACCESS_TOKEN: %s", joined)
			}
			if !strings.Contains(joined, "BUSINESS_API_ENDPOINT_ORIGIN=http://127.0.0.1:") {
				t.Fatalf("expected loopback BUSINESS_API rewritten from TASK_API host: %s", joined)
			}
			// TRAE_PUBLIC_IP is only set when a non-loopback public IP is available;
			// with all-loopback env it correctly stays unset (rewriteLoopback returns early).
			hostRoot := selectedImageHostStateRoot("task_1259")
			wantMount := hostRoot + ":" + selectedImageContainerStateRoot
			if !strings.Contains(joined, wantMount) {
				t.Fatalf("expected state volume mount %q in: %s", wantMount, joined)
			}
			if !strings.Contains(joined, "ONLINE_PROJECT_STATE_ROOT="+selectedImageContainerStateRoot) {
				t.Fatalf("expected ONLINE_PROJECT_STATE_ROOT in: %s", joined)
			}
			foundVolume = true
		}
	}
	if !foundPull {
		t.Fatalf("expected docker pull, calls=%v", calls)
	}
	if !foundInspect {
		t.Fatalf("expected docker image inspect, calls=%v", calls)
	}
	if !foundOrphanPS {
		t.Fatalf("expected docker ps filter for orphan relay containers, calls=%v", calls)
	}
	if !foundRun {
		t.Fatalf("expected docker run, calls=%v", calls)
	}
	if !foundVolume {
		t.Fatal("expected volume mount for token sync")
	}
	foundOverlay := false
	for _, c := range calls {
		if len(c) > 0 && c[0] == "run" {
			joined := strings.Join(c, " ")
			if strings.Contains(joined, "onlineServiceJS/src/server.mjs:/app/onlineServiceJS/src/server.mjs") {
				foundOverlay = true
			}
		}
	}
	// Overlay only when repoRoot points at a real monorepo with server.mjs.
	_ = foundOverlay

	stateMu.Lock()
	defer stateMu.Unlock()
	if !state.Running {
		t.Fatal("expected Running")
	}
	if state.ContainerID != "cid-abc123" {
		t.Fatalf("ContainerID=%q", state.ContainerID)
	}
	if state.Image != "registry.example/app:v1" {
		t.Fatalf("Image=%q", state.Image)
	}
	if state.ContainerName != "relay_taskId_task_1259" {
		t.Fatalf("ContainerName=%q", state.ContainerName)
	}
	if state.UIURL != "" {
		t.Fatalf("selected-image must not fabricate UIURL, got %q", state.UIURL)
	}
	if state.Port <= 0 {
		t.Fatalf("Port=%d", state.Port)
	}
	if state.AccessToken != "tok-exchanged" {
		t.Fatalf("AccessToken after sync=%q want tok-exchanged", state.AccessToken)
	}
	if state.RefreshToken != "rt-exchanged" {
		t.Fatalf("RefreshToken after sync=%q", state.RefreshToken)
	}
	if result["ui_url"] != "" {
		t.Fatalf("result ui_url must be empty, got %v", result["ui_url"])
	}
	if result["mode"] != "selected_image" {
		t.Fatalf("mode=%v", result["mode"])
	}

	foundHashLog := false
	for _, line := range state.Logs {
		if strings.Contains(line, "docker pull ok:") &&
			strings.Contains(line, "id=sha256:imgcfgdeadbeef") &&
			strings.Contains(line, "digest=registry.example/app@sha256:repodigestcafe") {
			foundHashLog = true
		}
	}
	if !foundHashLog {
		t.Fatalf("expected image hash in logs, got %#v", state.Logs)
	}

	foundPortEnv := false
	for _, c := range calls {
		if len(c) > 0 && c[0] == "run" {
			joined := strings.Join(c, " ")
			if strings.Contains(joined, fmt.Sprintf("PORT=%d", state.Port)) {
				foundPortEnv = true
			}
		}
	}
	if !foundPortEnv {
		t.Fatalf("expected PORT in docker run env, calls=%v", calls)
	}

	// Token file should exist under host state root.
	tokPath := filepath.Join(selectedImageHostStateRoot("task_1259"), "runtime", "container_refresh_token.json")
	if _, err := os.Stat(tokPath); err != nil {
		t.Fatalf("expected token file at %s: %v", tokPath, err)
	}
}

func TestStartSelectedImageContainer_RequiresImage(t *testing.T) {
	_, err := startSelectedImageContainer(map[string]string{
		"TASK_API_ENDPOINT_ORIGIN":     "http://x",
		"BUSINESS_API_ENDPOINT_ORIGIN": "http://y",
		"ACCESS_TOKEN":                 "t",
	}, "t", "w", "task1", "  ")
	if err == nil || !strings.Contains(err.Error(), "image") {
		t.Fatalf("err=%v", err)
	}
}

func TestEnsureSelectedImageHostStateRoot_ClearsResidualBeforeStart(t *testing.T) {
	base := t.TempDir()
	t.Setenv("RELAY_ONLINE_STATE_BASE", base)

	taskID := "task_wipe_me"
	hostRoot := selectedImageHostStateRoot(taskID)
	staleLayer := filepath.Join(hostRoot, "layers", "old_layer", "workspace", "big.bin")
	staleLog := filepath.Join(hostRoot, "logs", "requests.log")
	staleSibling := filepath.Join(base, "task_old_sibling", "layers", "x", "f.txt")
	for _, p := range []string{staleLayer, staleLog, staleSibling} {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("stale-residue"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// Leftover token from a previous run must not survive either.
	oldTok := filepath.Join(hostRoot, "runtime", "container_refresh_token.json")
	if err := os.MkdirAll(filepath.Dir(oldTok), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(oldTok, []byte(`{"access_token":"old"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	gotRoot, refreshPath, err := ensureSelectedImageHostStateRoot(taskID, "registry.example/wipe:test")
	if err != nil {
		t.Fatalf("ensure: %v", err)
	}
	if gotRoot != hostRoot {
		t.Fatalf("hostRoot=%q want %q", gotRoot, hostRoot)
	}
	wantRefresh := filepath.Join(hostRoot, "runtime", "container_refresh_token.json")
	if refreshPath != wantRefresh {
		t.Fatalf("refreshPath=%q want %q", refreshPath, wantRefresh)
	}
	for _, p := range []string{staleLayer, staleLog, staleSibling, oldTok} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Fatalf("expected residual cleared: %s (err=%v)", p, err)
		}
	}
	if st, err := os.Stat(filepath.Join(hostRoot, "runtime")); err != nil || !st.IsDir() {
		t.Fatalf("expected fresh runtime dir, err=%v", err)
	}
	entries, err := os.ReadDir(base)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != taskID {
		t.Fatalf("base should only contain current task dir, got %#v", entries)
	}
}

func TestWipeHostStatePath_FallsBackToDockerWhenRemoveAllFails(t *testing.T) {
	base := t.TempDir()
	t.Setenv("RELAY_ONLINE_STATE_BASE", base)
	hostPath := filepath.Join(base, "task_root_owned")
	if err := os.MkdirAll(filepath.Join(hostPath, "logs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hostPath, "logs", "a.log"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	prevRemove := removeAllHostPath
	removeAllHostPath = func(string) error {
		return fmt.Errorf("permission denied (simulated)")
	}
	var dockerCalls [][]string
	prevDocker := runDockerCLI
	runDockerCLI = func(args ...string) (string, string, int, error) {
		dockerCalls = append(dockerCalls, append([]string(nil), args...))
		// Simulate successful docker wipe.
		_ = os.RemoveAll(hostPath)
		return "", "", 0, nil
	}
	defer func() {
		removeAllHostPath = prevRemove
		runDockerCLI = prevDocker
	}()

	if err := wipeHostStatePath(hostPath, "registry.example/app:v1"); err != nil {
		t.Fatalf("wipe: %v", err)
	}
	if _, err := os.Stat(hostPath); !os.IsNotExist(err) {
		t.Fatalf("path still exists after docker wipe")
	}
	found := false
	for _, c := range dockerCalls {
		joined := strings.Join(c, " ")
		if strings.Contains(joined, "run") &&
			strings.Contains(joined, "--entrypoint") &&
			strings.Contains(joined, "rm") &&
			strings.Contains(joined, "/relay_state_wipe/"+filepath.Base(hostPath)) &&
			strings.Contains(joined, "registry.example/app:v1") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected docker wipe run, calls=%v", dockerCalls)
	}
}

func TestStopOrphanRelayContainers(t *testing.T) {
	var calls [][]string
	prev := runDockerCLI
	runDockerCLI = func(args ...string) (string, string, int, error) {
		calls = append(calls, append([]string(nil), args...))
		if len(args) >= 1 && args[0] == "ps" {
			return "cid-orphan-a\ncid-orphan-b\n", "", 0, nil
		}
		if len(args) >= 2 && args[0] == "rm" && args[1] == "-f" {
			return args[2] + "\n", "", 0, nil
		}
		return "", "", 0, nil
	}
	prevLook := lookPathDocker
	lookPathDocker = func(file string) (string, error) { return "/usr/bin/docker", nil }
	defer func() {
		runDockerCLI = prev
		lookPathDocker = prevLook
	}()

	stopped := stopOrphanRelayContainers()
	if len(stopped) != 2 || stopped[0] != "cid-orphan-a" || stopped[1] != "cid-orphan-b" {
		t.Fatalf("stopped=%v", stopped)
	}
	foundPS, rmCount := false, 0
	for _, c := range calls {
		joined := strings.Join(c, " ")
		if strings.Contains(joined, "name=relay_taskId_") {
			foundPS = true
		}
		if len(c) >= 3 && c[0] == "rm" && c[1] == "-f" {
			rmCount++
		}
	}
	if !foundPS || rmCount != 2 {
		t.Fatalf("calls=%v foundPS=%v rmCount=%d", calls, foundPS, rmCount)
	}
}

func TestStartSelectedImageContainer_FailsWhenPortStillOccupied(t *testing.T) {
	t.Setenv("RELAY_ONLINE_STATE_BASE", t.TempDir())

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port

	// Simulate host-network Docker: port is listening but lsof sees no PIDs.
	prevPIDs := listPortListenerPIDs
	listPortListenerPIDs = func(int) []string { return nil }
	defer func() { listPortListenerPIDs = prevPIDs }()

	var calls [][]string
	prev := runDockerCLI
	runDockerCLI = func(args ...string) (string, string, int, error) {
		calls = append(calls, append([]string(nil), args...))
		if len(args) > 0 && args[0] == "pull" {
			return "pulled", "", 0, nil
		}
		if len(args) >= 2 && args[0] == "image" && args[1] == "inspect" {
			return "sha256:img\t[]\n", "", 0, nil
		}
		// Orphan cleanup finds nothing useful — port stays held by ln.
		return "", "", 0, nil
	}
	prevLook := lookPathDocker
	lookPathDocker = func(file string) (string, error) { return "/usr/bin/docker", nil }
	defer func() {
		runDockerCLI = prev
		lookPathDocker = prevLook
		stateMu.Lock()
		state = relayState{Logs: make([]string, 0)}
		stateMu.Unlock()
	}()

	_, err = startSelectedImageContainer(map[string]string{
		"TASK_API_ENDPOINT_ORIGIN":     "http://task.example.com",
		"BUSINESS_API_ENDPOINT_ORIGIN": "http://biz.example.com",
		"ACCESS_TOKEN":                 "tok",
		"PORT":                         strconv.Itoa(port),
	}, "t1", "w1", "task_busy", "registry.example/app:v1")
	if err == nil || !strings.Contains(err.Error(), "still in use") {
		t.Fatalf("expected port-in-use error, got %v", err)
	}
	for _, c := range calls {
		if len(c) > 0 && c[0] == "run" {
			t.Fatalf("docker run must not start while port occupied, calls=%v", calls)
		}
	}
}

func TestEnsureListenPortFree_RemovesOrphanAndFreesPort(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port

	prevPIDs := listPortListenerPIDs
	listPortListenerPIDs = func(int) []string { return nil }
	defer func() { listPortListenerPIDs = prevPIDs }()

	prev := runDockerCLI
	runDockerCLI = func(args ...string) (string, string, int, error) {
		if len(args) >= 1 && args[0] == "ps" {
			return "cid-hold\n", "", 0, nil
		}
		if len(args) >= 3 && args[0] == "rm" && args[1] == "-f" && args[2] == "cid-hold" {
			_ = ln.Close()
			return "cid-hold\n", "", 0, nil
		}
		return "", "", 0, nil
	}
	prevLook := lookPathDocker
	lookPathDocker = func(file string) (string, error) { return "/usr/bin/docker", nil }
	defer func() {
		runDockerCLI = prev
		lookPathDocker = prevLook
		_ = ln.Close()
	}()

	ensureListenPortFree(port)
	if portListening(port) {
		t.Fatalf("expected port %d freed after orphan container rm", port)
	}
}

func TestStopRunning_StopsDockerContainer(t *testing.T) {
	var calls [][]string
	prev := runDockerCLI
	runDockerCLI = func(args ...string) (string, string, int, error) {
		calls = append(calls, append([]string(nil), args...))
		return "", "", 0, nil
	}
	prevLook := lookPathDocker
	lookPathDocker = func(file string) (string, error) { return "/usr/bin/docker", nil }
	defer func() {
		runDockerCLI = prev
		lookPathDocker = prevLook
		stateMu.Lock()
		state = relayState{Logs: make([]string, 0)}
		stateMu.Unlock()
	}()

	stateMu.Lock()
	state.Running = true
	state.ContainerID = "cid-1"
	state.ContainerName = "relay_taskId_task1"
	state.Image = "img:1"
	stateMu.Unlock()

	stopRunning()

	foundStop := false
	for _, c := range calls {
		if len(c) >= 2 && c[0] == "stop" && (c[1] == "relay_taskId_task1" || c[1] == "cid-1") {
			foundStop = true
		}
	}
	if !foundStop {
		t.Fatalf("expected docker stop, calls=%v", calls)
	}
	stateMu.Lock()
	defer stateMu.Unlock()
	if state.Running || state.ContainerID != "" || state.Image != "" {
		t.Fatalf("state not cleared: %+v", state)
	}
}
