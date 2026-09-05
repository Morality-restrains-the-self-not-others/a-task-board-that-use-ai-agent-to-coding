package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// TestForwardToContainerService_NotifiesCloudOnConnectionRefused — OPT-20260818-008。
// 转发到容器 server_url 收到 connection refused 时，须通知 cloud 清推测性地址并
// demote binding（POST /api/internal/cloud-server-config/container-unreachable/）。
func TestForwardToContainerService_NotifiesCloudOnConnectionRefused(t *testing.T) {
	var mu sync.Mutex
	notified := 0
	cloud := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/internal/cloud-server-config/container-unreachable/" {
			mu.Lock()
			notified++
			mu.Unlock()
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"cleared": true, "demoted": true})
	}))
	defer cloud.Close()

	prev := cfg
	cfg = serviceConfig{CloudServiceURL: cloud.URL, ForwardReadSec: 5, ForwardConnectSec: 1}
	t.Cleanup(func() { cfg = prev })

	sc := scope{TenantID: "t1", WorkspaceID: "ws1", TaskID: "task1", CommentID: "cmt1"}
	// 指向本机关闭端口 → connection refused
	status, _, _ := forwardToContainerService(context.Background(), sc, http.MethodGet, "http://127.0.0.1:1/", "tok", nil)
	if status != http.StatusBadGateway {
		t.Fatalf("status=%d want 502", status)
	}
	mu.Lock()
	n := notified
	mu.Unlock()
	if n != 1 {
		t.Fatalf("cloud notified=%d want 1 (connection refused)", n)
	}
}

// TestForwardToContainerService_NoNotifyOnSuccess — 上游正常时不得通知 cloud demote。
func TestForwardToContainerService_NoNotifyOnSuccess(t *testing.T) {
	var mu sync.Mutex
	notified := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer upstream.Close()

	cloud := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/internal/cloud-server-config/container-unreachable/" {
			mu.Lock()
			notified++
			mu.Unlock()
		}
		_ = json.NewEncoder(w).Encode(map[string]any{})
	}))
	defer cloud.Close()

	prev := cfg
	cfg = serviceConfig{CloudServiceURL: cloud.URL, ForwardReadSec: 5, ForwardConnectSec: 1}
	t.Cleanup(func() { cfg = prev })

	sc := scope{TenantID: "t1", WorkspaceID: "ws1", TaskID: "task1", CommentID: "cmt1"}
	status, _, _ := forwardToContainerService(context.Background(), sc, http.MethodGet, upstream.URL+"/x", "tok", nil)
	if status != http.StatusOK {
		t.Fatalf("status=%d want 200", status)
	}
	mu.Lock()
	n := notified
	mu.Unlock()
	if n != 0 {
		t.Fatalf("cloud notified=%d want 0 (upstream healthy)", n)
	}
}
