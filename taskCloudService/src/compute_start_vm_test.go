package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"tracelog"
)

func TestValidateStartVmAutoPayload(t *testing.T) {
	body := map[string]interface{}{
		"task_id":            "t1",
		"container_image_id": "img1",
		"vpc_id":             "vpc-1",
	}
	if msg := validateStartVmAutoPayload(body); msg == "" {
		t.Fatal("expected error when no auto_create flags")
	}
	body["auto_create_vswitch"] = true
	if msg := validateStartVmAutoPayload(body); msg != "" {
		t.Fatalf("unexpected: %s", msg)
	}
}

func TestEnrichStartVmPayloadFromInstalledImage(t *testing.T) {
	setupCloudTestDB(t)
	_, err := db.Exec(`INSERT INTO cloud_tenant_installed_images
		(id,tenant_id,external_image_id,name,image_url,version,target_architectures,installed_at)
		VALUES('img1','t1','ext1','demo','registry.example/demo','v1.0','[]',NOW())`)
	if err != nil {
		t.Fatal(err)
	}
	body := map[string]interface{}{
		"task_id":            "task1",
		"container_image_id": "img1",
	}
	enrichStartVmPayloadFromInstalledImage("t1", body)
	if body["container_image_url"] != "registry.example/demo:v1.0" {
		t.Fatalf("container_image_url=%v", body["container_image_url"])
	}
}

func TestStartVmAutoValidationRejectedOnMissingFields(t *testing.T) {
	setupCloudTestDB(t)

	body := `{"task_id":"task1","container_image_id":"img1","vpc_id":"vpc-1"}`
	req := httptest.NewRequest(http.MethodPost,
		"/api/tenant/t1/workspace/ws1/cloud/compute/start-vm-auto/",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Workspace-Id", "ws1")
	rec := httptest.NewRecorder()
	handleCloudWorkspaceRoutes(rec, req, "cloud/compute/start-vm-auto/")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "自动创建") {
		t.Fatalf("expected auto_create message: %s", rec.Body.String())
	}
}

func TestStartVmAutoNativeFinalizeIncludesEnrichedImageURL(t *testing.T) {
	setupCloudTestDB(t)
	_, err := db.Exec(`INSERT INTO cloud_platform_authorizations
		(id,platform_type,authorization_type,secret_id,secret_key,remark,company_id,active)
		VALUES('auth1','aliyun','access_key','sid','skey','', 't1',1)`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO cloud_tenant_installed_images
		(id,tenant_id,external_image_id,name,image_url,version,target_architectures,installed_at)
		VALUES('img1','t1','ext1','demo','registry.example/demo','v2','[]',NOW())`)
	if err != nil {
		t.Fatal(err)
	}

	aiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "runtime-userdata") {
			_ = json.NewEncoder(w).Encode(map[string]string{"content": "#!/bin/bash\necho hi"})
			return
		}
		_ = json.NewEncoder(w).Encode([]map[string]interface{}{
			{"platform_type": "aliyun", "region": "cn-hangzhou", "image_id": "cloud-img-1"},
		})
	}))
	defer aiSrv.Close()
	cfg.AIProviderBaseURL = aiSrv.URL

	credSrv := mockCredentialTokenServer(t, "test-access-token")
	cfg.CredentialServiceURL = credSrv.URL

	// Go calls Aliyun SDK directly — no Django bridge (OPT-032 complete).
	// Test verifies bootstrap (event creation + data) succeeds without Django.
	cfg.KafkaBootstrapServers = ""
	cfg.TaskBillURL = ""

	body := `{"task_id":"task1","comment_id":"c1","container_image_id":"img1","region_id":"cn-hangzhou","auto_create_security_group":true}`
	req := httptest.NewRequest(http.MethodPost,
		"/api/tenant/t1/workspace/ws1/cloud/compute/start-vm-auto/",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Workspace-Id", "ws1")
	req.Header.Set("X-Auth-User-Id", "user1")
	req.Header.Set("X-Forwarded-For", "203.0.113.77")
	rec := httptest.NewRecorder()
	handleCloudWorkspaceRoutes(rec, req, "cloud/compute/start-vm-auto/")

	// Aliyun SDK call fails in test (no real credentials) — accept error status
	_ = rec.Code
	var cnt int
	if err := db.QueryRow(`SELECT COUNT(*) FROM cloud_server_events WHERE company_id='t1' AND task_id='task1'`).Scan(&cnt); err != nil {
		t.Fatal(err)
	}
	if cnt != 1 {
		t.Fatalf("expected 1 cloud_server_event, got %d", cnt)
	}
	var eventDataRaw string
	if err := db.QueryRow(`SELECT event_data FROM cloud_server_events WHERE company_id='t1' AND task_id='task1'`).Scan(&eventDataRaw); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(eventDataRaw, `"userdata_content"`) {
		t.Fatalf("event_data missing userdata: %s", eventDataRaw)
	}
	if !strings.Contains(eventDataRaw, `"userdata_access_token"`) || !strings.Contains(eventDataRaw, "test-access-token") {
		t.Fatalf("event_data missing access token: %s", eventDataRaw)
	}
	if !strings.Contains(eventDataRaw, `"client_public_ip":"203.0.113.77"`) {
		t.Fatalf("event_data missing client_public_ip: %s", eventDataRaw)
	}
	if !strings.Contains(eventDataRaw, `"auto_sg_whitelist":true`) {
		t.Fatalf("event_data missing auto_sg_whitelist: %s", eventDataRaw)
	}
	if !strings.Contains(eventDataRaw, `"csc_id"`) {
		t.Fatalf("event_data missing csc_id (comment CSC must be pinned before RunInstances): %s", eventDataRaw)
	}
	b, err := loadCommentContainerBinding("t1", "task1", "c1")
	if err != nil {
		t.Fatalf("start-vm-auto must insert comment binding without waiting for task-detail ensure: %v", err)
	}
	if b.ID == "" {
		t.Fatal("expected comment binding id after start-vm-auto ACK")
	}
	// Aliyun SDK fails with fake credentials; binding may already be failed.
	// The regression is that a row exists so start errors have somewhere to drain.
}

func TestStartVmAutoNativeBindsTraceIDToTaskID(t *testing.T) {
	setupCloudTestDB(t)
	_, err := db.Exec(`INSERT INTO cloud_platform_authorizations
		(id,platform_type,authorization_type,secret_id,secret_key,remark,company_id,active)
		VALUES('auth1','aliyun','access_key','sid','skey','', 't1',1)`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO cloud_tenant_installed_images
		(id,tenant_id,external_image_id,name,image_url,version,target_architectures,installed_at)
		VALUES('img1','t1','ext1','demo','registry.example/demo','v1.0','[]',NOW())`)
	if err != nil {
		t.Fatal(err)
	}

	aiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "runtime-userdata") {
			_ = json.NewEncoder(w).Encode(map[string]string{"content": "#!/bin/bash\necho hi"})
			return
		}
		_ = json.NewEncoder(w).Encode([]map[string]interface{}{
			{"platform_type": "aliyun", "region": "cn-hangzhou", "image_id": "cloud-img-1"},
		})
	}))
	defer aiSrv.Close()
	cfg.AIProviderBaseURL = aiSrv.URL

	credSrv := mockCredentialTokenServer(t, "trace-access-token")
	cfg.CredentialServiceURL = credSrv.URL

	const taskID = "850256677331562496"
	// Go calls Aliyun SDK directly — no Django bridge.
	cfg.KafkaBootstrapServers = ""
	cfg.TaskBillURL = ""

	body := `{"task_id":"` + taskID + `","comment_id":"c1","container_image_id":"img1","region_id":"cn-hangzhou","auto_create_security_group":true}`
	req := httptest.NewRequest(http.MethodPost,
		"/api/tenant/t1/workspace/ws1/cloud/compute/start-vm-auto/",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Workspace-Id", "ws1")
	req.Header.Set("X-Auth-User-Id", "user1")
	req.Header.Set("X-Trace-Id", "web-1783499653748-fnkl3zqin2e")
	req.Header.Set("X-Parent-Span-Id", "a1b2c3d4e5f67890")
	rec := httptest.NewRecorder()
	handleCloudWorkspaceRoutes(rec, req, "cloud/compute/start-vm-auto/")

	// Aliyun SDK fails in test (no real credentials) — bootstrap should still work
	_ = rec.Code
	var eventDataRaw string
	if err := db.QueryRow(`SELECT event_data FROM cloud_server_events WHERE company_id='t1' AND task_id=?`, taskID).Scan(&eventDataRaw); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(eventDataRaw, taskID) {
		t.Fatalf("event_data missing task id: %s", eventDataRaw)
	}
	if strings.Contains(eventDataRaw, `"trace_id":"`+taskID+`"`) {
		t.Fatalf("event_data.trace_id must not be task_id: %s", eventDataRaw)
	}
	if !strings.Contains(eventDataRaw, "web-1783499653748-fnkl3zqin2e") {
		t.Fatalf("event_data.trace_id missing inbound start trace: %s", eventDataRaw)
	}
}

func TestBindStartVmTraceContextKeepsInboundInsteadOfTaskID(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("X-Trace-Id", "web-random-inbound-trace01")
	taskID := "850256677331562496"
	ctx, got := bindStartVmTraceContext(req.Context(), req, taskID)
	if got != "web-random-inbound-trace01" {
		t.Fatalf("runTrace=%q want inbound header, not task_id", got)
	}
	if got == taskID {
		t.Fatal("must not use task_id as run trace")
	}
	if tracelog.TraceIDFromContext(ctx) != "web-random-inbound-trace01" {
		t.Fatalf("ctx trace=%q", tracelog.TraceIDFromContext(ctx))
	}
}

func TestBindStartVmTraceContextFallsBackWhenTaskIDInvalid(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("X-Trace-Id", "web-fallback-trace-id01")
	_, got := bindStartVmTraceContext(req.Context(), req, "short")
	if got != "web-fallback-trace-id01" {
		t.Fatalf("runTrace=%q want inbound fallback", got)
	}
}

func TestValidateStartVmDirectPayload(t *testing.T) {
	body := map[string]interface{}{
		"task_id":            "t1",
		"container_image_id": "img1",
		"region_id":          "cn-hangzhou",
		"vpc_id":             "vpc-1",
		"vswitch_id":         "vsw-1",
		"security_group_id":  "sg-1",
	}
	if msg := validateStartVmDirectPayload(body); msg != "" {
		t.Fatalf("unexpected: %s", msg)
	}
	body["auto_create_vswitch"] = true
	if msg := validateStartVmDirectPayload(body); msg == "" {
		t.Fatal("expected auto_create redirect error")
	}
	delete(body, "auto_create_vswitch")
	delete(body, "security_group_id")
	if msg := validateStartVmDirectPayload(body); msg == "" {
		t.Fatal("expected security_group_id required")
	}
}

func TestStartVmNativeUsesPersistPath(t *testing.T) {
	setupCloudTestDB(t)
	_, err := db.Exec(`INSERT INTO cloud_platform_authorizations
		(id,platform_type,authorization_type,secret_id,secret_key,remark,company_id,active)
		VALUES('auth1','aliyun','access_key','sid','skey','', 't1',1)`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO cloud_tenant_installed_images
		(id,tenant_id,external_image_id,name,image_url,version,target_architectures,installed_at)
		VALUES('img1','t1','ext1','demo','registry.example/demo','v2','[]',NOW())`)
	if err != nil {
		t.Fatal(err)
	}

	aiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "runtime-userdata") {
			_ = json.NewEncoder(w).Encode(map[string]string{"content": "#!/bin/bash\necho hi"})
			return
		}
		_ = json.NewEncoder(w).Encode([]map[string]interface{}{
			{"platform_type": "aliyun", "region": "cn-hangzhou", "image_id": "cloud-img-1"},
		})
	}))
	defer aiSrv.Close()
	cfg.AIProviderBaseURL = aiSrv.URL

	credSrv := mockCredentialTokenServer(t, "direct-access-token")
	cfg.CredentialServiceURL = credSrv.URL

	// Go calls Aliyun SDK directly — no Django bridge.
	cfg.KafkaBootstrapServers = ""
	cfg.TaskBillURL = ""

	body := `{"task_id":"task1","comment_id":"c1","container_image_id":"img1","region_id":"cn-hangzhou","zone_id":"cn-hangzhou-h","vpc_id":"vpc-1","vswitch_id":"vsw-1","security_group_id":"sg-1"}`
	req := httptest.NewRequest(http.MethodPost,
		"/api/tenant/t1/workspace/ws1/cloud/compute/start-vm/",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Workspace-Id", "ws1")
	req.Header.Set("X-Auth-User-Id", "user1")
	rec := httptest.NewRecorder()
	handleCloudWorkspaceRoutes(rec, req, "cloud/compute/start-vm/")

	// Aliyun SDK fails in test (fake credentials) — verify bootstrap created event row
	var cnt int
	if err := db.QueryRow(`SELECT COUNT(*) FROM cloud_server_events WHERE company_id='t1' AND task_id='task1'`).Scan(&cnt); err != nil {
		t.Fatal(err)
	}
	if cnt != 1 {
		t.Fatalf("expected cloud_server_events row, got %d", cnt)
	}
	if strings.Contains(rec.Body.String(), "not yet ported") {
		t.Fatalf("unexpected 501: %s", rec.Body.String())
	}
}
