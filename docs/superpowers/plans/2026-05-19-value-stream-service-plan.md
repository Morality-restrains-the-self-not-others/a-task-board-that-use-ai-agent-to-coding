# valueStream Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a standalone Go CLI `valueStream` that reads a user-supplied YAML config, runs registered pytest unit-test files per value-stream step, exposes a Web UI for pass/fail testing, and **displays each step’s registered data fields** (`<runAll-service>.<table>.<column>`) with **provider application service** parsed from the field name (validated against runAll `config.yaml` when `runall_config` is set).

**Architecture:** Single Go package (`main`) in `valueStream/`, mirroring `runAll/` patterns: `config.go` + `status.go` + `runner.go` + `ui.go` + embedded `index.html`. No Django/DDD layering (ops tool only); in-memory `StatusStore` holds stream/step `TestRun` results. Global mutex ensures one pytest job at a time.

**Tech Stack:** Go 1.24, `gopkg.in/yaml.v3`, `embed`, `net/http`, `os/exec`

**Spec:** [docs/superpowers/specs/2026-05-19-value-stream-service-design.md](../specs/2026-05-19-value-stream-service-design.md)

**Example config (reference only):** [docs/examples/value-streams.example.yaml](../../examples/value-streams.example.yaml)

**Plan revision:** 2026-05-19 — integrates step `fields` + runAll service validation (no separate late task).

---

## Task index

| # | Deliverable |
|---|-------------|
| 1 | Go module `valueStream/` |
| 2 | `fields.go` + `runall.go` + `config.go`（YAML、`runall_config`、三段式字段校验） |
| 3 | `status.go`（含 `FieldView` / `StepView.fields`） |
| 4 | `runner.go`（pytest；整条流失败继续） |
| 5 | `ui.go`（`/api/streams` 含 fields；test step/stream） |
| 6 | `index.html`（状态灯 + **数据字段表**） |
| 7 | `main.go`（`--config` 必填） |
| 8 | README + `go test ./...` |
| 9 | 手工 E2E（example YAML + 真实 pytest） |

---

## File Map

| File | Responsibility |
|------|----------------|
| `valueStream/go.mod` | Module `valueStream`, yaml.v3 |
| `valueStream/config.go` | YAML structs, `LoadConfig`, validation, path resolution, field parse + runAll service whitelist |
| `valueStream/runall.go` | Minimal runAll YAML parse (`groups[].services[].name`) |
| `valueStream/fields.go` | `ParseFieldName(name) → provider, table, column` |
| `valueStream/config_test.go` | Config load/validate tests |
| `valueStream/status.go` | `StatusStore`, JSON DTOs, stream/step state |
| `valueStream/status_test.go` | Store behavior tests |
| `valueStream/runner.go` | `Runner`: pytest exec, `RunStep`, `RunStream` (continue on fail) |
| `valueStream/runner_test.go` | Mock pytest binary tests |
| `valueStream/ui.go` | `/api/streams`, `/api/test/step`, `/api/test/stream`, `/` |
| `valueStream/ui_test.go` | Handler tests |
| `valueStream/index.html` | Embedded dashboard |
| `valueStream/main.go` | Required `--config`, UI server, graceful shutdown |
| `valueStream/README.md` | Usage and YAML reference |

---

### Task 1: Initialize Go module

**Files:**
- Create: `valueStream/go.mod`
- Create: `valueStream/.gitignore` (optional, `valueStream` binary)

- [ ] **Step 1: Create module**

```bash
mkdir -p valueStream && cd valueStream && go mod init valueStream
```

- [ ] **Step 2: Add yaml dependency**

```bash
cd valueStream && go get gopkg.in/yaml.v3
```

- [ ] **Step 3: Verify module**

```bash
cd valueStream && go mod tidy
```

Expected: `go.mod` contains `go 1.24` (or repo Go version) and `gopkg.in/yaml.v3`.

- [ ] **Step 4: Commit**

```bash
git add valueStream/go.mod valueStream/go.sum
git commit -m "feat(valueStream): init Go module"
```

---

### Task 2: Config, data fields, and runAll service validation

**Files:**
- Create: `valueStream/fields.go`, `valueStream/fields_test.go`
- Create: `valueStream/runall.go`, `valueStream/runall_test.go`
- Create: `valueStream/config.go`
- Create: `valueStream/config_test.go`

- [ ] **Step 1: Write failing field + runAll tests**

Create `valueStream/fields_test.go`:

```go
func TestParseFieldName_Valid(t *testing.T) {
	p, tbl, col, err := ParseFieldName("saas-backend.accounts_user.email")
	if err != nil || p != "saas-backend" || tbl != "accounts_user" || col != "email" {
		t.Fatalf("got %q %q %q err=%v", p, tbl, col, err)
	}
}

func TestParseFieldName_InvalidSegments(t *testing.T) {
	_, _, _, err := ParseFieldName("saas-backend.email")
	if err == nil {
		t.Fatal("want error for two segments")
	}
}
```

Create `valueStream/runall_test.go` — temp YAML with `groups[].services[].name`, assert `LoadRunAllServiceNames` returns `saas-backend`, `git-oauth`.

- [ ] **Step 2: Implement `fields.go` and `runall.go`**

```go
// fields.go
var fieldNameRe = regexp.MustCompile(`^([a-z0-9-]+)\.([a-z0-9_]+)\.([a-z0-9_]+)$`)

func ParseFieldName(name string) (provider, table, column string, err error) {
	m := fieldNameRe.FindStringSubmatch(name)
	if m == nil {
		return "", "", "", fmt.Errorf("field name %q must be <service>.<table>.<column>", name)
	}
	return m[1], m[2], m[3], nil
}
```

```go
// runall.go — minimal struct, collect all services[].name
func LoadRunAllServiceNames(path string) (map[string]bool, error)
```

- [ ] **Step 3: Run field/runall tests**

```bash
cd valueStream && go test ./... -run 'TestParseFieldName|TestLoadRunAll' -v
```

Expected: PASS.

- [ ] **Step 4: Write failing config tests**

Create `valueStream/config_test.go`:

```go
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeConfig(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadConfig_Valid(t *testing.T) {
	dir := t.TempDir()
	wd := filepath.Join(dir, "proj")
	os.MkdirAll(wd, 0755)
	tf := filepath.Join(wd, "sample_test.py")
	os.WriteFile(tf, []byte("# test"), 0644)

	yaml := `
version: "1"
runner:
  working_dir: proj
  pytest_bin: pytest
value_streams:
  - name: flow-a
    description: demo
    steps:
      - name: step1
        test_file: sample_test.py
`
	path := writeConfig(t, dir, "cfg.yaml", yaml)
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Runner.WorkingDir != wd {
		t.Errorf("working_dir = %q, want %q", cfg.Runner.WorkingDir, wd)
	}
	if cfg.ValueStreams[0].Steps[0].TestFileAbs != tf {
		t.Errorf("TestFileAbs = %q, want %q", cfg.ValueStreams[0].Steps[0].TestFileAbs, tf)
	}
}

func TestLoadConfig_DuplicateStreamName(t *testing.T) {
	dir := t.TempDir()
	wd := filepath.Join(dir, "proj")
	os.MkdirAll(wd, 0755)
	os.WriteFile(filepath.Join(wd, "a.py"), []byte(""), 0644)
	yaml := `
version: "1"
runner:
  working_dir: proj
value_streams:
  - name: dup
    steps:
      - name: s1
        test_file: a.py
  - name: dup
    steps:
      - name: s1
        test_file: a.py
`
	_, err := LoadConfig(writeConfig(t, dir, "c.yaml", yaml))
	if err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("want duplicate stream error, got %v", err)
	}
}

func TestLoadConfig_MissingTestFile(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "proj"), 0755)
	yaml := `
version: "1"
runner:
  working_dir: proj
value_streams:
  - name: f
    steps:
      - name: s
        test_file: no_such.py
`
	_, err := LoadConfig(writeConfig(t, dir, "c.yaml", yaml))
	if err == nil || !strings.Contains(err.Error(), "no_such.py") {
		t.Fatalf("want missing test file error, got %v", err)
	}
}

func TestLoadConfig_InvalidFieldProvider(t *testing.T) {
	dir := t.TempDir()
	wd := filepath.Join(dir, "proj")
	os.MkdirAll(wd, 0755)
	os.WriteFile(filepath.Join(wd, "a.py"), []byte(""), 0644)
	runall := writeConfig(t, dir, "runall.yaml", `
version: "1"
groups:
  - name: g
    services:
      - name: saas-backend
        command: "true"
        health_check:
          url: "http://127.0.0.1:1"
`)
	yaml := `
version: "1"
runall_config: runall.yaml
runner:
  working_dir: proj
value_streams:
  - name: f
    steps:
      - name: s
        test_file: a.py
        fields:
          - name: unknown-svc.accounts_user.email
`
	_, err := LoadConfig(writeConfig(t, dir, "vs.yaml", yaml))
	if err == nil || !strings.Contains(err.Error(), "unknown-svc") {
		t.Fatalf("want unknown provider error, got %v", err)
	}
}
```

- [ ] **Step 5: Run config tests — expect FAIL**

```bash
cd valueStream && go test ./... -run TestLoadConfig -v
```

Expected: compile error or `LoadConfig` undefined.

- [ ] **Step 6: Implement `config.go`**

```go
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Version      string        `yaml:"version"`
	RunallConfig string        `yaml:"runall_config"`
	Runner       RunnerConfig  `yaml:"runner"`
	ValueStreams []ValueStream `yaml:"value_streams"`
	ConfigDir    string        `yaml:"-"`
}

type RunnerConfig struct {
	WorkingDir string            `yaml:"working_dir"`
	PytestBin  string            `yaml:"pytest_bin"`
	Env        map[string]string `yaml:"env"`
	PytestArgs []string          `yaml:"pytest_args"`
}

type ValueStream struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Steps       []Step `yaml:"steps"`
}

type StepField struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

type Step struct {
	Name         string      `yaml:"name"`
	TestFile     string      `yaml:"test_file"`
	Fields       []StepField `yaml:"fields"`
	TestFileAbs  string      `yaml:"-"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse yaml: %w", err)
	}
	abs, err := filepath.Abs(filepath.Dir(path))
	if err != nil {
		return nil, err
	}
	cfg.ConfigDir = abs
	cfg.fillDefaults()
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	cfg.resolvePaths()
	return &cfg, nil
}

func (c *Config) fillDefaults() {
	if c.Runner.PytestBin == "" {
		c.Runner.PytestBin = "pytest"
	}
}

func (c *Config) validate() error {
	if c.Version != "1" {
		return fmt.Errorf("unsupported version %q, want \"1\"", c.Version)
	}
	if c.Runner.WorkingDir == "" {
		return fmt.Errorf("runner.working_dir is required")
	}
	if len(c.ValueStreams) == 0 {
		return fmt.Errorf("at least one value_streams entry required")
	}
	seenStream := map[string]bool{}
	for _, vs := range c.ValueStreams {
		if vs.Name == "" {
			return fmt.Errorf("value stream name is required")
		}
		if seenStream[vs.Name] {
			return fmt.Errorf("duplicate value stream name %q", vs.Name)
		}
		seenStream[vs.Name] = true
		if len(vs.Steps) == 0 {
			return fmt.Errorf("stream %q: at least one step required", vs.Name)
		}
		seenStep := map[string]bool{}
		for _, st := range vs.Steps {
			if st.Name == "" {
				return fmt.Errorf("stream %q: step name required", vs.Name)
			}
			if seenStep[st.Name] {
				return fmt.Errorf("stream %q: duplicate step name %q", vs.Name, st.Name)
			}
			seenStep[st.Name] = true
			if st.TestFile == "" {
				return fmt.Errorf("stream %q step %q: test_file required", vs.Name, st.Name)
			}
			seenField := map[string]bool{}
			for _, f := range st.Fields {
				if _, _, _, err := ParseFieldName(f.Name); err != nil {
					return fmt.Errorf("stream %q step %q: %w", vs.Name, st.Name, err)
				}
				if seenField[f.Name] {
					return fmt.Errorf("stream %q step %q: duplicate field %q", vs.Name, st.Name, f.Name)
				}
				seenField[f.Name] = true
			}
		}
	}
	var runAllNames map[string]bool
	if c.RunallConfig != "" {
		p := filepath.Join(c.ConfigDir, c.RunallConfig)
		var err error
		runAllNames, err = LoadRunAllServiceNames(p)
		if err != nil {
			return fmt.Errorf("runall_config: %w", err)
		}
	} else {
		log.Printf("[warn] runall_config not set; field provider names not validated against runAll")
	}
	if runAllNames != nil {
		for _, vs := range c.ValueStreams {
			for _, st := range vs.Steps {
				for _, f := range st.Fields {
					prov, _, _, _ := ParseFieldName(f.Name)
					if !runAllNames[prov] {
						return fmt.Errorf("field %q: provider %q not in runAll config", f.Name, prov)
					}
				}
			}
		}
	}
	wd := c.runnerWorkingDirAbs()
	info, err := os.Stat(wd)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("runner.working_dir not found: %s", wd)
	}
	for i := range c.ValueStreams {
		for j := range c.ValueStreams[i].Steps {
			abs := filepath.Join(wd, c.ValueStreams[i].Steps[j].TestFile)
			if _, err := os.Stat(abs); err != nil {
				return fmt.Errorf("test_file not found: %s", abs)
			}
		}
	}
	return nil
}

func (c *Config) runnerWorkingDirAbs() string {
	return filepath.Join(c.ConfigDir, c.Runner.WorkingDir)
}

func (c *Config) resolvePaths() {
	wd := c.runnerWorkingDirAbs()
	for i := range c.ValueStreams {
		for j := range c.ValueStreams[i].Steps {
			c.ValueStreams[i].Steps[j].TestFileAbs = filepath.Join(wd, c.ValueStreams[i].Steps[j].TestFile)
		}
	}
}

func (c *Config) StreamByName(name string) (*ValueStream, error) {
	for i := range c.ValueStreams {
		if c.ValueStreams[i].Name == name {
			return &c.ValueStreams[i], nil
		}
	}
	return nil, fmt.Errorf("unknown stream %q", name)
}

func (vs *ValueStream) StepByName(name string) (*Step, error) {
	for i := range vs.Steps {
		if vs.Steps[i].Name == name {
			return &vs.Steps[i], nil
		}
	}
	return nil, fmt.Errorf("unknown step %q", name)
}
```

- [ ] **Step 7: Run all Task 2 tests — expect PASS**

```bash
cd valueStream && go test ./... -run 'TestLoadConfig|TestParseFieldName|TestLoadRunAll' -v
```

- [ ] **Step 8: Commit**

```bash
git add valueStream/config.go valueStream/config_test.go \
  valueStream/fields.go valueStream/fields_test.go \
  valueStream/runall.go valueStream/runall_test.go
git commit -m "feat(valueStream): config, step fields, runAll service validation"
```

---

### Task 3: Status store and JSON DTOs

**Files:**
- Create: `valueStream/status.go`
- Create: `valueStream/status_test.go`

- [ ] **Step 1: Write failing tests**

`valueStream/status_test.go`:

```go
package main

import (
	"testing"
	"time"
)

func TestStatusStore_StreamRunSetsFailedSteps(t *testing.T) {
	cfg := minimalConfig()
	store := NewStatusStore(cfg)
	store.BeginStreamRun("flow-a")
	store.SetStepRunning("flow-a", "s1")
	store.FinishStep("flow-a", "s1", TestRun{Passed: true, ExitCode: 0})
	store.SetStepRunning("flow-a", "s2")
	store.FinishStep("flow-a", "s2", TestRun{Passed: false, ExitCode: 1, LogTail: "fail"})
	store.SetStepRunning("flow-a", "s3")
	store.FinishStep("flow-a", "s3", TestRun{Passed: true, ExitCode: 0})
	store.EndStreamRun("flow-a")

	view := store.StreamView("flow-a")
	if view.StreamStatus != StreamStatusFailed {
		t.Fatalf("stream_status = %q, want failed", view.StreamStatus)
	}
	if len(view.FailedSteps) != 1 || view.FailedSteps[0] != "s2" {
		t.Fatalf("failed_steps = %v, want [s2]", view.FailedSteps)
	}
}

func minimalConfig() *Config {
	return &Config{
		ValueStreams: []ValueStream{{
			Name: "flow-a",
			Steps: []Step{
				{Name: "s1", TestFile: "a.py"},
				{Name: "s2", TestFile: "b.py"},
				{Name: "s3", TestFile: "c.py"},
			},
		}},
	}
}
```

- [ ] **Step 2: Run — expect FAIL**

```bash
cd valueStream && go test ./... -run TestStatusStore -v
```

- [ ] **Step 3: Implement `status.go`**

Key types and behavior:

```go
type StreamStatus string
const (
	StreamStatusUnknown StreamStatus = "unknown"
	StreamStatusRunning StreamStatus = "running"
	StreamStatusPassed  StreamStatus = "passed"
	StreamStatusFailed  StreamStatus = "failed"
)

type StepStatus string
const (
	StepStatusPending StepStatus = "pending"
	StepStatusRunning StepStatus = "running"
	StepStatusPassed  StepStatus = "passed"
	StepStatusFailed  StepStatus = "failed"
)

type TestRun struct {
	StartedAt  string `json:"started_at"`
	DurationMs int64  `json:"duration_ms"`
	ExitCode   int    `json:"exit_code"`
	Passed     bool   `json:"passed"`
	LogTail    string `json:"log_tail"`
}

type FieldView struct {
	Name            string `json:"name"`
	ProviderService string `json:"provider_service"`
	Table           string `json:"table"`
	Column          string `json:"column"`
	Description     string `json:"description,omitempty"`
}

type StepView struct {
	Name     string      `json:"name"`
	TestFile string      `json:"test_file"`
	Fields   []FieldView `json:"fields,omitempty"`
	Status   StepStatus  `json:"status"`
	LastRun  *TestRun    `json:"last_run,omitempty"`
}

type StreamView struct {
	Name              string       `json:"name"`
	Description       string       `json:"description"`
	StreamStatus      StreamStatus `json:"stream_status"`
	FailedSteps       []string     `json:"failed_steps,omitempty"`
	LastStreamRunAt   string       `json:"last_stream_run_at,omitempty"`
	Steps             []StepView   `json:"steps"`
}

type StreamsResponse struct {
	Streams []StreamView `json:"streams"`
}
```

`StatusStore` fields:

- `mu sync.RWMutex`
- `cfg *Config`
- `busy bool` + `busyReason string` (for 409)
- per stream: `streamStatus`, `failedSteps`, `lastStreamRunAt`
- per (stream, step): `stepStatus`, `lastRun *TestRun`

Methods:

- `NewStatusStore(cfg *Config) *StatusStore` — init all steps `pending`, streams `unknown`
- `TryAcquire() bool` / `Release()`
- `BeginStreamRun(stream)` → stream `running`, clear prior `failedSteps` for this run
- `EndStreamRun(stream)` → compute `passed`/`failed` from step statuses after run; set `lastStreamRunAt`
- `SetStepRunning`, `FinishStep` — update step; **do not** change stream-level status on single-step runs (MVP)
- `AllStreams() StreamsResponse` — build `StepView.Fields` from config `Step.Fields` via `ParseFieldName` (static metadata, not test results)
- `StreamView(name) StreamView`

**MVP rule:** `stream_status` only updates in `EndStreamRun` (whole stream). Single-step `FinishStep` updates step only.

- [ ] **Step 4: Run tests — expect PASS**

```bash
cd valueStream && go test ./... -run TestStatusStore -v
```

- [ ] **Step 5: Commit**

```bash
git add valueStream/status.go valueStream/status_test.go
git commit -m "feat(valueStream): in-memory status store"
```

---

### Task 4: Pytest runner (single step + full stream)

**Files:**
- Create: `valueStream/runner.go`
- Create: `valueStream/runner_test.go`

- [ ] **Step 1: Write failing test with mock pytest**

`runner_test.go` — create fake pytest script in temp dir:

```go
func TestRunner_RunStep_MockPytest(t *testing.T) {
	dir := t.TempDir()
	mockPytest := filepath.Join(dir, "mock-pytest")
	script := `#!/bin/sh
if [ "$1" = "fail.py" ]; then exit 1; fi
exit 0
`
	os.WriteFile(mockPytest, []byte(script), 0755)

	proj := filepath.Join(dir, "proj")
	os.MkdirAll(proj, 0755)
	passFile := filepath.Join(proj, "pass.py")
	failFile := filepath.Join(proj, "fail.py")
	os.WriteFile(passFile, []byte(""), 0644)
	os.WriteFile(failFile, []byte(""), 0644)

	cfg := &Config{
		ConfigDir: dir,
		Runner: RunnerConfig{WorkingDir: proj, PytestBin: mockPytest},
		ValueStreams: []ValueStream{{
			Name: "f",
			Steps: []Step{
				{Name: "ok", TestFile: "pass.py", TestFileAbs: passFile},
				{Name: "bad", TestFile: "fail.py", TestFileAbs: failFile},
			},
		}},
	}
	store := NewStatusStore(cfg)
	r := NewRunner(cfg, store)

	run, err := r.runPytest(context.Background(), passFile)
	if err != nil {
		t.Fatal(err)
	}
	if !run.Passed || run.ExitCode != 0 {
		t.Fatalf("pass run: %+v", run)
	}

	run2, err := r.runPytest(context.Background(), failFile)
	if err != nil {
		t.Fatal(err)
	}
	if run2.Passed || run2.ExitCode == 0 {
		t.Fatalf("fail run: %+v", run2)
	}
}

func TestRunner_RunStream_ContinuesOnFailure(t *testing.T) {
	// same mock setup; RunStream("f") must invoke both steps even when fail.py fails
	// assert store.StreamView failed_steps contains "bad" and ok step passed
}
```

- [ ] **Step 2: Run — expect FAIL**

```bash
cd valueStream && go test ./... -run TestRunner -v
```

- [ ] **Step 3: Implement `runner.go`**

```go
type Runner struct {
	cfg   *Config
	store *StatusStore
}

func NewRunner(cfg *Config, store *StatusStore) *Runner {
	return &Runner{cfg: cfg, store: store}
}

const logTailMaxLines = 80

func tailLines(s string, n int) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) <= n {
		return s
	}
	return strings.Join(lines[len(lines)-n:], "\n")
}

func (r *Runner) runPytest(ctx context.Context, testFileAbs string) (TestRun, error) {
	start := time.Now()
	args := append([]string{}, r.cfg.Runner.PytestArgs...)
	args = append(args, testFileAbs)
	cmd := exec.CommandContext(ctx, r.cfg.Runner.PytestBin, args...)
	cmd.Dir = r.cfg.runnerWorkingDirAbs()
	cmd.Env = mergeEnv(os.Environ(), r.cfg.Runner.Env)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	combined := stdout.String() + stderr.String()
	exitCode := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			exitCode = ee.ExitCode()
		} else {
			return TestRun{}, err
		}
	}
	passed := exitCode == 0
	return TestRun{
		StartedAt:  start.Format(time.RFC3339),
		DurationMs: time.Since(start).Milliseconds(),
		ExitCode:   exitCode,
		Passed:     passed,
		LogTail:    tailLines(combined, logTailMaxLines),
	}, nil
}

func (r *Runner) RunStep(ctx context.Context, streamName, stepName string) error {
	if !r.store.TryAcquire() {
		return errBusy
	}
	defer r.store.Release()
	vs, err := r.cfg.StreamByName(streamName)
	if err != nil {
		return err
	}
	st, err := vs.StepByName(stepName)
	if err != nil {
		return err
	}
	r.store.SetStepRunning(streamName, stepName)
	run, err := r.runPytest(ctx, st.TestFileAbs)
	if err != nil {
		r.store.FinishStep(streamName, stepName, TestRun{Passed: false, ExitCode: -1, LogTail: err.Error()})
		return nil
	}
	r.store.FinishStep(streamName, stepName, run)
	return nil
}

func (r *Runner) RunStream(ctx context.Context, streamName string) error {
	if !r.store.TryAcquire() {
		return errBusy
	}
	defer r.store.Release()
	vs, err := r.cfg.StreamByName(streamName)
	if err != nil {
		return err
	}
	r.store.BeginStreamRun(streamName)
	for _, st := range vs.Steps {
		select {
		case <-ctx.Done():
			r.store.EndStreamRun(streamName)
			return ctx.Err()
		default:
		}
		r.store.SetStepRunning(streamName, st.Name)
		run, err := r.runPytest(ctx, st.TestFileAbs)
		if err != nil {
			run = TestRun{Passed: false, ExitCode: -1, LogTail: err.Error()}
		}
		r.store.FinishStep(streamName, st.Name, run)
		// no break on failure — continue all steps
	}
	r.store.EndStreamRun(streamName)
	return nil
}

var errBusy = errors.New("a test is already running")
```

Add `mergeEnv` helper in same file.

- [ ] **Step 4: Run tests — expect PASS**

```bash
cd valueStream && go test ./... -run TestRunner -v
```

- [ ] **Step 5: Commit**

```bash
git add valueStream/runner.go valueStream/runner_test.go
git commit -m "feat(valueStream): pytest runner with continue-on-fail stream"
```

---

### Task 5: HTTP API and handlers

**Files:**
- Create: `valueStream/ui.go`
- Create: `valueStream/ui_test.go`

- [ ] **Step 1: Write failing tests**

```go
func TestAPIStreams(t *testing.T) {
	cfg := minimalConfig()
	store := NewStatusStore(cfg)
	mux := http.NewServeMux()
	r := NewRunner(cfg, store)
	registerUIHandlers(mux, store, r)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", "/api/streams", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("code %d", rec.Code)
	}
	var resp StreamsResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	if len(resp.Streams) != 1 {
		t.Fatalf("len %d", len(resp.Streams))
	}
}

func TestAPITestStream_Busy409(t *testing.T) {
	cfg := minimalConfig()
	store := NewStatusStore(cfg)
	store.busy = true // or expose TryAcquire that leaves busy
	// POST /api/test/stream while busy → 409
}
```

- [ ] **Step 2: Implement `ui.go`**

Handlers (mirror spec):

- `GET /api/streams` → JSON `StreamsResponse`
- `POST /api/test/step` — decode `{stream, step}`; `go runner.RunStep(context.Background(), ...)` if acquired else 409
- `POST /api/test/stream` — decode `{stream}`; async `RunStream`
- `GET /` — serve embedded `index.html`

Use `sync`/`TryAcquire` before spawning goroutine; return immediately with `{"status":"started"}` on success.

```go
//go:embed index.html
var indexHTML embed.FS

func registerUIHandlers(mux *http.ServeMux, store *StatusStore, runner *Runner) { ... }
func startUIServer(store *StatusStore, runner *Runner, port string) *http.Server { ... }
```

409 body: `{"error":"a test is already running"}`

- [ ] **Step 3: Run tests**

```bash
cd valueStream && go test ./... -run TestAPI -v
```

- [ ] **Step 4: Commit**

```bash
git add valueStream/ui.go valueStream/ui_test.go
git commit -m "feat(valueStream): HTTP API handlers"
```

---

### Task 6: Web UI (`index.html`)

**Files:**
- Create: `valueStream/index.html`

- [ ] **Step 1: Create dashboard HTML**

Base on `runAll/status.html` styling (dark theme, colored dots). Requirements:

- Header: title `valueStream`, subtitle「不依赖 runAll；需本机 pytest 与 Saas_project 测试依赖」
- Poll `GET /api/streams` every 2s
- Per stream card:
  - name, description
  - stream dot: gray=`unknown`, yellow=`running`, green=`passed`, red=`failed`
  - if `failed_steps.length`, show red list「失败环节: …」
  - button「测试整条流」→ `POST /api/test/stream` with `{stream:name}`
- Per step row (indented):
  - step name, basename of `test_file`
  - step dot from `status`
  - button「测试本环节」→ `POST /api/test/step`
  - **if `fields.length`:** table columns「数据字段 | 数据库表 | 列 | 应用服务」; optional `description` as second line or `title` attr
  - if `last_run.log_tail`, `<details>` expandable
- Disable all test buttons when any stream has `stream_status === "running"` or any step `running` (or track 409)

- [ ] **Step 2: Wire embed in `ui.go`** (if not done in Task 5)

- [ ] **Step 3: Manual smoke**

```bash
cd valueStream && go build -o valueStream .
./valueStream --config ../docs/examples/value-streams.example.yaml
# open http://localhost:9998 — page loads, streams listed
```

- [ ] **Step 4: Commit**

```bash
git add valueStream/index.html
git commit -m "feat(valueStream): Web UI dashboard"
```

---

### Task 7: CLI entrypoint (`main.go`)

**Files:**
- Create: `valueStream/main.go`

- [ ] **Step 1: Implement main**

```go
func main() {
	configPath := flag.String("config", "", "Path to YAML configuration file (required)")
	uiPort := flag.String("ui-port", ":9998", "Web UI listen address")
	flag.Parse()

	if *configPath == "" {
		fmt.Fprintln(os.Stderr, "error: --config is required")
		flag.Usage()
		os.Exit(2)
	}

	cfg, err := LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Config error: %v", err)
	}

	store := NewStatusStore(cfg)
	runner := NewRunner(cfg, store)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	srv := startUIServer(store, runner, *uiPort)
	log.Printf("valueStream UI: http://localhost%s", *uiPort)
	log.Printf("Config: %s", *configPath)

	<-ctx.Done()
	log.Println("Shutting down...")
	srv.Shutdown(context.Background())
}
```

- [ ] **Step 2: Build and verify required flag**

```bash
cd valueStream && go build -o valueStream .
./valueStream 2>&1; echo exit:$?
```

Expected: exit code 2, message `--config is required`.

```bash
./valueStream --config ../docs/examples/value-streams.example.yaml &
sleep 1
curl -s http://localhost:9998/api/streams | head -c 200
kill %1
```

- [ ] **Step 3: Commit**

```bash
git add valueStream/main.go
git commit -m "feat(valueStream): CLI with required --config"
```

---

### Task 8: README and full test suite

**Files:**
- Create: `valueStream/README.md`

- [ ] **Step 1: Write README**

Include:

- Build: `go build -o valueStream .`
- Usage: `./valueStream --config <path> [--ui-port :9998]`
- YAML: `runall_config`, `steps[].fields[].name` 格式 `<runAll服务>.<表>.<列>`（应用服务名须存在于 runAll `config.yaml`）
- Example: `docs/examples/value-streams.example.yaml`
- Note: pytest 需 `task2app` 环境；valueStream 不启动 runAll（`runall_config` 仅校验命名）

- [ ] **Step 2: Run all tests**

```bash
cd valueStream && go test ./... -v
```

Expected: all PASS.

- [ ] **Step 3: Commit**

```bash
git add valueStream/README.md
git commit -m "docs(valueStream): add README"
```

---

### Task 9: End-to-end manual verification (optional but recommended)

- [ ] **Step 1: Use example config against real Saas_project tests**

Prerequisite: `task2app` venv active, dependencies installed.

```bash
cd valueStream && go build -o valueStream .
./valueStream --config ../docs/examples/value-streams.example.yaml
```

In browser:

1. Confirm `user-auth` steps show **数据字段表**（如 `saas-backend.accounts_user.email` → 提供方 `saas-backend`）。
2. Click「测试整条流」— pass/fail dots; failed stream lists `failed_steps`.
3. Click「测试本环节」— only that row updates; stream-level dot unchanged until full stream run (MVP).

- [ ] **Step 2: Document any env issues in README** (e.g. `pytest` not on PATH)

---

## Spec Coverage Checklist

| Spec requirement | Task |
|------------------|------|
| Independent `valueStream/` | Task 1–8 |
| YAML `runner` + `value_streams` | Task 2 |
| `--config` required, no default file | Task 7 |
| `--ui-port` default `:9998` | Task 7 |
| pytest per `test_file` | Task 4 |
| Stream run continues on failure + `failed_steps` | Task 3, 4 |
| Stream status MVP (unknown until full run) | Task 3 |
| Single global test mutex / 409 | Task 3, 5 |
| GET `/api/streams`, POST step/stream | Task 5 |
| Web UI per spec | Task 6 |
| Startup validation (missing test file) | Task 2 |
| No runAll dependency | README Task 8 |
| Example config exists | already at `docs/examples/value-streams.example.yaml` |
| YAML `fields` + 三段式命名 | Task 2 |
| `runall_config` → runAll 服务名校验 | Task 2 |
| API/UI `fields` + `FieldView` | Task 3, 5 |
| UI 字段表 + 应用服务列 | Task 6 |

## DDD Note

This tool is **not** `Saas_project` backend code. No domain/infrastructure/interfaces split required. Internal model (`ValueStream`, `Step`, `TestRun`, `StatusStore`) matches the spec’s lightweight ops-tool domain only.

---

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-05-19-value-stream-service-plan.md`.

**Two execution options:**

1. **Subagent-Driven (recommended)** — dispatch a fresh subagent per task, review between tasks  
2. **Inline Execution** — implement task-by-task in this session with checkpoints  

Which approach do you want?
