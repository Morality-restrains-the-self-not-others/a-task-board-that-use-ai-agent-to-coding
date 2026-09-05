package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRecycleIdleMachinesClearsTimedOutIdleConfig(t *testing.T) {
	setupCloudTestDB(t)

	idleSince := time.Now().UTC().Add(-2 * time.Hour).Format("2006-01-02 15:04:05")
	_, err := db.Exec(`INSERT INTO cloud_workspace_machine_policies
		(company_id, workspace_id, idle_recycle_minutes, enabled_authorization_ids)
		VALUES ('t1','ws1',30,'[]')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO cloud_server_configs
		(id,company_id,workspace_id,task_id,platform,instance_id,public_ip,region,authorization_id,idle_since,last_runtime_status,created_at,updated_at)
		VALUES ('cfg-stale','t1','ws1','task-stale','mock','mock-stale','1.2.3.4','cn-test','auth1',?,'Running',NOW(),NOW())`,
		idleSince)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO cloud_server_config_histories
		(id,company_id,workspace_id,task_id,platform,instance_id,started_at,created_at)
		VALUES ('hist-stale','t1','ws1','task-stale','mock','mock-stale',NOW(),NOW())`)
	if err != nil {
		t.Fatal(err)
	}

	recycled, err := recycleIdleMachines(time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if recycled != 1 {
		t.Fatalf("recycled=%d want 1", recycled)
	}

	cfg, err := loadCloudServerConfig("t1", "ws1", "task-stale")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.InstanceID != "" || cfg.PublicIP != "" {
		t.Fatalf("config not cleared: instance_id=%q public_ip=%q", cfg.InstanceID, cfg.PublicIP)
	}

	open, err := loadOpenCloudServerConfigHistory("t1", "ws1", "task-stale")
	if err != nil {
		t.Fatal(err)
	}
	if open != nil {
		t.Fatal("open history should be closed after recycle")
	}
}

func TestInternalRecycleIdleMachinesEndpoint(t *testing.T) {
	setupCloudTestDB(t)

	idleSince := time.Now().UTC().Add(-90 * time.Minute).Format("2006-01-02 15:04:05")
	_, err := db.Exec(`INSERT INTO cloud_workspace_machine_policies
		(company_id, workspace_id, idle_recycle_minutes, enabled_authorization_ids)
		VALUES ('t1','ws1',60,'[]')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO cloud_server_configs
		(id,company_id,workspace_id,task_id,platform,instance_id,region,authorization_id,idle_since,last_runtime_status,created_at,updated_at)
		VALUES ('cfg-api','t1','ws1','task-api','mock','mock-api','cn-test','auth1',?,'Running',NOW(),NOW())`,
		idleSince)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost,
		"/api/internal/cloud/compute/recycle-idle-machines/", nil)
	rec := httptest.NewRecorder()
	handleInternalRecycleIdleMachines(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"recycled":1`) {
		t.Fatalf("expected recycled=1: %s", rec.Body.String())
	}
}

func TestRecycleIdleMachinesClearsAllCoResidentTasks(t *testing.T) {
	setupCloudTestDB(t)

	idleSince := time.Now().UTC().Add(-2 * time.Hour).Format("2006-01-02 15:04:05")
	_, err := db.Exec(`INSERT INTO cloud_workspace_machine_policies
		(company_id, workspace_id, idle_recycle_minutes, enabled_authorization_ids)
		VALUES ('t1','ws1',30,'[]')`)
	if err != nil {
		t.Fatal(err)
	}
	for _, spec := range []struct{ id, task string }{
		{"cfg-co-a", "task-co-a"},
		{"cfg-co-b", "task-co-b"},
	} {
		_, err = db.Exec(`INSERT INTO cloud_server_configs
			(id,company_id,workspace_id,task_id,platform,instance_id,public_ip,region,authorization_id,idle_since,last_runtime_status,created_at,updated_at)
			VALUES (?,?,?,?,?,?,?,?,?,?,'Running',NOW(),NOW())`,
			spec.id, "t1", "ws1", spec.task, "mock", "mock-shared", "1.2.3.4", "cn-test", "auth1", idleSince)
		if err != nil {
			t.Fatal(err)
		}
		_, err = db.Exec(`INSERT INTO cloud_server_config_histories
			(id,company_id,workspace_id,task_id,platform,instance_id,started_at,created_at)
			VALUES (?,?,?,?,?,?,NOW(),NOW())`,
			"hist-"+spec.task, "t1", "ws1", spec.task, "mock", "mock-shared")
		if err != nil {
			t.Fatal(err)
		}
	}

	recycled, err := recycleIdleMachines(time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if recycled != 2 {
		t.Fatalf("recycled=%d want 2 (both co-resident tasks)", recycled)
	}

	for _, taskID := range []string{"task-co-a", "task-co-b"} {
		cfg, err := loadCloudServerConfig("t1", "ws1", taskID)
		if err != nil {
			t.Fatal(err)
		}
		if cfg.InstanceID != "" || cfg.PublicIP != "" {
			t.Fatalf("task %s not cleared: instance_id=%q public_ip=%q", taskID, cfg.InstanceID, cfg.PublicIP)
		}
	}
}

// OPT-20260823-027：idle_recycle 只清库不调云 API，但必须在评论启动日志写入
// 带「触发：工作区机器空闲回收」的调度行，避免用户看到清理却不知道来源。
func TestRecycleIdleMachinesWritesTriggerLineToBindingLog(t *testing.T) {
	setupCloudTestDB(t)

	idleSince := time.Now().UTC().Add(-2 * time.Hour).Format("2006-01-02 15:04:05")
	if _, err := db.Exec(`INSERT INTO cloud_workspace_machine_policies
		(company_id, workspace_id, idle_recycle_minutes, enabled_authorization_ids)
		VALUES ('t1','ws1',30,'[]')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO cloud_server_configs
		(id,company_id,workspace_id,task_id,comment_id,platform,instance_id,region,authorization_id,idle_since,last_runtime_status,created_at,updated_at)
		VALUES ('cfg-idle-log','t1','ws1','task-idle-log','cmt-idle-log','mock','mock-idle-log','cn-test','auth1',?,'Running',NOW(),NOW())`,
		idleSince); err != nil {
		t.Fatal(err)
	}
	if _, err := insertCommentContainerBinding("t1", "task-idle-log", "cmt-idle-log", ccbExecutionIndependent, ""); err != nil {
		t.Fatal(err)
	}

	recycled, err := recycleIdleMachines(time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if recycled != 1 {
		t.Fatalf("recycled=%d want 1", recycled)
	}

	rows, err := listCommentContainerBindingLogsIn("ws1", "t1", "task-idle-log")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, l := range rows {
		if l.CommentID == "cmt-idle-log" && strings.Contains(l.Message, "触发：工作区机器空闲回收") {
			found = true
		}
	}
	if !found {
		t.Fatalf("idle_recycle trigger line missing in binding log: %+v", rows)
	}
}

func TestRecycleIdleMachinesSkipsInstanceWithBusyContainer(t *testing.T) {
	setupCloudTestDB(t)

	idleSince := time.Now().UTC().Add(-2 * time.Hour).Format("2006-01-02 15:04:05")
	_, err := db.Exec(`INSERT INTO cloud_workspace_machine_policies
		(company_id, workspace_id, idle_recycle_minutes, enabled_authorization_ids)
		VALUES ('t1','ws1',30,'[]')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO cloud_server_configs
		(id,company_id,workspace_id,task_id,platform,instance_id,public_ip,region,authorization_id,server_url,idle_since,last_runtime_status,created_at,updated_at)
		VALUES ('cfg-busy','t1','ws1','task-busy','mock','mock-mixed','1.2.3.4','cn-test','auth1','http://127.0.0.1:8080/',?,'Running',NOW(),NOW())`,
		idleSince)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO cloud_server_configs
		(id,company_id,workspace_id,task_id,platform,instance_id,public_ip,region,authorization_id,idle_since,last_runtime_status,created_at,updated_at)
		VALUES ('cfg-idle','t1','ws1','task-idle','mock','mock-mixed','1.2.3.4','cn-test','auth1',?,'Running',NOW(),NOW())`,
		idleSince)
	if err != nil {
		t.Fatal(err)
	}

	recycled, err := recycleIdleMachines(time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if recycled != 0 {
		t.Fatalf("recycled=%d want 0 (busy container blocks instance recycle)", recycled)
	}

	busyCfg, err := loadCloudServerConfig("t1", "ws1", "task-busy")
	if err != nil {
		t.Fatal(err)
	}
	if busyCfg.InstanceID != "mock-mixed" {
		t.Fatalf("busy instance_id=%q want mock-mixed", busyCfg.InstanceID)
	}
	idleCfg, err := loadCloudServerConfig("t1", "ws1", "task-idle")
	if err != nil {
		t.Fatal(err)
	}
	if idleCfg.InstanceID != "mock-mixed" {
		t.Fatalf("idle instance_id=%q want mock-mixed", idleCfg.InstanceID)
	}
}

