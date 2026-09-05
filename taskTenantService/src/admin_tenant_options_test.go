package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func withPlatformStaff(req *http.Request) *http.Request {
	req.Header.Set("X-Gateway-Auth-Verified", "1")
	req.Header.Set("X-User-Id", "staff1")
	req.Header.Set("X-User-Roles", "super_admin")
	return req
}

func getTenantOptions(t *testing.T, mux *http.ServeMux, search string) *httptest.ResponseRecorder {
	t.Helper()
	path := "/api/system-admin/accounts/admin/tenant-options/"
	if search != "" {
		path += "?search=" + search
	}
	req := withPlatformStaff(httptest.NewRequest(http.MethodGet, path, nil))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func parseTenantOptions(t *testing.T, rec *httptest.ResponseRecorder) []map[string]interface{} {
	t.Helper()
	var out []map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode %d %s: %v", rec.Code, rec.Body.String(), err)
	}
	return out
}

func TestAdminTenantOptionsRequiresAuth(t *testing.T) {
	mux := setupTestService(t)
	req := httptest.NewRequest(http.MethodGet, "/api/system-admin/accounts/admin/tenant-options/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAdminTenantOptionsRequiresPlatformStaff(t *testing.T) {
	mux := setupTestService(t)
	req := httptest.NewRequest(http.MethodGet, "/api/system-admin/accounts/admin/tenant-options/", nil)
	req.Header.Set("X-Gateway-Auth-Verified", "1")
	req.Header.Set("X-User-Id", "member1")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("non-staff got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAdminTenantOptionsIncludesEmailPhone(t *testing.T) {
	mux := setupTestService(t)
	prev := fetchCreatorContactsFn
	fetchCreatorContactsFn = func(userIDs []string) map[string]creatorContact {
		return map[string]creatorContact{
			"admin1": {Email: "owner@example.com", Phone: "13800138000"},
		}
	}
	t.Cleanup(func() { fetchCreatorContactsFn = prev })

	rec := getTenantOptions(t, mux, "")
	if rec.Code != 200 {
		t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
	}
	out := parseTenantOptions(t, rec)
	if len(out) == 0 {
		t.Fatal("expected companies")
	}
	found := false
	for _, item := range out {
		if item["id"] != "c1" {
			continue
		}
		found = true
		if item["email"] != "owner@example.com" || item["phone"] != "13800138000" {
			t.Fatalf("c1 contacts=%v", item)
		}
		if item["name"] != "Test Co" {
			t.Fatalf("name=%v", item["name"])
		}
	}
	if !found {
		t.Fatalf("missing c1: %v", out)
	}
}

func TestAdminTenantOptionsEmptyContactsAreBlank(t *testing.T) {
	mux := setupTestService(t)
	prev := fetchCreatorContactsFn
	fetchCreatorContactsFn = func(userIDs []string) map[string]creatorContact {
		return map[string]creatorContact{}
	}
	t.Cleanup(func() { fetchCreatorContactsFn = prev })

	out := parseTenantOptions(t, getTenantOptions(t, mux, ""))
	for _, item := range out {
		if item["id"] != "c1" {
			continue
		}
		if item["email"] != "" || item["phone"] != "" {
			t.Fatalf("want empty contacts, got %v", item)
		}
		return
	}
	t.Fatal("missing c1")
}

func TestAdminTenantOptionsSearchByCompanyID(t *testing.T) {
	mux := setupTestService(t)
	prevC, prevS := fetchCreatorContactsFn, searchCreatorIDsFn
	fetchCreatorContactsFn = func(userIDs []string) map[string]creatorContact { return map[string]creatorContact{} }
	searchCreatorIDsFn = func(query string) []string { return nil }
	t.Cleanup(func() {
		fetchCreatorContactsFn = prevC
		searchCreatorIDsFn = prevS
	})
	out := parseTenantOptions(t, getTenantOptions(t, mux, "c1"))
	if len(out) != 1 || out[0]["id"] != "c1" {
		t.Fatalf("id search got %v", out)
	}
}

func TestAdminTenantOptionsSearchByCreatorContact(t *testing.T) {
	mux := setupTestService(t)
	if _, err := db.Exec(`INSERT INTO tenant_company (id, name, creator_id) VALUES ('c-phone','我的公司','creator-b')`); err != nil {
		t.Fatalf("seed: %v", err)
	}
	prevC, prevS := fetchCreatorContactsFn, searchCreatorIDsFn
	searchCreatorIDsFn = func(query string) []string {
		if query == "13900001111" {
			return []string{"creator-b"}
		}
		return nil
	}
	fetchCreatorContactsFn = func(userIDs []string) map[string]creatorContact {
		return map[string]creatorContact{
			"creator-b": {Email: "b@example.com", Phone: "13900001111"},
		}
	}
	t.Cleanup(func() {
		fetchCreatorContactsFn = prevC
		searchCreatorIDsFn = prevS
	})
	out := parseTenantOptions(t, getTenantOptions(t, mux, "13900001111"))
	if len(out) != 1 || out[0]["id"] != "c-phone" {
		t.Fatalf("phone search got %v", out)
	}
	if out[0]["phone"] != "13900001111" || out[0]["email"] != "b@example.com" {
		t.Fatalf("contacts=%v", out[0])
	}
}

func TestAdminTenantOptionsAuthFailOpen(t *testing.T) {
	mux := setupTestService(t)
	prev := fetchCreatorContactsFn
	fetchCreatorContactsFn = fetchCreatorContactsFromAuth
	t.Cleanup(func() { fetchCreatorContactsFn = prev })
	oldURL := cfg.TaskAuthURL
	cfg.TaskAuthURL = "http://127.0.0.1:1"
	t.Cleanup(func() { cfg.TaskAuthURL = oldURL })

	rec := getTenantOptions(t, mux, "")
	if rec.Code != 200 {
		t.Fatalf("fail-open status %d body=%s", rec.Code, rec.Body.String())
	}
	out := parseTenantOptions(t, rec)
	if len(out) == 0 || out[0]["id"] != "c1" {
		t.Fatalf("fail-open should still list companies: %v", out)
	}
	if out[0]["email"] != "" || out[0]["phone"] != "" {
		t.Fatalf("fail-open contacts should be empty: %v", out[0])
	}
}

func TestFetchCreatorContactsFromAuthParsesPhone(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if r.Header.Get("X-TaskAuth-Internal-Secret") != "sec" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"results": map[string]interface{}{
				"u1": map[string]string{"email": "a@x.com", "phone": "13700000000"},
			},
		})
	}))
	t.Cleanup(srv.Close)
	oldURL, oldSecret := cfg.TaskAuthURL, cfg.InternalSecret
	cfg.TaskAuthURL = srv.URL
	cfg.InternalSecret = "sec"
	t.Cleanup(func() {
		cfg.TaskAuthURL = oldURL
		cfg.InternalSecret = oldSecret
	})
	got := fetchCreatorContactsFromAuth([]string{"u1"})
	if gotPath != "/api/internal/users/batch/details/" {
		t.Fatalf("path=%s", gotPath)
	}
	if got["u1"].Email != "a@x.com" || got["u1"].Phone != "13700000000" {
		t.Fatalf("got=%v", got)
	}
}
