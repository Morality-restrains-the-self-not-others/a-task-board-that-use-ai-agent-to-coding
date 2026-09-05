package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClearContainerReachabilityNative(t *testing.T) {
	setupCloudTestDB(t)
	now := "2026-07-08 00:00:00"
	_, err := db.Exec(`INSERT INTO cloud_server_configs
		(id,company_id,workspace_id,task_id,platform,instance_id,region,authorization_id,server_url,business_api_endpoint,container_vscode_url,created_at,updated_at)
		VALUES('csc1','t1','ws1','task-clear','aliyun','i-1','cn-hangzhou','auth1','http://srv','http://biz','http://vscode',?,?)`, now, now)
	if err != nil {
		t.Fatal(err)
	}

	ok, err := clearContainerReachabilityNative("t1", "ws1", "task-clear", "stop_vm", "i-1")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected cleared")
	}
	cfg, err := loadCloudServerConfig("t1", "ws1", "task-clear")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ServerURL != "" || cfg.InstanceID != "" {
		t.Fatalf("reachability not cleared: %+v", cfg)
	}
	if cfg.LastRuntimeStatus != machineRuntimeReleased {
		t.Fatalf("last_runtime_status=%q want Released after stop clear", cfg.LastRuntimeStatus)
	}
}

func TestInternalClearAfterStopEndpoint(t *testing.T) {
	setupCloudTestDB(t)
	now := "2026-07-08 00:00:00"
	_, err := db.Exec(`INSERT INTO cloud_server_configs
		(id,company_id,workspace_id,task_id,platform,instance_id,region,authorization_id,server_url,created_at,updated_at)
		VALUES('csc2','t1','ws1','task-api','aliyun','i-2','cn-hangzhou','auth1','http://srv',?,?)`, now, now)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/internal/cloud-server-config/clear-after-stop/",
		strings.NewReader(`{"tenant_id":"t1","workspace_id":"ws1","task_id":"task-api","stop_reason":"stop_vm"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleInternalCloudServerConfig(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestClearAfterStopScopesHistoryAndPreservesNewerBinding(t *testing.T) {
	setupCloudTestDB(t)
	now := "2026-07-18 00:00:00"
	_, err := db.Exec(`INSERT INTO cloud_server_configs
		(id,company_id,workspace_id,task_id,platform,instance_id,region,authorization_id,server_url,created_at,updated_at)
		VALUES('csc-new','t1','ws1','task-scope','aliyun','i-new','cn-hongkong','auth1','http://new',?,?)`, now, now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO cloud_server_config_histories
		(id,company_id,workspace_id,task_id,platform,platform_id,instance_id,region,zone_id,authorization_id,started_at,created_at)
		VALUES
		('h-old','t1','ws1','task-scope','aliyun',1,'i-old','cn-hongkong','cn-hongkong-d','auth1',?,?),
		('h-new','t1','ws1','task-scope','aliyun',1,'i-new','cn-hongkong','cn-hongkong-d','auth1',?,?)`,
		now, now, now, now)
	if err != nil {
		t.Fatal(err)
	}

	ok, err := clearContainerReachabilityNative("t1", "ws1", "task-scope", "superseded_by_new_start", "i-old")
	if err != nil || !ok {
		t.Fatalf("clear old: ok=%v err=%v", ok, err)
	}

	cfg, err := loadCloudServerConfig("t1", "ws1", "task-scope")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.InstanceID != "i-new" || cfg.ServerURL != "http://new" {
		t.Fatalf("newer CSC binding wiped: %+v", cfg)
	}
	oldH, err := loadCloudServerConfigHistoryByID("h-old")
	if err != nil {
		t.Fatal(err)
	}
	if oldH.StoppedAt == nil {
		t.Fatal("expected old history closed")
	}
	newH, err := loadCloudServerConfigHistoryByID("h-new")
	if err != nil {
		t.Fatal(err)
	}
	if newH.StoppedAt != nil {
		t.Fatalf("new history should stay open, stopped_at=%v", newH.StoppedAt)
	}
}

func insertReachabilityBinding(t *testing.T, taskID, commentID, status string) *CommentContainerBinding {
	t.Helper()
	b, err := insertCommentContainerBinding("t1", taskID, commentID, ccbExecutionIndependent, "")
	if err != nil {
		t.Fatalf("insert binding %s: %v", commentID, err)
	}
	if err := updateCommentContainerBindingStatus(b.ID, status); err != nil {
		t.Fatalf("set binding %s status=%s: %v", commentID, status, err)
	}
	b.Status = status
	return b
}

func TestClearContainerReachabilityReleasesMatchingBinding(t *testing.T) {
	setupCloudTestDB(t)
	now := "2026-08-14 00:00:00"
	_, err := db.Exec(`INSERT INTO cloud_server_configs
		(id,company_id,workspace_id,task_id,comment_id,platform,instance_id,region,authorization_id,server_url,created_at,updated_at)
		VALUES('csc-rel','t1','ws1','task-rel','c1','aliyun','i-rel','cn-hangzhou','auth1','http://srv',?,?)`, now, now)
	if err != nil {
		t.Fatal(err)
	}
	insertReachabilityBinding(t, "task-rel", "c1", ccbStatusStarting)

	if !commentBindingIsProvisioning("t1", "task-rel") {
		t.Fatal("starting binding should count as provisioning before stop")
	}

	cfg, err := loadCloudServerConfigForComment("t1", "ws1", "task-rel", "c1")
	if err != nil {
		t.Fatal(err)
	}
	ok, err := clearContainerReachabilityOnConfig(cfg, "t1", "ws1", "task-rel")
	if err != nil || !ok {
		t.Fatalf("clear: ok=%v err=%v", ok, err)
	}

	released, err := loadCommentContainerBinding("t1", "task-rel", "c1")
	if err != nil {
		t.Fatal(err)
	}
	if released.Status != ccbStatusReleased {
		t.Fatalf("binding status=%s want %s", released.Status, ccbStatusReleased)
	}
	if commentBindingIsProvisioning("t1", "task-rel") {
		t.Fatal("released binding must not count as provisioning")
	}

	logs, err := listCommentContainerBindingLogs("t1", "task-rel")
	if err != nil {
		t.Fatal(err)
	}
	foundReleased := false
	for _, l := range logs {
		if l.CommentID == "c1" && l.Stage == ccbStatusReleased {
			foundReleased = true
			if l.Message != ccbStageMessage(ccbStatusReleased) {
				t.Fatalf("released log message=%q", l.Message)
			}
		}
	}
	if !foundReleased {
		t.Fatal("expected released stage log for stopped comment")
	}
}

func TestClearContainerReachabilityDoesNotReleaseParallelBinding(t *testing.T) {
	setupCloudTestDB(t)
	now := "2026-08-14 00:00:00"
	_, err := db.Exec(`INSERT INTO cloud_server_configs
		(id,company_id,workspace_id,task_id,comment_id,platform,instance_id,region,authorization_id,server_url,created_at,updated_at)
		VALUES('csc-rel-p','t1','ws1','task-rel-p','c1','aliyun','i-rel-p','cn-hangzhou','auth1','http://srv',?,?)`, now, now)
	if err != nil {
		t.Fatal(err)
	}
	insertReachabilityBinding(t, "task-rel-p", "c1", ccbStatusRunning)
	insertReachabilityBinding(t, "task-rel-p", "c2", ccbStatusStarting)

	cfg, err := loadCloudServerConfigForComment("t1", "ws1", "task-rel-p", "c1")
	if err != nil {
		t.Fatal(err)
	}
	ok, err := clearContainerReachabilityOnConfig(cfg, "t1", "ws1", "task-rel-p")
	if err != nil || !ok {
		t.Fatalf("clear: ok=%v err=%v", ok, err)
	}

	stopped, err := loadCommentContainerBinding("t1", "task-rel-p", "c1")
	if err != nil {
		t.Fatal(err)
	}
	if stopped.Status != ccbStatusReleased {
		t.Fatalf("stopped comment binding status=%s want %s", stopped.Status, ccbStatusReleased)
	}
	parallel, err := loadCommentContainerBinding("t1", "task-rel-p", "c2")
	if err != nil {
		t.Fatal(err)
	}
	if parallel.Status != ccbStatusStarting {
		t.Fatalf("parallel binding status=%s want %s (must not be released)", parallel.Status, ccbStatusStarting)
	}
	if !commentBindingIsProvisioning("t1", "task-rel-p") {
		t.Fatal("parallel starting binding should still count as provisioning")
	}
}
