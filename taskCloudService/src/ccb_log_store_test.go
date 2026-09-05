package main

import (
	"testing"
)

func TestCCBLogShardsIsolateWorkspaces(t *testing.T) {
	setupCloudTestDB(t)
	for _, row := range []struct{ id, ws, task, url string }{
		{"cfg-iso-a", "ws1", "taskIsoA", "http://mock/a/"},
		{"cfg-iso-b", "ws2", "taskIsoB", "http://mock/b/"},
	} {
		if _, err := db.Exec(
			`INSERT INTO cloud_server_configs(id, company_id, workspace_id, task_id, comment_id, platform, server_url, region, zone_id, authorization_id)
			 VALUES (?,?,?,?,?,?,?,?,?,?)`,
			row.id, "t1", row.ws, row.task, "", "aliyun", row.url, "cn-test", "cn-test-a", "test",
		); err != nil {
			t.Fatalf("seed %s: %v", row.id, err)
		}
	}

	ba, err := insertCommentContainerBinding("t1", "taskIsoA", "cA", ccbExecutionIndependent, "")
	if err != nil {
		t.Fatal(err)
	}
	bb, err := insertCommentContainerBinding("t1", "taskIsoB", "cB", ccbExecutionIndependent, "")
	if err != nil {
		t.Fatal(err)
	}
	tableA, err := ccbLogTable("ws1")
	if err != nil {
		t.Fatal(err)
	}
	tableB, err := ccbLogTable("ws2")
	if err != nil {
		t.Fatal(err)
	}
	if tableA == tableB {
		t.Logf("ws1 and ws2 collided on %s (allowed); isolation still via workspace_id column", tableA)
	}

	rowsA, err := listCommentContainerBindingLogsIn("ws1", "t1", "taskIsoA")
	if err != nil {
		t.Fatal(err)
	}
	rowsB, err := listCommentContainerBindingLogsIn("ws2", "t1", "taskIsoB")
	if err != nil {
		t.Fatal(err)
	}
	if len(rowsA) == 0 {
		t.Fatalf("ws1 missing pending log; binding=%s", ba.ID)
	}
	if len(rowsB) == 0 {
		t.Fatalf("ws2 missing pending log; binding=%s", bb.ID)
	}
	for _, l := range rowsA {
		if l.WorkspaceID != "ws1" || l.CommentID == "cB" {
			t.Fatalf("ws1 list leaked foreign log: %+v", l)
		}
	}
	for _, l := range rowsB {
		if l.WorkspaceID != "ws2" || l.CommentID == "cA" {
			t.Fatalf("ws2 list leaked foreign log: %+v", l)
		}
	}

	var cntA, cntB, cntLegacy int
	if err := db.QueryRow("SELECT COUNT(*) FROM " + tableA + " WHERE workspace_id='ws1' AND task_id='taskIsoA'").Scan(&cntA); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow("SELECT COUNT(*) FROM " + tableB + " WHERE workspace_id='ws2' AND task_id='taskIsoB'").Scan(&cntB); err != nil {
		t.Fatal(err)
	}
	// 033_drop_legacy_ccb_logs.sql 已将遗留表 RENAME 为 deprecated 名（保留数据、禁误查旧表名）。
	if err := db.QueryRow("SELECT COUNT(*) FROM cloud_comment_container_binding_logs_deprecated_20260821 WHERE task_id IN ('taskIsoA','taskIsoB')").Scan(&cntLegacy); err != nil {
		t.Fatal(err)
	}
	if cntA != 0 || cntB != 0 {
		t.Fatalf("archived logs must leave shards empty, got A=%d B=%d tableA=%s tableB=%s", cntA, cntB, tableA, tableB)
	}
	if cntLegacy != 0 {
		t.Fatalf("legacy table must not receive new writes, got %d", cntLegacy)
	}
}

func TestCCBLogListRequiresWorkspace(t *testing.T) {
	setupCloudTestDB(t)
	_, err := listCommentContainerBindingLogsIn("", "t1", "task-no-csc")
	if err == nil {
		t.Fatal("list without workspace or CSC must fail")
	}
}

func TestCCBLogPersistsWithWorkspaceColumnNoTaskCSC(t *testing.T) {
	setupCloudTestDB(t)
	// 无任务级 CSC（cloud_server_configs 无该 task 行）：workspace_id 从 binding 列持久化，
	// server_failed 仍能落日志分片（OPT-20260821-003）。
	b, err := insertCommentContainerBinding("t1", "taskNoCsc", "cFail", ccbExecutionIndependent, "", "ws9")
	if err != nil {
		t.Fatal(err)
	}
	if bindingLogWorkspaceID(b) != "ws9" {
		t.Fatalf("bindingLogWorkspaceID=%q want ws9 (from column)", bindingLogWorkspaceID(b))
	}
	loaded, err := loadCommentContainerBinding("t1", "taskNoCsc", "cFail")
	if err != nil || loaded == nil {
		t.Fatalf("load: %v loaded=%v", err, loaded)
	}
	if loaded.WorkspaceID != "ws9" {
		t.Fatalf("loaded.WorkspaceID=%q want ws9", loaded.WorkspaceID)
	}

	if err := appendCommentContainerBindingLogMessage(loaded, ccbStageServerFailed, "启动失败：镜像市场 502"); err != nil {
		t.Fatalf("stage log err=%v", err)
	}
	rows, err := listCommentContainerBindingLogsIn("ws9", "t1", "taskNoCsc")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, l := range rows {
		if l.Stage == ccbStageServerFailed && l.CommentID == "cFail" && l.WorkspaceID == "ws9" {
			found = true
		}
	}
	if !found {
		t.Fatalf("server_failed log missing in ws9 shard: %+v", rows)
	}
}
