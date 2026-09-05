package main

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func seedStaleStartingReachabilityFixture(t *testing.T, suffix string, createdAt time.Time, runtime, serverURL, bindingStatus string) (cscID string, bindingID string) {
	t.Helper()
	companyID := "t-stale-" + suffix
	workspaceID := "ws-stale-" + suffix
	taskID := "taskStale" + suffix
	commentID := "cmtStale" + suffix
	cscID = "csc-stale-" + suffix
	_, err := db.Exec(
		`INSERT INTO cloud_server_configs(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, public_ip, server_url, region, zone_id, authorization_id, last_runtime_status, created_at, updated_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		cscID, companyID, workspaceID, taskID, commentID, "aliyun",
		"i-stale-"+suffix, "203.0.113.10", serverURL, "cn-test", "cn-test-a", "auth-stale",
		runtime, createdAt.UTC().Format("2006-01-02 15:04:05"), createdAt.UTC().Format("2006-01-02 15:04:05"),
	)
	if err != nil {
		t.Fatalf("seed csc: %v", err)
	}
	b, err := insertCommentContainerBinding(companyID, taskID, commentID, ccbExecutionIndependent, "", workspaceID)
	if err != nil {
		t.Fatalf("insert binding: %v", err)
	}
	if err := markCommentContainerBindingStartingWithCSC(b.ID, b.MockContainerName, cscID); err != nil {
		t.Fatalf("mark starting: %v", err)
	}
	if bindingStatus != ccbStatusStarting {
		if _, err := db.Exec(`UPDATE cloud_comment_container_bindings SET status=? WHERE id=?`, bindingStatus, b.ID); err != nil {
			t.Fatalf("set binding status: %v", err)
		}
	}
	if _, err := db.Exec(
		`UPDATE cloud_comment_container_bindings SET created_at=?, updated_at=? WHERE id=?`,
		createdAt.UTC().Format("2006-01-02 15:04:05"), createdAt.UTC().Format("2006-01-02 15:04:05"), b.ID,
	); err != nil {
		t.Fatalf("backdate binding: %v", err)
	}
	return cscID, b.ID
}

func TestStaleStartingReachability(t *testing.T) {
	setupCloudTestDB(t)
	now := time.Now().UTC()

	t.Run("json_includes_csc_runtime", func(t *testing.T) {
		cscID, _ := seedStaleStartingReachabilityFixture(t, "json", now, machineRuntimeRunning, "", ccbStatusStarting)
		b, err := loadCommentContainerBinding("t-stale-json", "taskStalejson", "cmtStalejson")
		if err != nil || b == nil {
			t.Fatalf("load: %v", err)
		}
		out := commentContainerBindingToJSON(b)
		if out["last_runtime_status"] != machineRuntimeRunning {
			t.Fatalf("last_runtime_status=%v want Running csc=%s", out["last_runtime_status"], cscID)
		}
		if out["has_server_url"] != false {
			t.Fatalf("has_server_url=%v want false", out["has_server_url"])
		}
		if out["public_ip"] != "203.0.113.10" {
			t.Fatalf("public_ip=%v", out["public_ip"])
		}
	})

	t.Run("skips_fresh_and_healthy", func(t *testing.T) {
		seedStaleStartingReachabilityFixture(t, "fresh", now.Add(-5*time.Minute), machineRuntimeRunning, "", ccbStatusStarting)
		seedStaleStartingReachabilityFixture(t, "url", now.Add(-40*time.Minute), machineRuntimeRunning, "http://203.0.113.11:8080", ccbStatusStarting)
		seedStaleStartingReachabilityFixture(t, "run", now.Add(-40*time.Minute), machineRuntimeRunning, "", ccbStatusRunning)

		n, err := reconcileStaleStartingReachability(now, 20*time.Minute)
		if err != nil {
			t.Fatalf("reconcile: %v", err)
		}
		if n != 0 {
			t.Fatalf("failed=%d want 0", n)
		}
		fresh, _ := loadCommentContainerBinding("t-stale-fresh", "taskStalefresh", "cmtStalefresh")
		if fresh == nil || fresh.Status != ccbStatusStarting {
			t.Fatalf("fresh status=%v want starting", fresh)
		}
	})

	t.Run("fails_old_running_without_server_url", func(t *testing.T) {
		cscID, bindingID := seedStaleStartingReachabilityFixture(t, "old", now.Add(-25*time.Minute), machineRuntimeRunning, "", ccbStatusStarting)

		n, err := reconcileStaleStartingReachability(now, 20*time.Minute)
		if err != nil {
			t.Fatalf("reconcile: %v", err)
		}
		if n != 1 {
			t.Fatalf("failed=%d want 1", n)
		}
		b, err := loadCommentContainerBinding("t-stale-old", "taskStaleold", "cmtStaleold")
		if err != nil || b == nil {
			t.Fatalf("load binding: %v id=%s", err, bindingID)
		}
		if b.Status != ccbStatusFailed {
			t.Fatalf("binding status=%s want failed", b.Status)
		}
		cfg, err := loadCloudServerConfigByID(cscID)
		if err != nil || cfg == nil {
			t.Fatalf("load csc: %v", err)
		}
		if cfg.LastRuntimeStatus != machineRuntimeRunning {
			t.Fatalf("last_runtime_status=%q want Running (VM 仍在跑，不得标 Failed)", cfg.LastRuntimeStatus)
		}
		if trim(cfg.ErrorReason) == "" || !strings.Contains(cfg.ErrorReason, "可达地址") {
			t.Fatalf("error_reason=%q want reachability timeout", cfg.ErrorReason)
		}
		out := commentContainerBindingToJSON(b)
		if out["error_reason"] == nil || !strings.Contains(fmt.Sprint(out["error_reason"]), "可达地址") {
			t.Fatalf("binding json error_reason=%v", out["error_reason"])
		}
		if out["last_runtime_status"] != machineRuntimeRunning {
			t.Fatalf("json last_runtime_status=%v want Running", out["last_runtime_status"])
		}
	})
}

func TestCommentContainerBindingJSONIncludesCSCRuntime(t *testing.T) {
	setupCloudTestDB(t)
	now := time.Now().UTC()
	cscID, _ := seedStaleStartingReachabilityFixture(t, "jsonTop", now, machineRuntimeRunning, "", ccbStatusStarting)
	b, err := loadCommentContainerBinding("t-stale-jsonTop", "taskStalejsonTop", "cmtStalejsonTop")
	if err != nil || b == nil {
		t.Fatalf("load: %v", err)
	}
	out := commentContainerBindingToJSON(b)
	if out["last_runtime_status"] != machineRuntimeRunning {
		t.Fatalf("last_runtime_status=%v want Running csc=%s", out["last_runtime_status"], cscID)
	}
	if out["has_server_url"] != false {
		t.Fatalf("has_server_url=%v want false", out["has_server_url"])
	}
	if out["public_ip"] != "203.0.113.10" {
		t.Fatalf("public_ip=%v", out["public_ip"])
	}
}

func TestCommentContainerBindingsToJSONBatchLoadsCSC(t *testing.T) {
	setupCloudTestDB(t)
	now := time.Now().UTC()
	type seed struct {
		suffix    string
		runtime   string
		serverURL string
		publicIP  string
	}
	seeds := []seed{
		{suffix: "bat0", runtime: machineRuntimeRunning, serverURL: "", publicIP: "203.0.113.40"},
		{suffix: "bat1", runtime: machineRuntimeRunning, serverURL: "http://203.0.113.41:8080", publicIP: "203.0.113.41"},
		{suffix: "bat2", runtime: machineRuntimeStarting, serverURL: "", publicIP: "203.0.113.42"},
	}
	rows := make([]CommentContainerBinding, 0, len(seeds))
	for _, s := range seeds {
		cscID, _ := seedStaleStartingReachabilityFixture(t, s.suffix, now, s.runtime, s.serverURL, ccbStatusStarting)
		if _, err := db.Exec(
			`UPDATE cloud_server_configs SET public_ip=?, last_runtime_status=?, server_url=? WHERE id=?`,
			s.publicIP, s.runtime, s.serverURL, cscID,
		); err != nil {
			t.Fatalf("update csc %s: %v", cscID, err)
		}
		b, err := loadCommentContainerBinding("t-stale-"+s.suffix, "taskStale"+s.suffix, "cmtStale"+s.suffix)
		if err != nil || b == nil {
			t.Fatalf("load %s: %v", s.suffix, err)
		}
		rows = append(rows, *b)
	}

	var byID int
	var batchIDs []string
	prevByID := cloudServerConfigByIDLoadHook
	prevBatch := cloudServerConfigsByIDsLoadHook
	cloudServerConfigByIDLoadHook = func(string) { byID++ }
	cloudServerConfigsByIDsLoadHook = func(ids []string) { batchIDs = append([]string(nil), ids...) }
	t.Cleanup(func() {
		cloudServerConfigByIDLoadHook = prevByID
		cloudServerConfigsByIDsLoadHook = prevBatch
	})

	out := commentContainerBindingsToJSON(rows)
	if byID != 0 {
		t.Fatalf("loadCloudServerConfigByID calls=%d want 0 (N+1)", byID)
	}
	if len(batchIDs) != 3 {
		t.Fatalf("batch IN size=%d ids=%v want 3", len(batchIDs), batchIDs)
	}
	if len(out) != 3 {
		t.Fatalf("json rows=%d want 3", len(out))
	}
	for i, s := range seeds {
		item := out[i]
		if item["last_runtime_status"] != s.runtime {
			t.Fatalf("[%d] last_runtime_status=%v want %s", i, item["last_runtime_status"], s.runtime)
		}
		if item["public_ip"] != s.publicIP {
			t.Fatalf("[%d] public_ip=%v want %s", i, item["public_ip"], s.publicIP)
		}
		wantURL := s.serverURL != ""
		if item["has_server_url"] != wantURL {
			t.Fatalf("[%d] has_server_url=%v want %v", i, item["has_server_url"], wantURL)
		}
	}
}

func TestReconcileStaleStartingReachabilitySkipsWhenHTTPReady(t *testing.T) {
	setupCloudTestDB(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(srv.Close)
	hostPort := strings.TrimPrefix(srv.URL, "http://")

	prevProbe := probeStaleStartingReachability
	probeStaleStartingReachability = defaultProbeStaleStartingReachability
	t.Cleanup(func() { probeStaleStartingReachability = prevProbe })

	now := time.Now().UTC()
	cscID, bindingID := seedStaleStartingReachabilityFixture(t, "httpok", now.Add(-25*time.Minute), machineRuntimeRunning, "", ccbStatusStarting)
	if _, err := db.Exec(`UPDATE cloud_server_configs SET public_ip=? WHERE id=?`, hostPort, cscID); err != nil {
		t.Fatalf("set public_ip: %v", err)
	}

	n, err := reconcileStaleStartingReachability(now, 20*time.Minute)
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if n != 0 {
		t.Fatalf("failed=%d want 0 when HTTP ready", n)
	}
	b, err := loadCommentContainerBinding("t-stale-httpok", "taskStalehttpok", "cmtStalehttpok")
	if err != nil || b == nil {
		t.Fatalf("load binding: %v id=%s", err, bindingID)
	}
	if b.Status != ccbStatusStarting {
		t.Fatalf("binding status=%s want starting (HTTP ready must not fail)", b.Status)
	}
}

func TestReconcileStaleStartingReachabilityFailsWhenPortRefused(t *testing.T) {
	setupCloudTestDB(t)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	hostPort := ln.Addr().String()
	if err := ln.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}

	prevProbe := probeStaleStartingReachability
	probeStaleStartingReachability = defaultProbeStaleStartingReachability
	t.Cleanup(func() { probeStaleStartingReachability = prevProbe })

	now := time.Now().UTC()
	cscID, bindingID := seedStaleStartingReachabilityFixture(t, "refused", now.Add(-25*time.Minute), machineRuntimeRunning, "", ccbStatusStarting)
	if _, err := db.Exec(`UPDATE cloud_server_configs SET public_ip=? WHERE id=?`, hostPort, cscID); err != nil {
		t.Fatalf("set public_ip: %v", err)
	}

	n, err := reconcileStaleStartingReachability(now, 20*time.Minute)
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if n != 1 {
		t.Fatalf("failed=%d want 1 when port refused", n)
	}
	b, err := loadCommentContainerBinding("t-stale-refused", "taskStalerefused", "cmtStalerefused")
	if err != nil || b == nil {
		t.Fatalf("load binding: %v id=%s", err, bindingID)
	}
	if b.Status != ccbStatusFailed {
		t.Fatalf("binding status=%s want failed", b.Status)
	}
}
