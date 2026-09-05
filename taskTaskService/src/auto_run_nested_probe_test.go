package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProbeNestedGitReposAccessPassesTenant(t *testing.T) {
	var gotCompany, gotTenant, gotHeaderTenant, gotUser string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/api/internal/nested-git-repos") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		gotCompany = r.URL.Query().Get("company_id")
		gotTenant = r.URL.Query().Get("tenant_id")
		gotHeaderTenant = r.Header.Get("X-Auth-Tenant-Id")
		gotUser = r.URL.Query().Get("user_id")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"parent_repo_url": r.URL.Query().Get("repo_url"),
			"nested_repos":    []interface{}{},
			"error":           "",
		})
	}))
	t.Cleanup(srv.Close)
	prev := cfg.ProjectServiceURL
	cfg.ProjectServiceURL = srv.URL
	t.Cleanup(func() { cfg.ProjectServiceURL = prev })

	reason := probeNestedGitReposAccess("1001", "877397588196749312", "https://gitlab.example/g/repo.git")
	if reason != "" {
		t.Fatalf("reason=%q want empty", reason)
	}
	if gotUser != "1001" {
		t.Fatalf("user_id=%q", gotUser)
	}
	if gotCompany != "877397588196749312" && gotTenant != "877397588196749312" {
		t.Fatalf("query company_id=%q tenant_id=%q want tenant", gotCompany, gotTenant)
	}
	if gotHeaderTenant != "877397588196749312" {
		t.Fatalf("X-Auth-Tenant-Id=%q", gotHeaderTenant)
	}
}
