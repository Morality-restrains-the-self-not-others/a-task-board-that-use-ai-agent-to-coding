package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeReloadTestYAML(t *testing.T, dir, svcName, healthURL string) string {
	t.Helper()
	path := filepath.Join(dir, "runAll.yaml")
	body := `version: "1"
groups:
  - name: g
    services:
      - name: ` + svcName + `
        start_command: "sleep 30"
        stop_command: "true"
        on_failure: skip
        health_check:
          url: "` + healthURL + `"
          timeout: 2
          retries: 2
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestReloadConfigFromDisk_NoopWhenCfgPathEmpty(t *testing.T) {
	runner, _ := testPreciseRestartRunner(t, &Config{Groups: []Group{{Services: []Service{
		{Name: "mem-old-svc", Command: "true"},
	}}}})
	added, err := runner.reloadConfigFromDisk()
	if err != nil {
		t.Fatalf("empty cfgPath should no-op, got %v", err)
	}
	if len(added) != 0 {
		t.Fatalf("added = %v, want empty", added)
	}
	if svc := runner.resolveRegisteredService("mem-old-svc"); svc == nil {
		t.Fatal("in-memory service should remain")
	}
}

func TestReloadConfigFromDisk_PicksUpNewService(t *testing.T) {
	health := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(health.Close)

	runner, store := testPreciseRestartRunner(t, &Config{Groups: []Group{{Services: []Service{
		{Name: "mem-old-svc", Command: "true"},
	}}}})
	store.Update("mem-old-svc", StatusHealthy, "")

	yamlPath := writeReloadTestYAML(t, t.TempDir(), "disk-new-svc", health.URL)
	runner.SetConfigPath(yamlPath)

	if got := runner.resolveRegisteredServices("disk-new-svc"); len(got) != 0 {
		t.Fatalf("stale memory should not know disk-new-svc, got %v", got)
	}

	added, err := runner.reloadConfigFromDisk()
	if err != nil {
		t.Fatalf("reloadConfigFromDisk: %v", err)
	}
	if len(added) != 1 || added[0] != "disk-new-svc" {
		t.Fatalf("added = %v, want [disk-new-svc]", added)
	}
	svcs := runner.resolveRegisteredServices("disk-new-svc")
	if len(svcs) != 1 || svcs[0].Name != "disk-new-svc" {
		t.Fatalf("resolve after reload = %v, want disk-new-svc", svcs)
	}
	stNew := store.Get("disk-new-svc")
	if stNew == nil || stNew.Status != StatusPending {
		t.Fatalf("new service store = %+v, want pending", stNew)
	}
	stOld := store.Get("mem-old-svc")
	if stOld == nil || stOld.Status != StatusHealthy {
		t.Fatalf("existing service store = %+v, want healthy (must not Init-reset)", stOld)
	}
}

func TestReloadConfigFromDisk_SkipsUnchangedMtime(t *testing.T) {
	health := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(health.Close)

	runner, _ := testPreciseRestartRunner(t, &Config{Groups: []Group{{Services: []Service{
		{Name: "mem-old-svc", Command: "true"},
	}}}})
	yamlPath := writeReloadTestYAML(t, t.TempDir(), "disk-new-svc", health.URL)
	runner.SetConfigPath(yamlPath)

	if _, err := runner.reloadConfigFromDisk(); err != nil {
		t.Fatalf("first reload: %v", err)
	}
	if svc := runner.resolveRegisteredService("disk-new-svc"); svc == nil {
		t.Fatal("expected disk-new-svc after first reload")
	}
	runner.cfg.Groups[0].Services[0].Name = "mutated-svc"

	if _, err := runner.reloadConfigFromDisk(); err != nil {
		t.Fatalf("second reload: %v", err)
	}
	if svc := runner.resolveRegisteredService("mutated-svc"); svc == nil {
		t.Fatal("unchanged mtime must skip LoadConfig and keep in-memory mutation")
	}
	if svc := runner.resolveRegisteredService("disk-new-svc"); svc != nil {
		t.Fatal("mtime skip must not restore yaml names")
	}
}

func TestReloadConfigFromDisk_ReloadsWhenMtimeChanges(t *testing.T) {
	health := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(health.Close)

	runner, _ := testPreciseRestartRunner(t, &Config{})
	dir := t.TempDir()
	yamlPath := writeReloadTestYAML(t, dir, "disk-new-svc", health.URL)
	runner.SetConfigPath(yamlPath)
	if _, err := runner.reloadConfigFromDisk(); err != nil {
		t.Fatalf("first reload: %v", err)
	}
	runner.cfg.Groups[0].Services[0].Name = "mutated-svc"

	writeReloadTestYAML(t, dir, "disk-new-svc", health.URL)
	future := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(yamlPath, future, future); err != nil {
		t.Fatal(err)
	}

	if _, err := runner.reloadConfigFromDisk(); err != nil {
		t.Fatalf("mtime-changed reload: %v", err)
	}
	if svc := runner.resolveRegisteredService("disk-new-svc"); svc == nil {
		t.Fatal("changed mtime must LoadConfig and restore yaml names")
	}
	if svc := runner.resolveRegisteredService("mutated-svc"); svc != nil {
		t.Fatal("changed mtime must not keep in-memory mutation")
	}
}

func TestReloadConfigFromDisk_InvalidYAMLKeepsMemory(t *testing.T) {
	runner, _ := testPreciseRestartRunner(t, &Config{Groups: []Group{{Services: []Service{
		{Name: "mem-old-svc", Command: "true"},
	}}}})
	bad := filepath.Join(t.TempDir(), "runAll.yaml")
	if err := os.WriteFile(bad, []byte("not: [valid: yaml: {{{"), 0o644); err != nil {
		t.Fatal(err)
	}
	runner.SetConfigPath(bad)

	if _, err := runner.reloadConfigFromDisk(); err == nil {
		t.Fatal("expected error for invalid YAML")
	}
	if svc := runner.resolveRegisteredService("mem-old-svc"); svc == nil {
		t.Fatal("invalid YAML must keep in-memory config")
	}
}

func TestPreciseRestart_ReloadsStaleMemoryConfig(t *testing.T) {
	health := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(health.Close)

	oldReleaseWait := servicePortReleaseWait
	servicePortReleaseWait = 300 * time.Millisecond
	t.Cleanup(func() { servicePortReleaseWait = oldReleaseWait })

	runner, store := testPreciseRestartRunner(t, &Config{})
	yamlPath := writeReloadTestYAML(t, t.TempDir(), "disk-new-svc", health.URL)
	runner.SetConfigPath(yamlPath)

	path := filepath.Join(t.TempDir(), "reg.txt")
	t.Setenv("RUNALL_PRECISE_RESTART_FILE", path)
	if err := writeRegisteredServices(path, []string{"disk-new-svc"}); err != nil {
		t.Fatal(err)
	}
	if !runner.TryBeginPreciseRestart("precise-restart-reload-test") {
		t.Fatal("TryBeginPreciseRestart failed")
	}
	defer runner.endPreciseRestart()

	keep, err := runner.PreciseRestart(context.Background(), "test-session")
	if err != nil {
		t.Fatalf("PreciseRestart: %v", err)
	}
	if len(keep) != 0 {
		t.Fatalf("keep = %v, want empty (disk-new-svc must not be unknown)", keep)
	}
	st := store.Get("disk-new-svc")
	if st == nil {
		t.Fatal("disk-new-svc missing from status store after reload")
	}
	deadline := time.Now().Add(10 * time.Second)
	for {
		st = store.Get("disk-new-svc")
		if st != nil && st.Status == StatusHealthy {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("disk-new-svc not healthy after reload restart: %+v", st)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func TestPreciseRestart_UnknownAfterReloadStillRetained(t *testing.T) {
	health := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(health.Close)

	runner, _ := testPreciseRestartRunner(t, &Config{})
	yamlPath := writeReloadTestYAML(t, t.TempDir(), "disk-new-svc", health.URL)
	runner.SetConfigPath(yamlPath)

	path := filepath.Join(t.TempDir(), "reg.txt")
	t.Setenv("RUNALL_PRECISE_RESTART_FILE", path)
	if err := writeRegisteredServices(path, []string{"ghost-svc"}); err != nil {
		t.Fatal(err)
	}

	keep, err := runner.PreciseRestart(context.Background(), "test-session")
	if err != nil {
		t.Fatalf("PreciseRestart: %v", err)
	}
	if len(keep) != 1 || keep[0] != "ghost-svc" {
		t.Fatalf("keep = %v, want [ghost-svc]", keep)
	}
	after, _ := readRegisteredServices(path)
	if len(after) != 1 || after[0] != "ghost-svc" {
		t.Fatalf("file after = %v, want [ghost-svc]", after)
	}
}

func TestPreciseRestart_ReloadFailureDoesNotBlockKnownService(t *testing.T) {
	health := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(health.Close)

	oldReleaseWait := servicePortReleaseWait
	servicePortReleaseWait = 300 * time.Millisecond
	t.Cleanup(func() { servicePortReleaseWait = oldReleaseWait })

	cfg := &Config{Groups: []Group{{Services: []Service{{
		Name:        "mem-known-svc",
		Command:     "sleep 30",
		HealthCheck: HealthCheck{URL: health.URL, Timeout: 2, Retries: 2, CheckInterval: 1, Backoff: Backoff{Initial: 0.1, Max: 0.2, Multiplier: 1.5}},
	}}}}}
	runner, store := testPreciseRestartRunner(t, cfg)
	store.Update("mem-known-svc", StatusStopped, "")

	bad := filepath.Join(t.TempDir(), "runAll.yaml")
	if err := os.WriteFile(bad, []byte("{{{{"), 0o644); err != nil {
		t.Fatal(err)
	}
	runner.SetConfigPath(bad)

	path := filepath.Join(t.TempDir(), "reg.txt")
	t.Setenv("RUNALL_PRECISE_RESTART_FILE", path)
	if err := writeRegisteredServices(path, []string{"mem-known-svc"}); err != nil {
		t.Fatal(err)
	}
	if !runner.TryBeginPreciseRestart("precise-restart-reload-fail-test") {
		t.Fatal("TryBeginPreciseRestart failed")
	}
	defer runner.endPreciseRestart()

	keep, err := runner.PreciseRestart(context.Background(), "test-session")
	if err != nil {
		t.Fatalf("PreciseRestart: %v", err)
	}
	if len(keep) != 0 {
		t.Fatalf("keep = %v, want empty; reload failure must not block known service", keep)
	}
}
