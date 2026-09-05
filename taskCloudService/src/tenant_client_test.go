package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestVerifyCompanyExists_UsesTenantServiceCreator(t *testing.T) {
	var gotPath, gotCompany string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotCompany = r.URL.Query().Get("company_id")
		_ = json.NewEncoder(w).Encode(map[string]any{"found": true, "creator_id": "u1"})
	}))
	t.Cleanup(srv.Close)

	prev := cfg.TaskTenantURL
	cfg.TaskTenantURL = srv.URL
	t.Cleanup(func() { cfg.TaskTenantURL = prev })

	if err := verifyCompanyExists("850256677331562496"); err != nil {
		t.Fatalf("verifyCompanyExists: %v", err)
	}
	if gotPath != "/api/internal/tenant/companies/creator" {
		t.Fatalf("path=%q", gotPath)
	}
	if gotCompany != "850256677331562496" {
		t.Fatalf("company_id=%q", gotCompany)
	}
}

func TestVerifyCompanyExists_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"found": false})
	}))
	t.Cleanup(srv.Close)

	prev := cfg.TaskTenantURL
	cfg.TaskTenantURL = srv.URL
	t.Cleanup(func() { cfg.TaskTenantURL = prev })

	err := verifyCompanyExists("missing-company")
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestVerifyTenantCompanyExistsFP_SkipsWhenTenantURLEmpty(t *testing.T) {
	prev := cfg.TaskTenantURL
	cfg.TaskTenantURL = ""
	t.Cleanup(func() { cfg.TaskTenantURL = prev })

	exists, err := verifyTenantCompanyExistsFP("any")
	if err != nil || !exists {
		t.Fatalf("exists=%v err=%v", exists, err)
	}
}
