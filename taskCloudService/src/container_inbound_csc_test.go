package main

import "testing"

func seedInboundCSC(t *testing.T, id, taskID, commentID, instanceID, publicIP string) {
	t.Helper()
	_, err := db.Exec(
		`INSERT INTO cloud_server_configs(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, public_ip, server_url, region, zone_id, authorization_id, last_runtime_status)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		id, "t1", "ws1", taskID, commentID, "aliyun", instanceID, publicIP,
		"http://"+publicIP+":8765/", "cn-qingdao", "cn-qingdao-b", "auth-1", "Running",
	)
	if err != nil {
		t.Fatalf("seed inbound CSC %s: %v", id, err)
	}
}

func TestResolveInboundCommentCSC_ExplicitCommentID(t *testing.T) {
	setupCloudTestDB(t)
	seedInboundCSC(t, "cfg-a", "task-inb", "cmt-a", "i-a", "203.0.113.10")
	scope := &validatedContainerScope{CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task-inb"}
	cfg, msg := resolveInboundCommentCSC(scope, map[string]any{"comment_id": "cmt-a"}, "")
	if msg != "" {
		t.Fatalf("msg=%q", msg)
	}
	if cfg == nil || cfg.CommentID != "cmt-a" {
		t.Fatalf("cfg=%+v", cfg)
	}
}

func TestResolveInboundCommentCSC_ContainerName(t *testing.T) {
	setupCloudTestDB(t)
	seedInboundCSC(t, "cfg-a", "task_inb", "cmt_inb1", "i-a", "203.0.113.11")
	scope := &validatedContainerScope{CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task_inb"}
	cfg, msg := resolveInboundCommentCSC(scope, map[string]any{
		"container_name": "task_inb_cmt_inb1",
	}, "")
	if msg != "" {
		t.Fatalf("msg=%q", msg)
	}
	if cfg == nil || cfg.CommentID != "cmt_inb1" {
		t.Fatalf("cfg=%+v", cfg)
	}
}

func TestResolveInboundCommentCSC_PublicIPWhenCommentMissing(t *testing.T) {
	setupCloudTestDB(t)
	seedInboundCSC(t, "cfg-a", "task-inb", "cmt-a", "i-a", "203.0.113.10")
	seedInboundCSC(t, "cfg-b", "task-inb", "cmt-b", "i-b", "203.0.113.20")
	scope := &validatedContainerScope{CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task-inb"}
	cfg, msg := resolveInboundCommentCSC(scope, map[string]any{
		"public_ip": "203.0.113.20",
	}, "")
	if msg != "" {
		t.Fatalf("msg=%q", msg)
	}
	if cfg == nil || cfg.CommentID != "cmt-b" {
		t.Fatalf("want cmt-b got %+v", cfg)
	}
}

func TestResolveInboundCommentCSC_ServerURLHostWhenCommentMissing(t *testing.T) {
	setupCloudTestDB(t)
	seedInboundCSC(t, "cfg-a", "task-inb", "cmt-a", "i-a", "203.0.113.10")
	scope := &validatedContainerScope{CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task-inb"}
	cfg, msg := resolveInboundCommentCSC(scope, map[string]any{
		"business_api_endpoint": "http://203.0.113.10:8765/api",
	}, "")
	if msg != "" {
		t.Fatalf("msg=%q", msg)
	}
	if cfg == nil || cfg.CommentID != "cmt-a" {
		t.Fatalf("cfg=%+v", cfg)
	}
}

func TestResolveInboundCommentCSC_ClientIPWhenCommentMissing(t *testing.T) {
	setupCloudTestDB(t)
	seedInboundCSC(t, "cfg-a", "task-inb", "cmt-a", "i-a", "203.0.113.10")
	seedInboundCSC(t, "cfg-b", "task-inb", "cmt-b", "i-b", "203.0.113.20")
	scope := &validatedContainerScope{CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task-inb"}
	cfg, msg := resolveInboundCommentCSC(scope, map[string]any{}, "203.0.113.10")
	if msg != "" {
		t.Fatalf("msg=%q", msg)
	}
	if cfg == nil || cfg.CommentID != "cmt-a" {
		t.Fatalf("cfg=%+v", cfg)
	}
}

func TestResolveInboundCommentCSC_UniqueCommentInstance(t *testing.T) {
	setupCloudTestDB(t)
	seedInboundCSC(t, "cfg-tpl", "task-inb", "", "i-old", "198.51.100.1")
	seedInboundCSC(t, "cfg-a", "task-inb", "cmt-a", "i-a", "203.0.113.10")
	scope := &validatedContainerScope{CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task-inb"}
	cfg, msg := resolveInboundCommentCSC(scope, map[string]any{}, "")
	if msg != "" {
		t.Fatalf("msg=%q", msg)
	}
	if cfg == nil || cfg.CommentID != "cmt-a" {
		t.Fatalf("cfg=%+v", cfg)
	}
}

func TestResolveInboundCommentCSC_TwoCommentsStillRequireHint(t *testing.T) {
	setupCloudTestDB(t)
	seedInboundCSC(t, "cfg-a", "task-inb", "cmt-a", "i-a", "203.0.113.10")
	seedInboundCSC(t, "cfg-b", "task-inb", "cmt-b", "i-b", "203.0.113.20")
	scope := &validatedContainerScope{CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task-inb"}
	_, msg := resolveInboundCommentCSC(scope, map[string]any{}, "")
	if msg != "缺少评论ID" {
		t.Fatalf("msg=%q want 缺少评论ID", msg)
	}
}
