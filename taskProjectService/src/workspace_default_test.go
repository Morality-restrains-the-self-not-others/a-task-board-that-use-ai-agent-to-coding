package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func createWorkspaceNamed(t *testing.T, tenantID, name string, isDefault bool) string {
	t.Helper()
	body := `{"name":"` + name + `","is_default":` + boolJSON(isDefault) + `}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/workspaces/tenant_id/"+tenantID+"/", strings.NewReader(body))
	req.Header.Set("X-Auth-Tenant-Id", tenantID)
	rec := httptest.NewRecorder()
	handleCreateWorkspace(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("create %q: %d %s", name, rec.Code, rec.Body.String())
	}
	var ws map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&ws); err != nil {
		t.Fatalf("decode create %q: %v", name, err)
	}
	id, _ := ws["id"].(string)
	if id == "" {
		t.Fatalf("missing id for %q", name)
	}
	return id
}

func boolJSON(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

func getWorkspaceDefaultFlag(t *testing.T, tenantID, wsID string) bool {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/projects/workspaces/tenant_id/"+tenantID+"/"+wsID+"/", nil)
	req.Header.Set("X-Auth-Tenant-Id", tenantID)
	req.Header.Set("X-Resource-Id", wsID)
	rec := httptest.NewRecorder()
	handleGetWorkspace(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("get %s: %d %s", wsID, rec.Code, rec.Body.String())
	}
	var ws map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&ws); err != nil {
		t.Fatalf("decode get %s: %v", wsID, err)
	}
	flag, _ := ws["is_default"].(bool)
	return flag
}

func TestUpdateWorkspaceSetDefaultClearsSiblings(t *testing.T) {
	setupTestDB(t)
	wsA := createWorkspaceNamed(t, "t1", "Alpha", true)
	wsB := createWorkspaceNamed(t, "t1", "Beta", false)
	if !getWorkspaceDefaultFlag(t, "t1", wsA) {
		t.Fatal("expected Alpha to be default after create")
	}

	patchBody := `{"is_default":true}`
	req := httptest.NewRequest(http.MethodPut, "/api/projects/workspaces/tenant_id/t1/"+wsB+"/", strings.NewReader(patchBody))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Resource-Id", wsB)
	rec := httptest.NewRecorder()
	handleUpdateWorkspace(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("set default Beta: %d %s", rec.Code, rec.Body.String())
	}

	if !getWorkspaceDefaultFlag(t, "t1", wsB) {
		t.Fatal("expected Beta to be default after update")
	}
	if getWorkspaceDefaultFlag(t, "t1", wsA) {
		t.Fatal("expected Alpha to lose default after Beta was set as default")
	}
}

func TestCreateWorkspaceSetDefaultClearsSiblings(t *testing.T) {
	setupTestDB(t)
	wsA := createWorkspaceNamed(t, "t1", "First Default", true)
	wsB := createWorkspaceNamed(t, "t1", "Second Default", true)

	if getWorkspaceDefaultFlag(t, "t1", wsA) {
		t.Fatal("expected First Default to lose default after second default create")
	}
	if !getWorkspaceDefaultFlag(t, "t1", wsB) {
		t.Fatal("expected Second Default to be the sole default")
	}
}

func TestUpdateWorkspaceDefaultDoesNotTouchOtherTenant(t *testing.T) {
	setupTestDB(t)
	wsT1 := createWorkspaceNamed(t, "t1", "Tenant1 Default", true)
	wsT2 := createWorkspaceNamed(t, "t2", "Tenant2 Default", true)
	wsT1b := createWorkspaceNamed(t, "t1", "Tenant1 Other", false)

	patchBody := `{"is_default":true}`
	req := httptest.NewRequest(http.MethodPut, "/api/projects/workspaces/tenant_id/t1/"+wsT1b+"/", strings.NewReader(patchBody))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Resource-Id", wsT1b)
	rec := httptest.NewRecorder()
	handleUpdateWorkspace(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("set default t1 other: %d %s", rec.Code, rec.Body.String())
	}

	if getWorkspaceDefaultFlag(t, "t1", wsT1) {
		t.Fatal("expected t1 original default to be cleared")
	}
	if !getWorkspaceDefaultFlag(t, "t1", wsT1b) {
		t.Fatal("expected t1 other to become default")
	}
	if !getWorkspaceDefaultFlag(t, "t2", wsT2) {
		t.Fatal("expected t2 default to remain unchanged")
	}
}
