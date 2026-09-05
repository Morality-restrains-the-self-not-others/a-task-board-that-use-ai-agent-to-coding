package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"authz"
	"gatewayauth"
)

func setupBudgetPermissionTestDB(t *testing.T) *saasHTTPStore {
	t.Helper()
	store := setupBudgetTestDB(t)
	store.creators["c1"] = "admin-u"
	store.putMember("c1", "admin-u", "m-admin", true)
	store.putMember("c1", "u1", "m1", false)
	seedBudgetTenant(t, "c1")
	return store
}

func TestTenantBudgetPermissionInternalCRUDAndEvaluate(t *testing.T) {
	setupBudgetPermissionTestDB(t)
	cfg.InternalSecret = ""

	upsertBody := `{
		"company_id":"c1",
		"items":[{"subject_type":"member","subject_id":"m1","can_raise_task_budget":true}]
	}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/internal/budget/tenant-permissions/upsert/", strings.NewReader(upsertBody))
	handleInternalBudgetRoutes(rec, req)
	if rec.Code != 200 {
		t.Fatalf("upsert status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/internal/budget/tenant-permissions/?company_id=c1", nil)
	handleInternalBudgetRoutes(rec, req)
	if rec.Code != 200 {
		t.Fatalf("list status=%d body=%s", rec.Code, rec.Body.String())
	}
	var listed map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &listed); err != nil {
		t.Fatal(err)
	}
	items, _ := listed["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("items=%v", listed)
	}

	evalBody := `{
		"company_id":"c1","user_id":"u1","member_id":"m1",
		"task_owner_member_id":"other","is_tenant_admin":false,"is_workspace_admin":false
	}`
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/internal/budget/tenant-permissions/evaluate-raise/", strings.NewReader(evalBody))
	handleInternalBudgetRoutes(rec, req)
	if rec.Code != 200 {
		t.Fatalf("eval status=%d body=%s", rec.Code, rec.Body.String())
	}
	var eval map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &eval); err != nil {
		t.Fatal(err)
	}
	if eval["allowed"] != true {
		t.Fatalf("eval=%v", eval)
	}
}

func TestUserTenantBudgetPermissionsAdminGate(t *testing.T) {
	setupBudgetPermissionTestDB(t)

	req := httptest.NewRequest(http.MethodGet, "/api/cloud/budget-permissions/tenant_id/c1/", nil)
	req.Header.Set(gatewayauth.HeaderAuthUserID, "u1")
	req.Header.Set("X-Auth-Tenant-Id", "c1")
	rec := httptest.NewRecorder()
	handleUserTenantBudgetPermissions(rec, req)
	if rec.Code != 200 {
		t.Fatalf("get status=%d body=%s", rec.Code, rec.Body.String())
	}

	patchBody := `{"items":[{"subject_type":"member","subject_id":"m1","can_raise_task_budget":true}]}`
	req = httptest.NewRequest(http.MethodPatch, "/api/cloud/budget-permissions/tenant_id/c1/", strings.NewReader(patchBody))
	req.Header.Set(gatewayauth.HeaderAuthUserID, "u1")
	req.Header.Set("X-Auth-Tenant-Id", "c1")
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	handleUserTenantBudgetPermissions(rec, req)
	if rec.Code != 403 {
		t.Fatalf("non-admin patch status=%d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPatch, "/api/cloud/budget-permissions/tenant_id/c1/", strings.NewReader(patchBody))
	req.Header.Set(gatewayauth.HeaderAuthUserID, "admin-u")
	req.Header.Set("X-Auth-Tenant-Id", "c1")
	req.Header.Set(authz.HeaderTenantPerms, "c1:cloud:manage") // v63: PDP 注入的租户权限码
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	handleUserTenantBudgetPermissions(rec, req)
	if rec.Code != 200 {
		t.Fatalf("admin patch status=%d body=%s", rec.Code, rec.Body.String())
	}
}
