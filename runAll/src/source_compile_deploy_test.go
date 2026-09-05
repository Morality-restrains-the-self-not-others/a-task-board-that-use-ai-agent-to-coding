package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPreciseRestartFile_DeployModeUsesSourceRoot(t *testing.T) {
	src := t.TempDir()
	t.Setenv("DEPLOY_MODE", "1")
	t.Setenv("SOURCE_ROOT", src)
	t.Setenv("RUNALL_PRECISE_RESTART_FILE", "")
	got := preciseRestartFile("/x/conf/runAll.yaml")
	want, err := filepath.Abs(filepath.Join(src, preciseRestartFileName))
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("preciseRestartFile = %q, want %q", got, want)
	}
}

func TestPreciseRestartFile_DeployModeReadsCutoverWhenEnvEmpty(t *testing.T) {
	src := t.TempDir()
	deploy := t.TempDir()
	cutover := filepath.Join(deploy, "cutover.env")
	body := "export DEPLOY_MODE=1\nexport SOURCE_ROOT=" + src + "\n"
	if err := os.WriteFile(cutover, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DEPLOY_MODE", "1")
	t.Setenv("SOURCE_ROOT", "")
	t.Setenv("RUNALL_PRECISE_RESTART_FILE", "")
	t.Setenv("CUTOVER_ENV", cutover)
	t.Setenv("DEPLOY_ROOT", deploy)
	got := preciseRestartFile(filepath.Join(deploy, "conf", "runAll.yaml"))
	want, err := filepath.Abs(filepath.Join(src, preciseRestartFileName))
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("preciseRestartFile = %q, want %q (must not use deploy-tree cwd .runall/)", got, want)
	}
}

func TestParseSourceRootFromCutoverFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cutover.env")
	body := "# comment\nexport DEPLOY_MODE=1\nexport SOURCE_ROOT=/tmp/ram-work\nexport RUNALL_SKIP_BUILD=1\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	got := parseSourceRootFromCutoverFile(path)
	if got != "/tmp/ram-work" {
		t.Fatalf("parseSourceRootFromCutoverFile = %q", got)
	}
	if parseSourceRootFromCutoverFile(filepath.Join(dir, "missing.env")) != "" {
		t.Fatal("missing file must return empty")
	}
}

func TestRequireSourceRoot_EmptyFails(t *testing.T) {
	t.Setenv("SOURCE_ROOT", "")
	t.Setenv("CUTOVER_ENV", "")
	t.Setenv("DEPLOY_ROOT", t.TempDir())
	if _, err := requireSourceRoot(); err == nil || !strings.Contains(err.Error(), "SOURCE_ROOT") {
		t.Fatalf("err = %v, want SOURCE_ROOT", err)
	}
}

func TestRequireSourceRoot_ReadsCutoverWhenEnvEmpty(t *testing.T) {
	src := t.TempDir()
	deploy := t.TempDir()
	cutover := filepath.Join(deploy, "cutover.env")
	if err := os.WriteFile(cutover, []byte("export SOURCE_ROOT="+src+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SOURCE_ROOT", "")
	t.Setenv("CUTOVER_ENV", cutover)
	t.Setenv("DEPLOY_ROOT", deploy)
	got, err := requireSourceRoot()
	if err != nil {
		t.Fatal(err)
	}
	if got != src {
		t.Fatalf("requireSourceRoot = %q, want %q", got, src)
	}
}

func TestDeployModePreciseRestart_EmptyRegistryDoesNotCompile(t *testing.T) {
	t.Setenv("DEPLOY_MODE", "1")
	t.Setenv("SOURCE_ROOT", t.TempDir())
	called := false
	old := prepareDeploySourceArtifactsFn
	prepareDeploySourceArtifactsFn = func(context.Context, []string, bool) error {
		called = true
		return nil
	}
	t.Cleanup(func() { prepareDeploySourceArtifactsFn = old })

	path := filepath.Join(t.TempDir(), "reg.txt")
	t.Setenv("RUNALL_PRECISE_RESTART_FILE", path)
	runner, _ := testPreciseRestartRunner(t, &Config{})
	if _, err := runner.PreciseRestart(context.Background(), "test-session"); err == nil {
		t.Fatal("expected error for empty registrations")
	}
	if called {
		t.Fatal("empty registry must not compile")
	}
}

func TestDeployModePreciseRestart_MissingSourceRoot(t *testing.T) {
	t.Setenv("DEPLOY_MODE", "1")
	t.Setenv("SOURCE_ROOT", "")
	t.Setenv("CUTOVER_ENV", "")
	t.Setenv("DEPLOY_ROOT", t.TempDir())
	old := prepareDeploySourceArtifactsFn
	prepareDeploySourceArtifactsFn = prepareDeploySourceArtifacts
	t.Cleanup(func() { prepareDeploySourceArtifactsFn = old })

	health := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(health.Close)
	cfg := &Config{Groups: []Group{{Services: []Service{{
		Name:        "svc-src-missing",
		Command:     "sleep 30",
		HealthCheck: HealthCheck{URL: health.URL, Timeout: 2, Retries: 2, CheckInterval: 1, Backoff: Backoff{Initial: 0.1, Max: 0.2, Multiplier: 1.5}},
	}}}}}
	runner, store := testPreciseRestartRunner(t, cfg)
	store.Init([]string{"svc-src-missing"})
	store.Update("svc-src-missing", StatusStopped, "")
	path := filepath.Join(t.TempDir(), "reg.txt")
	t.Setenv("RUNALL_PRECISE_RESTART_FILE", path)
	if err := writeRegisteredServices(path, []string{"svc-src-missing"}); err != nil {
		t.Fatal(err)
	}
	_, err := runner.PreciseRestart(context.Background(), "test-session")
	if err == nil || !strings.Contains(err.Error(), "SOURCE_ROOT") {
		t.Fatalf("err = %v, want SOURCE_ROOT", err)
	}
	after, _ := readRegisteredServices(path)
	if len(after) != 1 || after[0] != "svc-src-missing" {
		t.Fatalf("registry after = %v, want kept", after)
	}
	if st := store.Get("svc-src-missing"); st == nil || st.Status != StatusStopped {
		t.Fatalf("status = %v, want stopped (no restart)", st)
	}
}

func TestDeployModePreciseRestart_CompilesThenRestarts(t *testing.T) {
	t.Setenv("DEPLOY_MODE", "1")
	t.Setenv("SOURCE_ROOT", t.TempDir())
	t.Setenv("DEPLOY_ROOT", t.TempDir())
	var gotNames []string
	var gotAll bool
	old := prepareDeploySourceArtifactsFn
	prepareDeploySourceArtifactsFn = func(_ context.Context, names []string, all bool) error {
		gotNames = append([]string(nil), names...)
		gotAll = all
		return nil
	}
	t.Cleanup(func() { prepareDeploySourceArtifactsFn = old })

	health := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(health.Close)
	work := t.TempDir()
	buildMarker := filepath.Join(work, "deploy-build-ran")
	if err := os.WriteFile(filepath.Join(work, "build.sh"), []byte("#!/bin/bash\necho ran > '"+buildMarker+"'\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := &Config{Groups: []Group{{Services: []Service{{
		Name:         "task-auth",
		Command:      "sleep 30",
		BuildCommand: "./build.sh",
		WorkingDir:   work,
		HealthCheck:  HealthCheck{URL: health.URL, Timeout: 2, Retries: 2, CheckInterval: 1, Backoff: Backoff{Initial: 0.1, Max: 0.2, Multiplier: 1.5}},
	}}}}}
	runner, store := testPreciseRestartRunner(t, cfg)
	oldWait := servicePortReleaseWait
	servicePortReleaseWait = 300 * time.Millisecond
	t.Cleanup(func() { servicePortReleaseWait = oldWait })
	store.Init([]string{"task-auth"})
	store.Update("task-auth", StatusStopped, "")
	path := filepath.Join(t.TempDir(), "reg.txt")
	t.Setenv("RUNALL_PRECISE_RESTART_FILE", path)
	if err := writeRegisteredServices(path, []string{"task-auth"}); err != nil {
		t.Fatal(err)
	}
	if !runner.TryBeginPreciseRestart("precise-restart-test") {
		t.Fatal("TryBeginPreciseRestart failed")
	}
	defer runner.endPreciseRestart()

	keep, err := runner.PreciseRestart(context.Background(), "test-session")
	if err != nil {
		t.Fatalf("PreciseRestart: %v", err)
	}
	if len(keep) != 0 {
		t.Fatalf("keep = %v", keep)
	}
	if gotAll {
		t.Fatal("precise restart must not pass all=true")
	}
	if len(gotNames) != 1 || gotNames[0] != "task-auth" {
		t.Fatalf("compile names = %v, want [task-auth]", gotNames)
	}
	if _, err := os.Stat(buildMarker); err == nil {
		t.Fatal("deploy-tree build.sh must not run")
	}
	deadline := time.Now().Add(10 * time.Second)
	for {
		st := store.Get("task-auth")
		if st != nil && st.Status == StatusHealthy {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("status = %v, want healthy", st)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func TestDeployModePreciseRestart_CompileFailKeepsRegistry(t *testing.T) {
	t.Setenv("DEPLOY_MODE", "1")
	t.Setenv("SOURCE_ROOT", t.TempDir())
	old := prepareDeploySourceArtifactsFn
	prepareDeploySourceArtifactsFn = func(context.Context, []string, bool) error {
		return fmt.Errorf("compile failed: fake")
	}
	t.Cleanup(func() { prepareDeploySourceArtifactsFn = old })

	health := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(health.Close)
	cfg := &Config{Groups: []Group{{Services: []Service{{
		Name:        "task-auth",
		Command:     "sleep 30",
		HealthCheck: HealthCheck{URL: health.URL, Timeout: 2, Retries: 2, CheckInterval: 1, Backoff: Backoff{Initial: 0.1, Max: 0.2, Multiplier: 1.5}},
	}}}}}
	runner, store := testPreciseRestartRunner(t, cfg)
	store.Init([]string{"task-auth"})
	store.Update("task-auth", StatusStopped, "")
	path := filepath.Join(t.TempDir(), "reg.txt")
	t.Setenv("RUNALL_PRECISE_RESTART_FILE", path)
	if err := writeRegisteredServices(path, []string{"task-auth"}); err != nil {
		t.Fatal(err)
	}
	_, err := runner.PreciseRestart(context.Background(), "test-session")
	if err == nil {
		t.Fatal("expected compile error")
	}
	after, _ := readRegisteredServices(path)
	if len(after) != 1 || after[0] != "task-auth" {
		t.Fatalf("registry = %v, want kept", after)
	}
	if st := store.Get("task-auth"); st == nil || st.Status != StatusStopped {
		t.Fatalf("status = %v, want stopped", st)
	}
}

func TestDeployModeBuildAll_CompilesAllWithoutRestart(t *testing.T) {
	t.Setenv("DEPLOY_MODE", "1")
	t.Setenv("SOURCE_ROOT", t.TempDir())
	t.Setenv("DEPLOY_ROOT", t.TempDir())
	var gotAll bool
	var restartHookHits int
	oldPrep := prepareDeploySourceArtifactsFn
	prepareDeploySourceArtifactsFn = func(_ context.Context, names []string, all bool) error {
		gotAll = all
		if !all {
			t.Errorf("BuildAll must pass all=true, names=%v", names)
		}
		return nil
	}
	oldHook := restartServiceTestHook
	restartServiceTestHook = func(string) { restartHookHits++ }
	t.Cleanup(func() {
		prepareDeploySourceArtifactsFn = oldPrep
		restartServiceTestHook = oldHook
	})

	dir := t.TempDir()
	store := NewStatusStore()
	store.Init([]string{"ba-deploy-a"})
	store.Update("ba-deploy-a", StatusStopped, "")
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{{
			Name: "ba-deploy",
			Services: []Service{{
				Name:         "ba-deploy-a",
				Command:      "sleep 30",
				BuildCommand: "echo must-not-run",
				WorkingDir:   dir,
			}},
		}},
	}, store)
	if err != nil {
		t.Fatal(err)
	}
	store.Update("ba-deploy-a", StatusStopped, "")
	path := filepath.Join(t.TempDir(), "reg.txt")
	t.Setenv("RUNALL_PRECISE_RESTART_FILE", path)
	if err := writeRegisteredServices(path, []string{"ba-deploy-a"}); err != nil {
		t.Fatal(err)
	}
	result, err := runner.BuildAll(context.Background())
	if err != nil {
		t.Fatalf("BuildAll: %v", err)
	}
	if result == nil || result.Built < 1 {
		t.Fatalf("result = %+v, want built>=1", result)
	}
	if !gotAll {
		t.Fatal("expected --all compile")
	}
	if restartHookHits != 0 {
		t.Fatalf("restartService called %d times, want 0", restartHookHits)
	}
	if st := store.Get("ba-deploy-a"); st == nil || st.Status == StatusHealthy || st.Status == StatusRestarting {
		t.Fatalf("status = %v, want not restarted", st)
	}
	after, _ := readRegisteredServices(path)
	if len(after) != 0 {
		t.Fatalf("registrations after BuildAll = %v, want empty", after)
	}
}

func TestDeployModeBuildAll_MissingSourceRoot(t *testing.T) {
	t.Setenv("DEPLOY_MODE", "1")
	t.Setenv("SOURCE_ROOT", "")
	t.Setenv("CUTOVER_ENV", "")
	t.Setenv("DEPLOY_ROOT", t.TempDir())
	old := prepareDeploySourceArtifactsFn
	prepareDeploySourceArtifactsFn = prepareDeploySourceArtifacts
	t.Cleanup(func() { prepareDeploySourceArtifactsFn = old })
	store := NewStatusStore()
	store.Init([]string{"ba-src"})
	store.Update("ba-src", StatusStopped, "")
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{{Services: []Service{{
			Name:         "ba-src",
			Command:      "true",
			BuildCommand: "echo x",
			WorkingDir:   t.TempDir(),
		}}}},
	}, store)
	if err != nil {
		t.Fatal(err)
	}
	_, err = runner.BuildAll(context.Background())
	if err == nil || !strings.Contains(err.Error(), "SOURCE_ROOT") {
		t.Fatalf("err = %v, want SOURCE_ROOT", err)
	}
}

func TestRsyncConfLocal_SuccessAndCompileFailDoesNotWrite(t *testing.T) {
	source := t.TempDir()
	deploy := t.TempDir()
	if err := os.MkdirAll(filepath.Join(source, "conf-local"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "conf-local", "a.yaml"), []byte("from: source\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(deploy, "conf-local"), 0o755); err != nil {
		t.Fatal(err)
	}
	oldDest := filepath.Join(deploy, "conf-local", "old.yaml")
	if err := os.WriteFile(oldDest, []byte("keep-me\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("DEPLOY_MODE", "1")
	t.Setenv("SOURCE_ROOT", source)
	t.Setenv("DEPLOY_ROOT", deploy)
	scripts := filepath.Join(source, "scripts")
	if err := os.MkdirAll(scripts, 0o755); err != nil {
		t.Fatal(err)
	}
	failScript := "#!/usr/bin/env bash\nexit 1\n"
	if err := os.WriteFile(filepath.Join(scripts, "precise-compile.sh"), []byte(failScript), 0o755); err != nil {
		t.Fatal(err)
	}
	err := prepareDeploySourceArtifacts(context.Background(), []string{"task-auth"}, false)
	if err == nil {
		t.Fatal("expected compile failure")
	}
	got, _ := os.ReadFile(oldDest)
	if string(got) != "keep-me\n" {
		t.Fatalf("dest conf-local mutated on compile fail: %q", got)
	}

	okScript := "#!/usr/bin/env bash\nset -euo pipefail\nmkdir -p \"$SOURCE_ROOT/deploy-binaries\"\necho elf > \"$SOURCE_ROOT/deploy-binaries/runAll\"\nexit 0\n"
	if err := os.WriteFile(filepath.Join(scripts, "precise-compile.sh"), []byte(okScript), 0o755); err != nil {
		t.Fatal(err)
	}
	installSrc := filepath.Join("..", "scripts", "install-local-artifacts.sh")
	installDstDir := filepath.Join(source, "runAll", "scripts")
	if err := os.MkdirAll(installDstDir, 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(installSrc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(installDstDir, "install-local-artifacts.sh"), data, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := prepareDeploySourceArtifacts(context.Background(), nil, true); err != nil {
		t.Fatalf("prepare success path: %v", err)
	}
	aligned, err := os.ReadFile(filepath.Join(deploy, "conf-local", "a.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if string(aligned) != "from: source\n" {
		t.Fatalf("rsync dest a.yaml = %q", aligned)
	}
	if _, err := os.Stat(oldDest); err == nil {
		t.Fatal("rsync --delete should remove dest-only old.yaml")
	}
}

// OPT-20260902-012：rsync 先写同父目录 staging，失败时 dest 树与失败前一致、
// staging 被清理，不残留半写入。用假 rsync（exit 1）注入失败。
func TestRsyncConfLocal_AtomicFailureLeavesDestUntouched(t *testing.T) {
	source := t.TempDir()
	deploy := t.TempDir()
	if err := os.MkdirAll(filepath.Join(source, "conf-local"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "conf-local", "a.yaml"), []byte("new\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(deploy, "conf-local"), 0o755); err != nil {
		t.Fatal(err)
	}
	oldDest := filepath.Join(deploy, "conf-local", "old.yaml")
	if err := os.WriteFile(oldDest, []byte("old\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	fakeBin := filepath.Join(t.TempDir(), "bin")
	if err := os.MkdirAll(fakeBin, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fakeBin, "rsync"), []byte("#!/usr/bin/env bash\necho 'boom' >&2\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"))

	err := rsyncConfLocal(context.Background(), source, deploy)
	if err == nil {
		t.Fatal("expected fake rsync failure")
	}
	got, _ := os.ReadFile(oldDest)
	if string(got) != "old\n" {
		t.Fatalf("dest conf-local mutated on rsync failure: %q", got)
	}
	if _, err := os.Stat(filepath.Join(deploy, "conf-local", "a.yaml")); err == nil {
		t.Fatal("source file must not reach dest when rsync fails")
	}
	entries, _ := os.ReadDir(deploy)
	for _, e := range entries {
		if strings.Contains(e.Name(), "conf-local.incoming") || strings.Contains(e.Name(), "conf-local.old") {
			t.Fatalf("staging/backup not cleaned on failure: %s", e.Name())
		}
	}
}

// OPT-20260902-012：swap 交换成功路径——staging 成为新 destDir，备份被清理。
func TestSwapConfLocalDir_SwapsAndCleansBackup(t *testing.T) {
	parent := t.TempDir()
	destDir := filepath.Join(parent, "conf-local")
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(destDir, "old.yaml"), []byte("old\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	staging := filepath.Join(parent, "conf-local.incoming.42")
	if err := os.MkdirAll(staging, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(staging, "new.yaml"), []byte("new\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := swapConfLocalDir(staging, destDir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(destDir, "new.yaml")); err != nil {
		t.Fatalf("staging content missing after swap: %v", err)
	}
	if _, err := os.Stat(filepath.Join(destDir, "old.yaml")); err == nil {
		t.Fatal("old dest content should be replaced after swap")
	}
	entries, _ := os.ReadDir(parent)
	for _, e := range entries {
		if strings.Contains(e.Name(), "conf-local.old") || strings.Contains(e.Name(), "conf-local.incoming") {
			t.Fatalf("staging/backup should be cleaned, found: %s", e.Name())
		}
	}
}

func TestApplyCutoverEnvFile_LoadsSourceRoot(t *testing.T) {
	restoreCutoverKeys(t)
	_ = os.Unsetenv("SOURCE_ROOT")
	path := filepath.Join(t.TempDir(), "cutover.env")
	body := "export DEPLOY_MODE=1\nexport SOURCE_ROOT=/tmp/ram-work\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := applyCutoverEnvFile(path); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("SOURCE_ROOT") != "/tmp/ram-work" {
		t.Fatalf("SOURCE_ROOT=%q", os.Getenv("SOURCE_ROOT"))
	}
}
