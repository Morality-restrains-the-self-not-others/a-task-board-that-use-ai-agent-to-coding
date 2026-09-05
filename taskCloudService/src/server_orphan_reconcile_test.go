package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// OPT-20260816-027：孤儿对账改走 taskTaskService 批量 HTTP API，禁止直连 task-task 库。
func TestReconcileOrphanTaskCSCsViaHTTP(t *testing.T) {
	setupCloudTestDB(t)

	var called bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.URL.Path != "/api/internal/tasks/exists/" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		if r.Header.Get("X-Internal-Secret") != "sec" {
			t.Errorf("missing internal secret header")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"exists":{"task-alive":true,"task-dead":false}}`))
	}))
	defer srv.Close()

	prevURL := cfg.TaskServiceURL
	prevSecret := cfg.InternalSecret
	cfg.TaskServiceURL = srv.URL
	cfg.InternalSecret = "sec"
	defer func() {
		cfg.TaskServiceURL = prevURL
		cfg.InternalSecret = prevSecret
	}()

	now := time.Now().UTC()
	for _, row := range []struct{ id, taskID string }{
		{"csc-alive", "task-alive"},
		{"csc-dead", "task-dead"},
	} {
		if _, err := db.Exec(`INSERT INTO cloud_server_configs(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, region, zone_id, authorization_id, created_at, updated_at)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
			row.id, "t1", "ws1", row.taskID, "", "aliyun", "i-"+row.taskID, "cn-hangzhou", "cn-hangzhou-i", "auth1", now, now); err != nil {
			t.Fatal(err)
		}
	}

	cleaned, err := reconcileOrphanTaskCSCs(time.Now().UTC())
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if !called {
		t.Fatal("expected task service HTTP call")
	}
	if cleaned != 1 {
		t.Fatalf("cleaned=%d want 1", cleaned)
	}
	var released int
	if err := db.QueryRow(`SELECT COUNT(*) FROM cloud_server_configs WHERE id='csc-dead' AND terminal_released=1`).Scan(&released); err != nil {
		t.Fatal(err)
	}
	if released != 1 {
		t.Fatalf("csc-dead not marked terminal_released")
	}
	var aliveReleased int
	if err := db.QueryRow(`SELECT COUNT(*) FROM cloud_server_configs WHERE id='csc-alive' AND terminal_released=1`).Scan(&aliveReleased); err != nil {
		t.Fatal(err)
	}
	if aliveReleased != 0 {
		t.Fatalf("csc-alive must not be marked terminal_released")
	}
}

func TestReconcileOrphanTaskCSCsEmpty(t *testing.T) {
	setupCloudTestDB(t)
	cleaned, err := reconcileOrphanTaskCSCs(time.Now().UTC())
	if err != nil {
		t.Fatalf("reconcile empty: %v", err)
	}
	if cleaned != 0 {
		t.Fatalf("cleaned=%d want 0", cleaned)
	}
}

// OPT-20260816-027 回归：孤儿对账文件不得再引用跨库 DSN 直连。
func TestServerOrphanReconcileNoCrossTaskDSN(t *testing.T) {
	src, err := os.ReadFile("server_orphan_reconcile.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, needle := range []string{"ResolveMySQLDSN", "TASK_TASK_MYSQL_DSN", "sql.Open(\"mysql\""} {
		if strings.Contains(string(src), needle) {
			t.Fatalf("server_orphan_reconcile.go must not reference %q (cross-repo DSN banned)", needle)
		}
	}
}
