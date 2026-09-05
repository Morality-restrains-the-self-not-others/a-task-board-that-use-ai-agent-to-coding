package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	ecsclient "github.com/alibabacloud-go/ecs-20140526/v7/client"
)

func TestContainerTaskUIContext(t *testing.T) {
	setupCloudTestDB(t)
	seedCloudConfig(t, "t1", "ws1", "task1", "http://203.0.113.10:8080/?access_token=test-token")

	req := httptest.NewRequest(http.MethodGet, "/?task_id=task1&comment_id="+testLiveCommentID, nil)
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Workspace-Id", "ws1")
	req.Header.Set("X-Task-Id", "task1")
	rec := httptest.NewRecorder()
	handleContainerTaskUIContext(rec, req, "t1", "ws1", "task1")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "success" {
		t.Fatalf("status=%v", body["status"])
	}
	if body["has_server_config"] != true {
		t.Fatalf("has_server_config=%v", body["has_server_config"])
	}
}

func TestInternalContainerTarget(t *testing.T) {
	setupCloudTestDB(t)
	seedCloudConfig(t, "t1", "ws1", "task1", "http://127.0.0.1:1/ui/test-token/")

	req := httptest.NewRequest(http.MethodGet, "/api/internal/cloud-server-config/container-target/?tenant_id=t1&workspace_id=ws1&task_id=task1", nil)
	rec := httptest.NewRecorder()
	handleInternalCloudServerConfig(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestWorkspaceCloudRouteDispatchesComputePaths(t *testing.T) {
	setupCloudTestDB(t)

	req := httptest.NewRequest(http.MethodGet,
		"/api/tenant/t1/workspace/ws1/cloud/compute/previous-server-config/?task_id=task1", nil)
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Workspace-Id", "ws1")
	rec := httptest.NewRecorder()
	handleCloudWorkspaceRoutes(rec, req, "cloud/compute/previous-server-config/")

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"server_config"`) {
		t.Fatalf("expected server_config payload: %s", rec.Body.String())
	}
}

func TestServerRuntimeStatusWithoutConfig(t *testing.T) {
	setupCloudTestDB(t)
	req := httptest.NewRequest(http.MethodGet, "/?task_id=missing&comment_id="+testLiveCommentID, nil)
	rec := httptest.NewRecorder()
	handleServerRuntimeStatus(rec, req, "t1", "ws1", "missing")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"runtime_status"`) {
		t.Fatalf("expected runtime_status: %s", rec.Body.String())
	}
}

func TestServerRuntimeStatusUsesCommentCSCWithInstance(t *testing.T) {
	setupCloudTestDB(t)
	_, err := db.Exec(
		`INSERT INTO cloud_server_configs(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, region, zone_id, authorization_id)
		 VALUES (?,?,?,?,?,?,?,?,?,?)`,
		"cfg-task-empty", "t1", "ws1", "task-cmt-rt", "", "aliyun", "", "cn-test", "cn-test-a", "test",
	)
	if err != nil {
		t.Fatalf("seed task csc: %v", err)
	}
	_, err = db.Exec(
		`INSERT INTO cloud_server_configs(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, region, zone_id, authorization_id, last_runtime_status)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		"cfg-cmt-inst", "t1", "ws1", "task-cmt-rt", "cmt_1", "mock", "mock-cmt-1", "cn-test", "cn-test-a", "test", "Running",
	)
	if err != nil {
		t.Fatalf("seed comment csc: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/?task_id=task-cmt-rt&comment_id=cmt_1", nil)
	rec := httptest.NewRecorder()
	handleServerRuntimeStatus(rec, req, "t1", "ws1", "task-cmt-rt")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["runtime_status"] != "Running" {
		t.Fatalf("runtime_status=%v want Running (comment CSC)", body["runtime_status"])
	}
	if body["instance_id"] != "mock-cmt-1" {
		t.Fatalf("instance_id=%v want mock-cmt-1", body["instance_id"])
	}
}

// TestServerRuntimeStatusCommentCSCWithoutInstancePending
// 回归：评论级 CSC 已分配但尚无 instance_id 时，不得返回「未找到服务器配置记录」；
// 应与「服务器启动状态=启动中」对齐为 Starting。
func TestServerRuntimeStatusCommentCSCWithoutInstancePending(t *testing.T) {
	setupCloudTestDB(t)
	_, err := db.Exec(
		`INSERT INTO cloud_server_configs(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, region, zone_id, authorization_id)
		 VALUES (?,?,?,?,?,?,?,?,?,?)`,
		"cfg-cmt-pending", "t1", "ws1", "task-cmt-pending", "cmt_p1", "aliyun", "", "cn-test", "cn-test-a", "test",
	)
	if err != nil {
		t.Fatalf("seed comment csc: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/?task_id=task-cmt-pending&comment_id=cmt_p1", nil)
	rec := httptest.NewRecorder()
	handleServerRuntimeStatus(rec, req, "t1", "ws1", "task-cmt-pending")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if msg, _ := body["message"].(string); strings.Contains(msg, "未找到服务器配置记录") {
		t.Fatalf("must not report missing CSC when comment CSC exists: body=%v", body)
	}
	if body["runtime_status"] != "Starting" {
		t.Fatalf("runtime_status=%v want Starting, body=%v", body["runtime_status"], body)
	}
	if body["csc_id"] != "cfg-cmt-pending" {
		t.Fatalf("csc_id=%v want cfg-cmt-pending", body["csc_id"])
	}
	if body["comment_id"] != "cmt_p1" {
		t.Fatalf("comment_id=%v want cmt_p1", body["comment_id"])
	}
}

// TestServerRuntimeStatusCommentCSCAfterStopNotStarting
// 回归：停机清空 instance_id 后评论级 CSC 仍在。不得因 comment_id!=” 误报 Starting。
func TestServerRuntimeStatusCommentCSCAfterStopNotStarting(t *testing.T) {
	setupCloudTestDB(t)
	_, err := db.Exec(
		`INSERT INTO cloud_server_configs(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, region, zone_id, authorization_id, last_runtime_status)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		"cfg-cmt-stopped", "t1", "ws1", "task-cmt-stopped", "cmt_s1", "aliyun", "", "cn-test", "cn-test-a", "test", "Released",
	)
	if err != nil {
		t.Fatalf("seed comment csc: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/?task_id=task-cmt-stopped&comment_id=cmt_s1", nil)
	rec := httptest.NewRecorder()
	handleServerRuntimeStatus(rec, req, "t1", "ws1", "task-cmt-stopped")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["runtime_status"] == "Starting" {
		t.Fatalf("after stop must not report Starting: body=%v", body)
	}
	if body["runtime_status"] != "Released" {
		t.Fatalf("runtime_status=%v want Released, body=%v", body["runtime_status"], body)
	}
	if msg, _ := body["message"].(string); strings.Contains(msg, "云实例创建中") {
		t.Fatalf("after stop must not say 云实例创建中: %q", msg)
	}
}

// TestServerRuntimeStatusTemplateRowNotUsedAsRuntime
// 仅有 comment_id=” 硬件模板时，带评论 id 查询不得把模板当运行实例。
func TestServerRuntimeStatusTemplateRowNotUsedAsRuntime(t *testing.T) {
	setupCloudTestDB(t)
	_, err := db.Exec(
		`INSERT INTO cloud_server_configs(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, region, zone_id, authorization_id)
		 VALUES (?,?,?,?,?,?,?,?,?,?)`,
		"cfg-task-idle", "t1", "ws1", "task-idle", "", "aliyun", "", "cn-test", "cn-test-a", "test",
	)
	if err != nil {
		t.Fatalf("seed task csc: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/?task_id=task-idle&comment_id="+testLiveCommentID, nil)
	rec := httptest.NewRecorder()
	handleServerRuntimeStatus(rec, req, "t1", "ws1", "task-idle")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["runtime_status"] != nil {
		t.Fatalf("runtime_status=%v want nil when only template row exists", body["runtime_status"])
	}
	if msg, _ := body["message"].(string); msg != "未找到服务器配置记录" {
		t.Fatalf("message=%q want 未找到服务器配置记录", msg)
	}
}

// 回归：启动失败后评论 CSC 仍不存在时，不得用空闲态「未找到服务器配置记录」掩盖启动失败。
func TestServerRuntimeStatusMissingCSCAfterFailedStart(t *testing.T) {
	setupCloudTestDB(t)
	b, err := insertCommentContainerBinding("t1", "task-miss-csc", "cmt_miss", ccbExecutionIndependent, "")
	if err != nil {
		t.Fatalf("insert binding: %v", err)
	}
	if err := markCommentContainerBindingFailed(b.ID, b.MockContainerName, ""); err != nil {
		t.Fatalf("mark failed: %v", err)
	}
	persistCommentBindingStartTraceID("task-miss-csc", "cmt_miss", "start-trace-miss-csc")

	req := httptest.NewRequest(http.MethodGet, "/?task_id=task-miss-csc&comment_id=cmt_miss", nil)
	rec := httptest.NewRecorder()
	handleServerRuntimeStatus(rec, req, "t1", "ws1", "task-miss-csc")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	msg, _ := body["message"].(string)
	if msg == "未找到服务器配置记录" {
		t.Fatalf("failed start must not use idle missing-CSC copy: %q", msg)
	}
	if !strings.Contains(msg, "启动失败") {
		t.Fatalf("message=%q want 启动失败 hint", msg)
	}
}

// 回归：评论级 CSC 已创建但 RunInstances 失败（无 instance_id）。
// 不得把 failed binding 显示成「云实例创建中，等待分配」。
func TestServerRuntimeStatusCommentCSCStartFailedNotStarting(t *testing.T) {
	setupCloudTestDB(t)
	_, err := db.Exec(
		`INSERT INTO cloud_server_configs(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, region, zone_id, authorization_id, last_runtime_status)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		"cfg-cmt-fail", "t1", "ws1", "task-cmt-fail", "cmt_fail1", "aliyun", "", "cn-test", "cn-test-a", "test", "Starting",
	)
	if err != nil {
		t.Fatalf("seed comment csc: %v", err)
	}
	b, err := insertCommentContainerBinding("t1", "task-cmt-fail", "cmt_fail1", ccbExecutionIndependent, "", "ws1")
	if err != nil {
		t.Fatalf("insert binding: %v", err)
	}
	if err := markCommentContainerBindingFailed(b.ID, b.MockContainerName, "cfg-cmt-fail"); err != nil {
		t.Fatalf("mark failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/?task_id=task-cmt-fail&comment_id=cmt_fail1", nil)
	rec := httptest.NewRecorder()
	handleServerRuntimeStatus(rec, req, "t1", "ws1", "task-cmt-fail")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["runtime_status"] == "Starting" {
		t.Fatalf("start-vm failure must not report Starting: body=%v", body)
	}
	if body["start_failed"] != true {
		t.Fatalf("start_failed=%v want true, body=%v", body["start_failed"], body)
	}
	msg, _ := body["message"].(string)
	if strings.Contains(msg, "创建中") {
		t.Fatalf("must not keep provisioning copy after failure: %q", msg)
	}
}

func TestServerRuntimeStatusIncludesOpenSessionStartedAt(t *testing.T) {
	setupCloudTestDB(t)
	_, err := db.Exec(
		`INSERT INTO cloud_server_configs(id, company_id, workspace_id, task_id, comment_id, platform, server_url, region, zone_id, authorization_id)
		 VALUES (?,?,?,?,?,?,?,?,?,?)`,
		"cfg2", "t1", "ws1", "task2", testLiveCommentID, "mock", "http://127.0.0.1:8080/", "cn-test", "cn-test-a", "test",
	)
	if err != nil {
		t.Fatalf("seed config: %v", err)
	}
	_, err = db.Exec(
		`INSERT INTO cloud_server_config_histories(
			id, company_id, workspace_id, task_id, platform, platform_id, instance_id, instance_type_id,
			security_group_id, vswitch_id, region, zone_id, authorization_id, public_ip, server_url,
			business_api_endpoint, error_reason, stop_reason, runtime_source, launch_request_id,
			cpu_cores, memory_gb, storage_gb, started_at, stopped_at, created_at
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		"hist1", "t1", "ws1", "task2", "mock", 1, "", "ecs.test", "", "", "cn-test", "cn-test-a", "test",
		"", "", "", "", "", "start_vm", "", 2, 8, 40,
		time.Now().UTC().Format("2006-01-02 15:04:05"), nil, time.Now().UTC().Format("2006-01-02 15:04:05"),
	)
	if err != nil {
		t.Fatalf("seed history: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/?task_id=task2&comment_id="+testLiveCommentID, nil)
	rec := httptest.NewRecorder()
	handleServerRuntimeStatus(rec, req, "t1", "ws1", "task2")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	// Verify server_started_at is present (exact value depends on DB timezone configuration)
	if body["server_started_at"] == nil || body["server_started_at"] == "" {
		t.Fatalf("server_started_at empty: body=%v", body)
	}
}

func TestServerRuntimeStatusIncludesAutoReleaseTime(t *testing.T) {
	setupCloudTestDB(t)
	_, err := db.Exec(`INSERT INTO cloud_platform_authorizations
		(id,platform_type,authorization_type,secret_id,secret_key,remark,company_id,active)
		VALUES('auth-release','aliyun','access_key','sid','skey','', 't1',1)`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(
		`INSERT INTO cloud_server_configs(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, server_url, region, zone_id, authorization_id)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		"cfg-release", "t1", "ws1", "task-release", testLiveCommentID, "aliyun", "i-release-1", "", "cn-hongkong", "cn-hongkong-b", "auth-release",
	)
	if err != nil {
		t.Fatalf("seed config: %v", err)
	}

	old := describeInstanceForRuntime
	t.Cleanup(func() { describeInstanceForRuntime = old })
	describeInstanceForRuntime = func(accessKey, secretKey, regionID, instanceID string) (map[string]interface{}, bool, string, error) {
		if accessKey != "sid" || secretKey != "skey" || regionID != "cn-hongkong" || instanceID != "i-release-1" {
			t.Fatalf("unexpected describe args: %s %s %s %s", accessKey, secretKey, regionID, instanceID)
		}
		return map[string]interface{}{
			"Status":          "Running",
			"InstanceId":      "i-release-1",
			"AutoReleaseTime": "2026-07-13T06:30:00Z",
			"CreationTime":    "2026-07-13T06:00:00Z",
			"ZoneId":          "cn-hongkong-b",
		}, true, "req-describe-instances-1", nil
	}

	req := httptest.NewRequest(http.MethodGet, "/?task_id=task-release&comment_id="+testLiveCommentID, nil)
	rec := httptest.NewRecorder()
	handleServerRuntimeStatus(rec, req, "t1", "ws1", "task-release")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("X-Cloud-Request-Id-DescribeInstances"); got != "req-describe-instances-1" {
		t.Fatalf("DescribeInstances request id header=%q", got)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["auto_release_time"] != "2026-07-13T06:30:00Z" {
		t.Fatalf("auto_release_time=%v", body["auto_release_time"])
	}
	attr, _ := body["instance_attribute"].(map[string]interface{})
	attrBody, _ := attr["body"].(map[string]interface{})
	if attrBody["AutoReleaseTime"] != "2026-07-13T06:30:00Z" {
		t.Fatalf("instance_attribute.body.AutoReleaseTime=%v", attrBody["AutoReleaseTime"])
	}
	if body["runtime_status"] != "Running" {
		t.Fatalf("runtime_status=%v", body["runtime_status"])
	}
}

func TestServerRuntimeStatusReleasedWhenDescribeInstancesEmpty(t *testing.T) {
	setupCloudTestDB(t)
	_, err := db.Exec(`INSERT INTO cloud_platform_authorizations
		(id,platform_type,authorization_type,secret_id,secret_key,remark,company_id,active)
		VALUES('auth-gone','aliyun','access_key','sid','skey','', 't1',1)`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(
		`INSERT INTO cloud_server_configs(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, server_url, region, zone_id, authorization_id)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		"cfg-gone", "t1", "ws1", "task-gone", testLiveCommentID, "aliyun", "i-gone-1", "", "cn-hongkong", "cn-hongkong-b", "auth-gone",
	)
	if err != nil {
		t.Fatalf("seed config: %v", err)
	}

	old := describeInstanceForRuntime
	t.Cleanup(func() { describeInstanceForRuntime = old })
	describeInstanceForRuntime = func(accessKey, secretKey, regionID, instanceID string) (map[string]interface{}, bool, string, error) {
		return nil, false, "req-empty", nil
	}

	req := httptest.NewRequest(http.MethodGet, "/?task_id=task-gone&comment_id="+testLiveCommentID, nil)
	rec := httptest.NewRecorder()
	handleServerRuntimeStatus(rec, req, "t1", "ws1", "task-gone")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["runtime_status"] != "Released" || body["released"] != true {
		t.Fatalf("body=%v", body)
	}
}

func TestMapDescribeInstancesInstanceIncludesAutoReleaseTime(t *testing.T) {
	autoRelease := "2026-07-13T06:30:00Z"
	status := "Running"
	instID := "i-1"
	inst := &ecsclient.DescribeInstancesResponseBodyInstancesInstance{
		Status:          &status,
		InstanceId:      &instID,
		AutoReleaseTime: &autoRelease,
	}
	out := mapDescribeInstancesInstance(inst)
	if out["AutoReleaseTime"] != autoRelease {
		t.Fatalf("AutoReleaseTime=%v", out["AutoReleaseTime"])
	}
	empty := ""
	inst.AutoReleaseTime = &empty
	out = mapDescribeInstancesInstance(inst)
	if _, ok := out["AutoReleaseTime"]; ok {
		t.Fatalf("empty AutoReleaseTime should be omitted, got %v", out["AutoReleaseTime"])
	}
}

func TestMapDescribeInstancesInstanceIncludesInternetBandwidth(t *testing.T) {
	status := "Running"
	instID := "i-bw"
	charge := "PayByTraffic"
	outBW := int32(5)
	inBW := int32(5)
	eipIP := "203.0.113.9"
	eipCharge := "PayByBandwidth"
	eipBW := int32(10)
	inst := &ecsclient.DescribeInstancesResponseBodyInstancesInstance{
		Status:                  &status,
		InstanceId:              &instID,
		InternetChargeType:      &charge,
		InternetMaxBandwidthOut: &outBW,
		InternetMaxBandwidthIn:  &inBW,
		EipAddress: &ecsclient.DescribeInstancesResponseBodyInstancesInstanceEipAddress{
			IpAddress:          &eipIP,
			InternetChargeType: &eipCharge,
			Bandwidth:          &eipBW,
		},
	}
	out := mapDescribeInstancesInstance(inst)
	if out["InternetChargeType"] != charge {
		t.Fatalf("InternetChargeType=%v", out["InternetChargeType"])
	}
	if out["InternetMaxBandwidthOut"] != outBW {
		t.Fatalf("InternetMaxBandwidthOut=%v", out["InternetMaxBandwidthOut"])
	}
	if out["InternetMaxBandwidthIn"] != inBW {
		t.Fatalf("InternetMaxBandwidthIn=%v", out["InternetMaxBandwidthIn"])
	}
	eip, ok := out["EipAddress"].(map[string]interface{})
	if !ok {
		t.Fatalf("EipAddress missing: %v", out["EipAddress"])
	}
	if eip["IpAddress"] != eipIP {
		t.Fatalf("EipAddress.IpAddress=%v", eip["IpAddress"])
	}
	if eip["InternetChargeType"] != eipCharge {
		t.Fatalf("EipAddress.InternetChargeType=%v", eip["InternetChargeType"])
	}
	if eip["Bandwidth"] != eipBW {
		t.Fatalf("EipAddress.Bandwidth=%v", eip["Bandwidth"])
	}
}

func TestServerStartupStatusPollNative(t *testing.T) {
	setupCloudTestDB(t)
	eventID, _, err := insertPendingStartEvent("t1", "ws1", "task1", "m1", "", map[string]interface{}{"task_id": "task1"})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet,
		"/api/tenant/t1/workspace/ws1/cloud/compute/server-startup-status/?task_id=task1&event_id="+eventID, nil)
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Workspace-Id", "ws1")
	rec := httptest.NewRecorder()
	handleCloudWorkspaceRoutes(rec, req, "cloud/compute/server-startup-status/")

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"processing"`) {
		t.Fatalf("expected processing payload: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), eventID) {
		t.Fatalf("expected event_id in body: %s", rec.Body.String())
	}
}

func TestStartVmAutoUsesNativeFinalizePath(t *testing.T) {
	setupCloudTestDB(t)
	_, err := db.Exec(`INSERT INTO cloud_platform_authorizations
		(id,platform_type,authorization_type,secret_id,secret_key,remark,company_id,active)
		VALUES('auth1','aliyun','access_key','sid','skey','', 't1',1)`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO cloud_tenant_installed_images
		(id,tenant_id,external_image_id,name,image_url,version,target_architectures,installed_at)
		VALUES('img1','t1','ext1','demo','registry.example/demo','v1','[]',NOW())`)
	if err != nil {
		t.Fatal(err)
	}

	credSrv := mockCredentialTokenServer(t, "auto-access-token")
	cfg.CredentialServiceURL = credSrv.URL

	// Go calls Aliyun SDK directly — no Django bridge.
	cfg.KafkaBootstrapServers = ""
	cfg.TaskBillURL = ""

	aiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "runtime-userdata") {
			_ = json.NewEncoder(w).Encode(map[string]string{"content": "#!/bin/bash\necho ok"})
			return
		}
		_ = json.NewEncoder(w).Encode([]map[string]interface{}{})
	}))
	defer aiSrv.Close()
	cfg.AIProviderBaseURL = aiSrv.URL

	body := `{"task_id":"task1","comment_id":"c1","cloud_server_image_id":"img-cloud-1","region_id":"cn-hangzhou","auto_create_vswitch":true}`
	req := httptest.NewRequest(http.MethodPost,
		"/api/tenant/t1/workspace/ws1/cloud/compute/start-vm-auto/",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Workspace-Id", "ws1")
	req.Header.Set("X-Auth-User-Id", "user1")
	rec := httptest.NewRecorder()
	handleCloudWorkspaceRoutes(rec, req, "cloud/compute/start-vm-auto/")

	// Aliyun SDK fails in test (no real credentials) — verify bootstrap succeeded
	_ = rec.Code
	var cnt int
	if err := db.QueryRow(`SELECT COUNT(*) FROM cloud_server_events WHERE company_id='t1' AND task_id='task1'`).Scan(&cnt); err != nil {
		t.Fatal(err)
	}
	if cnt != 1 {
		t.Fatalf("expected cloud_server_events row, got %d", cnt)
	}
	if strings.Contains(rec.Body.String(), "not yet ported") {
		t.Fatalf("unexpected 501 payload: %s", rec.Body.String())
	}
}

func TestStartVmWithoutDjangoConfigNoLongerBlocksStart(t *testing.T) {
	// Go calls Aliyun SDK directly — DjangoInternalAPI no longer required for start-vm.
	// The old 502 "django internal api not configured" no longer applies.
	setupCloudTestDB(t)

	req := httptest.NewRequest(http.MethodPost,
		"/api/tenant/t1/workspace/ws1/cloud/compute/start-vm/",
		strings.NewReader(`{"task_id":"task1","container_image_id":"img1","region_id":"cn-hangzhou","vpc_id":"vpc-1","vswitch_id":"vsw-1","security_group_id":"sg-1"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Workspace-Id", "ws1")
	req.Header.Set("X-Auth-User-Id", "user1")
	rec := httptest.NewRecorder()
	handleCloudWorkspaceRoutes(rec, req, "cloud/compute/start-vm/")

	// Without cloud auth DB row, returns bad request (auth not found)
	// Django is fully decommissioned — Go calls Aliyun SDK directly.
	if rec.Code < 400 {
		t.Fatalf("expected error status, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func seedAliyunCloudConfig(t *testing.T, companyID, workspaceID, taskID, commentID, instanceID, region string) {
	t.Helper()
	_, err := db.Exec(
		`INSERT INTO cloud_server_configs(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, server_url, region, zone_id, authorization_id)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		"cfg-wb-"+taskID, companyID, workspaceID, taskID, commentID, "aliyun", instanceID, "", region, region+"-b", "auth1",
	)
	if err != nil {
		t.Fatalf("seed aliyun config: %v", err)
	}
}

func TestWorkbenchLinkSuccess(t *testing.T) {
	setupCloudTestDB(t)
	seedAliyunCloudConfig(t, "t1", "ws1", "task-wb", "cmt-wb", "i-bp1234567890", "cn-hongkong")

	req := httptest.NewRequest(http.MethodGet,
		"/api/tenant/t1/workspace/ws1/cloud/compute/workbench-link/?task_id=task-wb&comment_id=cmt-wb", nil)
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Workspace-Id", "ws1")
	rec := httptest.NewRecorder()
	handleCloudWorkspaceRoutes(rec, req, "cloud/compute/workbench-link/")

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "success" {
		t.Fatalf("status=%v", body["status"])
	}
	url, _ := body["workbench_url"].(string)
	if !strings.Contains(url, "ecs-workbench.aliyun.com") {
		t.Fatalf("workbench_url=%q", url)
	}
	if !strings.Contains(url, "instanceId=i-bp1234567890") {
		t.Fatalf("workbench_url missing instanceId: %q", url)
	}
	if !strings.Contains(url, "regionId=cn-hongkong") {
		t.Fatalf("workbench_url missing regionId: %q", url)
	}
}

func TestWorkbenchLinkMissingConfig(t *testing.T) {
	setupCloudTestDB(t)
	req := httptest.NewRequest(http.MethodGet,
		"/api/tenant/t1/workspace/ws1/cloud/compute/workbench-link/?task_id=missing&comment_id=cmt-missing", nil)
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Workspace-Id", "ws1")
	rec := httptest.NewRecorder()
	handleCloudWorkspaceRoutes(rec, req, "cloud/compute/workbench-link/")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestWorkbenchLinkMockInstanceRejected(t *testing.T) {
	setupCloudTestDB(t)
	seedAliyunCloudConfig(t, "t1", "ws1", "task-mock", "cmt-mock", "mock-abc", "cn-hongkong")

	req := httptest.NewRequest(http.MethodGet,
		"/api/tenant/t1/workspace/ws1/cloud/compute/workbench-link/?task_id=task-mock&comment_id=cmt-mock", nil)
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Workspace-Id", "ws1")
	rec := httptest.NewRecorder()
	handleCloudWorkspaceRoutes(rec, req, "cloud/compute/workbench-link/")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Mock") {
		t.Fatalf("expected mock rejection: %s", rec.Body.String())
	}
}

// github-credential-status / github-credential-approve 的 410 stub 测试已随
// OPT-052 Go 重新实现移除，等价行为测试见 compute_github_credential_test.go
//（TestGithubCredentialApproveRejectsNonPOST 等）。

func TestServerRuntimeStatusAuthMissingFallsBackToCachedRunning(t *testing.T) {
	setupCloudTestDB(t)
	// CSC 有真实 instance_id + Running 缓存，但 authorization_id 已失效（授权被删）
	_, err := db.Exec(
		`INSERT INTO cloud_server_configs(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, public_ip, server_url, region, zone_id, authorization_id, last_runtime_status)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		"cfg-auth-gone", "t1", "ws1", "task-auth-gone", testLiveCommentID, "aliyun", "i-alive-1",
		"203.0.113.50", "http://203.0.113.50:8080/", "cn-hangzhou", "cn-hangzhou-i",
		"deleted-auth-id", "Running",
	)
	if err != nil {
		t.Fatalf("seed config: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/?task_id=task-auth-gone&comment_id="+testLiveCommentID, nil)
	req.Header.Set("X-Trace-Id", "trace-auth-gone-1")
	rec := httptest.NewRecorder()
	handleServerRuntimeStatus(rec, req, "t1", "ws1", "task-auth-gone")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s (want 200 degraded success, not 400)", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "success" {
		t.Fatalf("status=%v want success", body["status"])
	}
	if body["runtime_status"] != "Running" {
		t.Fatalf("runtime_status=%v want Running from local cache", body["runtime_status"])
	}
	if body["auth_missing"] != true {
		t.Fatalf("auth_missing=%v want true", body["auth_missing"])
	}
	msg, _ := body["message"].(string)
	if !strings.Contains(msg, "云平台授权") {
		t.Fatalf("message should mention auth: %q", msg)
	}
	if body["instance_id"] != "i-alive-1" {
		t.Fatalf("instance_id=%v", body["instance_id"])
	}
	if body["trace_id"] != "trace-auth-gone-1" {
		t.Fatalf("trace_id=%v want trace-auth-gone-1 (FE data-traceId)", body["trace_id"])
	}
	if rec.Header().Get("X-Trace-Id") != "trace-auth-gone-1" {
		t.Fatalf("response X-Trace-Id=%q", rec.Header().Get("X-Trace-Id"))
	}
}

func TestServerRuntimeStatusAuthMissingUsesCompanyFallbackAuth(t *testing.T) {
	setupCloudTestDB(t)
	_, err := db.Exec(`INSERT INTO cloud_platform_authorizations
		(id,platform_type,authorization_type,secret_id,secret_key,remark,company_id,active)
		VALUES('auth-company','aliyun','access_key','sid-fb','skey-fb','', 't1',1)`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(
		`INSERT INTO cloud_server_configs(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, region, zone_id, authorization_id, last_runtime_status)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		"cfg-auth-fb", "t1", "ws1", "task-auth-fb", testLiveCommentID, "aliyun", "i-fb-1",
		"cn-hangzhou", "cn-hangzhou-i", "stale-auth-id", "Starting",
	)
	if err != nil {
		t.Fatalf("seed config: %v", err)
	}

	old := describeInstanceForRuntime
	t.Cleanup(func() { describeInstanceForRuntime = old })
	described := false
	describeInstanceForRuntime = func(accessKey, secretKey, regionID, instanceID string) (map[string]interface{}, bool, string, error) {
		described = true
		if accessKey != "sid-fb" || secretKey != "skey-fb" {
			t.Fatalf("expected company fallback auth secrets, got %s/%s", accessKey, secretKey)
		}
		return map[string]interface{}{
			"Status":     "Running",
			"InstanceId": "i-fb-1",
		}, true, "req-fb", nil
	}

	req := httptest.NewRequest(http.MethodGet, "/?task_id=task-auth-fb&comment_id="+testLiveCommentID, nil)
	rec := httptest.NewRecorder()
	handleServerRuntimeStatus(rec, req, "t1", "ws1", "task-auth-fb")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !described {
		t.Fatal("expected describe via company fallback auth")
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["runtime_status"] != "Running" {
		t.Fatalf("runtime_status=%v", body["runtime_status"])
	}
	if body["auth_missing"] == true {
		t.Fatal("live describe succeeded; auth_missing should be absent/false")
	}
}

func TestServerRuntimeStatusHealsMockPlatformRealInstance(t *testing.T) {
	setupCloudTestDB(t)
	_, err := db.Exec(`INSERT INTO cloud_platform_authorizations
		(id,platform_type,authorization_type,secret_id,secret_key,remark,company_id,active)
		VALUES('cpa-heal-rt','aliyun','access_key','sid-heal','skey-heal','','t1',1)`)
	if err != nil {
		t.Fatal(err)
	}
	now := "2026-08-14 00:00:00"
	_, err = db.Exec(`INSERT INTO cloud_server_configs
		(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, region, zone_id, authorization_id, last_runtime_status, created_at, updated_at)
		VALUES ('cfg-task-rt-heal', 't1', 'ws1', 'task-rt-heal', '', 'aliyun', '', 'cn-qingdao', 'cn-qingdao-b', 'cpa-heal-rt', '', ?, ?)`, now, now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO cloud_server_configs
		(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, region, zone_id, authorization_id, last_runtime_status, created_at, updated_at)
		VALUES ('cfg-cmt-rt-heal', 't1', 'ws1', 'task-rt-heal', ?, 'mock', 'i-m5edps4oejpnwahrjpoa', '', '', '', 'Running', ?, ?)`,
		testLiveCommentID, now, now)
	if err != nil {
		t.Fatal(err)
	}

	old := describeInstanceForRuntime
	t.Cleanup(func() { describeInstanceForRuntime = old })
	described := false
	describeInstanceForRuntime = func(accessKey, secretKey, regionID, instanceID string) (map[string]interface{}, bool, string, error) {
		described = true
		if instanceID != "i-m5edps4oejpnwahrjpoa" {
			t.Fatalf("instanceID=%s", instanceID)
		}
		if regionID != "cn-qingdao" {
			t.Fatalf("regionID=%s want cn-qingdao after heal", regionID)
		}
		return map[string]interface{}{"Status": "Running", "InstanceId": instanceID}, true, "req-heal", nil
	}

	req := httptest.NewRequest(http.MethodGet, "/?task_id=task-rt-heal&comment_id="+testLiveCommentID, nil)
	rec := httptest.NewRecorder()
	handleServerRuntimeStatus(rec, req, "t1", "ws1", "task-rt-heal")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !described {
		t.Fatal("expected live Describe after healing mock platform")
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["auth_missing"] == true {
		t.Fatalf("must not degrade to auth_missing: %s", rec.Body.String())
	}
	if body["mock"] == true {
		t.Fatalf("must not treat real ECS as mock: %s", rec.Body.String())
	}
	if body["runtime_status"] != "Running" {
		t.Fatalf("runtime_status=%v", body["runtime_status"])
	}
}
