package useraccountdeletionexecute

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// OPT-20260824-017：taskAuth 实际监听 8003（conf/auth/task-auth/config.yaml port: 8003）。
// 兜底默认从 8001 修正为 8003，避免 execute-due 扫描在无 env 注入时打错端口。
func TestNewTaskAuthExecuteClientDefaultBaseURLIs8003(t *testing.T) {
	t.Setenv("TASK_AUTH_INTERNAL_URL", "")
	t.Setenv("TASK_AUTH_INTERNAL_SECRET", "")
	t.Setenv("SHARED_INTERNAL_SECRET", "")
	c := NewTaskAuthExecuteClient()
	if c.BaseURL != "http://127.0.0.1:8003" {
		t.Fatalf("default BaseURL = %q, want http://127.0.0.1:8003", c.BaseURL)
	}
}

func TestNewTaskAuthExecuteClientUsesEnvAndTrimsSlash(t *testing.T) {
	t.Setenv("TASK_AUTH_INTERNAL_URL", "http://127.0.0.1:8003/")
	t.Setenv("TASK_AUTH_INTERNAL_SECRET", "sekret")
	c := NewTaskAuthExecuteClient()
	if c.BaseURL != "http://127.0.0.1:8003" {
		t.Fatalf("BaseURL = %q, want trimmed http://127.0.0.1:8003", c.BaseURL)
	}
	if c.InternalSecret != "sekret" {
		t.Fatalf("InternalSecret = %q, want sekret", c.InternalSecret)
	}
}

// ExecuteOnce 应命中 execute-due 路径并透传内部密钥头。
func TestTaskAuthExecuteClientExecuteOnceHitsExecuteDuePath(t *testing.T) {
	var gotPath, gotSecret string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotSecret = r.Header.Get("X-Internal-Secret")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"processed":2}`))
	}))
	defer srv.Close()

	c := &TaskAuthExecuteClient{
		BaseURL:        strings.TrimRight(srv.URL, "/"),
		InternalSecret: "test-secret",
		HTTPClient:     srv.Client(),
	}
	if err := c.ExecuteOnce(context.Background()); err != nil {
		t.Fatalf("ExecuteOnce() error = %v", err)
	}
	if gotPath != executeDuePath {
		t.Fatalf("request path = %q, want %q", gotPath, executeDuePath)
	}
	if gotSecret != "test-secret" {
		t.Fatalf("X-Internal-Secret = %q, want test-secret", gotSecret)
	}
}
