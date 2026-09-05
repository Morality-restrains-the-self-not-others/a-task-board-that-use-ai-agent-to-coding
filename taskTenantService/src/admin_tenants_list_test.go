package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func getAdminTenants(t *testing.T, mux *http.ServeMux, query string) *httptest.ResponseRecorder {
	t.Helper()
	path := "/api/system-admin/accounts/admin/tenants/"
	if query != "" {
		path += "?" + query
	}
	req := withPlatformStaff(httptest.NewRequest(http.MethodGet, path, nil))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func parseAdminTenants(t *testing.T, rec *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var out map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode %d %s: %v", rec.Code, rec.Body.String(), err)
	}
	return out
}

func tenantItems(t *testing.T, body map[string]interface{}) []map[string]interface{} {
	t.Helper()
	raw, ok := body["items"].([]interface{})
	if !ok {
		t.Fatalf("items missing: %v", body)
	}
	out := make([]map[string]interface{}, 0, len(raw))
	for _, v := range raw {
		m, ok := v.(map[string]interface{})
		if !ok {
			t.Fatalf("item not object: %v", v)
		}
		out = append(out, m)
	}
	return out
}

func TestAdminTenantsRequiresAuth(t *testing.T) {
	mux := setupTestService(t)
	req := httptest.NewRequest(http.MethodGet, "/api/system-admin/accounts/admin/tenants/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAdminTenantsRequiresPlatformStaff(t *testing.T) {
	mux := setupTestService(t)
	req := httptest.NewRequest(http.MethodGet, "/api/system-admin/accounts/admin/tenants/", nil)
	req.Header.Set("X-Gateway-Auth-Verified", "1")
	req.Header.Set("X-User-Id", "member1")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("non-staff got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAdminTenantsListsSeedCompany(t *testing.T) {
	mux := setupTestService(t)
	prev := fetchCreatorContactsFn
	fetchCreatorContactsFn = func(userIDs []string) map[string]creatorContact {
		return map[string]creatorContact{
			"admin1": {Email: "owner@example.com", Phone: "13800138000"},
		}
	}
	t.Cleanup(func() { fetchCreatorContactsFn = prev })

	rec := getAdminTenants(t, mux, "")
	if rec.Code != 200 {
		t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
	}
	body := parseAdminTenants(t, rec)
	items := tenantItems(t, body)
	if len(items) == 0 || items[0]["id"] != "c1" {
		t.Fatalf("expected c1, got %v", items)
	}
	if items[0]["name"] != "Test Co" {
		t.Fatalf("name=%v", items[0]["name"])
	}
	if items[0]["email"] != "owner@example.com" || items[0]["phone"] != "13800138000" {
		t.Fatalf("contacts=%v", items[0])
	}
	total, _ := body["total"].(float64)
	if total < 1 {
		t.Fatalf("total=%v", body["total"])
	}
}

func TestAdminTenantsPagination(t *testing.T) {
	mux := setupTestService(t)
	if _, err := db.Exec(`INSERT INTO tenant_company (id, name, creator_id) VALUES ('c2','Beta Co','admin1')`); err != nil {
		t.Fatalf("seed c2: %v", err)
	}
	prev := fetchCreatorContactsFn
	fetchCreatorContactsFn = func(userIDs []string) map[string]creatorContact { return map[string]creatorContact{} }
	t.Cleanup(func() { fetchCreatorContactsFn = prev })

	page1 := parseAdminTenants(t, getAdminTenants(t, mux, "limit=1&offset=0"))
	page2 := parseAdminTenants(t, getAdminTenants(t, mux, "limit=1&offset=1"))
	items1 := tenantItems(t, page1)
	items2 := tenantItems(t, page2)
	if len(items1) != 1 || len(items2) != 1 {
		t.Fatalf("page sizes %d %d", len(items1), len(items2))
	}
	if items1[0]["id"] == items2[0]["id"] {
		t.Fatalf("pages overlapped: %v %v", items1[0]["id"], items2[0]["id"])
	}
	total, _ := page1["total"].(float64)
	if total < 2 {
		t.Fatalf("total=%v", page1["total"])
	}
}

func TestAdminTenantsAuthFailOpen(t *testing.T) {
	mux := setupTestService(t)
	prev := fetchCreatorContactsFn
	fetchCreatorContactsFn = fetchCreatorContactsFromAuth
	t.Cleanup(func() { fetchCreatorContactsFn = prev })
	oldURL := cfg.TaskAuthURL
	cfg.TaskAuthURL = "http://127.0.0.1:1"
	t.Cleanup(func() { cfg.TaskAuthURL = oldURL })

	rec := getAdminTenants(t, mux, "")
	if rec.Code != 200 {
		t.Fatalf("fail-open status %d body=%s", rec.Code, rec.Body.String())
	}
	items := tenantItems(t, parseAdminTenants(t, rec))
	if len(items) == 0 || items[0]["id"] != "c1" {
		t.Fatalf("fail-open should still list companies: %v", items)
	}
	if items[0]["email"] != "" || items[0]["phone"] != "" {
		t.Fatalf("fail-open contacts should be empty: %v", items[0])
	}
}

func TestAdminTenantsSearchDoesNotCapAtEighty(t *testing.T) {
	mux := setupTestService(t)
	prevC, prevS := fetchCreatorContactsFn, searchCreatorIDsFn
	fetchCreatorContactsFn = func(userIDs []string) map[string]creatorContact { return map[string]creatorContact{} }
	searchCreatorIDsFn = func(query string) []string { return nil }
	t.Cleanup(func() {
		fetchCreatorContactsFn = prevC
		searchCreatorIDsFn = prevS
	})
	for i := 0; i < 90; i++ {
		id := fmt.Sprintf("acme%02d", i)
		name := fmt.Sprintf("Acme Co %02d", i)
		if _, err := db.Exec(`INSERT INTO tenant_company (id, name, creator_id) VALUES (?,?,?)`, id, name, "admin1"); err != nil {
			t.Fatalf("seed %s: %v", id, err)
		}
	}
	rec := getAdminTenants(t, mux, "search=Acme&limit=100&offset=0")
	if rec.Code != 200 {
		t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
	}
	body := parseAdminTenants(t, rec)
	items := tenantItems(t, body)
	if len(items) != 90 {
		t.Fatalf("search must not snap limit>200 to 80; got %d items want 90; total=%v", len(items), body["total"])
	}
	total, _ := body["total"].(float64)
	if int(total) != 90 {
		t.Fatalf("total=%v want 90", body["total"])
	}
}

func TestAdminTenantsMethodNotAllowed(t *testing.T) {
	mux := setupTestService(t)
	req := withPlatformStaff(httptest.NewRequest(http.MethodPost, "/api/system-admin/accounts/admin/tenants/", nil))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST got %d", rec.Code)
	}
}
