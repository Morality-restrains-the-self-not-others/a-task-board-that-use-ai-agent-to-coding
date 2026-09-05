# runAll Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Go CLI that reads a YAML config, starts services in DAG order, health-checks them, and serves a Web UI status dashboard.

**Architecture:** Single Go package (`main`) in `runAll/` directory. 6 source files + 1 embedded HTML + example YAML. Uses `gopkg.in/yaml.v3` for parsing, `embed.FS` for the UI, zero other external dependencies.

**Tech Stack:** Go 1.24, `gopkg.in/yaml.v3`, `embed`, `net/http`, `os/exec`

---

### Task 1: Initialize Go module and example config

**Files:**
- Create: `runAll/go.mod`
- Create: `runAll/config.yaml`

- [ ] **Step 1: Create go.mod**

```bash
mkdir -p runAll && cd runAll && go mod init runAll
```

- [ ] **Step 2: Write example config**

Write `runAll/config.yaml`:

```yaml
version: "1"
groups:
  - name: infrastructure
    services:
      - name: redis
        command: "redis-server --port 6379"
        health_check:
          url: "http://localhost:6379"
          timeout: 30
          retries: 10
          backoff:
            initial: 1.0
            max: 8.0
            multiplier: 2.0
        on_failure: exit

      - name: kafka
        command: "docker start kafka"
        health_check:
          url: "http://localhost:9092"

  - name: apps
    services:
      - name: saas-backend
        command: "python manage.py runserver 0.0.0.0:8000"
        health_check:
          url: "http://localhost:8000/api/health"
        depends_on: [redis, kafka]
        working_dir: ./task2app/Saas_project

      - name: go-relay
        command: "./go_relayToTrae"
        health_check:
          url: "http://localhost:9090/healthz"
        depends_on: [kafka]

      - name: frontend
        command: "npm run dev"
        health_check:
          url: "http://localhost:5173"
        depends_on: [saas-backend]
        on_failure: skip
```

- [ ] **Step 3: Add yaml dependency**

```bash
cd runAll && go get gopkg.in/yaml.v3
```

- [ ] **Step 4: Commit**

```bash
git add runAll/go.mod runAll/go.sum runAll/config.yaml
git commit -m "feat(runAll): init Go module and example config"
```

---

### Task 2: Config parsing and validation

**Files:**
- Create: `runAll/config.go`
- Create: `runAll/config_test.go`

- [ ] **Step 1: Write failing test for config loading**

Write `runAll/config_test.go`:

```go
package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_ValidYAML(t *testing.T) {
	yaml := `
version: "1"
groups:
  - name: infra
    services:
      - name: redis
        command: "redis-server"
        health_check:
          url: "http://localhost:6379"
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte(yaml), 0644)

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Version != "1" {
		t.Errorf("version = %q, want %q", cfg.Version, "1")
	}
	if len(cfg.Groups) != 1 {
		t.Fatalf("groups len = %d, want 1", len(cfg.Groups))
	}
	svc := cfg.Groups[0].Services[0]
	if svc.Name != "redis" {
		t.Errorf("service name = %q, want %q", svc.Name, "redis")
	}
}

func TestLoadConfig_Defaults(t *testing.T) {
	yaml := `
version: "1"
groups:
  - name: infra
    services:
      - name: svc
        command: "echo hi"
        health_check:
          url: "http://localhost:8080"
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte(yaml), 0644)

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	svc := cfg.Groups[0].Services[0]
	if svc.OnFailure != "exit" {
		t.Errorf("on_failure default = %q, want %q", svc.OnFailure, "exit")
	}
	if svc.HealthCheck.Timeout != 30 {
		t.Errorf("timeout default = %d, want 30", svc.HealthCheck.Timeout)
	}
	if svc.HealthCheck.Retries != 10 {
		t.Errorf("retries default = %d, want 10", svc.HealthCheck.Retries)
	}
	if svc.HealthCheck.Backoff.Initial != 1.0 {
		t.Errorf("backoff.initial default = %f, want 1.0", svc.HealthCheck.Backoff.Initial)
	}
	if svc.HealthCheck.Backoff.Max != 8.0 {
		t.Errorf("backoff.max default = %f, want 8.0", svc.HealthCheck.Backoff.Max)
	}
	if svc.HealthCheck.Backoff.Multiplier != 2.0 {
		t.Errorf("backoff.multiplier default = %f, want 2.0", svc.HealthCheck.Backoff.Multiplier)
	}
}

func TestLoadConfig_DuplicateServiceNames(t *testing.T) {
	yaml := `
version: "1"
groups:
  - name: g1
    services:
      - name: dup
        command: "a"
        health_check:
          url: "http://localhost:1"
  - name: g2
    services:
      - name: dup
        command: "b"
        health_check:
          url: "http://localhost:2"
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte(yaml), 0644)

	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected error for duplicate service names")
	}
}

func TestLoadConfig_MissingDependsOn(t *testing.T) {
	yaml := `
version: "1"
groups:
  - name: g1
    services:
      - name: svc
        command: "a"
        health_check:
          url: "http://localhost:1"
        depends_on: [nonexistent]
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte(yaml), 0644)

	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected error for missing depends_on reference")
	}
}

func TestLoadConfig_InvalidOnFailure(t *testing.T) {
	yaml := `
version: "1"
groups:
  - name: g1
    services:
      - name: svc
        command: "a"
        health_check:
          url: "http://localhost:1"
        on_failure: panic
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte(yaml), 0644)

	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected error for invalid on_failure")
	}
}

func TestFlattenServices(t *testing.T) {
	cfg := &Config{
		Groups: []Group{
			{Name: "g1", Services: []Service{
				{Name: "a"}, {Name: "b"},
			}},
			{Name: "g2", Services: []Service{
				{Name: "c"},
			}},
		},
	}
	got := cfg.Flatten()
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	names := []string{got[0].Name, got[1].Name, got[2].Name}
	expected := []string{"a", "b", "c"}
	for i, n := range names {
		if n != expected[i] {
			t.Errorf("names[%d] = %q, want %q", i, n, expected[i])
		}
	}
}

func TestLoadConfig_MissingCommand(t *testing.T) {
	yaml := `
version: "1"
groups:
  - name: g1
    services:
      - name: svc
        health_check:
          url: "http://localhost:1"
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte(yaml), 0644)

	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected error for missing command")
	}
}

func TestLoadConfig_MissingHealthCheckURL(t *testing.T) {
	yaml := `
version: "1"
groups:
  - name: g1
    services:
      - name: svc
        command: "echo hi"
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte(yaml), 0644)

	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected error for missing health_check.url")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd runAll && go test -v -run TestLoadConfig ./...
```
Expected: compilation errors (types not defined).

- [ ] **Step 3: Write config.go**

Write `runAll/config.go`:

```go
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Version string  `yaml:"version"`
	Groups  []Group `yaml:"groups"`
}

type Group struct {
	Name     string    `yaml:"name"`
	Services []Service `yaml:"services"`
}

type Service struct {
	Name        string            `yaml:"name"`
	Command     string            `yaml:"command"`
	WorkingDir  string            `yaml:"working_dir"`
	Env         map[string]string `yaml:"env"`
	DependsOn   []string          `yaml:"depends_on"`
	OnFailure   string            `yaml:"on_failure"`
	HealthCheck HealthCheck       `yaml:"health_check"`
}

type HealthCheck struct {
	URL     string  `yaml:"url"`
	Timeout int     `yaml:"timeout"`
	Retries int     `yaml:"retries"`
	Backoff Backoff `yaml:"backoff"`
}

type Backoff struct {
	Initial    float64 `yaml:"initial"`
	Max        float64 `yaml:"max"`
	Multiplier float64 `yaml:"multiplier"`
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

	cfg.fillDefaults()
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	absDir, err := filepath.Abs(filepath.Dir(path))
	if err != nil {
		return nil, fmt.Errorf("resolve config dir: %w", err)
	}
	cfg.resolveWorkingDirs(absDir)

	return &cfg, nil
}

func (c *Config) fillDefaults() {
	for gi := range c.Groups {
		for si := range c.Groups[gi].Services {
			svc := &c.Groups[gi].Services[si]
			if svc.OnFailure == "" {
				svc.OnFailure = "exit"
			}
			if svc.HealthCheck.Timeout == 0 {
				svc.HealthCheck.Timeout = 30
			}
			if svc.HealthCheck.Retries == 0 {
				svc.HealthCheck.Retries = 10
			}
			if svc.HealthCheck.Backoff.Initial == 0 {
				svc.HealthCheck.Backoff.Initial = 1.0
			}
			if svc.HealthCheck.Backoff.Max == 0 {
				svc.HealthCheck.Backoff.Max = 8.0
			}
			if svc.HealthCheck.Backoff.Multiplier == 0 {
				svc.HealthCheck.Backoff.Multiplier = 2.0
			}
		}
	}
}

func (c *Config) validate() error {
	names := map[string]bool{}
	for _, g := range c.Groups {
		for _, svc := range g.Services {
			if svc.Name == "" {
				return fmt.Errorf("service name is required")
			}
			if svc.Command == "" {
				return fmt.Errorf("service %q: command is required", svc.Name)
			}
			if svc.HealthCheck.URL == "" {
				return fmt.Errorf("service %q: health_check.url is required", svc.Name)
			}
			if svc.OnFailure != "exit" && svc.OnFailure != "skip" {
				return fmt.Errorf("service %q: on_failure must be 'exit' or 'skip', got %q", svc.Name, svc.OnFailure)
			}
			if names[svc.Name] {
				return fmt.Errorf("duplicate service name: %q", svc.Name)
			}
			names[svc.Name] = true
		}
	}
	// Validate depends_on references
	for _, g := range c.Groups {
		for _, svc := range g.Services {
			for _, dep := range svc.DependsOn {
				if !names[dep] {
					return fmt.Errorf("service %q: depends_on %q does not exist", svc.Name, dep)
				}
			}
		}
	}
	return nil
}

func (c *Config) resolveWorkingDirs(configDir string) {
	for gi := range c.Groups {
		for si := range c.Groups[gi].Services {
			svc := &c.Groups[gi].Services[si]
			if svc.WorkingDir != "" && !filepath.IsAbs(svc.WorkingDir) {
				svc.WorkingDir = filepath.Join(configDir, svc.WorkingDir)
			}
		}
	}
}

func (c *Config) Flatten() []Service {
	var result []Service
	for _, g := range c.Groups {
		result = append(result, g.Services...)
	}
	return result
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd runAll && go test -v -run TestLoadConfig ./...
```
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add runAll/config.go runAll/config_test.go
git commit -m "feat(runAll): add config parsing with validation"
```

---

### Task 3: DAG construction and topological sort

**Files:**
- Create: `runAll/dag.go`
- Create: `runAll/dag_test.go`

- [ ] **Step 1: Write failing tests for DAG**

Write `runAll/dag_test.go`:

```go
package main

import (
	"strings"
	"testing"
)

func makeService(name string, deps ...string) Service {
	return Service{
		Name:      name,
		Command:   "echo " + name,
		HealthCheck: HealthCheck{URL: "http://localhost/" + name},
		DependsOn: deps,
	}
}

func TestBuildDAG_NoDependencies(t *testing.T) {
	services := []Service{
		makeService("a"),
		makeService("b"),
		makeService("c"),
	}
	levels, err := BuildDAG(services)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// All should be in one level since no dependencies
	if len(levels) != 1 {
		t.Fatalf("levels = %d, want 1", len(levels))
	}
	if len(levels[0].Services) != 3 {
		t.Fatalf("level 0 size = %d, want 3", len(levels[0].Services))
	}
}

func TestBuildDAG_LinearChain(t *testing.T) {
	services := []Service{
		makeService("c", "b"),
		makeService("b", "a"),
		makeService("a"),
	}
	levels, err := BuildDAG(services)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(levels) != 3 {
		t.Fatalf("levels = %d, want 3", len(levels))
	}
	if levels[0].Services[0].Name != "a" {
		t.Errorf("level 0 = %q, want 'a'", levels[0].Services[0].Name)
	}
	if levels[1].Services[0].Name != "b" {
		t.Errorf("level 1 = %q, want 'b'", levels[1].Services[0].Name)
	}
	if levels[2].Services[0].Name != "c" {
		t.Errorf("level 2 = %q, want 'c'", levels[2].Services[0].Name)
	}
}

func TestBuildDAG_DiamondDependency(t *testing.T) {
	// a → b, a → c, b → d, c → d
	services := []Service{
		makeService("a"),
		makeService("b", "a"),
		makeService("c", "a"),
		makeService("d", "b", "c"),
	}
	levels, err := BuildDAG(services)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(levels) != 3 {
		t.Fatalf("levels = %d, want 3", len(levels))
	}
	// Level 0: a
	// Level 1: b, c (parallel)
	// Level 2: d
	if len(levels[0].Services) != 1 || levels[0].Services[0].Name != "a" {
		t.Errorf("level 0: want [a]")
	}
	if len(levels[1].Services) != 2 {
		t.Errorf("level 1 size = %d, want 2", len(levels[1].Services))
	}
	if len(levels[2].Services) != 1 || levels[2].Services[0].Name != "d" {
		t.Errorf("level 2: want [d]")
	}
}

func TestBuildDAG_CycleDetection(t *testing.T) {
	services := []Service{
		makeService("a", "b"),
		makeService("b", "a"),
	}
	_, err := BuildDAG(services)
	if err == nil {
		t.Fatal("expected error for cyclic dependency")
	}
	if !strings.Contains(err.Error(), "cycle") {
		t.Errorf("error should mention 'cycle', got: %v", err)
	}
}

func TestBuildDAG_SelfReference(t *testing.T) {
	services := []Service{
		makeService("a", "a"),
	}
	_, err := BuildDAG(services)
	if err == nil {
		t.Fatal("expected error for self-referencing dependency")
	}
}

func TestBuildDAG_EmptyList(t *testing.T) {
	levels, err := BuildDAG([]Service{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(levels) != 0 {
		t.Fatalf("levels = %d, want 0", len(levels))
	}
}

func TestBuildDAG_ThreeLevels(t *testing.T) {
	services := []Service{
		makeService("frontend", "backend"),
		makeService("backend", "db", "cache"),
		makeService("db"),
		makeService("cache"),
	}
	levels, err := BuildDAG(services)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(levels) != 3 {
		t.Fatalf("levels = %d, want 3", len(levels))
	}
	// Level 0: db, cache
	// Level 1: backend
	// Level 2: frontend
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd runAll && go test -v -run TestBuildDAG ./...
```
Expected: compilation error (BuildDAG not defined).

- [ ] **Step 3: Write dag.go**

Write `runAll/dag.go`:

```go
package main

import "fmt"

type ServiceNode struct {
	Service    Service
	DependsOn  []string
	Dependents []string
	InDegree   int
}

type ExecutionLevel struct {
	Services []*ServiceNode
}

func BuildDAG(services []Service) ([]ExecutionLevel, error) {
	if len(services) == 0 {
		return nil, nil
	}

	nodes := make(map[string]*ServiceNode, len(services))
	for _, svc := range services {
		nodes[svc.Name] = &ServiceNode{
			Service:   svc,
			DependsOn: svc.DependsOn,
		}
	}

	// Compute dependents and indegrees
	for _, node := range nodes {
		for _, depName := range node.DependsOn {
			dep := nodes[depName]
			dep.Dependents = append(dep.Dependents, node.Service.Name)
		}
		node.InDegree = len(node.DependsOn)
	}

	// Kahn's algorithm
	var levels []ExecutionLevel
	processed := 0

	// First level: nodes with indegree 0
	var currentLevel []*ServiceNode
	for _, node := range nodes {
		if node.InDegree == 0 {
			currentLevel = append(currentLevel, node)
		}
	}

	for len(currentLevel) > 0 {
		levels = append(levels, ExecutionLevel{Services: currentLevel})
		processed += len(currentLevel)

		var nextLevel []*ServiceNode
		for _, node := range currentLevel {
			for _, depName := range node.Dependents {
				dep := nodes[depName]
				dep.InDegree--
				if dep.InDegree == 0 {
					nextLevel = append(nextLevel, dep)
				}
			}
		}
		currentLevel = nextLevel
	}

	if processed != len(nodes) {
		// Find a cycle to report
		for _, node := range nodes {
			if node.InDegree > 0 {
				return nil, fmt.Errorf("cycle detected involving service %q", node.Service.Name)
			}
		}
	}

	return levels, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd runAll && go test -v -run TestBuildDAG ./...
```
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add runAll/dag.go runAll/dag_test.go
git commit -m "feat(runAll): add DAG construction and topological sort"
```

---

### Task 4: Health check with exponential backoff

**Files:**
- Create: `runAll/health.go`
- Create: `runAll/health_test.go`

- [ ] **Step 1: Write failing tests for health check**

Write `runAll/health_test.go`:

```go
package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestWaitHealthy_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := HealthCheck{
		URL:     srv.URL,
		Timeout: 5,
		Retries: 3,
		Backoff: Backoff{Initial: 0.001, Max: 0.01, Multiplier: 2.0},
	}

	ctx := context.Background()
	err := waitHealthy(ctx, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWaitHealthy_RetryThenSuccess(t *testing.T) {
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := HealthCheck{
		URL:     srv.URL,
		Timeout: 5,
		Retries: 5,
		Backoff: Backoff{Initial: 0.001, Max: 0.01, Multiplier: 2.0},
	}

	ctx := context.Background()
	err := waitHealthy(ctx, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if attempts != 3 {
		t.Errorf("attempts = %d, want 3", attempts)
	}
}

func TestWaitHealthy_ExhaustRetries(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	cfg := HealthCheck{
		URL:     srv.URL,
		Timeout: 2,
		Retries: 3,
		Backoff: Backoff{Initial: 0.001, Max: 0.01, Multiplier: 2.0},
	}

	ctx := context.Background()
	err := waitHealthy(ctx, cfg)
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
}

func TestWaitHealthy_ContextCancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	cfg := HealthCheck{
		URL:     srv.URL,
		Timeout: 60,
		Retries: 100,
		Backoff: Backoff{Initial: 0.5, Max: 5.0, Multiplier: 2.0},
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	err := waitHealthy(ctx, cfg)
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd runAll && go test -v -run TestWaitHealthy ./...
```
Expected: compilation error (waitHealthy not defined).

- [ ] **Step 3: Write health.go**

Write `runAll/health.go`:

```go
package main

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

func waitHealthy(ctx context.Context, cfg HealthCheck) error {
	interval := time.Duration(cfg.Backoff.Initial * float64(time.Second))
	maxInterval := time.Duration(cfg.Backoff.Max * float64(time.Second))

	deadline := time.Now().Add(time.Duration(cfg.Timeout) * time.Second)

	for i := 0; i < cfg.Retries; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("health check timed out after %ds", cfg.Timeout)
		}

		resp, err := http.Get(cfg.URL)
		if err == nil && resp.StatusCode >= 200 && resp.StatusCode < 400 {
			resp.Body.Close()
			return nil
		}
		if resp != nil {
			resp.Body.Close()
		}

		interval = time.Duration(float64(interval) * cfg.Backoff.Multiplier)
		if interval > maxInterval {
			interval = maxInterval
		}
	}

	return fmt.Errorf("health check failed after %d retries", cfg.Retries)
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd runAll && go test -v -run TestWaitHealthy ./...
```
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add runAll/health.go runAll/health_test.go
git commit -m "feat(runAll): add health check with exponential backoff"
```

---

### Task 5: Status store

**Files:**
- Create: `runAll/status.go`
- Create: `runAll/status_test.go`

- [ ] **Step 1: Write failing tests for StatusStore**

Write `runAll/status_test.go`:

```go
package main

import (
	"testing"
)

func TestStatusStore_UpdateAndGet(t *testing.T) {
	store := NewStatusStore()
	names := []string{"a", "b"}
	store.Init(names)

	store.Update("a", StatusStarting, "")
	store.Update("b", StatusHealthy, "")

	all := store.All()
	if len(all) != 2 {
		t.Fatalf("len = %d, want 2", len(all))
	}

	a := store.Get("a")
	if a.Status != StatusStarting {
		t.Errorf("a status = %q, want %q", a.Status, StatusStarting)
	}

	b := store.Get("b")
	if b.Status != StatusHealthy {
		t.Errorf("b status = %q, want %q", b.Status, StatusHealthy)
	}
}

func TestStatusStore_InitSetsPending(t *testing.T) {
	store := NewStatusStore()
	store.Init([]string{"redis", "kafka"})

	for _, name := range []string{"redis", "kafka"} {
		s := store.Get(name)
		if s.Status != StatusPending {
			t.Errorf("%s status = %q, want %q", name, s.Status, StatusPending)
		}
	}
}

func TestStatusStore_UpdatePreservesFields(t *testing.T) {
	store := NewStatusStore()
	store.Init([]string{"test-svc"})

	store.Update("test-svc", StatusStarting, "")
	store.Update("test-svc", StatusStarting, "") // Idempotent check

	got := store.Get("test-svc")
	if got.Status != StatusStarting {
		t.Errorf("status = %q, want %q", got.Status, StatusStarting)
	}
}

func TestStatusStore_MarkError(t *testing.T) {
	store := NewStatusStore()
	store.Init([]string{"fail-svc"})

	store.Update("fail-svc", StatusFailed, "connection refused")

	got := store.Get("fail-svc")
	if got.Status != StatusFailed {
		t.Errorf("status = %q, want %q", got.Status, StatusFailed)
	}
	if got.Error != "connection refused" {
		t.Errorf("error = %q, want %q", got.Error, "connection refused")
	}
}

func TestStatusStore_SetDependsOn(t *testing.T) {
	store := NewStatusStore()
	store.Init([]string{"backend", "frontend"})
	store.SetDependsOn("frontend", []DepStatus{
		{Name: "backend", Status: StatusHealthy},
	})

	got := store.Get("frontend")
	if len(got.DependsOn) != 1 {
		t.Fatalf("depends_on len = %d, want 1", len(got.DependsOn))
	}
	if got.DependsOn[0].Name != "backend" || got.DependsOn[0].Status != StatusHealthy {
		t.Errorf("depends_on[0] = %+v, want {backend healthy}", got.DependsOn[0])
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd runAll && go test -v -run TestStatusStore ./...
```
Expected: compilation error (NewStatusStore not defined).

- [ ] **Step 3: Write status.go**

Write `runAll/status.go`:

```go
package main

import (
	"sync"
	"time"
)

type Status string

const (
	StatusPending  Status = "pending"
	StatusStarting Status = "starting"
	StatusRetrying Status = "retrying"
	StatusHealthy  Status = "healthy"
	StatusFailed   Status = "failed"
	StatusSkipped  Status = "skipped"
)

type ServiceStatus struct {
	Name      string      `json:"name"`
	Status    Status      `json:"status"`
	DependsOn []DepStatus `json:"depends_on"`
	Command   string      `json:"command"`
	URL       string      `json:"url"`
	PID       int         `json:"pid"`
	StartedAt string      `json:"started_at"`
	Error     string      `json:"error,omitempty"`
}

type DepStatus struct {
	Name   string `json:"name"`
	Status Status `json:"status"`
}

type StatusStore struct {
	mu       sync.RWMutex
	services map[string]*ServiceStatus
}

func NewStatusStore() *StatusStore {
	return &StatusStore{
		services: make(map[string]*ServiceStatus),
	}
}

func (s *StatusStore) Init(names []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, name := range names {
		s.services[name] = &ServiceStatus{
			Name:   name,
			Status: StatusPending,
		}
	}
}

func (s *StatusStore) Update(name string, status Status, errMsg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	svc, ok := s.services[name]
	if !ok {
		return
	}
	svc.Status = status
	if status == StatusStarting && svc.StartedAt == "" {
		svc.StartedAt = time.Now().Format(time.RFC3339)
	}
	if errMsg != "" {
		svc.Error = errMsg
	}
}

func (s *StatusStore) SetPID(name string, pid int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if svc, ok := s.services[name]; ok {
		svc.PID = pid
	}
}

func (s *StatusStore) SetDependsOn(name string, deps []DepStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if svc, ok := s.services[name]; ok {
		svc.DependsOn = deps
	}
}

func (s *StatusStore) SetCommand(name, command string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if svc, ok := s.services[name]; ok {
		svc.Command = command
	}
}

func (s *StatusStore) SetURL(name, url string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if svc, ok := s.services[name]; ok {
		svc.URL = url
	}
}

func (s *StatusStore) UpdateDependencyStatus(name string, status Status) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, svc := range s.services {
		for i, dep := range svc.DependsOn {
			if dep.Name == name {
				svc.DependsOn[i].Status = status
			}
		}
	}
}

func (s *StatusStore) Get(name string) *ServiceStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.services[name]
}

func (s *StatusStore) All() []*ServiceStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*ServiceStatus, 0, len(s.services))
	for _, svc := range s.services {
		result = append(result, svc)
	}
	return result
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd runAll && go test -v -run TestStatusStore ./...
```
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add runAll/status.go runAll/status_test.go
git commit -m "feat(runAll): add thread-safe status store"
```

---

### Task 6: Web UI server and embedded HTML

**Files:**
- Create: `runAll/ui.go`
- Create: `runAll/status.html`
- Create: `runAll/ui_test.go`

- [ ] **Step 1: Write failing test for UI server**

Write `runAll/ui_test.go`:

```go
package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAPIStatus(t *testing.T) {
	store := NewStatusStore()
	store.Init([]string{"redis", "kafka"})
	store.Update("redis", StatusHealthy, "")
	store.Update("kafka", StatusStarting, "")

	mux := http.NewServeMux()
	registerUIHandlers(mux, store)

	req := httptest.NewRequest("GET", "/api/status", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var result []*ServiceStatus
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("json decode: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("len = %d, want 2", len(result))
	}
}

func TestUIHomePage(t *testing.T) {
	store := NewStatusStore()
	store.Init([]string{"svc"})

	mux := http.NewServeMux()
	registerUIHandlers(mux, store)

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	contentType := rec.Header().Get("Content-Type")
	if contentType == "" || (contentType != "" && contentType[:9] != "text/html") {
		t.Errorf("Content-Type = %q, want text/html", contentType)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd runAll && go test -v -run TestAPIStatus ./...
```
Expected: compilation error (registerUIHandlers not defined).

- [ ] **Step 3: Write status.html**

Write `runAll/status.html`:

```html
<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>runAll Status</title>
<style>
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, monospace; background: #1a1a2e; color: #e0e0e0; padding: 24px; }
  h1 { font-size: 18px; margin-bottom: 4px; color: #e0e0e0; }
  .subtitle { font-size: 12px; color: #888; margin-bottom: 20px; }
  .service { display: flex; align-items: center; gap: 10px; padding: 10px 14px; margin-bottom: 4px; background: #16213e; border-radius: 6px; }
  .service:hover { background: #1c2a4a; }
  .name { font-weight: 600; min-width: 140px; }
  .status { font-size: 13px; min-width: 80px; }
  .deps { display: flex; gap: 8px; flex-wrap: wrap; }
  .dep { display: flex; align-items: center; gap: 4px; font-size: 12px; background: #0f3460; padding: 2px 8px; border-radius: 10px; }
  .dot { width: 10px; height: 10px; border-radius: 50%; flex-shrink: 0; }
  .dot.green { background: #4ade80; box-shadow: 0 0 6px #4ade80; }
  .dot.yellow { background: #facc15; box-shadow: 0 0 6px #facc15; }
  .dot.gray { background: #6b7280; }
  .dot.red { background: #ef4444; box-shadow: 0 0 6px #ef4444; }
  .dot.dark { background: #374151; }
  .error-msg { color: #ef4444; font-size: 12px; margin-left: 8px; }
  .level-header { font-size: 11px; color: #666; text-transform: uppercase; letter-spacing: 1px; margin: 16px 0 6px 0; }
  .uptime { font-size: 12px; color: #888; margin-top: 20px; }
</style>
</head>
<body>
<h1>runAll Status</h1>
<div class="subtitle">auto-refresh every 2s</div>
<div id="services"></div>
<div class="uptime" id="uptime"></div>
<script>
const startTime = Date.now();
const dotClass = {
  healthy: 'green', starting: 'yellow', retrying: 'yellow',
  pending: 'gray', failed: 'red', skipped: 'dark'
};

async function refresh() {
  const resp = await fetch('/api/status');
  const data = await resp.json();
  const container = document.getElementById('services');
  let html = '';
  let level = -1;
  let seenNewLevel = false;

  // Group by DAG level (approximate by depends_on depth)
  const byLevel = new Map();
  for (const svc of data) {
    const depth = svc.depends_on ? svc.depends_on.length : 0;
    if (!byLevel.has(depth)) byLevel.set(depth, []);
    byLevel.get(depth).push(svc);
  }

  for (const [depth, services] of byLevel) {
    html += `<div class="level-header">Level ${depth}</div>`;
    for (const svc of services) {
      const cls = dotClass[svc.status] || 'gray';
      html += `<div class="service">`;
      html += `<span class="dot ${cls}"></span>`;
      html += `<span class="name">${esc(svc.name)}</span>`;
      html += `<span class="status">${svc.status}</span>`;
      if (svc.depends_on && svc.depends_on.length > 0) {
        html += `<span class="deps">`;
        for (const dep of svc.depends_on) {
          const dcls = dotClass[dep.status] || 'gray';
          html += `<span class="dep"><span class="dot ${dcls}"></span>${esc(dep.name)}</span>`;
        }
        html += `</span>`;
      }
      if (svc.error) {
        html += `<span class="error-msg">${esc(svc.error)}</span>`;
      }
      html += `</div>`;
    }
  }
  container.innerHTML = html;

  const elapsed = Math.floor((Date.now() - startTime) / 1000);
  document.getElementById('uptime').textContent = `Uptime: ${elapsed}s`;
}

function esc(s) {
  const el = document.createElement('span');
  el.textContent = s;
  return el.innerHTML;
}

refresh();
setInterval(refresh, 2000);
</script>
</body>
</html>
```

- [ ] **Step 4: Write ui.go**

Write `runAll/ui.go`:

```go
package main

import (
	"embed"
	"encoding/json"
	"net/http"
)

//go:embed status.html
var statusHTML embed.FS

func registerUIHandlers(mux *http.ServeMux, store *StatusStore) {
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(store.All())
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		data, _ := statusHTML.ReadFile("status.html")
		w.Write(data)
	})
}

func startUIServer(store *StatusStore, port string) *http.Server {
	mux := http.NewServeMux()
	registerUIHandlers(mux, store)

	srv := &http.Server{Addr: port, Handler: mux}
	go srv.ListenAndServe()
	return srv
}
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
cd runAll && go test -v -run "TestAPIStatus|TestUIHomePage" ./...
```
Expected: all PASS.

- [ ] **Step 6: Commit**

```bash
git add runAll/ui.go runAll/status.html runAll/ui_test.go
git commit -m "feat(runAll): add Web UI server with embedded dashboard"
```

---

### Task 7: Runner — service lifecycle orchestration

**Files:**
- Create: `runAll/runner.go`

- [ ] **Step 1: Write runner.go**

Write `runAll/runner.go`:

```go
package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

type Runner struct {
	cfg       *Config
	store     *StatusStore
	levels    []ExecutionLevel
	processes map[string]*exec.Cmd
	mu        sync.Mutex
}

func NewRunner(cfg *Config, store *StatusStore) (*Runner, error) {
	services := cfg.Flatten()
	names := make([]string, len(services))
	for i, svc := range services {
		names[i] = svc.Name
	}
	store.Init(names)

	// Populate command and URL in store for UI display
	for _, svc := range services {
		store.SetCommand(svc.Name, svc.Command)
		store.SetURL(svc.Name, svc.HealthCheck.URL)
	}

	// Build dependency status references
	for _, svc := range services {
		deps := make([]DepStatus, len(svc.DependsOn))
		for i, depName := range svc.DependsOn {
			deps[i] = DepStatus{Name: depName, Status: StatusPending}
		}
		store.SetDependsOn(svc.Name, deps)
	}

	levels, err := BuildDAG(services)
	if err != nil {
		return nil, err
	}

	return &Runner{
		cfg:       cfg,
		store:     store,
		levels:    levels,
		processes: make(map[string]*exec.Cmd),
	}, nil
}

func (r *Runner) Run(ctx context.Context, daemon bool) error {
	for _, level := range r.levels {
		if err := r.executeLevel(ctx, level); err != nil {
			return err
		}
	}

	log.Println("All services healthy.")

	if daemon {
		log.Println("Daemon mode: exiting.")
		return nil
	}

	log.Println("Running. Press Ctrl+C to stop.")
	<-ctx.Done()
	log.Println("Shutting down...")
	r.Shutdown()
	return nil
}

func (r *Runner) executeLevel(ctx context.Context, level ExecutionLevel) error {
	var wg sync.WaitGroup
	errCh := make(chan error, len(level.Services))

	for _, node := range level.Services {
		// Check if any dependency was skipped/failed
		skip := false
		for _, depName := range node.Service.DependsOn {
			depStatus := r.store.Get(depName)
			if depStatus.Status == StatusFailed || depStatus.Status == StatusSkipped {
				r.store.Update(node.Service.Name, StatusSkipped, fmt.Sprintf("dependency %s is %s", depName, depStatus.Status))
				log.Printf("[%s] SKIPPED: dependency %s is %s", node.Service.Name, depName, depStatus.Status)
				skip = true
				break
			}
		}
		if skip {
			continue
		}

		wg.Add(1)
		go func(node *ServiceNode) {
			defer wg.Done()
			if err := r.startAndCheck(ctx, node); err != nil {
				errCh <- err
			}
		}(node)
	}

	wg.Wait()
	close(errCh)

	// Collect errors
	var firstExitErr error
	for err := range errCh {
		if firstExitErr == nil && err != nil {
			firstExitErr = err
		}
	}

	// If any exit-type failure occurred, stop everything
	if firstExitErr != nil {
		r.Shutdown()
		return firstExitErr
	}

	return nil
}

func (r *Runner) startAndCheck(ctx context.Context, node *ServiceNode) error {
	svc := node.Service
	r.store.Update(svc.Name, StatusStarting, "")

	cmd := exec.CommandContext(ctx, "sh", "-c", svc.Command)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if svc.WorkingDir != "" {
		cmd.Dir = svc.WorkingDir
	}
	if len(svc.Env) > 0 {
		env := os.Environ()
		for k, v := range svc.Env {
			env = append(env, k+"="+v)
		}
		cmd.Env = env
	}

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		r.store.Update(svc.Name, StatusFailed, err.Error())
		if svc.OnFailure == "exit" {
			return fmt.Errorf("[%s] failed to start: %w", svc.Name, err)
		}
		log.Printf("[%s] failed to start, skipping: %v", svc.Name, err)
		return nil
	}

	r.mu.Lock()
	r.processes[svc.Name] = cmd
	r.mu.Unlock()
	r.store.SetPID(svc.Name, cmd.Process.Pid)

	go streamOutput(stdout, svc.Name)
	go streamOutput(stderr, svc.Name)

	// Health check
	r.store.Update(svc.Name, StatusRetrying, "")
	err := waitHealthy(ctx, svc.HealthCheck)
	if err != nil {
		r.store.Update(svc.Name, StatusFailed, err.Error())
		log.Printf("[%s] health check failed: %v", svc.Name, err)
		if svc.OnFailure == "exit" {
			return fmt.Errorf("[%s] health check failed: %w", svc.Name, err)
		}
		return nil
	}

	r.store.Update(svc.Name, StatusHealthy, "")
	r.store.UpdateDependencyStatus(svc.Name, StatusHealthy)
	log.Printf("[%s] healthy (%s)", svc.Name, svc.HealthCheck.URL)
	return nil
}

func (r *Runner) Shutdown() {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Shutdown in reverse order
	for i := len(r.levels) - 1; i >= 0; i-- {
		level := r.levels[i]
		var wg sync.WaitGroup
		for _, node := range level.Services {
			cmd, ok := r.processes[node.Service.Name]
			if !ok || cmd.Process == nil {
				continue
			}
			wg.Add(1)
			go func(name string, cmd *exec.Cmd) {
				defer wg.Done()
				log.Printf("[%s] sending SIGTERM", name)
				syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)

				done := make(chan struct{})
				go func() {
					cmd.Wait()
					close(done)
				}()
				select {
				case <-done:
					log.Printf("[%s] stopped", name)
				case <-time.After(5 * time.Second):
					log.Printf("[%s] did not stop, sending SIGKILL", name)
					syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
					cmd.Wait()
				}
			}(node.Service.Name, cmd)
		}
		wg.Wait()
	}
}

func streamOutput(r io.Reader, name string) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		log.Printf("[%s] %s", name, scanner.Text())
	}
}
```

- [ ] **Step 2: Commit**

```bash
git add runAll/runner.go
git commit -m "feat(runAll): add service lifecycle runner"
```

---

### Task 8: main.go — wire everything together

**Files:**
- Create: `runAll/main.go`

- [ ] **Step 1: Write main.go**

Write `runAll/main.go`:

```go
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	configPath := flag.String("config", "config.yaml", "Path to YAML configuration file")
	daemon := flag.Bool("daemon", false, "Start services and exit (no Web UI)")
	uiPort := flag.String("ui-port", ":9999", "Web UI listen address")
	flag.Parse()

	cfg, err := LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Config error: %v", err)
	}

	store := NewStatusStore()
	runner, err := NewRunner(cfg, store)
	if err != nil {
		log.Fatalf("Setup error: %v", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Start Web UI (only in foreground mode)
	if !*daemon {
		srv := startUIServer(store, *uiPort)
		log.Printf("Web UI: http://localhost%s", *uiPort)
		defer srv.Close()
	}

	if err := runner.Run(ctx, *daemon); err != nil {
		log.Printf("Exiting due to error: %v", err)
		os.Exit(1)
	}
}
```

- [ ] **Step 2: Build and verify**

```bash
cd runAll && go build -o runAll .
```
Expected: successful build.

- [ ] **Step 3: Run all tests**

```bash
cd runAll && go test -v ./...
```
Expected: all PASS.

- [ ] **Step 4: Commit**

```bash
git add runAll/main.go
git commit -m "feat(runAll): wire up main entry point"
```

---

### Task 9: Restart button and API

**Files:**
- Modify: `runAll/status.go` — add `StatusRestarting`
- Modify: `runAll/runner.go` — add `RestartService` method
- Modify: `runAll/ui.go` — add `POST /api/restart` handler
- Modify: `runAll/status.html` — add restart button per service
- Modify: `runAll/main.go` — pass runner to UI

- [ ] **Step 1: Add StatusRestarting to status.go**

Edit `runAll/status.go`, add the new status constant after `StatusSkipped`:

```go
const (
	StatusPending    Status = "pending"
	StatusStarting   Status = "starting"
	StatusRetrying   Status = "retrying"
	StatusHealthy    Status = "healthy"
	StatusFailed     Status = "failed"
	StatusSkipped    Status = "skipped"
	StatusRestarting Status = "restarting"
)
```

- [ ] **Step 2: Add RestartService to runner.go**

Add this method to `runner.go` after the `Shutdown()` method:

```go
func (r *Runner) RestartService(ctx context.Context, name string) error {
	svc := r.findService(name)
	if svc == nil {
		return fmt.Errorf("service %q not found", name)
	}

	current := r.store.Get(name)
	if current == nil {
		return fmt.Errorf("service %q not found", name)
	}
	if current.Status != StatusHealthy && current.Status != StatusFailed {
		return fmt.Errorf("service %q is %s, can only restart healthy or failed services", name, current.Status)
	}

	r.store.Update(name, StatusRestarting, "")

	// Stop existing process
	r.stopProcess(name)

	// Start and health check
	node := &ServiceNode{Service: *svc}
	if err := r.startAndCheck(ctx, node); err != nil {
		return err
	}

	return nil
}

func (r *Runner) findService(name string) *Service {
	for _, g := range r.cfg.Groups {
		for _, svc := range g.Services {
			if svc.Name == name {
				return &svc
			}
		}
	}
	return nil
}

func (r *Runner) stopProcess(name string) {
	r.mu.Lock()
	cmd, ok := r.processes[name]
	if ok {
		delete(r.processes, name)
	}
	r.mu.Unlock()

	if !ok || cmd == nil || cmd.Process == nil {
		return
	}

	log.Printf("[%s] restarting: sending SIGTERM", name)
	syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)

	done := make(chan struct{})
	go func() {
		cmd.Wait()
		close(done)
	}()
	select {
	case <-done:
		log.Printf("[%s] stopped for restart", name)
	case <-time.After(5 * time.Second):
		log.Printf("[%s] did not stop, sending SIGKILL", name)
		syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		cmd.Wait()
	}
}
```

- [ ] **Step 3: Add POST /api/restart to ui.go**

Change `registerUIHandlers` signature to accept `*Runner`, add restart endpoint:

```go
import (
	"embed"
	"encoding/json"
	"log"
	"net/http"
)

func registerUIHandlers(mux *http.ServeMux, store *StatusStore, runner *Runner) {
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(store.All())
	})

	mux.HandleFunc("/api/restart", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var body struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if body.Name == "" {
			http.Error(w, "name is required", http.StatusBadRequest)
			return
		}

		log.Printf("[api] restart request for %s", body.Name)
		if err := runner.RestartService(r.Context(), body.Name); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		data, _ := statusHTML.ReadFile("status.html")
		w.Write(data)
	})
}

func startUIServer(store *StatusStore, runner *Runner, port string) *http.Server {
	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner)

	srv := &http.Server{Addr: port, Handler: mux}
	go srv.ListenAndServe()
	return srv
}
```

Also update the existing ui_test.go tests to pass `nil` for runner in `registerUIHandlers` calls (they only test GET endpoints, the runner is not used).

- [ ] **Step 4: Add restart button to status.html**

Replace the service row rendering in the `refresh()` function. Change this section in `runAll/status.html`:

```javascript
      html += `<div class="service">`;
      html += `<span class="dot ${cls}"></span>`;
      html += `<span class="name">${esc(svc.name)}</span>`;
      html += `<span class="status">${svc.status}</span>`;
```

To add the restart button after the status span:

```javascript
      html += `<div class="service">`;
      html += `<span class="dot ${cls}"></span>`;
      html += `<span class="name">${esc(svc.name)}</span>`;
      html += `<span class="status">${svc.status}</span>`;
      if (svc.status === 'healthy' || svc.status === 'failed') {
        html += `<button class="restart-btn" onclick="restartService('${esc(svc.name)}')" title="Restart">&#x21bb;</button>`;
      }
```

Add the restart function to the `<script>` block:

```javascript
async function restartService(name) {
  try {
    const resp = await fetch('/api/restart', {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({name: name})
    });
    const result = await resp.json();
    if (!resp.ok) {
      alert('Restart failed: ' + (result.error || 'unknown error'));
    }
  } catch (err) {
    alert('Restart failed: ' + err.message);
  }
}
```

Add the button style to the `<style>` block:

```css
  .restart-btn { background: none; border: 1px solid #555; color: #ccc; cursor: pointer; font-size: 16px; padding: 2px 8px; border-radius: 4px; line-height: 1; }
  .restart-btn:hover { background: #333; color: #fff; border-color: #888; }
```

- [ ] **Step 5: Update main.go to pass runner to the UI**

Edit the `startUIServer` call:

```go
	// Start Web UI (only in foreground mode)
	if !*daemon {
		srv := startUIServer(store, runner, *uiPort)
		log.Printf("Web UI: http://localhost%s", *uiPort)
		defer srv.Close()
	}
```

- [ ] **Step 6: Run all tests**

```bash
cd runAll && go test -v ./...
```
Expected: all PASS.

- [ ] **Step 7: Commit**

```bash
git add runAll/status.go runAll/runner.go runAll/ui.go runAll/status.html runAll/main.go runAll/ui_test.go
git commit -m "feat(runAll): add restart button and API endpoint"
```
