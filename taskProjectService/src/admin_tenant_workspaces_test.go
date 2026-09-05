package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAdminTenantWorkspacesRequiresAuth(t *testing.T) {
	setupTestDB(t)
	req := httptest.NewRequest(http.MethodGet, "/api/system-admin/tenant-workspaces/tenant_id/t1/", nil)
	rec := httptest.NewRecorder()
	handleAdminTenantWorkspaces(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAdminTenantWorkspacesRequiresPlatformStaff(t *testing.T) {
	setupTestDB(t)
	req := httptest.NewRequest(http.MethodGet, "/api/system-admin/tenant-workspaces/tenant_id/t1/", nil)
	req.Header.Set("X-Gateway-Auth-Verified", "1")
	req.Header.Set("X-User-Id", "u1")
	rec := httptest.NewRecorder()
	handleAdminTenantWorkspaces(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("non-staff got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAdminTenantWorkspacesListsAllWithoutMineFilter(t *testing.T) {
	setupTestDB(t)

	body := `{"name":"Open WS"}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/workspaces/tenant_id/t1/", strings.NewReader(body))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	rec := httptest.NewRecorder()
	handleCreateWorkspace(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("create: %d %s", rec.Code, rec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/system-admin/tenant-workspaces/tenant_id/t1/", nil)
	listReq.Header.Set("X-Gateway-Auth-Verified", "1")
	listReq.Header.Set("X-User-Id", "staff1")
	listReq.Header.Set("X-User-Roles", "super_admin")
	listRec := httptest.NewRecorder()
	handleAdminTenantWorkspaces(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list: %d %s", listRec.Code, listRec.Body.String())
	}
	var out map[string]interface{}
	if err := json.Unmarshal(listRec.Body.Bytes(), &out); err != nil {
		t.Fatalf("json: %v", err)
	}
	items, _ := out["items"].([]interface{})
	if len(items) < 1 {
		t.Fatalf("expected at least one workspace, got %v", out)
	}
	row, _ := items[0].(map[string]interface{})
	if row["name"] != "Open WS" {
		t.Fatalf("name=%v", row["name"])
	}
}
