package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

// OPT-20260816-032：relayToTrae 状态推送改为事件驱动，删除 1.5s ticker。

func TestNoTickerInPushLoop(t *testing.T) {
	raw, err := os.ReadFile("push.go")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "time.NewTicker") {
		t.Fatal("push.go must not use time.NewTicker (event-driven push, OPT-20260816-032)")
	}
}

func newPushCounterServer(t *testing.T) (string, *int, *sync.Mutex) {
	t.Helper()
	var mu sync.Mutex
	var pushes int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		pushes++
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ack":1,"status":"ok"}`))
	}))
	t.Cleanup(server.Close)
	return server.URL, &pushes, &mu
}

func resetPushTestState(t *testing.T, serverURL string) {
	t.Helper()
	stateMu.Lock()
	defer stateMu.Unlock()
	state.TokenSyncPending = false
	state.AccessToken = ""
	state.RefreshToken = ""
	state.AccessTokenExpiresAt = time.Time{}
	state.Running = false
	state.Port = 0
	registeredTasks = make(map[string]*RegisteredTask)
	taskSeq = make(map[string]int)
}

func TestHandleRegister_PushesAtLeastOnce(t *testing.T) {
	serverURL, pushes, mu := newPushCounterServer(t)
	resetPushTestState(t, serverURL)

	body := `{"tenant_id":"t1","workspace_id":"w1","task_id":"push-event-task","task_api_endpoint_origin":"` +
		serverURL + `","access_token":"tok","comment_id":"cmt-push-event"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/register", bytes.NewReader([]byte(body)))
	rec := httptest.NewRecorder()
	handleRegister(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("handleRegister status = %d, want 200 (body=%s)", rec.Code, rec.Body.String())
	}

	mu.Lock()
	got := *pushes
	mu.Unlock()
	if got < 1 {
		t.Fatalf("expected >=1 push after register (event-driven), got %d", got)
	}
}

func TestPushAllRegisteredTasks_NoPeriodicRequests(t *testing.T) {
	serverURL, pushes, mu := newPushCounterServer(t)
	resetPushTestState(t, serverURL)

	stateMu.Lock()
	registerTaskLocked("t1", "w1", "push-no-loop-task", serverURL+"/api", "tok", "cmt-push-no-loop")
	stateMu.Unlock()

	pushAllRegisteredTasks()

	mu.Lock()
	got := *pushes
	mu.Unlock()
	if got != 1 {
		t.Fatalf("expected exactly 1 push, got %d", got)
	}

	// 原 1.5s 轮询已删除：等待超过一个轮询周期后不应再有新请求。
	time.Sleep(1700 * time.Millisecond)
	mu.Lock()
	got = *pushes
	mu.Unlock()
	if got != 1 {
		t.Fatalf("expected no periodic pushes after event push, got %d", got)
	}
}

func TestHandleStop_NoPeriodicRequests(t *testing.T) {
	serverURL, pushes, mu := newPushCounterServer(t)
	resetPushTestState(t, serverURL)

	stateMu.Lock()
	registerTaskLocked("t1", "w1", "push-stop-task", serverURL+"/api", "tok", "cmt-push-stop")
	stateMu.Unlock()

	req := httptest.NewRequest(http.MethodPost, "/v1/stop", nil)
	rec := httptest.NewRecorder()
	handleStop(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("handleStop status = %d, want 200 (body=%s)", rec.Code, rec.Body.String())
	}

	mu.Lock()
	got := *pushes
	mu.Unlock()
	if got != 1 {
		t.Fatalf("expected exactly 1 push after stop (event-driven), got %d", got)
	}

	// 停机后不再周期请求。
	time.Sleep(1700 * time.Millisecond)
	mu.Lock()
	got = *pushes
	mu.Unlock()
	if got != 1 {
		t.Fatalf("expected no periodic pushes after stop, got %d", got)
	}
}
