package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTenantCloudZonesMock(t *testing.T) {
	setupCloudTestDB(t)
seedCloudAuth(t, "850256677331562496", "862031128628060160")
	t.Setenv("USE_IN_MEMORY_CLOUD", "true")

	req := httptest.NewRequest(http.MethodGet,
		"/api/tenant/850256677331562496/cloud-platform/862031128628060160/cloud/zones/?region_id=cn-hongkong", nil)
	req.Header.Set("X-Auth-Tenant-Id", "850256677331562496")
	req.Header.Set("X-User-Id", "test-user")
	rec := httptest.NewRecorder()

	path := strings.TrimPrefix(req.URL.Path, "/api/tenant/")
	parts := strings.SplitN(path, "/", 2)
	tenantID := parts[0]
	rest := strings.TrimPrefix(path, tenantID+"/cloud-platform/")
	authID := strings.SplitN(rest, "/", 2)[0]
	sub := strings.TrimPrefix(rest, authID+"/cloud/")
	handleCloudPlatformRoutes(rec, req, tenantID, authID, sub)

	if rec.Code != 200 {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"id":"cn-hongkong`) {
		t.Fatalf("expected zone id in response: %s", body)
	}
}

func TestTenantCloudZonesNotFound(t *testing.T) {
	setupCloudTestDB(t)
t.Setenv("USE_IN_MEMORY_CLOUD", "true")

	req := httptest.NewRequest(http.MethodGet,
		"/api/tenant/t1/cloud-platform/missing/cloud/zones/?region_id=cn-hongkong", nil)
	req.Header.Set("X-User-Id", "test-user")
	rec := httptest.NewRecorder()
	handleCloudPlatformRoutes(rec, req, "t1", "missing", "zones/")

	if rec.Code != 404 {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "未找到指定租户下的云平台授权") {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestTenantCloudRegionsMock(t *testing.T) {
	setupCloudTestDB(t)
seedCloudAuth(t, "t1", "auth1")
	t.Setenv("USE_IN_MEMORY_CLOUD", "true")

	req := httptest.NewRequest(http.MethodGet,
		"/api/tenant/t1/cloud-platform/auth1/cloud/regions/?platform_type=aliyun", nil)
	req.Header.Set("X-User-Id", "test-user")
	rec := httptest.NewRecorder()
	handleCloudPlatformRoutes(rec, req, "t1", "auth1", "regions/")

	if rec.Code != 200 {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"id":"cn-hangzhou"`) {
		t.Fatalf("expected hangzhou region: %s", rec.Body.String())
	}
}
