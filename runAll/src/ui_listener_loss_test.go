package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// OPT-20260817-001 回归：UI 监听丢失（bind 失败 / 监听器死亡）与 shutdown-self
// 完成后进程必须退出，且不得顺带停掉托管服务（防止残留 runAll 在 SIGTERM 时
// 误停正在运行的服务，也防止 bind 失败时新实例去停兄弟实例的服务）。

func TestUIListenerLost_CancelsRunLoopAndSkipsServiceShutdown(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{Version: "1", Groups: []Group{
		{
			Name: "platform",
			Services: []Service{
				{Name: "svc", Command: "./run.sh", HealthCheck: HealthCheck{URL: "http://127.0.0.1:8002/api/health/"}},
			},
		},
	}}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	uiListenerLost(runner, cancel, context.DeadlineExceeded)

	select {
	case <-ctx.Done():
		// cancel fired → runner.Run returns → process exits instead of lingering.
	default:
		t.Fatal("uiListenerLost did not cancel the run loop context")
	}
	if !runner.skipShutdownServices {
		t.Fatal("uiListenerLost must set skipShutdownServices so managed services stay up")
	}
}

func TestStartUIServer_BindFailureCancelsRunLoop(t *testing.T) {
	// Invalid bind address: SO_REUSEPORT (ADR-0058) can share a live port, so
	// occupying 127.0.0.1:0 no longer proves Listen failure.
	store := NewStatusStore()
	runner, err := NewRunner(&Config{Version: "1"}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_ = startUIServer(store, runner, "256.256.256.256:1", cancel, newRunAllExitReason())

	select {
	case <-ctx.Done():
	case <-time.After(3 * time.Second):
		t.Fatal("bind failure did not cancel the run loop — a UI-less runAll would linger")
	}
	if !runner.skipShutdownServices {
		t.Fatal("bind failure must skip managed service shutdown so the sibling instance's services are not torn down")
	}
}

func TestShutdownSelf_CancelsRunLoopAndSkipsServiceShutdown(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{Version: "1"}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	exitReason := newRunAllExitReason()

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, cancel, exitReason)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/shutdown-self", strings.NewReader("")))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("body = %#v, want status=ok", body)
	}

	select {
	case <-ctx.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("shutdown-self did not cancel the run loop — the process must exit after shutdown-self")
	}
	if !runner.skipShutdownServices {
		t.Fatal("shutdown-self must set skipShutdownServices")
	}
	source, detail := exitReason.get()
	if source != "shutdown-self" || detail != "/api/shutdown-self" {
		t.Fatalf("exit reason = %q/%q, want shutdown-self//api/shutdown-self", source, detail)
	}
}
