package wechatmpcleanup

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// OPT-20260826-003：taskAuth 实际监听 8003；兜底默认对齐其它 timer worker。
func TestNewTaskAuthCleanupClientDefaultBaseURLIs8003(t *testing.T) {
	t.Setenv("TASK_AUTH_INTERNAL_URL", "")
	t.Setenv("TASK_AUTH_INTERNAL_SECRET", "")
	t.Setenv("SHARED_INTERNAL_SECRET", "")
	c := NewTaskAuthCleanupClient()
	if c.BaseURL != "http://127.0.0.1:8003" {
		t.Fatalf("default BaseURL = %q, want http://127.0.0.1:8003", c.BaseURL)
	}
}

func TestNewTaskAuthCleanupClientUsesEnvAndTrimsSlash(t *testing.T) {
	t.Setenv("TASK_AUTH_INTERNAL_URL", "http://127.0.0.1:8003/")
	t.Setenv("TASK_AUTH_INTERNAL_SECRET", "sekret")
	c := NewTaskAuthCleanupClient()
	if c.BaseURL != "http://127.0.0.1:8003" {
		t.Fatalf("BaseURL = %q, want trimmed http://127.0.0.1:8003", c.BaseURL)
	}
	if c.InternalSecret != "sekret" {
		t.Fatalf("InternalSecret = %q, want sekret", c.InternalSecret)
	}
}

// CleanupOnce 应命中 wechat-mp-cleanup 路径并透传 taskAuth 期望的 X-TaskAuth-Internal-Secret 头。
func TestTaskAuthCleanupClientCleanupOnceHitsCleanupPath(t *testing.T) {
	var gotPath, gotSecret string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotSecret = r.Header.Get("X-TaskAuth-Internal-Secret")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"pending_deleted":2,"ticket_deleted":1}`))
	}))
	defer srv.Close()

	c := &TaskAuthCleanupClient{
		BaseURL:        strings.TrimRight(srv.URL, "/"),
		InternalSecret: "test-secret",
		HTTPClient:     srv.Client(),
	}
	if err := c.CleanupOnce(context.Background()); err != nil {
		t.Fatalf("CleanupOnce() error = %v", err)
	}
	if gotPath != cleanupPath {
		t.Fatalf("request path = %q, want %q", gotPath, cleanupPath)
	}
	if gotSecret != "test-secret" {
		t.Fatalf("X-TaskAuth-Internal-Secret = %q, want test-secret", gotSecret)
	}
}

// CleanupOnce 对 4xx/5xx 应返回错误而非吞掉。
func TestTaskAuthCleanupClientCleanupOnceReturnsErrorOnNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		_, _ = w.Write([]byte(`oops`))
	}))
	defer srv.Close()

	c := &TaskAuthCleanupClient{
		BaseURL:    strings.TrimRight(srv.URL, "/"),
		HTTPClient: srv.Client(),
	}
	if err := c.CleanupOnce(context.Background()); err == nil {
		t.Fatal("CleanupOnce() expected error on HTTP 500, got nil")
	}
}

// CleanupTickInterval 默认 24h，非正环境变量回退默认。
func TestCleanupTickIntervalDefaultAndFallback(t *testing.T) {
	t.Setenv("WECHAT_MP_CLEANUP_TICK_SEC", "")
	if got := CleanupTickInterval(); got != 24*3600e9 {
		t.Fatalf("default tick = %v, want 24h", got)
	}
	t.Setenv("WECHAT_MP_CLEANUP_TICK_SEC", "0")
	if got := CleanupTickInterval(); got != 24*3600e9 {
		t.Fatalf("fallback tick = %v, want 24h", got)
	}
	t.Setenv("WECHAT_MP_CLEANUP_TICK_SEC", "3600")
	if got := CleanupTickInterval(); got != 3600e9 {
		t.Fatalf("configured tick = %v, want 1h", got)
	}
}
