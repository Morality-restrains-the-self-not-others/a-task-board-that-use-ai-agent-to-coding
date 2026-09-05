package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPersistStartVmInstanceBindingZeroRowsErrors(t *testing.T) {
	setupCloudTestDB(t)
	err := persistStartVmInstanceBinding(map[string]interface{}{
		"task_id":    "task-missing",
		"comment_id": "cmt_missing",
		"csc_id":     "csc-does-not-exist",
	}, "i-orphan-bind", "req-z")
	if err == nil {
		t.Fatal("persist 0 rows must error")
	}
}

// 回归：RunInstances 已返回 instance_id，但评论 CSC 行尚未创建时，
// persist 须插入评论行，禁止孤儿实例 + runtime「未找到服务器配置记录」。
func TestPersistStartVmInstanceBindingInsertsMissingCommentCSC(t *testing.T) {
	setupCloudTestDB(t)
	err := persistStartVmInstanceBinding(map[string]interface{}{
		"task_id":             "task-insert-csc",
		"comment_id":          "cmt_insert",
		"company_id":          "t1",
		"workspace_id":        "ws1",
		"cloud_platform_type": "aliyun",
		"authorization_id":    "auth-insert",
		"region_id":           "cn-hangzhou",
		"zone_id":             "cn-hangzhou-h",
	}, "i-inserted", "req-ins")
	if err != nil {
		t.Fatalf("persist insert: %v", err)
	}
	cfg, err := loadCloudServerConfigForComment("t1", "ws1", "task-insert-csc", "cmt_insert")
	if err != nil || cfg == nil {
		t.Fatalf("load: err=%v", err)
	}
	if cfg.InstanceID != "i-inserted" {
		t.Fatalf("instance_id=%q", cfg.InstanceID)
	}
	if cfg.LastRuntimeStatus != machineRuntimeStarting {
		t.Fatalf("status=%q", cfg.LastRuntimeStatus)
	}
	if cfg.Platform != "aliyun" || cfg.AuthorizationID != "auth-insert" {
		t.Fatalf("meta platform=%s auth=%s", cfg.Platform, cfg.AuthorizationID)
	}
}

func TestPersistStartVmInstanceBindingFallsBackToCommentID(t *testing.T) {
	setupCloudTestDB(t)
	now := "2026-08-14 00:00:00"
	_, err := db.Exec(`INSERT INTO cloud_server_configs
		(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, region, zone_id, authorization_id, created_at, updated_at)
		VALUES ('csc-fb', 't1', 'ws1', 'task-fb', 'cmt_fb', 'aliyun', '', 'cn-hangzhou', 'cn-hangzhou-i', 'auth1', ?, ?)`, now, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := persistStartVmInstanceBinding(map[string]interface{}{
		"task_id":    "task-fb",
		"comment_id": "cmt_fb",
		"csc_id":     "csc-wrong-id",
	}, "i-fallback", "req-fb"); err != nil {
		t.Fatalf("fallback persist: %v", err)
	}
	cfg, err := loadCloudServerConfigForComment("t1", "ws1", "task-fb", "cmt_fb")
	if err != nil || cfg == nil {
		t.Fatalf("load: err=%v", err)
	}
	if cfg.InstanceID != "i-fallback" {
		t.Fatalf("instance_id=%q", cfg.InstanceID)
	}
}

func TestPersistStartVmInstanceBindingClearsTerminalReleased(t *testing.T) {
	setupCloudTestDB(t)
	now := "2026-08-22 00:00:00"
	_, err := db.Exec(`INSERT INTO cloud_server_configs
		(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, region, zone_id, authorization_id, terminal_released, last_runtime_status, created_at, updated_at)
		VALUES ('csc-rebind', 't1', 'ws1', 'task-rebind', 'cmt_rebind', 'aliyun', '', 'cn-hangzhou', 'cn-hangzhou-i', 'auth1', 1, 'Released', ?, ?)`, now, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := persistStartVmInstanceBinding(map[string]interface{}{
		"task_id":    "task-rebind",
		"comment_id": "cmt_rebind",
		"csc_id":     "csc-rebind",
	}, "i-rebind", "req-rebind"); err != nil {
		t.Fatalf("persist: %v", err)
	}
	var released int
	var status, instance string
	if err := db.QueryRow(`SELECT COALESCE(terminal_released,0), COALESCE(last_runtime_status,''), COALESCE(instance_id,'') FROM cloud_server_configs WHERE id='csc-rebind'`).
		Scan(&released, &status, &instance); err != nil {
		t.Fatal(err)
	}
	if released != 0 {
		t.Fatalf("terminal_released=%d want 0 after new instance bind", released)
	}
	if instance != "i-rebind" {
		t.Fatalf("instance_id=%q", instance)
	}
	if status != machineRuntimeStarting {
		t.Fatalf("status=%q want %s", status, machineRuntimeStarting)
	}
}

func TestPersistUnscopedSkipsTaskLevel(t *testing.T) {
	setupCloudTestDB(t)
	now := "2026-08-14 00:00:00"
	_, err := db.Exec(`INSERT INTO cloud_server_configs
		(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, region, zone_id, authorization_id, created_at, updated_at)
		VALUES ('csc-tpl-unscoped', 't1', 'ws1', 'task-unscoped', '', 'aliyun', '', 'cn-hangzhou', 'cn-hangzhou-i', 'auth1', ?, ?)`, now, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := persistStartVmInstanceBinding(map[string]interface{}{
		"task_id": "task-unscoped",
	}, "i-should-not-land", "req-x"); err == nil {
		t.Fatal("unscoped persist must error")
	}
	cfg, err := loadCloudServerConfig("t1", "ws1", "task-unscoped")
	if err != nil || cfg == nil {
		t.Fatalf("load template: %v cfg=%v", err, cfg)
	}
	if cfg.InstanceID != "" {
		t.Fatalf("task-level instance_id=%q want empty", cfg.InstanceID)
	}
}

func TestBindStartVmInstanceOrErrorEmptyInstance(t *testing.T) {
	err := bindStartVmInstanceOrError(map[string]interface{}{
		"task_id": "task-empty", "comment_id": "cmt_e",
	}, "", "req")
	if err == nil {
		t.Fatal("empty instance must error")
	}
	if !strings.Contains(err.Error(), "instance_id") {
		t.Fatalf("err=%v", err)
	}
}

func TestUpsertEmptyInstancePreservesCommentInstance(t *testing.T) {
	setupCloudTestDB(t)
	if err := upsertCloudServerConfig(CloudServerConfig{
		ID: "csc-keep", CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task-keep",
		CommentID: "cmt_keep", Platform: "aliyun", InstanceID: "i-keep",
		LastRuntimeStatus: machineRuntimeStarting, PublicIP: "1.2.3.4",
		ServerURL: "http://1.2.3.4:8080", Region: "cn-test", ZoneID: "a",
	}); err != nil {
		t.Fatal(err)
	}
	if err := upsertCloudServerConfig(CloudServerConfig{
		ID: "csc-wipe", CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task-keep",
		CommentID: "cmt_keep", Platform: "aliyun", Region: "cn-test", ZoneID: "a",
	}); err != nil {
		t.Fatal(err)
	}
	cfg, err := loadCloudServerConfigForComment("t1", "ws1", "task-keep", "cmt_keep")
	if err != nil || cfg == nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.InstanceID != "i-keep" {
		t.Fatalf("instance wiped: %q", cfg.InstanceID)
	}
	if cfg.LastRuntimeStatus != machineRuntimeStarting {
		t.Fatalf("status wiped: %q", cfg.LastRuntimeStatus)
	}
	if cfg.PublicIP != "1.2.3.4" {
		t.Fatalf("public_ip wiped: %q", cfg.PublicIP)
	}
}

func TestUpsertReleasedClearsCommentInstance(t *testing.T) {
	setupCloudTestDB(t)
	if err := upsertCloudServerConfig(CloudServerConfig{
		ID: "csc-rel", CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task-rel-up",
		CommentID: "cmt_rel_up", Platform: "aliyun", InstanceID: "i-rel",
		LastRuntimeStatus: machineRuntimeRunning, Region: "cn-test", ZoneID: "a",
	}); err != nil {
		t.Fatal(err)
	}
	if err := upsertCloudServerConfig(CloudServerConfig{
		ID: "csc-rel", CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task-rel-up",
		CommentID: "cmt_rel_up", Platform: "aliyun", InstanceID: "",
		LastRuntimeStatus: machineRuntimeReleased, Region: "cn-test", ZoneID: "a",
	}); err != nil {
		t.Fatal(err)
	}
	cfg, err := loadCloudServerConfigForComment("t1", "ws1", "task-rel-up", "cmt_rel_up")
	if err != nil || cfg == nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.InstanceID != "" {
		t.Fatalf("released must clear instance, got %q", cfg.InstanceID)
	}
	if cfg.LastRuntimeStatus != machineRuntimeReleased {
		t.Fatalf("status=%q", cfg.LastRuntimeStatus)
	}
}

func TestUpsertEmptyTaskLevelClearsInstance(t *testing.T) {
	setupCloudTestDB(t)
	if err := upsertCloudServerConfig(CloudServerConfig{
		ID: "tpl-clr", CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task-clr",
		Platform: "aliyun", InstanceID: "i-tpl", LastRuntimeStatus: machineRuntimeRunning,
		Region: "cn-test", ZoneID: "a",
	}); err != nil {
		t.Fatal(err)
	}
	if err := upsertCloudServerConfig(CloudServerConfig{
		ID: "tpl-clr", CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task-clr",
		Platform: "aliyun", Region: "cn-test", ZoneID: "a",
	}); err != nil {
		t.Fatal(err)
	}
	cfg, err := loadCloudServerConfig("t1", "ws1", "task-clr")
	if err != nil || cfg == nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.InstanceID != "" {
		t.Fatalf("task-level unbind must clear instance, got %q", cfg.InstanceID)
	}
}

func TestPersistPublicIPPromotesRunning(t *testing.T) {
	setupCloudTestDB(t)
	if err := upsertCloudServerConfig(CloudServerConfig{
		ID: "csc-ip", CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task-ip",
		CommentID: "cmt_ip", Platform: "aliyun", InstanceID: "i-ip",
		LastRuntimeStatus: machineRuntimeStarting, Region: "cn-test", ZoneID: "a",
	}); err != nil {
		t.Fatal(err)
	}
	// 即使调用方传入推测性 serverURL，也不得落库（仅 VM 公网 IP + Running）。
	persistCloudServerPublicIPByInstanceID("i-ip", "8.8.8.8", "http://8.8.8.8:8080")
	cfg, err := loadCloudServerConfigForComment("t1", "ws1", "task-ip", "cmt_ip")
	if err != nil || cfg == nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.PublicIP != "8.8.8.8" {
		t.Fatalf("public_ip=%q", cfg.PublicIP)
	}
	if cfg.LastRuntimeStatus != machineRuntimeRunning {
		t.Fatalf("status=%q want Running", cfg.LastRuntimeStatus)
	}
	if cfg.ServerURL != "" {
		t.Fatalf("server_url=%q want empty (register-reachability only)", cfg.ServerURL)
	}
	if commentCSCHasRuntime(cfg) {
		t.Fatal("public_ip must not make commentCSCHasRuntime true")
	}
}

func TestHealCommentCSCInstanceFromDescribe(t *testing.T) {
	setupCloudTestDB(t)
	_, err := db.Exec(`INSERT INTO cloud_platform_authorizations
		(id,platform_type,authorization_type,secret_id,secret_key,remark,company_id,active)
		VALUES('auth-heal','aliyun','access_key','sid','skey','','t1',1)`)
	if err != nil {
		t.Fatal(err)
	}
	if err := upsertCloudServerConfig(CloudServerConfig{
		ID: "csc-heal", CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task_heal1",
		CommentID: "cmt_heal", Platform: "aliyun", InstanceID: "",
		AuthorizationID: "auth-heal", Region: "cn-hangzhou", ZoneID: "i",
	}); err != nil {
		t.Fatal(err)
	}
	old := describeInstancesByName
	oldDesc := describeInstanceForRuntime
	t.Cleanup(func() {
		describeInstancesByName = old
		describeInstanceForRuntime = oldDesc
	})
	describeInstancesByName = func(accessKey, secretKey, regionID, instanceName string) ([]map[string]interface{}, string, error) {
		switch instanceName {
		case "task_heal1_cmt_heal":
			return []map[string]interface{}{
				{"InstanceId": "i-healed", "Status": "Running"},
			}, "req", nil
		default:
			t.Fatalf("must not lookup task-level InstanceName %q", instanceName)
			return nil, "", nil
		}
	}
	describeInstanceForRuntime = func(accessKey, secretKey, regionID, instanceID string) (map[string]interface{}, bool, string, error) {
		return map[string]interface{}{"Status": "Running", "InstanceId": instanceID}, true, "req", nil
	}

	req := httptest.NewRequest(http.MethodGet, "/?task_id=task_heal1&comment_id=cmt_heal", nil)
	rec := httptest.NewRecorder()
	handleServerRuntimeStatus(rec, req, "t1", "ws1", "task_heal1")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	cfg, err := loadCloudServerConfigForComment("t1", "ws1", "task_heal1", "cmt_heal")
	if err != nil || cfg == nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.InstanceID != "i-healed" {
		t.Fatalf("healed instance_id=%q", cfg.InstanceID)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if msg, _ := body["message"].(string); strings.Contains(msg, "等待分配") {
		t.Fatalf("healed row must not wait for alloc: %v", body)
	}
}

func TestHealDoesNotStealOtherCommentInstance(t *testing.T) {
	setupCloudTestDB(t)
	_, err := db.Exec(`INSERT INTO cloud_platform_authorizations
		(id,platform_type,authorization_type,secret_id,secret_key,remark,company_id,active)
		VALUES('auth-steal2','aliyun','access_key','sid','skey','','t1',1)`)
	if err != nil {
		t.Fatal(err)
	}
	if err := upsertCloudServerConfig(CloudServerConfig{
		ID: "csc-owner2", CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task_steal2",
		CommentID: "cmt_owner2", Platform: "aliyun", InstanceID: "i-owned",
		AuthorizationID: "auth-steal2", Region: "cn-hangzhou", ZoneID: "i",
		LastRuntimeStatus: machineRuntimeRunning,
	}); err != nil {
		t.Fatal(err)
	}
	if err := upsertCloudServerConfig(CloudServerConfig{
		ID: "csc-empty2", CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task_steal2",
		CommentID: "cmt_empty2", Platform: "aliyun", InstanceID: "",
		AuthorizationID: "auth-steal2", Region: "cn-hangzhou", ZoneID: "i",
	}); err != nil {
		t.Fatal(err)
	}
	old := describeInstancesByName
	t.Cleanup(func() { describeInstancesByName = old })
	describeInstancesByName = func(accessKey, secretKey, regionID, instanceName string) ([]map[string]interface{}, string, error) {
		return []map[string]interface{}{{"InstanceId": "i-owned", "Status": "Running"}}, "req", nil
	}

	req := httptest.NewRequest(http.MethodGet, "/?task_id=task_steal2&comment_id=cmt_empty2", nil)
	rec := httptest.NewRecorder()
	handleServerRuntimeStatus(rec, req, "t1", "ws1", "task_steal2")
	cfg, err := loadCloudServerConfigForComment("t1", "ws1", "task_steal2", "cmt_empty2")
	if err != nil || cfg == nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.InstanceID != "" {
		t.Fatalf("must not steal i-owned: %q", cfg.InstanceID)
	}
}

func TestBootProgressPromotesStartingToRunning(t *testing.T) {
	setupCloudTestDB(t)
	cfg := &CloudServerConfig{
		ID: "csc-boot", CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task-boot",
		CommentID: "cmt_boot", Platform: "aliyun", InstanceID: "i-boot",
		LastRuntimeStatus: machineRuntimeStarting, Region: "cn-test", ZoneID: "a",
	}
	if err := upsertCloudServerConfig(*cfg); err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	handleBootProgress(rec, context.Background(), cfg, map[string]any{
		"progress": 10, "message": "开始容器初始化",
	}, "t1", "ws1", "task-boot")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	got, err := loadCloudServerConfigForComment("t1", "ws1", "task-boot", "cmt_boot")
	if err != nil || got == nil {
		t.Fatalf("load: %v", err)
	}
	if got.LastRuntimeStatus != machineRuntimeRunning {
		t.Fatalf("status=%q want Running", got.LastRuntimeStatus)
	}
}

func TestServerRuntimeStatusBoundMockNotWaitingAlloc(t *testing.T) {
	setupCloudTestDB(t)
	if err := upsertCloudServerConfig(CloudServerConfig{
		ID: "csc-bound", CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task-bound",
		CommentID: "cmt_bound", Platform: "aliyun", InstanceID: "mock-bound-1",
		LastRuntimeStatus: machineRuntimeStarting, Region: "cn-test", ZoneID: "a",
	}); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/?task_id=task-bound&comment_id=cmt_bound", nil)
	rec := httptest.NewRecorder()
	handleServerRuntimeStatus(rec, req, "t1", "ws1", "task-bound")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["instance_id"] != "mock-bound-1" {
		t.Fatalf("instance_id=%v", body["instance_id"])
	}
	if msg, _ := body["message"].(string); strings.Contains(msg, "等待分配") {
		t.Fatalf("bound instance must not wait for alloc: %v", body)
	}
	if body["runtime_status"] != "Running" {
		t.Fatalf("mock runtime_status=%v want Running", body["runtime_status"])
	}
}

// T11：公网 IP 落库不得把 binding 从 starting 升到 running（须等 register-reachability）。
func TestPublicIPPersistDoesNotPromoteBindingToRunning(t *testing.T) {
	setupCommentContainerBindingTest(t, "task-pubip")

	body := `{"comment_id":"cmt_pubip","execution_mode":"independent"}`
	req := httptest.NewRequest(http.MethodPost,
		"/api/tenant/t1/task/task-pubip/comment-container-bindings/",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	rec := httptest.NewRecorder()
	handleCommentContainerBindingsRoutes(rec, req, "t1", "task-pubip", "", "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", rec.Code, rec.Body.String())
	}

	adv := httptest.NewRequest(http.MethodPost,
		"/api/tenant/t1/task/task-pubip/comment-container-bindings/advance/", nil)
	adv.Header.Set("X-Auth-Tenant-Id", "t1")
	advRec := httptest.NewRecorder()
	handleCommentContainerBindingsRoutes(advRec, adv, "t1", "task-pubip", "", "advance/")
	if advRec.Code != http.StatusOK {
		t.Fatalf("advance status=%d body=%s", advRec.Code, advRec.Body.String())
	}

	b, err := loadCommentContainerBinding("t1", "task-pubip", "cmt_pubip")
	if err != nil {
		t.Fatal(err)
	}
	if b.Status != ccbStatusStarting {
		t.Fatalf("status=%s want starting", b.Status)
	}
	csc, err := loadCloudServerConfigByID(b.CSCID)
	if err != nil {
		t.Fatal(err)
	}
	csc.InstanceID = "i-pubip-1"
	csc.Platform = "aliyun"
	if err := upsertCloudServerConfig(*csc); err != nil {
		t.Fatal(err)
	}

	persistCloudServerPublicIPByInstanceID("i-pubip-1", "203.0.113.50", "http://203.0.113.50:8080")

	rows := []CommentContainerBinding{*b}
	promoted, err := ccbTryPromoteStartingToRunning(&rows, 0)
	if err != nil {
		t.Fatal(err)
	}
	if promoted {
		t.Fatal("public_ip alone must not promote binding to running")
	}
	b2, err := loadCommentContainerBinding("t1", "task-pubip", "cmt_pubip")
	if err != nil {
		t.Fatal(err)
	}
	if b2.Status != ccbStatusStarting {
		t.Fatalf("after public_ip status=%s want starting", b2.Status)
	}

	csc.ServerURL = "http://203.0.113.50:8080/"
	if err := upsertCloudServerConfig(*csc); err != nil {
		t.Fatal(err)
	}
	promoted, err = ccbTryPromoteStartingToRunning(&rows, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !promoted {
		t.Fatal("want promote after real server_url")
	}
	if rows[0].Status != ccbStatusRunning {
		t.Fatalf("status=%s want running", rows[0].Status)
	}
}
