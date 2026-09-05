package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func seedBindingForUnreachableTest(t *testing.T, id, companyID, taskID, commentID, status string) {
	t.Helper()
	_, err := db.Exec(
		`INSERT INTO cloud_comment_container_bindings(
			id, company_id, task_id, comment_id, execution_mode, depends_on_comment_id,
			status, mock_container_name, csc_id, created_at, updated_at
		) VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		id, companyID, taskID, commentID, "independent", "",
		status, "mock-"+commentID, "cfg-"+commentID,
		"2026-08-19 00:00:00", "2026-08-19 00:00:00",
	)
	if err != nil {
		t.Fatalf("seed binding: %v", err)
	}
}

// TestInternalContainerUnreachable_ClearsServerURLAndDemotesRunning — OPT-20260818-008。
// 网关 dial 容器 server_url 收到 connection refused 时，cloud 须清推测性地址并把
// running binding 降回 starting。
func TestInternalContainerUnreachable_ClearsServerURLAndDemotesRunning(t *testing.T) {
	setupCloudTestDB(t)
	_, err := db.Exec(
		`INSERT INTO cloud_server_configs(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, server_url, business_api_endpoint, region, zone_id, authorization_id)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		"cfg-u1", "t1", "ws1", "task-u1", "cmt-u1", "aliyun", "i-u1",
		"http://1.2.3.4:8080", "http://1.2.3.4:8080/api", "cn-test", "cn-test-a", "test",
	)
	if err != nil {
		t.Fatalf("seed csc: %v", err)
	}
	seedBindingForUnreachableTest(t, "b-u1", "t1", "task-u1", "cmt-u1", ccbStatusRunning)

	req := httptest.NewRequest(http.MethodPost,
		"/api/internal/cloud-server-config/container-unreachable/?tenant_id=t1&workspace_id=ws1&task_id=task-u1&comment_id=cmt-u1", nil)
	rec := httptest.NewRecorder()
	handleInternalCloudServerConfig(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["cleared"] != true {
		t.Fatalf("cleared=%v want true", body["cleared"])
	}
	if body["demoted"] != true {
		t.Fatalf("demoted=%v want true", body["demoted"])
	}

	cfg, err := loadCloudServerConfigByID("cfg-u1")
	if err != nil {
		t.Fatalf("load csc: %v", err)
	}
	if trim(cfg.ServerURL) != "" || trim(cfg.BusinessAPIEndpoint) != "" {
		t.Fatalf("server_url=%q business_api_endpoint=%q want cleared", cfg.ServerURL, cfg.BusinessAPIEndpoint)
	}

	b, err := loadCommentContainerBinding("t1", "task-u1", "cmt-u1")
	if err != nil {
		t.Fatalf("load binding: %v", err)
	}
	if b.Status != ccbStatusStarting {
		t.Fatalf("binding status=%s want starting", b.Status)
	}
}

// TestInternalContainerUnreachable_StartingBindingStaysStarting — 非 running binding 不降级。
func TestInternalContainerUnreachable_StartingBindingStaysStarting(t *testing.T) {
	setupCloudTestDB(t)
	_, err := db.Exec(
		`INSERT INTO cloud_server_configs(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, server_url, business_api_endpoint, region, zone_id, authorization_id)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		"cfg-u2", "t1", "ws1", "task-u2", "cmt-u2", "aliyun", "i-u2",
		"http://1.2.3.4:8080", "http://1.2.3.4:8080/api", "cn-test", "cn-test-a", "test",
	)
	if err != nil {
		t.Fatalf("seed csc: %v", err)
	}
	seedBindingForUnreachableTest(t, "b-u2", "t1", "task-u2", "cmt-u2", ccbStatusStarting)

	req := httptest.NewRequest(http.MethodPost,
		"/api/internal/cloud-server-config/container-unreachable/?tenant_id=t1&workspace_id=ws1&task_id=task-u2&comment_id=cmt-u2", nil)
	rec := httptest.NewRecorder()
	handleInternalCloudServerConfig(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["cleared"] != true {
		t.Fatalf("cleared=%v want true", body["cleared"])
	}
	if body["demoted"] != false {
		t.Fatalf("demoted=%v want false (already starting)", body["demoted"])
	}
	b, err := loadCommentContainerBinding("t1", "task-u2", "cmt-u2")
	if err != nil {
		t.Fatalf("load binding: %v", err)
	}
	if b.Status != ccbStatusStarting {
		t.Fatalf("binding status=%s want starting", b.Status)
	}
}

// TestInternalContainerUnreachable_IdempotentNoCSC — 无 CSC/无地址时幂等成功。
func TestInternalContainerUnreachable_IdempotentNoCSC(t *testing.T) {
	setupCloudTestDB(t)
	req := httptest.NewRequest(http.MethodPost,
		"/api/internal/cloud-server-config/container-unreachable/?tenant_id=t1&task_id=task-missing", nil)
	rec := httptest.NewRecorder()
	handleInternalCloudServerConfig(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}
