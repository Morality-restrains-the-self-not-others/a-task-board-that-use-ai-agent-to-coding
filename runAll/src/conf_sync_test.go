package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"runAll/src/domain"
)

func TestAPIConfSync_Accepted(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join("..", ".."))
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{{
			Name: "platform",
			Services: []Service{{
				Name:         "task-auth",
				StartCommand: "true",
				WorkingDir:   filepath.Join(repoRoot, "taskAuth"),
				HealthCheck:  HealthCheck{URL: "http://localhost:1/health"},
			}},
		}},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/conf/sync", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202", rec.Code)
	}
}

func TestSyncMonorepoConf_RunsScript(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join("..", ".."))
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{{
			Services: []Service{{
				Name:         "task-auth",
				StartCommand: "true",
				WorkingDir:   filepath.Join(repoRoot, "taskAuth"),
				HealthCheck:  HealthCheck{URL: "http://localhost:1/health"},
			}},
		}},
	}, NewStatusStore())
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	if err := runner.SyncMonorepoConf(context.Background()); err != nil {
		t.Fatalf("SyncMonorepoConf: %v", err)
	}
}

type countingDevToolLogRecorder struct {
	mu    sync.Mutex
	start int
}

func (c *countingDevToolLogRecorder) Append(tool string, message string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if tool == domain.ToolConfSync && strings.Contains(message, "开始生成配置副本") {
		c.start++
	}
	return nil
}
func (c *countingDevToolLogRecorder) Tail(tool string, lines int) ([]string, error) {
	return nil, nil
}
func (c *countingDevToolLogRecorder) Clear(tool string) error { return nil }
func (c *countingDevToolLogRecorder) ClearAll() error         { return nil }

func TestBuildAll_SyncsMonorepoConfOnce(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join("..", ".."))
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{{
			Name: "platform",
			Services: []Service{
				{Name: "svc-a", StartCommand: "true", BuildCommand: "true", WorkingDir: filepath.Join(repoRoot, "taskAuth"), HealthCheck: HealthCheck{URL: "http://localhost:1/health"}},
				{Name: "svc-b", StartCommand: "true", BuildCommand: "true", WorkingDir: filepath.Join(repoRoot, "taskAuth"), HealthCheck: HealthCheck{URL: "http://localhost:1/health"}},
				{Name: "svc-c", StartCommand: "true", BuildCommand: "true", WorkingDir: filepath.Join(repoRoot, "taskAuth"), HealthCheck: HealthCheck{URL: "http://localhost:1/health"}},
			},
		}},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	rec := &countingDevToolLogRecorder{}
	runner.devToolLogRecorder = rec
	for _, name := range []string{"svc-a", "svc-b", "svc-c"} {
		store.Update(name, StatusStopped, "")
	}
	runID, ok := runner.TryBeginBuildAllRun()
	if !ok {
		t.Fatal("TryBeginBuildAllRun failed")
	}
	defer runner.EndBulk("build-all")
	_ = runID

	if _, err := runner.BuildAll(context.Background()); err != nil {
		t.Fatalf("BuildAll: %v", err)
	}
	rec.mu.Lock()
	got := rec.start
	rec.mu.Unlock()
	if got != 1 {
		t.Fatalf("conf-sync start count = %d, want 1 (once per BuildAll, not per service)", got)
	}
}
