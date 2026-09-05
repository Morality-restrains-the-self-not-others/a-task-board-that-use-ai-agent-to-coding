# go_run_container Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rewrite the Python Flask `mock_run_container` service as a single Go binary using only standard library.

**Architecture:** Single Go binary with net/http, sync.Mutex + map for in-memory job state, os/exec for Docker CLI. Same API contract, same port_config.json config, zero external dependencies.

**Tech Stack:** Go 1.22+, standard library only (net/http, sync, os/exec, encoding/json, regexp)

---

## File Structure

| File | Responsibility |
|------|---------------|
| `go_run_container/go.mod` | Module definition |
| `go_run_container/config.go` | Config loading from port_config.json + env vars |
| `go_run_container/config_test.go` | Config unit tests |
| `go_run_container/secret.go` | Auth middleware (X-Mock-Run-Container-Secret) |
| `go_run_container/secret_test.go` | Auth unit tests |
| `go_run_container/jobs.go` | In-memory job store with sync.Mutex |
| `go_run_container/jobs_test.go` | Job store unit tests |
| `go_run_container/docker.go` | Docker CLI via os/exec |
| `go_run_container/docker_test.go` | Docker unit tests (platform inference, sensitive keys, name validation) |
| `go_run_container/server.go` | HTTP handlers for all 6 endpoints |
| `go_run_container/server_test.go` | HTTP handler tests via httptest |
| `go_run_container/main.go` | Entry point, wires everything |
| `go_run_container/start.sh` | Build + run script |
| `go_run_container/README.md` | Usage docs |

---

### Task 1: Initialize Go module and project directory

**Files:**
- Create: `go_run_container/go.mod`

- [ ] **Step 1: Create directory and initialize Go module**

```bash
mkdir -p /Users/task2app/gitClone/ramDisk/ram-mount/go_run_container
cd /Users/task2app/gitClone/ramDisk/ram-mount/go_run_container
go mod init go_run_container
```

Expected: `go.mod` created with `module go_run_container` and `go 1.22` (or system Go version). Verify the Go version is at least 1.22 for enhanced routing support.

```bash
go version
```

Expected: `go version go1.22.x` or higher.

- [ ] **Step 2: Commit**

```bash
cd /Users/task2app/gitClone/ramDisk/ram-mount/go_run_container
git init
git add go.mod
git commit -m "feat: initialize Go module for go_run_container"
```

---

### Task 2: Config loading (config.go)

**Files:**
- Create: `go_run_container/config.go`
- Create: `go_run_container/config_test.go`

- [ ] **Step 1: Write failing config tests**

Create `go_run_container/config_test.go`:

```go
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestConfigDefaults(t *testing.T) {
	cfg := LoadConfig()
	if cfg.Host != "127.0.0.1" {
		t.Errorf("expected default host 127.0.0.1, got %s", cfg.Host)
	}
	if cfg.Port != 8796 {
		t.Errorf("expected default port 8796, got %d", cfg.Port)
	}
	if cfg.Secret != "" {
		t.Errorf("expected empty default secret, got %s", cfg.Secret)
	}
}

func TestConfigFromEnvVars(t *testing.T) {
	os.Setenv("MOCK_RUN_CONTAINER_HOST", "0.0.0.0")
	os.Setenv("MOCK_RUN_CONTAINER_PORT", "9999")
	os.Setenv("MOCK_RUN_CONTAINER_SECRET", "test-secret")
	defer func() {
		os.Unsetenv("MOCK_RUN_CONTAINER_HOST")
		os.Unsetenv("MOCK_RUN_CONTAINER_PORT")
		os.Unsetenv("MOCK_RUN_CONTAINER_SECRET")
	}()

	cfg := LoadConfig()
	if cfg.Host != "0.0.0.0" {
		t.Errorf("expected host 0.0.0.0, got %s", cfg.Host)
	}
	if cfg.Port != 9999 {
		t.Errorf("expected port 9999, got %d", cfg.Port)
	}
	if cfg.Secret != "test-secret" {
		t.Errorf("expected secret test-secret, got %s", cfg.Secret)
	}
}

func TestConfigFromPortConfigJSON(t *testing.T) {
	// Create a temporary port_config.json
	tmpDir := t.TempDir()
	confDir := filepath.Join(tmpDir, "conf")
	os.MkdirAll(confDir, 0755)

	portConfig := map[string]interface{}{
		"mock_run_container": map[string]interface{}{
			"host":   "10.0.0.1",
			"port":   float64(8888),
			"secret": "json-secret",
		},
	}
	data, _ := json.Marshal(portConfig)
	os.WriteFile(filepath.Join(confDir, "port_config.json"), data, 0644)

	// Change to tmpDir so the relative path resolves
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	cfg := LoadConfig()
	if cfg.Host != "10.0.0.1" {
		t.Errorf("expected host 10.0.0.1 from JSON, got %s", cfg.Host)
	}
	if cfg.Port != 8888 {
		t.Errorf("expected port 8888 from JSON, got %d", cfg.Port)
	}
	if cfg.Secret != "json-secret" {
		t.Errorf("expected secret json-secret from JSON, got %s", cfg.Secret)
	}
}

func TestConfigEnvOverridesJSON(t *testing.T) {
	tmpDir := t.TempDir()
	confDir := filepath.Join(tmpDir, "conf")
	os.MkdirAll(confDir, 0755)

	portConfig := map[string]interface{}{
		"mock_run_container": map[string]interface{}{
			"host":   "json-host",
			"port":   float64(7777),
			"secret": "json-secret",
		},
	}
	data, _ := json.Marshal(portConfig)
	os.WriteFile(filepath.Join(confDir, "port_config.json"), data, 0644)

	os.Setenv("MOCK_RUN_CONTAINER_PORT", "6666")
	defer os.Unsetenv("MOCK_RUN_CONTAINER_PORT")

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	cfg := LoadConfig()
	// Port should come from env, host and secret from JSON
	if cfg.Host != "json-host" {
		t.Errorf("expected host json-host from JSON, got %s", cfg.Host)
	}
	if cfg.Port != 6666 {
		t.Errorf("expected port 6666 from env override, got %d", cfg.Port)
	}
	if cfg.Secret != "json-secret" {
		t.Errorf("expected secret json-secret from JSON, got %s", cfg.Secret)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd /Users/task2app/gitClone/ramDisk/ram-mount/go_run_container && go test -v -run TestConfig
```

Expected: compilation error — `LoadConfig` not defined.

- [ ] **Step 3: Implement config.go**

Create `go_run_container/config.go`:

```go
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
)

type Config struct {
	Host   string
	Port   int
	Secret string
}

func LoadConfig() Config {
	cfg := Config{
		Host: "127.0.0.1",
		Port: 8796,
	}

	// Try to read port_config.json relative to the binary's parent (task2app/)
	// Look for ../task2app/conf/port_config.json from the current working directory
	configPath := filepath.Join("..", "task2app", "conf", "port_config.json")
	if data, err := os.ReadFile(configPath); err == nil {
		var root map[string]json.RawMessage
		if json.Unmarshal(data, &root) == nil {
			if block, ok := root["mock_run_container"]; ok {
				var pc struct {
					Host   string `json:"host"`
					Port   int    `json:"port"`
					Secret string `json:"secret"`
				}
				if json.Unmarshal(block, &pc) == nil {
					if pc.Host != "" {
						cfg.Host = pc.Host
					}
					if pc.Port != 0 {
						cfg.Port = pc.Port
					}
					// secret from JSON can be empty (no auth);
					// we distinguish "not present" by checking raw keys
					cfg.Secret = pc.Secret
				}
			}
		}
	}

	// Env var overrides
	if v := os.Getenv("MOCK_RUN_CONTAINER_HOST"); v != "" {
		cfg.Host = v
	}
	if v := os.Getenv("MOCK_RUN_CONTAINER_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			cfg.Port = p
		}
	}
	if v := os.Getenv("MOCK_RUN_CONTAINER_SECRET"); v != "" {
		cfg.Secret = v
	}

	return cfg
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd /Users/task2app/gitClone/ramDisk/ram-mount/go_run_container && go test -v -run TestConfig
```

Expected: all 4 config tests PASS.

- [ ] **Step 5: Commit**

```bash
cd /Users/task2app/gitClone/ramDisk/ram-mount/go_run_container
git add config.go config_test.go
git commit -m "feat: add config loading from port_config.json and env vars"
```

---

### Task 3: Secret authentication middleware (secret.go)

**Files:**
- Create: `go_run_container/secret.go`
- Create: `go_run_container/secret_test.go`

- [ ] **Step 1: Write failing secret tests**

Create `go_run_container/secret_test.go`:

```go
package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthNoSecretConfigured(t *testing.T) {
	handler := authMiddleware("", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/v1/jobs", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("expected 200 when no secret configured, got %d", rec.Code)
	}
}

func TestAuthCorrectSecret(t *testing.T) {
	handler := authMiddleware("my-secret", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/v1/jobs", nil)
	req.Header.Set("X-Mock-Run-Container-Secret", "my-secret")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("expected 200 with correct secret, got %d", rec.Code)
	}
}

func TestAuthWrongSecret(t *testing.T) {
	handler := authMiddleware("my-secret", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/v1/jobs", nil)
	req.Header.Set("X-Mock-Run-Container-Secret", "wrong")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != 401 {
		t.Errorf("expected 401 with wrong secret, got %d", rec.Code)
	}
}

func TestAuthMissingHeader(t *testing.T) {
	handler := authMiddleware("my-secret", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/v1/jobs", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != 401 {
		t.Errorf("expected 401 with missing header, got %d", rec.Code)
	}
}

func TestAuthHealthEndpointAlwaysPublic(t *testing.T) {
	handler := authMiddleware("my-secret", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("expected 200 for /health without secret, got %d", rec.Code)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd /Users/task2app/gitClone/ramDisk/ram-mount/go_run_container && go test -v -run TestAuth
```

Expected: compilation error — `authMiddleware` not defined.

- [ ] **Step 3: Implement secret.go**

Create `go_run_container/secret.go`:

```go
package main

import (
	"encoding/json"
	"net/http"
)

func authMiddleware(secret string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}
		if secret == "" {
			next.ServeHTTP(w, r)
			return
		}
		got := r.Header.Get("X-Mock-Run-Container-Secret")
		if got != secret {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"detail": "unauthorized"})
			return
		}
		next.ServeHTTP(w, r)
	})
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd /Users/task2app/gitClone/ramDisk/ram-mount/go_run_container && go test -v -run TestAuth
```

Expected: all 5 auth tests PASS.

- [ ] **Step 5: Commit**

```bash
cd /Users/task2app/gitClone/ramDisk/ram-mount/go_run_container
git add secret.go secret_test.go
git commit -m "feat: add secret-based auth middleware"
```

---

### Task 4: Job state management (jobs.go)

**Files:**
- Create: `go_run_container/jobs.go`
- Create: `go_run_container/jobs_test.go`

- [ ] **Step 1: Write failing job store tests**

Create `go_run_container/jobs_test.go`:

```go
package main

import (
	"sync"
	"testing"
)

func TestCreateJob(t *testing.T) {
	store := NewJobStore()
	job := store.Create()

	if job.ID == "" {
		t.Error("expected non-empty job ID")
	}
	if job.Done {
		t.Error("expected job not done")
	}
	if len(job.Logs) != 0 {
		t.Error("expected empty logs for new job")
	}
}

func TestGetJob(t *testing.T) {
	store := NewJobStore()
	job := store.Create()

	got := store.Get(job.ID)
	if got == nil {
		t.Fatal("expected to find job")
	}
	if got.ID != job.ID {
		t.Errorf("expected job ID %s, got %s", job.ID, got.ID)
	}
}

func TestGetJobNotFound(t *testing.T) {
	store := NewJobStore()
	got := store.Get("nonexistent")
	if got != nil {
		t.Error("expected nil for nonexistent job")
	}
}

func TestAppendLog(t *testing.T) {
	store := NewJobStore()
	job := store.Create()

	store.AppendLog(job.ID, "line 1")
	store.AppendLog(job.ID, "line 2")

	got := store.Get(job.ID)
	if len(got.Logs) != 2 {
		t.Fatalf("expected 2 log lines, got %d", len(got.Logs))
	}
	if got.Logs[0] != "line 1" {
		t.Errorf("expected 'line 1', got '%s'", got.Logs[0])
	}
	if got.Logs[1] != "line 2" {
		t.Errorf("expected 'line 2', got '%s'", got.Logs[1])
	}
}

func TestAppendLogEmptyLine(t *testing.T) {
	store := NewJobStore()
	job := store.Create()

	store.AppendLog(job.ID, "")
	store.AppendLog(job.ID, "valid")

	got := store.Get(job.ID)
	if len(got.Logs) != 1 {
		t.Fatalf("expected 1 log line (empty skipped), got %d", len(got.Logs))
	}
}

func TestSetContainerID(t *testing.T) {
	store := NewJobStore()
	job := store.Create()

	store.SetContainerID(job.ID, "abc123")

	got := store.Get(job.ID)
	if got.ContainerID != "abc123" {
		t.Errorf("expected container_id abc123, got %s", got.ContainerID)
	}
}

func TestFinishJob(t *testing.T) {
	store := NewJobStore()
	job := store.Create()

	store.Finish(job.ID, "", map[string]string{
		"container_id": "def456",
		"image":        "alpine:latest",
	})

	got := store.Get(job.ID)
	if !got.Done {
		t.Error("expected job done")
	}
	if got.Error != "" {
		t.Errorf("expected no error, got %s", got.Error)
	}
	if got.Result["container_id"] != "def456" {
		t.Errorf("expected result container_id def456, got %v", got.Result)
	}
}

func TestFinishJobWithError(t *testing.T) {
	store := NewJobStore()
	job := store.Create()

	store.Finish(job.ID, "docker_pull_failed", nil)

	got := store.Get(job.ID)
	if !got.Done {
		t.Error("expected job done")
	}
	if got.Error != "docker_pull_failed" {
		t.Errorf("expected error docker_pull_failed, got %s", got.Error)
	}
}

func TestGetLogsWithCursor(t *testing.T) {
	store := NewJobStore()
	job := store.Create()

	for i := 0; i < 5; i++ {
		store.AppendLog(job.ID, "log line")
	}

	got := store.Get(job.ID)
	logs := got.Logs[2:] // cursor=2
	if len(logs) != 3 {
		t.Errorf("expected 3 logs from cursor 2, got %d", len(logs))
	}
}

func TestConcurrentAccess(t *testing.T) {
	store := NewJobStore()
	job := store.Create()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			store.AppendLog(job.ID, "line")
		}()
	}
	wg.Wait()

	got := store.Get(job.ID)
	if len(got.Logs) != 100 {
		t.Errorf("expected 100 log lines after concurrent writes, got %d", len(got.Logs))
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd /Users/task2app/gitClone/ramDisk/ram-mount/go_run_container && go test -v -run TestJob
```

Expected: compilation error — types not defined.

- [ ] **Step 3: Implement jobs.go**

Create `go_run_container/jobs.go`:

```go
package main

import (
	"sync"

	"github.com/google/uuid"
)
```

Wait — standard library only, no external deps. Let me generate UUIDs manually.

Create `go_run_container/jobs.go`:

```go
package main

import (
	"crypto/rand"
	"fmt"
	"sync"
)

type Job struct {
	ID          string
	Logs        []string
	Done        bool
	Error       string
	Result      map[string]string
	ContainerID string
}

type JobStore struct {
	mu   sync.Mutex
	jobs map[string]*Job
}

func NewJobStore() *JobStore {
	return &JobStore{
		jobs: make(map[string]*Job),
	}
}

func (s *JobStore) Create() *Job {
	id := newJobID()
	job := &Job{
		ID:     id,
		Logs:   make([]string, 0),
		Result: make(map[string]string),
	}
	s.mu.Lock()
	s.jobs[id] = job
	s.mu.Unlock()
	return job
}

func (s *JobStore) Get(id string) *Job {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.jobs[id]
}

func (s *JobStore) AppendLog(jobID, line string) {
	if line == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if job, ok := s.jobs[jobID]; ok {
		job.Logs = append(job.Logs, line)
	}
}

func (s *JobStore) SetContainerID(jobID, containerID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if job, ok := s.jobs[jobID]; ok {
		job.ContainerID = containerID
	}
}

func (s *JobStore) Finish(jobID, errStr string, result map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if job, ok := s.jobs[jobID]; ok {
		job.Done = true
		job.Error = errStr
		if result != nil {
			job.Result = result
		}
	}
}

func newJobID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd /Users/task2app/gitClone/ramDisk/ram-mount/go_run_container && go test -v -run "TestCreate|TestGet|TestAppend|TestSet|TestFinish|TestConcurrent"
```

Expected: all job store tests PASS.

- [ ] **Step 5: Commit**

```bash
cd /Users/task2app/gitClone/ramDisk/ram-mount/go_run_container
git add jobs.go jobs_test.go
git commit -m "feat: add in-memory job store with mutex"
```

---

### Task 5: Docker operations (docker.go)

**Files:**
- Create: `go_run_container/docker.go`
- Create: `go_run_container/docker_test.go`

- [ ] **Step 1: Write failing docker tests**

Create `go_run_container/docker_test.go`:

```go
package main

import (
	"testing"
)

func TestContainerNameForTask(t *testing.T) {
	name, err := containerNameForTask("myTask_123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "taskId_myTask_123" {
		t.Errorf("expected taskId_myTask_123, got %s", name)
	}
}

func TestContainerNameForTaskInvalid(t *testing.T) {
	tests := []string{"", "   ", "has spaces", "中文", "special!char"}
	for _, tid := range tests {
		_, err := containerNameForTask(tid)
		if err == nil {
			t.Errorf("expected error for invalid task_id %q", tid)
		}
	}
}

func TestInferPlatformFromImage(t *testing.T) {
	tests := []struct {
		image    string
		expected string
	}{
		{"ubuntu:latest", ""},
		{"myimage-x86_64:v1", "linux/amd64"},
		{"myimage-amd64:v1", "linux/amd64"},
		{"myimage-arm64:v1", "linux/arm64"},
		{"myimage-aarch64:v1", "linux/arm64"},
		{"", ""},
	}
	for _, tc := range tests {
		got := inferPlatformFromImage(tc.image)
		if got != tc.expected {
			t.Errorf("inferPlatformFromImage(%q) = %q, want %q", tc.image, got, tc.expected)
		}
	}
}

func TestIsSensitiveEnvKey(t *testing.T) {
	sensitive := []string{"TOKEN", "SECRET", "PASSWORD", "KEY", "AUTH", "MY_TOKEN", "API_SECRET", "DB_PASSWORD", "ENCRYPTION_KEY", "AUTH_TOKEN"}
	for _, k := range sensitive {
		if !isSensitiveEnvKey(k) {
			t.Errorf("expected %q to be sensitive", k)
		}
	}

	notSensitive := []string{"ACCESS_TOKEN", "ACCESSTOKEN", "access_token", "Accesstoken", "HOME", "PATH", "USER", "DEBUG"}
	for _, k := range notSensitive {
		if isSensitiveEnvKey(k) {
			t.Errorf("expected %q to NOT be sensitive", k)
		}
	}
}

func TestFormatEnvForLog(t *testing.T) {
	env := map[string]string{
		"ACCESS_TOKEN": "abc123",
		"MY_SECRET":    "xyz789",
		"HOME":         "/root",
	}
	lines := formatEnvForLog(env)

	// ACCESS_TOKEN should be visible
	foundAccessToken := false
	foundMasked := false
	for _, line := range lines {
		if line == "-e ACCESS_TOKEN=abc123" {
			foundAccessToken = true
		}
		if line == "-e MY_SECRET=***" {
			foundMasked = true
		}
	}
	if !foundAccessToken {
		t.Error("expected ACCESS_TOKEN value to be visible")
	}
	if !foundMasked {
		t.Error("expected MY_SECRET value to be masked")
	}
}

func TestDockerExeOrEmpty(t *testing.T) {
	// docker should exist in test environment since we're testing a docker wrapper
	result := dockerExeOrEmpty()
	// This may or may not find docker depending on environment,
	// but should not panic and should return a string
	_ = result
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd /Users/task2app/gitClone/ramDisk/ram-mount/go_run_container && go test -v -run "TestContainerName|TestInferPlatform|TestIsSensitive|TestFormatEnv|TestDocker"
```

Expected: compilation error — functions not defined.

- [ ] **Step 3: Implement docker.go**

Create `go_run_container/docker.go`:

```go
package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

var taskIDRe = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func containerNameForTask(taskID string) (string, error) {
	tid := strings.TrimSpace(taskID)
	if tid == "" || !taskIDRe.MatchString(tid) {
		return "", fmt.Errorf("invalid task_id")
	}
	return "taskId_" + tid, nil
}

func dockerExeOrEmpty() string {
	p, _ := exec.LookPath("docker")
	return p
}

type ContainerStatus struct {
	Running       bool   `json:"running"`
	ContainerName string `json:"container_name"`
	ContainerID   string `json:"container_id"`
	Status        string `json:"status"`
}

func dockerInspect(ctx context.Context, dockerExe, containerName string) (map[string]interface{}, error) {
	cmd := exec.CommandContext(ctx, dockerExe, "inspect", "--type", "container", containerName)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var rows []map[string]interface{}
	if err := json.Unmarshal(out, &rows); err != nil || len(rows) == 0 {
		return nil, fmt.Errorf("no container found")
	}
	return rows[0], nil
}

func containerStatus(ctx context.Context, dockerExe, containerName string) ContainerStatus {
	row, err := dockerInspect(ctx, dockerExe, containerName)
	if err != nil {
		return ContainerStatus{
			Running:       false,
			ContainerName: containerName,
			Status:        "not_found",
		}
	}
	state, _ := row["State"].(map[string]interface{})
	running := false
	statusText := "exited"
	if state != nil {
		if r, ok := state["Running"].(bool); ok {
			running = r
		}
		if s, ok := state["Status"].(string); ok && s != "" {
			statusText = s
		} else if running {
			statusText = "running"
		}
	}
	containerID := ""
	if id, ok := row["Id"].(string); ok {
		containerID = id
	}
	return ContainerStatus{
		Running:       running,
		ContainerName: containerName,
		ContainerID:   containerID,
		Status:        statusText,
	}
}

func inferPlatformFromImage(image string) string {
	lower := strings.ToLower(image)
	if strings.Contains(lower, "x86_64") || strings.Contains(lower, "amd64") {
		return "linux/amd64"
	}
	if strings.Contains(lower, "arm64") || strings.Contains(lower, "aarch64") {
		return "linux/arm64"
	}
	return ""
}

func isSensitiveEnvKey(key string) bool {
	upper := strings.ToUpper(key)
	// ACCESS_TOKEN is NOT masked so developers can debug token exchange issues
	if upper == "ACCESS_TOKEN" || upper == "ACCESSTOKEN" {
		return false
	}
	for _, token := range []string{"TOKEN", "SECRET", "PASSWORD", "KEY", "AUTH"} {
		if strings.Contains(upper, token) {
			return true
		}
	}
	return false
}

func formatEnvForLog(env map[string]string) []string {
	var out []string
	for k, v := range env {
		if isSensitiveEnvKey(k) {
			out = append(out, fmt.Sprintf("-e %s=***", k))
		} else {
			out = append(out, fmt.Sprintf("-e %s=%s", k, v))
		}
	}
	return out
}

func removeContainerIfExists(ctx context.Context, dockerExe, containerName string) error {
	_, err := dockerInspect(ctx, dockerExe, containerName)
	if err != nil {
		return nil // doesn't exist, nothing to do
	}
	cmd := exec.CommandContext(ctx, dockerExe, "rm", "-f", containerName)
	return cmd.Run()
}
```

Wait — I used `json.Unmarshal` but didn't import `encoding/json` and `fmt`. Let me fix the imports in the actual file. Also `json` is used in `dockerInspect` but needs to be imported. Let me also look at my `fmt` calls. Actually, `fmt.Errorf` requires `fmt`, `fmt.Sprintf` also requires `fmt`. And `encoding/json` for `json.Unmarshal`. Let me fix the docker.go file.

Actually, the code above has import issues. Let me make sure the actual file has correct imports:

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd /Users/task2app/gitClone/ramDisk/ram-mount/go_run_container && go test -v -run "TestContainerName|TestInferPlatform|TestIsSensitive|TestFormatEnv|TestDocker"
```

Expected: all docker tests PASS.

- [ ] **Step 5: Commit**

```bash
cd /Users/task2app/gitClone/ramDisk/ram-mount/go_run_container
git add docker.go docker_test.go
git commit -m "feat: add Docker CLI operations"
```

---

### Task 6: HTTP server handlers (server.go)

**Files:**
- Create: `go_run_container/server.go`
- Create: `go_run_container/server_test.go`

- [ ] **Step 1: Write failing server tests**

Create `go_run_container/server_test.go`:

```go
package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestServer() *server {
	cfg := Config{Host: "127.0.0.1", Port: 8796, Secret: "test-secret"}
	store := NewJobStore()
	return newServer(cfg, store)
}

func TestHealthEndpoint(t *testing.T) {
	srv := newTestServer()
	mux := srv.routes()

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	var body map[string]bool
	json.NewDecoder(rec.Body).Decode(&body)
	if !body["ok"] {
		t.Error("expected ok:true")
	}
}

func TestCreateJobNoSecret(t *testing.T) {
	// Server with NO secret configured
	cfg := Config{Host: "127.0.0.1", Port: 8796, Secret: ""}
	srv := newServer(cfg, NewJobStore())
	mux := srv.routes()

	body := map[string]interface{}{
		"image": "alpine:latest",
		"env":   map[string]string{"FOO": "bar"},
	}
	payload, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/v1/jobs", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != 202 {
		t.Errorf("expected 202, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["job_id"] == "" {
		t.Error("expected non-empty job_id")
	}
}

func TestCreateJobWithSecret(t *testing.T) {
	srv := newTestServer()
	mux := srv.routes()

	body := map[string]interface{}{
		"image":   "alpine:latest",
		"task_id": "testTask",
	}
	payload, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/v1/jobs", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Mock-Run-Container-Secret", "test-secret")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != 202 {
		t.Errorf("expected 202, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["job_id"] == "" {
		t.Error("expected non-empty job_id")
	}
	if resp["container_name"] != "taskId_testTask" {
		t.Errorf("expected container_name taskId_testTask, got %v", resp["container_name"])
	}
}

func TestCreateJobUnauthorized(t *testing.T) {
	srv := newTestServer()
	mux := srv.routes()

	body := map[string]interface{}{"image": "alpine:latest"}
	payload, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/v1/jobs", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	// No secret header
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != 401 {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestCreateJobNoImage(t *testing.T) {
	srv := newTestServer()
	mux := srv.routes()

	body := map[string]interface{}{"image": ""}
	payload, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/v1/jobs", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Mock-Run-Container-Secret", "test-secret")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != 400 {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestCreateJobInvalidEnv(t *testing.T) {
	srv := newTestServer()
	mux := srv.routes()

	body := map[string]interface{}{
		"image": "alpine:latest",
		"env":   "not_an_object",
	}
	payload, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/v1/jobs", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Mock-Run-Container-Secret", "test-secret")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != 400 {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestGetJobNotFound(t *testing.T) {
	srv := newTestServer()
	mux := srv.routes()

	req := httptest.NewRequest("GET", "/v1/jobs/nonexistent", nil)
	req.Header.Set("X-Mock-Run-Container-Secret", "test-secret")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != 404 {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestGetJobWithLogs(t *testing.T) {
	srv := newTestServer()
	// Create a job directly in the store
	job := srv.store.Create()
	srv.store.AppendLog(job.ID, "line one")
	srv.store.AppendLog(job.ID, "line two")
	mux := srv.routes()

	req := httptest.NewRequest("GET", "/v1/jobs/"+job.ID, nil)
	req.Header.Set("X-Mock-Run-Container-Secret", "test-secret")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&resp)

	logs, ok := resp["logs"].([]interface{})
	if !ok || len(logs) != 2 {
		t.Fatalf("expected 2 logs, got %v", resp["logs"])
	}
	if resp["done"] != false {
		t.Error("expected done:false")
	}
}

func TestGetJobWithCursor(t *testing.T) {
	srv := newTestServer()
	job := srv.store.Create()
	srv.store.AppendLog(job.ID, "a")
	srv.store.AppendLog(job.ID, "b")
	srv.store.AppendLog(job.ID, "c")
	mux := srv.routes()

	req := httptest.NewRequest("GET", "/v1/jobs/"+job.ID+"?cursor=1", nil)
	req.Header.Set("X-Mock-Run-Container-Secret", "test-secret")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	var resp map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&resp)

	logs := resp["logs"].([]interface{})
	if len(logs) != 2 {
		t.Errorf("expected 2 logs from cursor=1, got %d", len(logs))
	}
	nextCursor, ok := resp["next_cursor"].(float64)
	if !ok || nextCursor != 3 {
		t.Errorf("expected next_cursor=3, got %v", resp["next_cursor"])
	}
}

func TestTaskContainerStatus(t *testing.T) {
	srv := newTestServer()
	mux := srv.routes()

	req := httptest.NewRequest("GET", "/v1/tasks/testTask/container/status", nil)
	req.Header.Set("X-Mock-Run-Container-Secret", "test-secret")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	// Returns 500 if docker not found, or 200 with status — both are valid
	if rec.Code != 200 && rec.Code != 500 {
		t.Errorf("expected 200 or 500, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestTaskContainerStatusInvalidTaskID(t *testing.T) {
	srv := newTestServer()
	mux := srv.routes()

	req := httptest.NewRequest("GET", "/v1/tasks/invalid!id/container/status", nil)
	req.Header.Set("X-Mock-Run-Container-Secret", "test-secret")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != 400 {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestStopJobNotFound(t *testing.T) {
	srv := newTestServer()
	mux := srv.routes()

	req := httptest.NewRequest("POST", "/v1/jobs/nonexistent/stop", nil)
	req.Header.Set("X-Mock-Run-Container-Secret", "test-secret")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != 404 {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd /Users/task2app/gitClone/ramDisk/ram-mount/go_run_container && go test -v -run "TestHealth|TestCreate|TestGet|TestTask|TestStop"
```

Expected: compilation error — `server` type and `newServer` not defined.

- [ ] **Step 3: Implement server.go**

Create `go_run_container/server.go`:

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type server struct {
	cfg   Config
	store *JobStore
}

func newServer(cfg Config, store *JobStore) *server {
	return &server{cfg: cfg, store: store}
}

func (s *server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("POST /v1/jobs", s.handleCreateJob)
	mux.HandleFunc("GET /v1/jobs/{job_id}", s.handleGetJob)
	mux.HandleFunc("POST /v1/jobs/{job_id}/stop", s.handleStopJob)
	mux.HandleFunc("GET /v1/tasks/{task_id}/container/status", s.handleTaskContainerStatus)
	mux.HandleFunc("POST /v1/tasks/{task_id}/container/stop", s.handleStopTaskContainer)
	return authMiddleware(s.cfg.Secret, mux)
}

func (s *server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (s *server) handleCreateJob(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Image          string            `json:"image"`
		Env            map[string]string `json:"env"`
		TaskID         string            `json:"task_id"`
		DockerPlatform string            `json:"docker_platform"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, 400, map[string]string{"detail": "invalid json"})
		return
	}

	image := strings.TrimSpace(body.Image)
	if image == "" {
		writeJSON(w, 400, map[string]string{"detail": "image required"})
		return
	}

	containerName := ""
	if taskID := strings.TrimSpace(body.TaskID); taskID != "" {
		var err error
		containerName, err = containerNameForTask(taskID)
		if err != nil {
			writeJSON(w, 400, map[string]string{"detail": "invalid task_id"})
			return
		}
	}

	envPayload := body.Env
	if envPayload == nil {
		envPayload = make(map[string]string)
	}

	job := s.store.Create()

	go runJob(
		s.store,
		job.ID,
		image,
		envPayload,
		strings.TrimSpace(body.DockerPlatform),
		containerName,
	)

	writeJSON(w, 202, map[string]string{
		"job_id":         job.ID,
		"container_name": containerName,
	})
}

func (s *server) handleGetJob(w http.ResponseWriter, r *http.Request) {
	jobID := r.PathValue("job_id")
	job := s.store.Get(jobID)
	if job == nil {
		writeJSON(w, 404, map[string]string{"detail": "not found"})
		return
	}

	cursor := 0
	if c := r.URL.Query().Get("cursor"); c != "" {
		if parsed, err := strconv.Atoi(c); err == nil && parsed > 0 {
			cursor = parsed
		}
	}

	var logs []string
	if cursor < len(job.Logs) {
		logs = job.Logs[cursor:]
	} else {
		logs = make([]string, 0)
	}

	writeJSON(w, 200, map[string]interface{}{
		"logs":         logs,
		"next_cursor":  len(job.Logs),
		"done":         job.Done,
		"error":        job.Error,
		"result":       job.Result,
		"container_id": job.ContainerID,
	})
}

func (s *server) handleStopJob(w http.ResponseWriter, r *http.Request) {
	jobID := r.PathValue("job_id")
	job := s.store.Get(jobID)
	if job == nil {
		writeJSON(w, 404, map[string]string{"detail": "job not found"})
		return
	}

	containerID := job.ContainerID
	if containerID == "" {
		if job.Done {
			writeJSON(w, 409, map[string]string{"detail": "job already finished"})
		} else {
			writeJSON(w, 409, map[string]string{"detail": "container not started yet"})
		}
		return
	}

	dockerExe := dockerExeOrEmpty()
	if dockerExe == "" {
		s.store.AppendLog(jobID, "[错误] 未检测到 docker，无法停止容器。")
		writeJSON(w, 500, map[string]string{"detail": "docker not found"})
		return
	}

	s.store.AppendLog(jobID, fmt.Sprintf("收到停止容器请求: %s", containerID))
	cmd := exec.Command(dockerExe, "stop", containerID)
	out, err := cmd.CombinedOutput()
	combined := strings.TrimSpace(string(out))
	if err != nil {
		lowered := strings.ToLower(combined)
		if strings.Contains(lowered, "no such container") || strings.Contains(lowered, "is not running") {
			s.store.AppendLog(jobID, fmt.Sprintf("容器已结束或未运行: %s", containerID))
			writeJSON(w, 200, map[string]string{"status": "ok", "container_id": containerID})
			return
		}
		s.store.AppendLog(jobID, fmt.Sprintf("[错误] docker stop 失败: %s", combined))
		writeJSON(w, 502, map[string]string{"detail": combined})
		return
	}

	s.store.AppendLog(jobID, fmt.Sprintf("容器停止成功: %s", containerID))
	writeJSON(w, 202, map[string]string{"status": "accepted", "container_id": containerID})
}

func (s *server) handleTaskContainerStatus(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("task_id")
	containerName, err := containerNameForTask(taskID)
	if err != nil {
		writeJSON(w, 400, map[string]string{"detail": "invalid task_id"})
		return
	}

	dockerExe := dockerExeOrEmpty()
	if dockerExe == "" {
		writeJSON(w, 500, map[string]string{"detail": "docker not found"})
		return
	}

	ctx := context.Background()
	status := containerStatus(ctx, dockerExe, containerName)
	writeJSON(w, 200, status)
}

func (s *server) handleStopTaskContainer(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("task_id")
	containerName, err := containerNameForTask(taskID)
	if err != nil {
		writeJSON(w, 400, map[string]string{"detail": "invalid task_id"})
		return
	}

	dockerExe := dockerExeOrEmpty()
	if dockerExe == "" {
		writeJSON(w, 500, map[string]string{"detail": "docker not found"})
		return
	}

	ctx := context.Background()
	status := containerStatus(ctx, dockerExe, containerName)
	if !status.Running {
		writeJSON(w, 200, map[string]string{
			"status":         "ok",
			"container_name": containerName,
			"container_id":   status.ContainerID,
			"message":        "container not running",
		})
		return
	}

	cmd := exec.CommandContext(ctx, dockerExe, "stop", containerName)
	out, err := cmd.CombinedOutput()
	combined := strings.TrimSpace(string(out))
	if err != nil {
		lowered := strings.ToLower(combined)
		if strings.Contains(lowered, "no such container") || strings.Contains(lowered, "is not running") {
			writeJSON(w, 200, map[string]string{
				"status":         "ok",
				"container_name": containerName,
				"message":        "container already stopped",
			})
			return
		}
		writeJSON(w, 502, map[string]string{"detail": combined})
		return
	}

	writeJSON(w, 202, map[string]string{
		"status":         "accepted",
		"container_name": containerName,
		"container_id":   status.ContainerID,
	})
}

func runJob(store *JobStore, jobID, image string, envPayload map[string]string, dockerPlatform, containerName string) {
	defer func() {
		if r := recover(); r != nil {
			store.AppendLog(jobID, fmt.Sprintf("[错误] panic: %v", r))
			store.Finish(jobID, fmt.Sprintf("%v", r), nil)
		}
	}()

	dockerExe := dockerExeOrEmpty()
	if dockerExe == "" {
		store.AppendLog(jobID, "[错误] 未检测到 docker，请先安装 Docker。")
		store.Finish(jobID, "docker_not_found", nil)
		return
	}

	if image == "" {
		store.AppendLog(jobID, "[错误] 镜像地址为空。")
		store.Finish(jobID, "image_required", nil)
		return
	}

	resolvedPlatform := dockerPlatform
	if resolvedPlatform == "" {
		resolvedPlatform = strings.TrimSpace(os.Getenv("MOCK_RUN_CONTAINER_DOCKER_PLATFORM"))
	}
	if resolvedPlatform == "" {
		resolvedPlatform = inferPlatformFromImage(image)
	}

	// docker pull
	pullArgs := []string{"pull"}
	if resolvedPlatform != "" {
		store.AppendLog(jobID, fmt.Sprintf("按平台拉取镜像: %s", resolvedPlatform))
		pullArgs = append(pullArgs, "--platform", resolvedPlatform)
	}
	pullArgs = append(pullArgs, image)
	store.AppendLog(jobID, fmt.Sprintf("开始拉取镜像: %s", image))
	store.AppendLog(jobID, fmt.Sprintf("执行命令: docker %s", strings.Join(pullArgs, " ")))

	pullCode, pullOutput := streamCommand(store, jobID, dockerExe, pullArgs...)
	if pullCode != 0 && resolvedPlatform == "" && strings.Contains(strings.ToLower(pullOutput), "no matching manifest for linux/arm64") {
		resolvedPlatform = "linux/amd64"
		store.AppendLog(jobID, "检测到 arm64 清单缺失，自动回退到 --platform linux/amd64 重试")
		retryArgs := []string{"pull", "--platform", resolvedPlatform, image}
		store.AppendLog(jobID, fmt.Sprintf("执行命令: docker %s", strings.Join(retryArgs, " ")))
		pullCode, _ = streamCommand(store, jobID, dockerExe, retryArgs...)
	}
	if pullCode != 0 {
		store.AppendLog(jobID, fmt.Sprintf("[错误] docker pull 失败，退出码=%d", pullCode))
		store.Finish(jobID, "docker_pull_failed", nil)
		return
	}

	// Remove existing container if named
	if containerName != "" {
		ctx := context.Background()
		removeContainerIfExists(ctx, dockerExe, containerName)
	}

	// docker run
	store.AppendLog(jobID, "开始运行容器...")
	runArgs := []string{"run", "-d", "--rm", "--network", "host"}
	if resolvedPlatform != "" {
		runArgs = append(runArgs, "--platform", resolvedPlatform)
	}
	if containerName != "" {
		runArgs = append(runArgs, "--name", containerName)
	}
	for k, v := range envPayload {
		if v != "" {
			runArgs = append(runArgs, "-e", fmt.Sprintf("%s=%s", k, v))
		}
	}
	runArgs = append(runArgs, image)

	// Build log-safe version of the command
	logRunArgs := []string{"run", "-d", "--rm", "--network", "host"}
	if resolvedPlatform != "" {
		logRunArgs = append(logRunArgs, "--platform", resolvedPlatform)
	}
	if containerName != "" {
		logRunArgs = append(logRunArgs, "--name", containerName)
	}
	for _, line := range formatEnvForLog(envPayload) {
		logRunArgs = append(logRunArgs, line)
	}
	logRunArgs = append(logRunArgs, image)
	store.AppendLog(jobID, fmt.Sprintf("执行命令: docker %s", strings.Join(logRunArgs, " ")))

	cmd := exec.Command(dockerExe, runArgs...)
	out, err := cmd.CombinedOutput()
	combined := strings.TrimSpace(string(out))
	if err != nil {
		store.AppendLog(jobID, fmt.Sprintf("[错误] docker run 失败: %s", combined))
		store.Finish(jobID, "docker_run_failed", nil)
		return
	}

	containerID := strings.TrimSpace(string(out))
	store.AppendLog(jobID, fmt.Sprintf("容器启动成功: %s", containerID))
	store.SetContainerID(jobID, containerID)

	// docker logs -f
	store.AppendLog(jobID, "开始采集容器日志...")
	store.AppendLog(jobID, fmt.Sprintf("执行命令: docker logs -f %s", containerID))
	logsCode, logsOutput := streamCommand(store, jobID, dockerExe, "logs", "-f", containerID)
	if logsCode != 0 {
		logsText := strings.TrimSpace(logsOutput)
		if logsText != "" {
			store.AppendLog(jobID, fmt.Sprintf("[警告] 容器日志采集退出码=%d: %s", logsCode, logsText))
		} else {
			store.AppendLog(jobID, fmt.Sprintf("[警告] 容器日志采集退出码=%d", logsCode))
		}
	}
	store.AppendLog(jobID, "容器日志流结束")
	store.Finish(jobID, "", map[string]string{
		"container_id":   containerID,
		"container_name": containerName,
		"image":          image,
		"docker_platform": resolvedPlatform,
	})
}

func streamCommand(store *JobStore, jobID, exe string, args ...string) (int, string) {
	cmd := exec.Command(exe, args...)
	stdout, _ := cmd.StdoutPipe()
	cmd.Stderr = cmd.Stdout // merge stderr into stdout

	if err := cmd.Start(); err != nil {
		msg := fmt.Sprintf("[错误] 启动命令失败: %v", err)
		store.AppendLog(jobID, msg)
		return -1, msg
	}

	var allOutput strings.Builder
	buf := make([]byte, 4096)
	for {
		n, readErr := stdout.Read(buf)
		if n > 0 {
			chunk := string(buf[:n])
			allOutput.WriteString(chunk)
			for _, line := range strings.Split(chunk, "\n") {
				line = strings.TrimRight(line, "\r")
				store.AppendLog(jobID, line)
			}
		}
		if readErr != nil {
			break
		}
	}

	exitCode := 0
	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}
	return exitCode, allOutput.String()
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
```

- [ ] **Step 4: Run server tests (non-Docker tests should pass)**

```bash
cd /Users/task2app/gitClone/ramDisk/ram-mount/go_run_container && go test -v -run "TestHealth|TestCreate|TestGet|TestStopJobNotFound"
```

Expected: Tests that don't need Docker PASS. Tests involving Docker (`TestTaskContainerStatus`) may fail if Docker not available — that's expected.

- [ ] **Step 5: Commit**

```bash
cd /Users/task2app/gitClone/ramDisk/ram-mount/go_run_container
git add server.go server_test.go
git commit -m "feat: add HTTP server handlers for all 6 endpoints"
```

---

### Task 7: Entry point and build script (main.go, start.sh, README.md)

**Files:**
- Create: `go_run_container/main.go`
- Create: `go_run_container/start.sh`
- Create: `go_run_container/README.md`

- [ ] **Step 1: Create main.go**

```go
package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	cfg := LoadConfig()
	store := NewJobStore()
	srv := newServer(cfg, store)

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	log.Printf("go_run_container starting on %s (secret=%t)", addr, cfg.Secret != "")
	if err := http.ListenAndServe(addr, srv.routes()); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
```

- [ ] **Step 2: Create start.sh**

```bash
#!/usr/bin/env bash
set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$HERE"

echo "Building go_run_container..."
go build -o go_run_container .

echo "Starting go_run_container..."
exec ./go_run_container
```

Make it executable:

```bash
chmod +x /Users/task2app/gitClone/ramDisk/ram-mount/go_run_container/start.sh
```

- [ ] **Step 3: Create README.md**

```markdown
# go_run_container

Go rewrite of mock_run_container — local Docker container runner for mockStart.

## API

Same API as the Python mock_run_container service:

- `GET  /health` — health check
- `POST /v1/jobs` — create container run job
- `GET  /v1/jobs/{job_id}?cursor=N` — poll logs
- `POST /v1/jobs/{job_id}/stop` — stop by job_id
- `GET  /v1/tasks/{task_id}/container/status` — container status
- `POST /v1/tasks/{task_id}/container/stop` — stop by task_id

## Run

```bash
./start.sh
```

Or directly:

```bash
go build -o go_run_container .
MOCK_RUN_CONTAINER_HOST=127.0.0.1 MOCK_RUN_CONTAINER_PORT=8796 ./go_run_container
```

## Config

1. Reads `../task2app/conf/port_config.json` → `mock_run_container` block
2. Env var overrides: `MOCK_RUN_CONTAINER_HOST`, `MOCK_RUN_CONTAINER_PORT`, `MOCK_RUN_CONTAINER_SECRET`
3. Defaults: `127.0.0.1:8796`, no auth

## Auth

If `MOCK_RUN_CONTAINER_SECRET` is set, requests (except `/health`) must include `X-Mock-Run-Container-Secret` header.
```

- [ ] **Step 4: Verify it compiles**

```bash
cd /Users/task2app/gitClone/ramDisk/ram-mount/go_run_container && go build -o /dev/null .
```

Expected: compilation succeeds with no errors.

- [ ] **Step 5: Run all tests**

```bash
cd /Users/task2app/gitClone/ramDisk/ram-mount/go_run_container && go test ./... -v -count=1
```

Expected: all tests PASS (Docker-dependent tests may be skipped or fail gracefully if no Docker daemon).

- [ ] **Step 6: Commit**

```bash
cd /Users/task2app/gitClone/ramDisk/ram-mount/go_run_container
git add main.go start.sh README.md
git commit -m "feat: add entry point, build script, and docs"
```

---

## Integration Verification (after all tasks)

- [ ] Start the server: `cd go_run_container && go run .`
- [ ] Health check: `curl http://127.0.0.1:8796/health`
- [ ] Create job with auth: `curl -X POST http://127.0.0.1:8796/v1/jobs -H 'Content-Type: application/json' -H 'X-Mock-Run-Container-Secret: dev-secret' -d '{"image":"alpine:latest","env":{"FOO":"bar"}}'`
- [ ] Poll logs: `curl http://127.0.0.1:8796/v1/jobs/<job_id> -H 'X-Mock-Run-Container-Secret: dev-secret'`
- [ ] Verify Django client compatibility: the existing `cloud/services/mock_run_container.py` should work unchanged
