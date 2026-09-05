package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	dbload "dbload"
)

func TestMain(m *testing.M) {
	// Avoid real Aliyun calls from orphan reconcile during package tests.
	describeInstancesByName = func(accessKey, secretKey, regionID, instanceName string) ([]map[string]interface{}, string, error) {
		return nil, "test", nil
	}
	// Avoid real egress probes (ifconfig.me / icanhazip.com) from start-vm tests:
	// no external network in the test env, and cgo DNS can outlive the client
	// timeout, hanging the whole package (pre-existing flaky).
	detectPlatformEgressCIDRFn = func() string { return "203.0.113.77/32" }
	// Inbound terminal guard fail-open by default so heartbeat tests do not
	// block on TaskServiceURL (5s timeout) or 410 live tasks.
	lookupTaskTerminalKindsFn = func(ids []string) (map[string]string, error) {
		return map[string]string{}, nil
	}
	stsAssumeRoleAPI = func(in stsReleaseMintInput) (map[string]any, error) {
		return nil, fmt.Errorf("unexpected live AssumeRole in tests role=%s", in.RoleARN)
	}
	// 文档网段 203.0.113.0/24 不可达；默认视为端口拒绝，避免 reconcile 测例卡住。
	probeStaleStartingReachability = func(*CloudServerConfig) bool { return false }
	os.Exit(m.Run())
}

func setupCloudTestDB(t *testing.T) {
	t.Helper()
	prevAsync := startVmExecuteAsync
	startVmExecuteAsync = false
	t.Cleanup(func() { startVmExecuteAsync = prevAsync })
	// OPT-20260821-019: 终态短 TTL 缓存是进程内状态，每个测试须从干净缓存开始。
	resetTerminalKindCache()
	testDSN, cleanup, err := dbload.OpenTestMySQLClonedFromDir(
		"task-cloud", repoRoot(), "dataMigrate/taskCloudService",
		func(dsn string) error {
			if err := openDB(dsn); err != nil {
				return err
			}
			defer db.Close()
			return runDataMigrate(repoRoot())
		},
	)
	if err != nil {
		t.Skipf("MySQL not available: %v", err)
	}
	t.Cleanup(cleanup)
	if err := openDB(testDSN); err != nil {
		t.Fatalf("openDB: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	// Clean slate between tests that share a DB connection:
	// truncate the core tables so each test starts fresh.
	t.Cleanup(func() {
		for _, tbl := range []string{
			"cloud_server_configs", "cloud_server_config_histories",
			"cloud_platform_authorizations", "cloud_platform_authorization_active_methods",
			"cloud_oauth_tokens", "cloud_vendor_platform_credentials", "cloud_server_events",
			"cloud_tenant_installed_images", "cloud_access_key_iam_associations",
			"cloud_comment_container_bindings", "cloud_comment_container_binding_logs_deprecated_20260821",
			"cloud_workspace_machine_policies",
			"cloud_binding_pending_start_trace",
			"cloud_task_repo_github_bindings",
			"cloud_server_config_defaults",
			"cloud_job_execution_event",
			"cloud_layer_graph_snapshot",
			"cloud_job_step_full_object",
			"cloud_comment_startup_log_object",
			"cloud_stop_request",
		} {
			_, _ = db.Exec(fmt.Sprintf("DELETE FROM %s", tbl))
		}
		for _, tbl := range ccbLogShardTables() {
			_, _ = db.Exec(fmt.Sprintf("DELETE FROM %s", tbl))
		}
		for _, tbl := range jobExecutionEventShardTables() {
			_, _ = db.Exec(fmt.Sprintf("DELETE FROM %s", tbl))
		}
	})
}

func setupTestBudgetDB(t *testing.T) error {
	t.Helper()
	dsn, cleanup, err := dbload.OpenTestMySQL("task-budget", repoRoot())
	if err != nil {
		t.Skipf("MySQL not available for budget: %v", err)
		return err
	}
	t.Cleanup(cleanup)
	return openBudgetDBForTest(dsn)
}

const testLiveCommentID = "cmt-rt"

func seedCloudConfig(t *testing.T, companyID, workspaceID, taskID, serverURL string) {
	t.Helper()
	_, err := db.Exec(
		`INSERT INTO cloud_server_configs(id, company_id, workspace_id, task_id, comment_id, platform, server_url, region, zone_id, authorization_id)
		 VALUES (?,?,?,?,?,?,?,?,?,?)`,
		"cfg1", companyID, workspaceID, taskID, "", "aliyun", serverURL, "cn-test", "cn-test-a", "test",
	)
	if err != nil {
		t.Fatalf("seed template config: %v", err)
	}
	_, err = db.Exec(
		`INSERT INTO cloud_server_configs(id, company_id, workspace_id, task_id, comment_id, platform, server_url, region, zone_id, authorization_id)
		 VALUES (?,?,?,?,?,?,?,?,?,?)`,
		"cfg1-cmt", companyID, workspaceID, taskID, testLiveCommentID, "aliyun", serverURL, "cn-test", "cn-test-a", "test",
	)
	if err != nil {
		t.Fatalf("seed comment config: %v", err)
	}
}

// TestRunDataMigrateIdempotent 回归 OPT-20260809-016：启动自动迁移后重复执行
// runDataMigrate 不重复应用（data_migrate_log step_key 去重），幂等安全。
func TestRunDataMigrateIdempotent(t *testing.T) {
	setupCloudTestDB(t) // 内部已执行一次 runDataMigrate
	countApplied := func() int {
		var n int
		if err := db.QueryRow(`SELECT COUNT(*) FROM data_migrate_log`).Scan(&n); err != nil {
			t.Fatalf("count data_migrate_log: %v", err)
		}
		return n
	}
	before := countApplied()
	if before == 0 {
		t.Fatal("expected migrations applied by setup")
	}
	if err := runDataMigrate(repoRoot()); err != nil {
		t.Fatalf("re-run runDataMigrate: %v", err)
	}
	after := countApplied()
	if after != before {
		t.Fatalf("idempotency broken: applied %d → %d", before, after)
	}
}

func TestValidateAICommentPostMissingServerURL(t *testing.T) {
	setupCloudTestDB(t)
	status, payload := validateAICommentPost("t1", "ws1", "task1", map[string]interface{}{
		"content": "hello",
	}, "")
	if status != 400 {
		t.Fatalf("status=%d want 400", status)
	}
	if payload["detail"] == nil {
		t.Fatalf("expected detail, got %v", payload)
	}
}

func TestValidateAICommentPostOK(t *testing.T) {
	setupCloudTestDB(t)
	seedCloudConfig(t, "t1", "ws1", "task1", "http://127.0.0.1:9999/")
	status, payload := validateAICommentPost("t1", "ws1", "task1", map[string]interface{}{
		"content":      "hello",
		"command_kind": "shell",
	}, "")
	if status != 200 {
		t.Fatalf("status=%d payload=%v", status, payload)
	}
}

func TestValidateAICommentPostRejectsTraeWithBothAnchors(t *testing.T) {
	gate := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/requirements/task-gate" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"clone_done": true})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer gate.Close()

	setupCloudTestDB(t)
	seedCloudConfig(t, "t1", "ws1", "task1", gate.URL+"/ui/test-token/")

	status, payload := validateAICommentPost("t1", "ws1", "task1", map[string]interface{}{
		"content":       "hello",
		"command_kind":  "trae",
		"parent_job_id": "job-1",
		"repo_layer_id": "layer-1",
	}, "")
	if status != 400 {
		t.Fatalf("status=%d want 400 payload=%v", status, payload)
	}
}

func TestInternalValidateAICommentPostHTTP(t *testing.T) {
	setupCloudTestDB(t)
	seedCloudConfig(t, "t1", "ws1", "task1", "http://127.0.0.1:9999/")

	body := `{"tenant_id":"t1","workspace_id":"ws1","task_id":"task1","content":"hi","command_kind":"shell"}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/cloud-server-config/validate-ai-comment-post/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleInternalValidateAICommentPost(rec, req)
	if rec.Code != 200 {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestInternalLookupConfig(t *testing.T) {
	setupCloudTestDB(t)
	seedCloudConfig(t, "t1", "ws1", "task1", "http://127.0.0.1:9999/")

	req := httptest.NewRequest(http.MethodGet, "/api/internal/cloud-server-config/lookup/?tenant_id=t1&workspace_id=ws1&task_id=task1", nil)
	rec := httptest.NewRecorder()
	handleInternalLookupConfig(rec, req)
	if rec.Code != 200 {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out["server_url"] != "http://127.0.0.1:9999/" {
		t.Fatalf("server_url=%v", out["server_url"])
	}
}

func TestImportCloudServerConfigs(t *testing.T) {
	setupCloudTestDB(t)
	count, err := importCloudServerConfigs([]CloudServerConfig{{
		ID: "imp1", CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task1",
		CommentID: "cmt-imp1", Platform: "mock", ServerURL: "http://example.com/",
	}})
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if count != 1 {
		t.Fatalf("count=%d", count)
	}
	cfg, err := loadCloudServerConfigForComment("t1", "ws1", "task1", "cmt-imp1")
	if err != nil || cfg == nil || cfg.ServerURL != "http://example.com/" {
		t.Fatalf("load: err=%v cfg=%v", err, cfg)
	}
}
