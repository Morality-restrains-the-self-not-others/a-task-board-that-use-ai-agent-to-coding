package main

import "testing"

func TestMigrateCloudServerConfigMultiCommentIndexes(t *testing.T) {
	setupCloudTestDB(t)
	var n int
	if err := db.QueryRow(
		`SELECT COUNT(DISTINCT INDEX_NAME) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='cloud_server_configs' AND INDEX_NAME='idx_csc_workspace_task'`,
	).Scan(&n); err != nil || n == 0 {
		t.Fatalf("idx_csc_workspace_task missing n=%d err=%v", n, err)
	}
	if err := db.QueryRow(
		`SELECT COUNT(DISTINCT INDEX_NAME) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='cloud_server_configs' AND INDEX_NAME='idx_csc_workspace_task_comment'`,
	).Scan(&n); err != nil || n == 0 {
		t.Fatalf("idx_csc_workspace_task_comment missing n=%d err=%v", n, err)
	}
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='cloud_server_configs' AND INDEX_NAME='idx_csc_company_task'`,
	).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("legacy idx_csc_company_task should be dropped, n=%d", n)
	}
}

func TestEnsureCommentCSCRejectsWhenTemplateIllegal(t *testing.T) {
	setupCloudTestDB(t)
	_, err := ensureCommentCloudServerConfig("t-none", "ws-none", "task-none", "cmt-none")
	if err == nil {
		t.Fatal("ensure without real template must fail")
	}
	_, loadErr := loadCloudServerConfigForComment("t-none", "ws-none", "task-none", "cmt-none")
	if loadErr == nil {
		t.Fatal("must not persist mock comment CSC")
	}
}

func TestEnsureCommentCSCClonesLegalTemplate(t *testing.T) {
	setupCloudTestDB(t)
	if err := upsertCloudServerConfig(CloudServerConfig{
		ID: "cfg-tpl", CompanyID: "t-new", WorkspaceID: "ws-new", TaskID: "task-new",
		Platform: "aliyun", Region: "cn-qingdao", ZoneID: "cn-qingdao-b", AuthorizationID: "cpa-1",
	}); err != nil {
		t.Fatal(err)
	}
	got, err := ensureCommentCloudServerConfig("t-new", "ws-new", "task-new", "cmt-new")
	if err != nil {
		t.Fatalf("ensure: %v", err)
	}
	if got.Platform != "aliyun" || got.Region != "cn-qingdao" || got.AuthorizationID != "cpa-1" {
		t.Fatalf("cloned meta platform=%s region=%s auth=%s", got.Platform, got.Region, got.AuthorizationID)
	}
}

func TestEnsureCommentCSCPreservesUserOverride(t *testing.T) {
	setupCloudTestDB(t)
	now := "2026-08-14 00:00:00"
	_, err := db.Exec(`INSERT INTO cloud_server_configs
		(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, region, zone_id, authorization_id, created_at, updated_at)
		VALUES ('cfg-tpl-ov', 't-ov', 'ws-ov', 'task-ov', '', 'aliyun', '', 'cn-qingdao', 'cn-qingdao-b', 'cpa-tpl', ?, ?)`, now, now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO cloud_server_configs
		(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, region, zone_id, authorization_id, created_at, updated_at)
		VALUES ('cfg-cmt-ov', 't-ov', 'ws-ov', 'task-ov', 'cmt-ov', 'aliyun', 'i-user', 'cn-hangzhou', 'cn-hangzhou-h', 'cpa-user', ?, ?)`, now, now)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ensureCommentCloudServerConfig("t-ov", "ws-ov", "task-ov", "cmt-ov")
	if err != nil {
		t.Fatal(err)
	}
	if got.Region != "cn-hangzhou" || got.AuthorizationID != "cpa-user" || got.InstanceID != "i-user" {
		t.Fatalf("must keep comment override region=%s auth=%s inst=%s", got.Region, got.AuthorizationID, got.InstanceID)
	}
}

// 存量脏行：评论 mock + 空 region/auth，模板合法 → 只补非法字段。
func TestEnsureCommentCSCUpgradesMockFromTaskBase(t *testing.T) {
	setupCloudTestDB(t)
	companyID := "t-upgrade"
	workspaceID := "ws-upgrade"
	taskID := "task-upgrade"
	commentID := "cmt-upgrade"
	now := "2026-08-14 00:00:00"
	_, err := db.Exec(`INSERT INTO cloud_server_configs
		(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, region, zone_id, authorization_id, created_at, updated_at)
		VALUES ('cfg-cmt-dirty', ?, ?, ?, ?, 'mock', 'i-dirty', '', '', '', ?, ?)`,
		companyID, workspaceID, taskID, commentID, now, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := upsertCloudServerConfig(CloudServerConfig{
		ID: "cfg-tpl-up", CompanyID: companyID, WorkspaceID: workspaceID, TaskID: taskID,
		Platform: "aliyun", Region: "cn-qingdao", ZoneID: "cn-qingdao-b", AuthorizationID: "cpa-1",
	}); err != nil {
		t.Fatal(err)
	}

	again, err := ensureCommentCloudServerConfig(companyID, workspaceID, taskID, commentID)
	if err != nil {
		t.Fatalf("ensure fill: %v", err)
	}
	if again.Platform != "aliyun" || again.Region != "cn-qingdao" || again.AuthorizationID != "cpa-1" {
		t.Fatalf("filled meta platform=%s region=%s auth=%s", again.Platform, again.Region, again.AuthorizationID)
	}
	if again.InstanceID != "i-dirty" {
		t.Fatalf("must keep instance, got %q", again.InstanceID)
	}
}

// 评论 CSC 从 mock 升级为真实云平台后，failed binding 须回到 starting，否则后续 start-vm 成功也无法被调度闭环。
func TestEnsureCommentCSCUpgradeRecoversFailedBinding(t *testing.T) {
	setupCloudTestDB(t)
	companyID := "t-recov"
	workspaceID := "ws-recov"
	taskID := "task-recov"
	commentID := "cmt-recov"

	now := "2026-08-14 00:00:00"
	_, err := db.Exec(`INSERT INTO cloud_server_configs
		(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, region, zone_id, authorization_id, created_at, updated_at)
		VALUES ('cfg-cmt-recov', ?, ?, ?, ?, 'mock', '', '', '', '', ?, ?)`,
		companyID, workspaceID, taskID, commentID, now, now)
	if err != nil {
		t.Fatal(err)
	}
	b, err := insertCommentContainerBinding(companyID, taskID, commentID, ccbExecutionIndependent, "")
	if err != nil {
		t.Fatalf("insert binding: %v", err)
	}
	if err := markCommentContainerBindingFailed(b.ID, b.MockContainerName, "cfg-cmt-recov"); err != nil {
		t.Fatalf("mark failed: %v", err)
	}

	base := CloudServerConfig{
		ID:              genID("csc"),
		CompanyID:       companyID,
		WorkspaceID:     workspaceID,
		TaskID:          taskID,
		CommentID:       "",
		Platform:        "aliyun",
		Region:          "cn-qingdao",
		AuthorizationID: "cpa-recov",
	}
	if err := upsertCloudServerConfig(base); err != nil {
		t.Fatalf("upsert base: %v", err)
	}

	if _, err := ensureCommentCloudServerConfig(companyID, workspaceID, taskID, commentID); err != nil {
		t.Fatalf("ensure upgrade: %v", err)
	}

	got, err := loadCommentContainerBinding(companyID, taskID, commentID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != ccbStatusStarting {
		t.Fatalf("binding status=%s want %s after CSC platform upgrade", got.Status, ccbStatusStarting)
	}
}
