package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWorkspaceMachinePolicyGetDefaults(t *testing.T) {
	setupCloudTestDB(t)

	req := httptest.NewRequest(http.MethodGet,
		"/api/tenant/t1/workspace/ws1/cloud/compute/workspace-machine-policy/", nil)
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Workspace-Id", "ws1")
	rec := httptest.NewRecorder()
	handleCloudWorkspaceRoutes(rec, req, "cloud/compute/workspace-machine-policy/")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "success" {
		t.Fatalf("status=%v", body["status"])
	}
	if body["idle_recycle_minutes"] != float64(5) {
		t.Fatalf("idle_recycle_minutes=%v want 5", body["idle_recycle_minutes"])
	}
	ids, ok := body["enabled_authorization_ids"].([]interface{})
	if !ok || len(ids) != 0 {
		t.Fatalf("enabled_authorization_ids=%v want []", body["enabled_authorization_ids"])
	}
}

func TestWorkspaceMachinePolicyPutAndGet(t *testing.T) {
	setupCloudTestDB(t)

	putBody := `{"idle_recycle_minutes":15,"enabled_authorization_ids":["auth-a","auth-b"]}`
	req := httptest.NewRequest(http.MethodPut,
		"/api/tenant/t1/workspace/ws1/cloud/compute/workspace-machine-policy/",
		strings.NewReader(putBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Workspace-Id", "ws1")
	rec := httptest.NewRecorder()
	handleCloudWorkspaceRoutes(rec, req, "cloud/compute/workspace-machine-policy/")
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT status=%d body=%s", rec.Code, rec.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet,
		"/api/tenant/t1/workspace/ws1/cloud/compute/workspace-machine-policy/", nil)
	getReq.Header.Set("X-Auth-Tenant-Id", "t1")
	getReq.Header.Set("X-Workspace-Id", "ws1")
	getRec := httptest.NewRecorder()
	handleCloudWorkspaceRoutes(getRec, getReq, "cloud/compute/workspace-machine-policy/")
	if getRec.Code != http.StatusOK {
		t.Fatalf("GET status=%d body=%s", getRec.Code, getRec.Body.String())
	}

	var body map[string]interface{}
	if err := json.Unmarshal(getRec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["idle_recycle_minutes"] != float64(15) {
		t.Fatalf("idle_recycle_minutes=%v want 15", body["idle_recycle_minutes"])
	}
	ids, ok := body["enabled_authorization_ids"].([]interface{})
	if !ok || len(ids) != 2 || ids[0] != "auth-a" || ids[1] != "auth-b" {
		t.Fatalf("enabled_authorization_ids=%v", body["enabled_authorization_ids"])
	}
}

func TestWorkspaceMachinePolicyPutRejectsNegativeMinutes(t *testing.T) {
	setupCloudTestDB(t)

	req := httptest.NewRequest(http.MethodPut,
		"/api/tenant/t1/workspace/ws1/cloud/compute/workspace-machine-policy/",
		strings.NewReader(`{"idle_recycle_minutes":-1}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Workspace-Id", "ws1")
	rec := httptest.NewRecorder()
	handleCloudWorkspaceRoutes(rec, req, "cloud/compute/workspace-machine-policy/")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}
