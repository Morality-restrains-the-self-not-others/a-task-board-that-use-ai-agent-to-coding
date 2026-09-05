package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGroupActiveTenantCompaniesSkipsInactiveAndSorts(t *testing.T) {
	got := groupActiveTenantCompanies([]tenantMemberHit{
		{UserID: "u1", CompanyID: "c2", CompanyName: "Zeta", IsActive: true},
		{UserID: "u1", CompanyID: "c1", CompanyName: "Acme", IsActive: true},
		{UserID: "u1", CompanyID: "c9", CompanyName: "Dead Co", IsActive: false},
		{UserID: "u2", CompanyID: "c3", CompanyName: "", IsActive: true},
		{UserID: "u1", CompanyID: "c1", CompanyName: "Acme Dup", IsActive: true},
	})
	u1 := got["u1"]
	if len(u1) != 2 {
		t.Fatalf("u1 companies=%v", u1)
	}
	if u1[0]["id"] != "c1" || u1[0]["name"] != "Acme" {
		t.Fatalf("first should be Acme, got %v", u1[0])
	}
	if u1[1]["id"] != "c2" || u1[1]["name"] != "Zeta" {
		t.Fatalf("second should be Zeta, got %v", u1[1])
	}
	u2 := got["u2"]
	if len(u2) != 1 || u2[0]["id"] != "c3" || u2[0]["name"] != "c3" {
		t.Fatalf("empty name should fall back to id, got %v", u2)
	}
	if _, ok := got["u3"]; ok {
		t.Fatalf("unexpected u3")
	}
}

func TestGroupActiveTenantCompaniesEmpty(t *testing.T) {
	got := groupActiveTenantCompanies(nil)
	if len(got) != 0 {
		t.Fatalf("expected empty, got %v", got)
	}
}

func TestFetchTenantCompaniesBatchPostsUserIDs(t *testing.T) {
	var gotBody struct {
		UserIDs []string `json:"user_ids"`
	}
	var gotSecret string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/internal/tenant/members/batch-get/" && r.URL.Path != "/api/internal/tenant/members/batch-get" {
			http.NotFound(w, r)
			return
		}
		gotSecret = r.Header.Get("X-Internal-Secret")
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"members": []map[string]interface{}{
				{"user_id": "u1", "company_id": "c1", "company_name": "Acme", "is_active": true},
				{"user_id": "u1", "company_id": "c2", "company_name": "Beta", "is_active": false},
			},
		})
	}))
	defer srv.Close()
	prevURL := cfg.TenantServiceURL
	prevSec := cfg.InternalSecret
	cfg.TenantServiceURL = srv.URL
	cfg.InternalSecret = "tenant-secret"
	t.Cleanup(func() {
		cfg.TenantServiceURL = prevURL
		cfg.InternalSecret = prevSec
	})

	got := fetchTenantCompaniesBatch(t.Context(), []string{"u1", "u2"})
	if len(gotBody.UserIDs) != 2 || gotBody.UserIDs[0] != "u1" || gotBody.UserIDs[1] != "u2" {
		t.Fatalf("posted user_ids=%v", gotBody.UserIDs)
	}
	if gotSecret != "tenant-secret" {
		t.Fatalf("expected internal secret header, got %q", gotSecret)
	}
	u1 := got["u1"]
	if len(u1) != 1 || u1[0]["id"] != "c1" || u1[0]["name"] != "Acme" {
		t.Fatalf("expected only active Acme, got %v", u1)
	}
}

func TestFetchTenantCompaniesBatchEmptyInput(t *testing.T) {
	got := fetchTenantCompaniesBatch(t.Context(), nil)
	if len(got) != 0 {
		t.Fatalf("expected empty, got %v", got)
	}
}

func TestFetchTenantCompaniesBatchDoesNotFailOnDownService(t *testing.T) {
	prev := cfg.TenantServiceURL
	cfg.TenantServiceURL = "http://127.0.0.1:1"
	t.Cleanup(func() { cfg.TenantServiceURL = prev })
	got := fetchTenantCompaniesBatch(t.Context(), []string{"u1"})
	if len(got) != 0 {
		t.Fatalf("expected empty on down service, got %v", got)
	}
}

func TestFetchTenantCompaniesBatchEmptyURL(t *testing.T) {
	prev := cfg.TenantServiceURL
	cfg.TenantServiceURL = ""
	t.Cleanup(func() { cfg.TenantServiceURL = prev })
	got := fetchTenantCompaniesBatch(t.Context(), []string{"u1"})
	if len(got) != 0 {
		t.Fatalf("expected empty when URL unset, got %v", got)
	}
}
