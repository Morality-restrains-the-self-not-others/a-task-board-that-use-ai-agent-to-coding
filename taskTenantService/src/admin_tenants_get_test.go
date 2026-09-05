package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func getAdminTenant(t *testing.T, mux *http.ServeMux, id string) *httptest.ResponseRecorder {
	t.Helper()
	req := withPlatformStaff(httptest.NewRequest(http.MethodGet, "/api/system-admin/accounts/admin/tenants/"+id+"/", nil))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestAdminTenantGetRequiresAuth(t *testing.T) {
	mux := setupTestService(t)
	req := httptest.NewRequest(http.MethodGet, "/api/system-admin/accounts/admin/tenants/c1/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAdminTenantGetRequiresPlatformStaff(t *testing.T) {
	mux := setupTestService(t)
	req := httptest.NewRequest(http.MethodGet, "/api/system-admin/accounts/admin/tenants/c1/", nil)
	req.Header.Set("X-Gateway-Auth-Verified", "1")
	req.Header.Set("X-User-Id", "member1")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("non-staff got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAdminTenantGetNotFound(t *testing.T) {
	mux := setupTestService(t)
	rec := getAdminTenant(t, mux, "missing-co")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing tenant got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAdminTenantGetReturnsSeedCompany(t *testing.T) {
	mux := setupTestService(t)
	prev := fetchCreatorContactsFn
	fetchCreatorContactsFn = func(userIDs []string) map[string]creatorContact {
		return map[string]creatorContact{
			"admin1": {Email: "owner@example.com", Phone: "13800138000"},
		}
	}
	t.Cleanup(func() { fetchCreatorContactsFn = prev })

	rec := getAdminTenant(t, mux, "c1")
	if rec.Code != 200 {
		t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
	}
	body := parseAdminTenants(t, rec)
	if body["id"] != "c1" || body["name"] != "Test Co" {
		t.Fatalf("header=%v", body)
	}
	if body["email"] != "owner@example.com" || body["phone"] != "13800138000" {
		t.Fatalf("contacts=%v", body)
	}
}
