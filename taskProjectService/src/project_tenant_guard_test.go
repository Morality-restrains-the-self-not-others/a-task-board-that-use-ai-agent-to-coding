package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProjectCompanyID(t *testing.T) {
	if got := projectCompanyID(nil); got != "" {
		t.Fatalf("nil detail company=%q", got)
	}
	if got := projectCompanyID(map[string]interface{}{"company": "  co1  "}); got != "co1" {
		t.Fatalf("company=%q", got)
	}
}

func TestRejectIfProjectNotInTenantMismatch(t *testing.T) {
	detail := map[string]interface{}{"id": "proj_1", "company": "co-a"}
	req := httptest.NewRequest(http.MethodGet, "/api/projects/proj_1/tenant_id/co-b/", nil)
	req.Header.Set("X-Auth-Tenant-Id", "co-b")
	req.Header.Set("X-Trace-Id", "trace-tenant-mismatch")
	rec := httptest.NewRecorder()
	if !rejectIfProjectNotInTenant(rec, req, detail, "co-b") {
		t.Fatal("expected mismatch reject")
	}
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "project not found") {
		t.Fatalf("body=%s", rec.Body.String())
	}
}

func TestRejectIfProjectNotInTenantSameTenantAllows(t *testing.T) {
	detail := map[string]interface{}{"id": "proj_1", "company": "co-a"}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	if rejectIfProjectNotInTenant(rec, req, detail, "co-a") {
		t.Fatal("same tenant must not reject")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestRejectIfProjectNotInTenantEmptyTenantFailOpen(t *testing.T) {
	detail := map[string]interface{}{"id": "proj_1", "company": "co-a"}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	if rejectIfProjectNotInTenant(rec, req, detail, "") {
		t.Fatal("empty tenant (legacy unit tests) must fail-open")
	}
}

func TestGetProjectOtherTenantNotFound(t *testing.T) {
	setupTestDB(t)
	mux := http.NewServeMux()
	mountRoutes(mux)

	createBody := `{"name":"TenantIsolationProj"}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/tenant_id/t1/", strings.NewReader(createBody))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var created map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	pid, _ := created["id"].(string)
	if pid == "" {
		t.Fatal("expected created project id")
	}

	req = httptest.NewRequest(http.MethodGet, "/api/projects/"+pid+"/tenant_id/t2/", nil)
	req.Header.Set("X-Auth-Tenant-Id", "t2")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("cross-tenant GET: expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "project not found") {
		t.Fatalf("body=%s", rec.Body.String())
	}
}

func TestDeleteProjectOtherTenantNotFound(t *testing.T) {
	setupTestDB(t)
	mux := http.NewServeMux()
	mountRoutes(mux)

	createBody := `{"name":"TenantIsolationDelete"}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/tenant_id/t1/", strings.NewReader(createBody))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var created map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	pid, _ := created["id"].(string)
	if pid == "" {
		t.Fatal("expected created project id")
	}

	req = httptest.NewRequest(http.MethodDelete, "/api/projects/"+pid+"/tenant_id/t2/", nil)
	req.Header.Set("X-Auth-Tenant-Id", "t2")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("cross-tenant DELETE: expected 404, got %d: %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/projects/"+pid+"/tenant_id/t1/", nil)
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("owner GET after rejected delete: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestUpdateProjectOtherTenantNotFound(t *testing.T) {
	setupTestDB(t)
	mux := http.NewServeMux()
	mountRoutes(mux)

	createBody := `{"name":"TenantIsolationUpdate"}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/tenant_id/t1/", strings.NewReader(createBody))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var created map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	pid, _ := created["id"].(string)
	if pid == "" {
		t.Fatal("expected created project id")
	}

	req = httptest.NewRequest(http.MethodPatch, "/api/projects/"+pid+"/tenant_id/t2/", strings.NewReader(`{"name":"Hijacked"}`))
	req.Header.Set("X-Auth-Tenant-Id", "t2")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("cross-tenant PATCH: expected 404, got %d: %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/projects/"+pid+"/tenant_id/t1/", nil)
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("owner GET: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var got map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode get: %v", err)
	}
	if got["name"] != "TenantIsolationUpdate" {
		t.Fatalf("name mutated by cross-tenant PATCH: %v", got["name"])
	}
}
