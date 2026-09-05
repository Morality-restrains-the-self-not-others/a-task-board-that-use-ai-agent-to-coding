package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	// Cutover shells export DEPLOY_MODE=1; that must not leak into `go test`.
	// Otherwise BuildGroup skips compile (resolveBuildCommand returns "") and
	// runner_test fails with built=0.
	_ = os.Unsetenv("DEPLOY_MODE")
	_ = os.Unsetenv("CUTOVER_ENV")
	_ = os.Unsetenv("SOURCE_ROOT")
	_ = os.Unsetenv("DEPLOY_ROOT")
	os.Exit(m.Run())
}

func TestDeployModeActive_FalseByDefault(t *testing.T) {
	if deployModeActive() {
		t.Fatal("package tests must start with DEPLOY_MODE unset so source-tree compile tests still run")
	}
}

func TestResolveWorkingDirs_DeployModeFlatBinIgnoresSourceServiceDir(t *testing.T) {
	t.Setenv("DEPLOY_MODE", "1")
	root := t.TempDir()
	confDir := filepath.Join(root, "conf")
	if err := os.MkdirAll(confDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := &Config{Groups: []Group{{Services: []Service{{
		Name:         "task-auth",
		StartCommand: "./bin/taskAuth",
		WorkingDir:   "taskAuth",
	}}}}}
	cfg.resolveWorkingDirs(confDir)
	got := cfg.Groups[0].Services[0].WorkingDir
	if got != root {
		t.Fatalf("working_dir=%q want deploy root %q (no taskAuth/ source tree)", got, root)
	}
}

func TestResolveWorkingDirs_DeployModeValueStreamUsesConfUnderRoot(t *testing.T) {
	t.Setenv("DEPLOY_MODE", "1")
	root := t.TempDir()
	confDir := filepath.Join(root, "conf")
	if err := os.MkdirAll(confDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := &Config{Groups: []Group{{Services: []Service{{
		Name:         "value-stream",
		StartCommand: "./bin/valueStream --config ../conf/value-stream.yaml --ui-port :9998",
		WorkingDir:   "valueStream",
	}}}}}
	cfg.resolveWorkingDirs(confDir)
	svc := cfg.Groups[0].Services[0]
	if svc.WorkingDir != root {
		t.Fatalf("working_dir=%q want %q", svc.WorkingDir, root)
	}
	if !strings.Contains(svc.StartCommand, "--config conf/value-stream.yaml") {
		t.Fatalf("start_command=%q want conf/value-stream.yaml under deploy root", svc.StartCommand)
	}
	if strings.Contains(svc.StartCommand, "../conf/") {
		t.Fatalf("start_command still uses ../conf: %q", svc.StartCommand)
	}
}

func TestSourceLessFlatBinRoot_TrueWhenRunAllBinAndNoTaskAuth(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "bin", "runAll"), []byte("elf"), 0o755); err != nil {
		t.Fatal(err)
	}
	if !sourceLessFlatBinRoot(root) {
		t.Fatal("expected source-less clone-run root")
	}
}

func TestSourceLessFlatBinRoot_FalseWhenTaskAuthPresent(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "bin", "runAll"), []byte("elf"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "taskAuth"), 0o755); err != nil {
		t.Fatal(err)
	}
	if sourceLessFlatBinRoot(root) {
		t.Fatal("monorepo-like tree with taskAuth/ must not infer deploy layout")
	}
}

func TestResolveWorkingDirs_SourceLessTreeInfersDeployLayoutWithoutEnv(t *testing.T) {
	t.Setenv("DEPLOY_MODE", "")
	root := t.TempDir()
	confDir := filepath.Join(root, "conf")
	if err := os.MkdirAll(confDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "bin", "runAll"), []byte("elf"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := &Config{Groups: []Group{{Services: []Service{
		{
			Name:         "task-auth",
			StartCommand: "./bin/taskAuth",
			WorkingDir:   "taskAuth",
		},
		{
			Name:         "value-stream",
			StartCommand: "./bin/valueStream --config ../conf/value-stream.yaml --ui-port :9998",
			WorkingDir:   "valueStream",
		},
	}}}}
	cfg.resolveWorkingDirs(confDir)
	if got := cfg.Groups[0].Services[0].WorkingDir; got != root {
		t.Fatalf("task-auth working_dir=%q want clone-run root %q", got, root)
	}
	vs := cfg.Groups[0].Services[1]
	if vs.WorkingDir != root {
		t.Fatalf("value-stream working_dir=%q want %q", vs.WorkingDir, root)
	}
	if !strings.Contains(vs.StartCommand, "--config conf/value-stream.yaml") {
		t.Fatalf("start_command=%q want conf/ under deploy root", vs.StartCommand)
	}
	if strings.Contains(vs.StartCommand, "../conf/") {
		t.Fatalf("start_command still uses ../conf: %q", vs.StartCommand)
	}
}

func TestResolveWorkingDirs_WithoutDeployModeKeepsServiceDir(t *testing.T) {
	t.Setenv("DEPLOY_MODE", "")
	root := t.TempDir()
	confDir := filepath.Join(root, "conf")
	svcDir := filepath.Join(root, "taskAuth")
	if err := os.MkdirAll(confDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(svcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := &Config{Groups: []Group{{Services: []Service{{
		Name:         "task-auth",
		StartCommand: "./bin/taskAuth",
		WorkingDir:   "taskAuth",
	}}}}}
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })
	cfg.resolveWorkingDirs(confDir)
	got := cfg.Groups[0].Services[0].WorkingDir
	if got != svcDir {
		t.Fatalf("working_dir=%q want source service dir %q", got, svcDir)
	}
}

func TestResolveWorkingDirs_DeployModeLeavesRecipeWorkingDir(t *testing.T) {
	t.Setenv("DEPLOY_MODE", "1")
	root := t.TempDir()
	confDir := filepath.Join(root, "conf")
	events := filepath.Join(root, "taskEvents")
	if err := os.MkdirAll(confDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(events, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := &Config{Groups: []Group{{Services: []Service{
		{
			Name:         "docker-mysql",
			StartCommand: "bash dockerInfra/mysql/run.sh start",
			WorkingDir:   ".",
		},
		{
			Name:         "task-events-x",
			StartCommand: "bash run.sh start billing_transaction_created/1_process_billing_transaction",
			WorkingDir:   "taskEvents",
		},
	}}}}
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })
	cfg.resolveWorkingDirs(confDir)
	if cfg.Groups[0].Services[0].WorkingDir != root {
		t.Fatalf("mysql working_dir=%q want %q", cfg.Groups[0].Services[0].WorkingDir, root)
	}
	if cfg.Groups[0].Services[1].WorkingDir != events {
		t.Fatalf("events working_dir=%q want recipe dir %q", cfg.Groups[0].Services[1].WorkingDir, events)
	}
}

func TestResolveWorkingDirs_DeployModeRewritesServiceStopScript(t *testing.T) {
	t.Setenv("DEPLOY_MODE", "1")
	root := t.TempDir()
	confDir := filepath.Join(root, "conf")
	if err := os.MkdirAll(confDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := &Config{Groups: []Group{{Services: []Service{{
		Name:         "task-git-oauth",
		StartCommand: "./bin/taskGitOauth",
		StopCommand:  "bash scripts/runall-stop.sh",
		WorkingDir:   "taskGitOauth",
		HealthCheck:  HealthCheck{URL: "http://127.0.0.1:8002/api/health/"},
	}}}}}
	cfg.resolveWorkingDirs(confDir)
	svc := cfg.Groups[0].Services[0]
	if svc.WorkingDir != root {
		t.Fatalf("working_dir=%q want %q", svc.WorkingDir, root)
	}
	if !strings.Contains(svc.StopCommand, "lsof -ti:8002") {
		t.Fatalf("stop_command=%q want portable lsof :8002", svc.StopCommand)
	}
}
