package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWorkspaceContainerImageAtModeEnabledDefaultTrue(t *testing.T) {
	setupTestDB(t)

	body := `{"name":"AT Mode WS"}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspaces/", strings.NewReader(body))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	rec := httptest.NewRecorder()
	handleCreateWorkspace(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("create workspace: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var created map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&created)
	wsID := created["id"].(string)

	if v, ok := created["container_image_at_mode_enabled"].(bool); !ok || !v {
		t.Errorf("create response: expected container_image_at_mode_enabled=true (default), got %v", created["container_image_at_mode_enabled"])
	}

	reqGet := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/workspaces/"+wsID+"/", nil)
	reqGet.Header.Set("X-Auth-Tenant-Id", "t1")
	reqGet.Header.Set("X-Resource-Id", wsID)
	recGet := httptest.NewRecorder()
	handleGetWorkspace(recGet, reqGet)
	if recGet.Code != http.StatusOK {
		t.Fatalf("get workspace: expected 200, got %d: %s", recGet.Code, recGet.Body.String())
	}
	var got map[string]interface{}
	json.NewDecoder(recGet.Body).Decode(&got)
	if v, ok := got["container_image_at_mode_enabled"].(bool); !ok || !v {
		t.Errorf("get response: expected container_image_at_mode_enabled=true (default), got %v", got["container_image_at_mode_enabled"])
	}
}

func TestWorkspaceContainerImageAtModeEnabledPatchFalse(t *testing.T) {
	setupTestDB(t)

	body := `{"name":"AT Mode Patch WS"}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspaces/", strings.NewReader(body))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	rec := httptest.NewRecorder()
	handleCreateWorkspace(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("create workspace: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var created map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&created)
	wsID := created["id"].(string)

	patchBody := `{"container_image_at_mode_enabled":false}`
	reqPatch := httptest.NewRequest(http.MethodPatch, "/api/tenant/t1/workspaces/"+wsID+"/", strings.NewReader(patchBody))
	reqPatch.Header.Set("X-Auth-Tenant-Id", "t1")
	reqPatch.Header.Set("X-Resource-Id", wsID)
	recPatch := httptest.NewRecorder()
	handleUpdateWorkspace(recPatch, reqPatch)
	if recPatch.Code != http.StatusOK {
		t.Fatalf("patch workspace: expected 200, got %d: %s", recPatch.Code, recPatch.Body.String())
	}
	var patched map[string]interface{}
	json.NewDecoder(recPatch.Body).Decode(&patched)
	if v, ok := patched["container_image_at_mode_enabled"].(bool); !ok || v {
		t.Errorf("patch response: expected container_image_at_mode_enabled=false, got %v", patched["container_image_at_mode_enabled"])
	}

	reqGet := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/workspaces/"+wsID+"/", nil)
	reqGet.Header.Set("X-Auth-Tenant-Id", "t1")
	reqGet.Header.Set("X-Resource-Id", wsID)
	recGet := httptest.NewRecorder()
	handleGetWorkspace(recGet, reqGet)
	if recGet.Code != http.StatusOK {
		t.Fatalf("get workspace after patch: expected 200, got %d: %s", recGet.Code, recGet.Body.String())
	}
	var got map[string]interface{}
	json.NewDecoder(recGet.Body).Decode(&got)
	if v, ok := got["container_image_at_mode_enabled"].(bool); !ok || v {
		t.Errorf("get after patch: expected container_image_at_mode_enabled=false, got %v", got["container_image_at_mode_enabled"])
	}
}
