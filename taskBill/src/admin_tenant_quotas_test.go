package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAdminTenantQuotasRequiresAuth(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/system-admin/tenant-quotas/tenant_id/9610000099/", nil)
	rec := httptest.NewRecorder()
	handleSystemAdminTenantQuotas(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAdminTenantQuotasRequiresPlatformStaff(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/system-admin/tenant-quotas/tenant_id/9610000099/", nil)
	req.Header.Set("X-Gateway-Auth-Verified", "1")
	req.Header.Set("X-User-Roles", "member")
	rec := httptest.NewRecorder()
	handleSystemAdminTenantQuotas(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("non-staff got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAdminTenantQuotasReturnsTaskPostKeys(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()

	req := staffAdminOrderReq(t, http.MethodGet, "/api/system-admin/tenant-quotas/tenant_id/9610000099/")
	rec := httptest.NewRecorder()
	handleSystemAdminTenantQuotas(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json: %v", err)
	}
	if _, ok := body["task_post_quota"]; !ok {
		t.Fatalf("missing task_post_quota: %v", body)
	}
}
