package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// OPT-20260816-028：泄漏实例对账迁到 taskEvents cloud_csc_reconcile timer，
// taskCloudService 仅暴露一次性 HTTP 端点（不再自带进程内 ticker）。
func TestHandleInternalReconcileLeakedServersCleans(t *testing.T) {
	setupCloudTestDB(t)
	now := time.Now().UTC()
	if _, err := db.Exec(`INSERT INTO cloud_server_configs
		(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, region, zone_id, authorization_id, server_url, terminal_released, last_runtime_status, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		"csc-leaked", "t1", "ws1", "task-leaked", "", "aliyun", "i-leak", "cn-hangzhou", "cn-hangzhou-i", "auth1",
		"http://127.0.0.1:18080", 1, "Starting", now, now); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/internal/cloud/compute/reconcile-leaked-servers/", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	handleInternalReconcileLeakedServers(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"cleaned":1`) {
		t.Fatalf("cleaned not reflected in body: %s", rec.Body.String())
	}

	var url string
	if err := db.QueryRow(`SELECT COALESCE(server_url,'') FROM cloud_server_configs WHERE id='csc-leaked'`).Scan(&url); err != nil {
		t.Fatal(err)
	}
	if url != "" {
		t.Fatalf("server_url=%q want empty after leak cleanup", url)
	}
}

func TestHandleInternalReconcileLeakedServersMethodNotAllowed(t *testing.T) {
	setupCloudTestDB(t)
	req := httptest.NewRequest(http.MethodGet, "/api/internal/cloud/compute/reconcile-leaked-servers/", nil)
	rec := httptest.NewRecorder()
	handleInternalReconcileLeakedServers(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d want 405", rec.Code)
	}
}

func TestHandleInternalReconcileLeakedServersForbiddenWithoutSecret(t *testing.T) {
	setupCloudTestDB(t)
	prev := cfg.InternalSecret
	cfg.InternalSecret = "sec"
	defer func() { cfg.InternalSecret = prev }()

	req := httptest.NewRequest(http.MethodPost, "/api/internal/cloud/compute/reconcile-leaked-servers/", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	handleInternalReconcileLeakedServers(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d want 403", rec.Code)
	}
}

// OPT-20260816-028 回归：移除进程内 ticker 后，文件不得再引用 time.NewTicker/禁用 env。
func TestServerReleaseReconcileNoLegacyTicker(t *testing.T) {
	src, err := os.ReadFile("server_release_reconcile.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, needle := range []string{"time.NewTicker", "RECONCILE_LEAKED_TICK_SEC", "startLeakedServerReconcileTicker"} {
		if strings.Contains(string(src), needle) {
			t.Fatalf("server_release_reconcile.go must not reference %q (ticker 已迁 timer)", needle)
		}
	}
}
