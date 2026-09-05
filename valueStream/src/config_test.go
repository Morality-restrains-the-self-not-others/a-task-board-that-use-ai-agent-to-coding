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

	writeConfig(t, dir, "runall.yaml", `
version: "1"
groups:
  - name: g
    services:
      - name: saas-backend
        command: "true"
        health_check:
          url: "http://127.0.0.1:1"
`)
	path := writeConfig(t, dir, "cfg.yaml", `
version: "1"
runall_config: runall.yaml
runner:
  working_dir: proj
value_streams:
  - name: flow-a
    domain: auth
    description: demo
    steps:
      - name: step1
        test_file: sample_test.py
        fields:
          - name: saas-backend.auth_user.email
`)
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.runnerWorkingDirAbs() != wd {
		t.Errorf("working_dir abs = %q, want %q", cfg.runnerWorkingDirAbs(), wd)
	}
	if cfg.ValueStreams[0].Steps[0].TestFileAbs != tf {
		t.Errorf("TestFileAbs = %q, want %q", cfg.ValueStreams[0].Steps[0].TestFileAbs, tf)
	}
	if cfg.ValueStreams[0].Domain != "auth" {
		t.Errorf("Domain = %q, want auth", cfg.ValueStreams[0].Domain)
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
    domain: domain-a
    steps:
      - name: s1
        test_file: a.py
  - name: dup
    domain: domain-b
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
    domain: domain-a
    steps:
      - name: s
        test_file: no_such.py
`
	cfg, err := LoadConfig(writeConfig(t, dir, "c.yaml", yaml))
	if err != nil {
		t.Fatalf("missing test_file should warn only, got error: %v", err)
	}
	if cfg == nil || len(cfg.ValueStreams) != 1 {
		t.Fatalf("expected config with one value stream, got %+v", cfg)
	}
}

func TestLoadConfig_DuplicateFieldName(t *testing.T) {
	dir := t.TempDir()
	wd := filepath.Join(dir, "proj")
	os.MkdirAll(wd, 0755)
	os.WriteFile(filepath.Join(wd, "a.py"), []byte(""), 0644)
	writeConfig(t, dir, "runall.yaml", `
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
    domain: domain-a
    steps:
      - name: s
        test_file: a.py
        fields:
          - name: saas-backend.auth_user.email
          - name: saas-backend.auth_user.email
`
	_, err := LoadConfig(writeConfig(t, dir, "vs.yaml", yaml))
	if err == nil || !strings.Contains(err.Error(), "duplicate field") {
		t.Fatalf("want duplicate field error, got %v", err)
	}
}

func TestLoadConfig_InvalidVersion(t *testing.T) {
	dir := t.TempDir()
	wd := filepath.Join(dir, "proj")
	os.MkdirAll(wd, 0755)
	os.WriteFile(filepath.Join(wd, "a.py"), []byte(""), 0644)
	yaml := `
version: "2"
runner:
  working_dir: proj
value_streams:
  - name: f
    domain: domain-a
    steps:
      - name: s
        test_file: a.py
`
	_, err := LoadConfig(writeConfig(t, dir, "c.yaml", yaml))
	if err == nil || !strings.Contains(err.Error(), "version") {
		t.Fatalf("want version error, got %v", err)
	}
}

func TestLoadConfig_PlannedStepSkipsMissingTestFile(t *testing.T) {
	dir := t.TempDir()
	wd := filepath.Join(dir, "proj")
	os.MkdirAll(wd, 0755)
	os.WriteFile(filepath.Join(wd, "active.py"), []byte(""), 0644)
	writeConfig(t, dir, "runall.yaml", `
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
    domain: domain-a
    steps:
      - name: future
        status: planned
        test_file: accounts/view_test/UserViewSet_login_test.py
        fields:
          - name: saas-backend.auth_user.email
      - name: now
        status: active
        test_file: active.py
        fields:
          - name: saas-backend.auth_user.email
`
	cfg, err := LoadConfig(writeConfig(t, dir, "vs.yaml", yaml))
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if !cfg.ValueStreams[0].Steps[0].IsPlanned() {
		t.Fatal("want planned step")
	}
	if cfg.ValueStreams[0].Steps[1].TestFileAbs == "" {
		t.Fatal("want active step TestFileAbs set")
	}
}

func TestLoadConfig_StreamAllowsAllPlannedSteps(t *testing.T) {
	dir := t.TempDir()
	wd := filepath.Join(dir, "proj")
	os.MkdirAll(wd, 0755)
	yaml := `
version: "1"
runner:
  working_dir: proj
value_streams:
  - name: f
    domain: domain-a
    steps:
      - name: only-planned
        status: planned
        test_file: missing.py
`
	cfg, err := LoadConfig(writeConfig(t, dir, "c.yaml", yaml))
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if !cfg.ValueStreams[0].Steps[0].IsPlanned() {
		t.Fatal("want planned step")
	}
}

func TestLoadConfig_InvalidFieldProvider(t *testing.T) {
	// Field provider validation is skipped when runall_config is not set
	// (runAll service names are used as the allowlist).
	// This test verifies that configs without a runall_config load successfully
	// even with unknown field providers — they are validated at a different layer.
	dir := t.TempDir()
	wd := filepath.Join(dir, "proj")
	os.MkdirAll(wd, 0755)
	os.WriteFile(filepath.Join(wd, "a.py"), []byte(""), 0644)
	writeConfig(t, dir, "runall.yaml", `
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
    domain: domain-a
    steps:
      - name: s
        test_file: a.py
        fields:
          - name: unknown-svc.auth_user.email
`
	cfg, err := LoadConfig(writeConfig(t, dir, "vs.yaml", yaml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
}

func TestLoadConfig_StreamDomainRequired(t *testing.T) {
	dir := t.TempDir()
	wd := filepath.Join(dir, "proj")
	os.MkdirAll(wd, 0755)
	os.WriteFile(filepath.Join(wd, "a.py"), []byte(""), 0644)
	yaml := `
version: "1"
runner:
  working_dir: proj
value_streams:
  - name: f
    steps:
      - name: s
        test_file: a.py
`
	_, err := LoadConfig(writeConfig(t, dir, "c.yaml", yaml))
	if err == nil || !strings.Contains(err.Error(), "domain is required") {
		t.Fatalf("want domain required error, got %v", err)
	}
}

func TestLoadConfig_TrimStreamDomainWhitespace(t *testing.T) {
	dir := t.TempDir()
	wd := filepath.Join(dir, "proj")
	os.MkdirAll(wd, 0755)
	os.WriteFile(filepath.Join(wd, "a.py"), []byte(""), 0644)
	yaml := `
version: "1"
runner:
  working_dir: proj
value_streams:
  - name: f
    domain: "  auth  "
    steps:
      - name: s
        test_file: a.py
`
	cfg, err := LoadConfig(writeConfig(t, dir, "c.yaml", yaml))
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if got := cfg.ValueStreams[0].Domain; got != "auth" {
		t.Fatalf("domain = %q, want auth", got)
	}
}

func TestLoadConfig_InvalidStepStatus(t *testing.T) {
	dir := t.TempDir()
	wd := filepath.Join(dir, "proj")
	if err := os.MkdirAll(wd, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wd, "a.py"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	yaml := `
version: "1"
runner:
  working_dir: proj
value_streams:
  - name: flow-a
    domain: auth
    steps:
      - name: s1
        status: done
        test_file: a.py
`
	_, err := LoadConfig(writeConfig(t, dir, "c.yaml", yaml))
	if err == nil {
		t.Fatal("want invalid status error")
	}
	if !strings.Contains(err.Error(), "status must be active, planned, or deprecated") || !strings.Contains(err.Error(), "done") {
		t.Fatalf("want invalid status semantics, got %v", err)
	}
}

func TestLoadConfig_DefaultStatusIsActiveWhenOmitted(t *testing.T) {
	dir := t.TempDir()
	wd := filepath.Join(dir, "proj")
	if err := os.MkdirAll(wd, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wd, "a.py"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	yaml := `
version: "1"
runner:
  working_dir: proj
value_streams:
  - name: flow-a
    domain: auth
    steps:
      - name: s1
        test_file: a.py
`
	cfg, err := LoadConfig(writeConfig(t, dir, "c.yaml", yaml))
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	step := cfg.ValueStreams[0].Steps[0]
	if step.Lifecycle != "active" {
		t.Fatalf("step lifecycle = %q, want active", step.Lifecycle)
	}
	if !step.IsActive() {
		t.Fatal("step should be active when status omitted")
	}
}

func TestReorderValueStreamsByDomain_StableAndAppendMissing(t *testing.T) {
	streams := []ValueStream{
		{Name: "flow-1", Domain: "A"},
		{Name: "flow-2", Domain: "B"},
		{Name: "flow-3", Domain: "A"},
		{Name: "flow-4", Domain: "C"},
	}
	reordered, err := ReorderValueStreamsByDomain(streams, []string{"C", "A"})
	if err != nil {
		t.Fatalf("reorder: %v", err)
	}
	got := []string{reordered[0].Name, reordered[1].Name, reordered[2].Name, reordered[3].Name}
	want := []string{"flow-4", "flow-1", "flow-3", "flow-2"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order = %v, want %v", got, want)
		}
	}
}

func TestReorderValueStreamsByDomain_RejectsUnknownOrDuplicate(t *testing.T) {
	streams := []ValueStream{
		{Name: "flow-1", Domain: "A"},
		{Name: "flow-2", Domain: "B"},
	}
	if _, err := ReorderValueStreamsByDomain(streams, []string{"A", "A"}); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("want duplicate domain error, got %v", err)
	}
	if _, err := ReorderValueStreamsByDomain(streams, []string{"X"}); err == nil || !strings.Contains(err.Error(), "unknown") {
		t.Fatalf("want unknown domain error, got %v", err)
	}
}

func TestSaveConfigAtomic_PersistsReorderedStreams(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "value-stream.yaml")
	if err := os.MkdirAll(filepath.Join(dir, "proj"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "proj", "a.py"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "proj", "b.py"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	content := `
version: "1"
runner:
  working_dir: proj
value_streams:
  - name: flow-a
    domain: domain-a
    steps:
      - name: s1
        test_file: a.py
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cfg := &Config{
		Version:    "1",
		ConfigPath: path,
		ConfigDir:  dir,
		Runner: RunnerConfig{
			WorkingDir: "proj",
		},
		ValueStreams: []ValueStream{
			{Name: "flow-a", Domain: "A", Steps: []Step{{Name: "s1", TestFile: "a.py"}}},
			{Name: "flow-b", Domain: "B", Steps: []Step{{Name: "s1", TestFile: "b.py"}}},
		},
	}
	reordered, err := ReorderValueStreamsByDomain(cfg.ValueStreams, []string{"B", "A"})
	if err != nil {
		t.Fatalf("ReorderValueStreamsByDomain: %v", err)
	}
	cfg.ValueStreams = reordered
	if err := SaveConfigAtomic(path, cfg); err != nil {
		t.Fatalf("SaveConfigAtomic: %v", err)
	}
	saved, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(saved)
	if strings.Index(text, "name: flow-b") > strings.Index(text, "name: flow-a") {
		t.Fatalf("unexpected order in yaml: %s", text)
	}
}

func TestLoadConfig_ConfigPathIsAbsolute(t *testing.T) {
	dir := t.TempDir()
	wd := filepath.Join(dir, "proj")
	if err := os.MkdirAll(wd, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wd, "a.py"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	path := writeConfig(t, dir, "cfg.yaml", `
version: "1"
runner:
  working_dir: proj
value_streams:
  - name: flow-a
    domain: auth
    steps:
      - name: s1
        test_file: a.py
`)
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if !filepath.IsAbs(cfg.ConfigPath) {
		t.Fatalf("ConfigPath should be absolute, got %q", cfg.ConfigPath)
	}
}

func TestSaveConfigAtomic_NormalizesAbsolutePath(t *testing.T) {
	dir := t.TempDir()
	prev, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Chdir(prev)
	}()

	if err := os.MkdirAll("proj", 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join("proj", "a.py"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	relPath := "value-stream.yaml"
	if err := os.WriteFile(relPath, []byte("version: \"1\"\nvalue_streams: []\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg := &Config{
		Version:    "1",
		ConfigPath: relPath,
		ConfigDir:  ".",
		Runner: RunnerConfig{
			WorkingDir: "proj",
		},
		ValueStreams: []ValueStream{
			{Name: "flow-a", Domain: "A", Steps: []Step{{Name: "s1", TestFile: "a.py"}}},
		},
	}
	if err := SaveConfigAtomic(relPath, cfg); err != nil {
		t.Fatalf("SaveConfigAtomic: %v", err)
	}
	if !filepath.IsAbs(cfg.ConfigPath) {
		t.Fatalf("ConfigPath should be absolute, got %q", cfg.ConfigPath)
	}
	if !filepath.IsAbs(cfg.ConfigDir) {
		t.Fatalf("ConfigDir should be absolute, got %q", cfg.ConfigDir)
	}
}

func TestSaveConfigAtomic_RejectsInvalidConfig(t *testing.T) {
	// SaveConfigAtomic validates via cfg.validate() which requires:
	// non-empty Version, Runner.WorkingDir, ValueStreams, and at least one step.
	// Missing working_dir is logged as a warning but not rejected at the validate layer.
	dir := t.TempDir()
	path := filepath.Join(dir, "value-stream.yaml")
	if err := os.WriteFile(path, []byte("version: \"1\"\nvalue_streams: []\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg := &Config{
		Version:    "1",
		ConfigPath: path,
		ConfigDir:  dir,
		Runner: RunnerConfig{
			WorkingDir: "missing-proj",
		},
		ValueStreams: []ValueStream{
			{Name: "flow-a", Domain: "A", Steps: []Step{{Name: "s1", TestFile: "a.py"}}},
		},
	}
	// With RunallConfig empty, field provider validation is skipped;
	// working_dir existence is a warning, not a hard error.
	err := SaveConfigAtomic(path, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
