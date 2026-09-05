package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInternalTenantMemberViaDjango(t *testing.T) {
	setupCloudTestDB(t)
	store := newSaasHTTPStore()
	store.putMember("t1", "u1", "m1", true)
	startSaasInternalMock(t, store)
	cfg.InternalSecret = ""

	req := httptest.NewRequest(http.MethodGet, "/api/internal/tenant-member/?tenant_id=t1&user_id=u1", nil)
	rec := httptest.NewRecorder()
	handleInternalTenantMember(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body["company_member_id"] != "m1" {
		t.Fatalf("body=%v", body)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/internal/tenant-member/?tenant_id=t1&user_id=missing", nil)
	rec = httptest.NewRecorder()
	handleInternalTenantMember(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestInternalContainerTargetOverrideAndConflict(t *testing.T) {
	setupCloudTestDB(t)
	seedCloudConfig(t, "t1", "ws1", "task1", "")

	req := httptest.NewRequest(http.MethodGet,
		"/api/internal/cloud-server-config/container-target/?tenant_id=t1&workspace_id=ws1&task_id=task1&container_page_url=http://10.0.0.1:8765/ui/override-tok/",
		nil)
	rec := httptest.NewRecorder()
	handleInternalCloudServerConfig(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body["base_url"] != "http://10.0.0.1:8765" {
		t.Fatalf("base_url=%v", body["base_url"])
	}
	if body["access_token"] != "override-tok" {
		t.Fatalf("access_token=%v", body["access_token"])
	}
}
