package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// seedCloudServerConfigDefault inserts a row into cloud_server_config_defaults for testing.
func seedCloudServerConfigDefault(t *testing.T, companyID, authID, platformType, region string) {
	t.Helper()
	id := fmt.Sprintf("cscd-%s-%s", companyID, authID)
	_, err := db.Exec(
		`INSERT INTO cloud_server_config_defaults
			(id, company_id, authorization_id, platform_type, region, zone_id, vpc_id, vswitch_id,
			 security_group_id, payment_type, bandwidth_charging_mode, bandwidth,
			 cpu_cores, memory_gb, instance_type, system_disk_category, data_disk_category)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE region=VALUES(region), updated_at=CURRENT_TIMESTAMP`,
		id, companyID, authID, platformType, region, "cn-test-a", "vpc-test", "vsw-test", "sg-test",
		"PostPaid", "PayByTraffic", 1, "2", "4", "ecs.g6.large", "cloud_essd", "cloud_essd",
	)
	if err != nil {
		t.Fatalf("seed cloud_server_config_default: %v", err)
	}
	// Also clean up after test
	t.Cleanup(func() {
		db.Exec(`DELETE FROM cloud_server_config_defaults WHERE id=?`, id)
	})
}

func TestHandleWorkspaceCloudPlatformsListEmpty(t *testing.T) {
	setupCloudTestDB(t)

	req := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/workspaces/ws1/cloud/platforms/", nil)
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Workspace-Id", "ws1")
	rec := httptest.NewRecorder()
	handleWorkspaceCloudPlatforms(rec, req, "t1", "ws1", "")

	if rec.Code != 200 {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	var body map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	platforms, ok := body["platforms"].([]interface{})
	if !ok {
		t.Fatalf("expected platforms array, got %T: %v", body["platforms"], body)
	}
	if len(platforms) != 0 {
		t.Fatalf("expected empty platforms, got %d", len(platforms))
	}
}

func TestHandleWorkspaceCloudPlatformsListWithData(t *testing.T) {
	setupCloudTestDB(t)
	seedCloudAuth(t, "t1", "auth-platform-1")

	req := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/workspaces/ws1/cloud/platforms/", nil)
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Workspace-Id", "ws1")
	rec := httptest.NewRecorder()
	handleWorkspaceCloudPlatforms(rec, req, "t1", "ws1", "")

	if rec.Code != 200 {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	var body map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	platforms, ok := body["platforms"].([]interface{})
	if !ok {
		t.Fatalf("expected platforms array, got %T: %v", body["platforms"], body)
	}
	if len(platforms) < 1 {
		t.Fatalf("expected at least 1 platform, got %d", len(platforms))
	}
}

func TestHandleWorkspaceCloudPlatformsListPostMethodNotAllowed(t *testing.T) {
	setupCloudTestDB(t)

	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspaces/ws1/cloud/platforms/", nil)
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Workspace-Id", "ws1")
	rec := httptest.NewRecorder()
	handleWorkspaceCloudPlatforms(rec, req, "t1", "ws1", "")

	if rec.Code != 405 {
		t.Fatalf("status=%d want 405 body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleWorkspaceCloudPlatformsDefaultConfigEmpty(t *testing.T) {
	setupCloudTestDB(t)

	req := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/workspaces/ws1/cloud/platforms/default-config", nil)
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Workspace-Id", "ws1")
	rec := httptest.NewRecorder()
	handleWorkspaceCloudPlatforms(rec, req, "t1", "ws1", "default-config")

	if rec.Code != 200 {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	var body map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["status"] != "success" {
		t.Fatalf("expected status=success, got %v", body["status"])
	}
	// Empty data is OK — no configs seeded
}

func TestHandleWorkspaceCloudPlatformsDefaultConfigWithData(t *testing.T) {
	setupCloudTestDB(t)
	seedCloudServerConfigDefault(t, "t1", "auth-dc-1", "aliyun", "cn-hangzhou")

	req := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/workspaces/ws1/cloud/platforms/default-config", nil)
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Workspace-Id", "ws1")
	rec := httptest.NewRecorder()
	handleWorkspaceCloudPlatforms(rec, req, "t1", "ws1", "default-config")

	if rec.Code != 200 {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	var body map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["status"] != "success" {
		t.Fatalf("expected status=success, got %v", body["status"])
	}
	data, ok := body["data"].([]interface{})
	if !ok {
		t.Fatalf("expected data array, got %T: %v", body["data"], body)
	}
	if len(data) < 1 {
		t.Fatalf("expected at least 1 default config, got %d", len(data))
	}
}

func TestHandleWorkspaceCloudPlatformsDefaultConfigPostMethodNotAllowed(t *testing.T) {
	setupCloudTestDB(t)

	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspaces/ws1/cloud/platforms/default-config", nil)
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Workspace-Id", "ws1")
	rec := httptest.NewRecorder()
	handleWorkspaceCloudPlatforms(rec, req, "t1", "ws1", "default-config")

	if rec.Code != 405 {
		t.Fatalf("status=%d want 405 body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleWorkspaceCloudPlatformsUnknownSubRoute(t *testing.T) {
	setupCloudTestDB(t)

	req := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/workspaces/ws1/cloud/platforms/unknown", nil)
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Workspace-Id", "ws1")
	rec := httptest.NewRecorder()
	handleWorkspaceCloudPlatforms(rec, req, "t1", "ws1", "unknown")

	if rec.Code != 404 {
		t.Fatalf("status=%d want 404 body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleWorkspaceCloudPlatformsQueryParamsPassthrough(t *testing.T) {
	setupCloudTestDB(t)
	seedCloudAuth(t, "t1", "auth-qp-1")

	// sub="" allows query params (strings.HasPrefix(sub, "?"))
	req := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/workspaces/ws1/cloud/platforms/?extra=1", nil)
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Workspace-Id", "ws1")
	rec := httptest.NewRecorder()
	handleWorkspaceCloudPlatforms(rec, req, "t1", "ws1", "?extra=1")

	if rec.Code != 200 {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	var body map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	_, ok := body["platforms"].([]interface{})
	if !ok {
		t.Fatalf("expected platforms array: %v", body)
	}
}

func TestHandleWorkspaceCloudPlatformsCredentialsMasked(t *testing.T) {
	setupCloudTestDB(t)
	seedCloudAuth(t, "t1", "auth-masked-1")

	req := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/workspaces/ws1/cloud/platforms/", nil)
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Workspace-Id", "ws1")
	rec := httptest.NewRecorder()
	handleWorkspaceCloudPlatforms(rec, req, "t1", "ws1", "")

	if rec.Code != 200 {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	body := rec.Body.String()
	if strings.Contains(body, "AKTEST1234") || strings.Contains(body, "SKTEST5678") {
		t.Fatalf("expected masked credentials, got full values: %s", body)
	}
	if !strings.Contains(body, "AKTE") || !strings.Contains(body, "1234") {
		t.Fatalf("expected access key prefix/suffix visible: %s", body)
	}
}
