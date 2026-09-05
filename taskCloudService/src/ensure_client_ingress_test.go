package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestNormalizePublicHostCIDR(t *testing.T) {
	if got := normalizePublicHostCIDR("203.0.113.9"); got != "203.0.113.9/32" {
		t.Fatalf("got %q", got)
	}
	if got := normalizePublicHostCIDR("203.0.113.9/32"); got != "203.0.113.9/32" {
		t.Fatalf("got %q", got)
	}
	if got := normalizePublicHostCIDR("2001:db8::1"); got != "2001:db8::1/128" {
		t.Fatalf("got %q", got)
	}
	if got := normalizePublicHostCIDR("0.0.0.0/0"); got != "" {
		t.Fatalf("full-open must be rejected, got %q", got)
	}
	if got := normalizePublicHostCIDR("10.0.0.0/24"); got != "" {
		t.Fatalf("non-host cidr must be rejected, got %q", got)
	}
	if got := normalizePublicHostCIDR("127.0.0.1"); got != "" {
		t.Fatalf("loopback must be rejected, got %q", got)
	}
	if got := normalizePublicHostCIDR("10.1.2.3"); got != "" {
		t.Fatalf("private must be rejected, got %q", got)
	}
}

func TestCollectEnsureClientIngressIPs(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("X-Forwarded-For", "198.51.100.7, 10.0.0.1")
	body := map[string]interface{}{
		"client_public_ip":  "203.0.113.10",
		"client_public_ips": []interface{}{"203.0.113.10", "2001:db8::2", "bad"},
	}
	ips := collectEnsureClientIngressIPs(req, body)
	joined := strings.Join(ips, ",")
	if !strings.Contains(joined, "203.0.113.10") || !strings.Contains(joined, "198.51.100.7") {
		t.Fatalf("ips=%v", ips)
	}
	// dedupe
	count := 0
	for _, ip := range ips {
		if ip == "203.0.113.10" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected dedupe, ips=%v", ips)
	}
}

func TestHandleEnsureClientIngressMockSkipped(t *testing.T) {
	setupCloudTestDB(t)
	seedCloudConfig(t, "t1", "ws1", "task-ens", "http://127.0.0.1:8765/")

	req := httptest.NewRequest(http.MethodPost,
		"/api/tenant/t1/workspace/ws1/cloud/compute/ensure-client-ingress/?task_id=task-ens",
		strings.NewReader(`{"client_public_ip":"203.0.113.55"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Workspace-Id", "ws1")
	rec := httptest.NewRecorder()
	handleEnsureClientIngress(rec, req, "t1", "ws1", "task-ens")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["status"] != "success" || payload["skipped"] != true {
		t.Fatalf("payload=%v", payload)
	}
}

func TestHandleEnsureClientIngressHealsCommentMock(t *testing.T) {
	setupCloudTestDB(t)
	now := "2026-08-14 00:00:00"
	_, err := db.Exec(`INSERT INTO cloud_server_configs
		(id, company_id, workspace_id, task_id, comment_id, platform, region, zone_id, authorization_id, security_group_id, created_at, updated_at)
		VALUES ('cfg-tpl-ing', 't1', 'ws1', 'task-ing', '', 'aliyun', 'cn-qingdao', 'cn-qingdao-b', 'cpa-ing', 'sg-ing', ?, ?)`, now, now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO cloud_server_configs
		(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, region, authorization_id, created_at, updated_at)
		VALUES ('cfg-cmt-ing', 't1', 'ws1', 'task-ing', 'c-ing', 'mock', 'i-ing', '', '', ?, ?)`, now, now)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost,
		"/api/tenant/t1/workspace/ws1/cloud/compute/ensure-client-ingress/?task_id=task-ing&comment_id=c-ing",
		strings.NewReader(`{"client_public_ip":"203.0.113.55"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Workspace-Id", "ws1")
	rec := httptest.NewRecorder()
	handleEnsureClientIngress(rec, req, "t1", "ws1", "task-ing")
	if rec.Code != http.StatusOK && rec.Code != http.StatusBadGateway {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["skipped"] == true && payload["reason"] == "local_or_mock_platform" {
		t.Fatalf("comment mock must be healed before skip-as-mock: %v", payload)
	}
	reloaded, err := loadCloudServerConfigForComment("t1", "ws1", "task-ing", "c-ing")
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Platform != "aliyun" || reloaded.Region != "cn-qingdao" || reloaded.AuthorizationID != "cpa-ing" {
		t.Fatalf("healed platform=%s region=%s auth=%s", reloaded.Platform, reloaded.Region, reloaded.AuthorizationID)
	}
}

func TestHandleEnsureClientIngressMissingIP(t *testing.T) {
	setupCloudTestDB(t)
	req := httptest.NewRequest(http.MethodPost,
		"/api/tenant/t1/workspace/ws1/cloud/compute/ensure-client-ingress/?task_id=task-x",
		strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = ""
	rec := httptest.NewRecorder()
	handleEnsureClientIngress(rec, req, "t1", "ws1", "task-x")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestLiveAuthorizeClientIngressIdempotent(t *testing.T) {
	if os.Getenv("LIVE_SG_AUTHORIZE") != "1" {
		t.Skip("set LIVE_SG_AUTHORIZE=1 to run")
	}
	setupCloudTestDB(t)
	var region, sgID, authID string
	if err := db.QueryRow(`SELECT region, security_group_id, authorization_id FROM cloud_server_configs WHERE task_id=? AND IFNULL(security_group_id,'')!='' LIMIT 1`,
		"task_13912900205675601865").Scan(&region, &sgID, &authID); err != nil {
		t.Fatal(err)
	}
	auth, err := loadCloudAuthByID(authID)
	if err != nil || auth == nil {
		t.Fatal(err)
	}
	_, already, err := aliyunAuthorizeClientIngress(auth.SecretID, auth.SecretKey, region, sgID, "47.86.27.42")
	if err != nil {
		t.Fatal(err)
	}
	if !already {
		t.Log("rule newly added (unexpected if already whitelisted)")
	}
}
