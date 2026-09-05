package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestListTenantGitlabLocalRepoURLsFiltersInternal(t *testing.T) {
	setupTestDB(t)
	oldResolver := providerResolver
	providerResolver = &ProviderResolver{
		entries: []providerEntry{
			{
				Provider:        "gitlab",
				ServiceProvider: "gitlab-local",
				ProviderKey:     "gitlab:gitlab-local",
				Host:            "gitlab.daydaymoney.com",
				Netloc:          "gitlab.daydaymoney.com",
			},
		},
	}
	t.Cleanup(func() { providerResolver = oldResolver })

	_, err := db.Exec(`INSERT INTO project_entries(id,name,company_id) VALUES('p1','n','tenant-a')`)
	if err != nil {
		t.Fatalf("insert project: %v", err)
	}
	_, err = db.Exec(`INSERT INTO project_repos(id,project_id,repo_url,created_at,updated_at) VALUES
		('r1','p1','https://gitlab.daydaymoney.com/u/a.git',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
		('r2','p1','https://github.com/o/b.git',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`)
	if err != nil {
		t.Fatalf("insert repos: %v", err)
	}

	urls, err := listTenantGitlabLocalRepoURLs("tenant-a")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(urls) != 1 || !strings.Contains(urls[0], "gitlab.daydaymoney.com") {
		t.Fatalf("urls=%#v", urls)
	}
}

func TestMeasureTenantGitlabLocalDiskUsageSums(t *testing.T) {
	setupTestDB(t)
	clearDiskSizeCache()
	t.Cleanup(clearDiskSizeCache)

	gitlabSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v4/groups/tenant-tenant-b" {
			// Step 1: group lookup
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": 42})
			return
		}
		if strings.Contains(r.URL.Path, "/api/v4/groups/42/projects") {
			// Step 2: list group projects with statistics
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"path_with_namespace": "tenant-tenant-b/r",
					"web_url":             "https://gitlab.daydaymoney.com/tenant-tenant-b/r",
					"statistics":          map[string]interface{}{"repository_size": 4096},
				},
			})
			return
		}
		if !strings.Contains(r.URL.Path, "/api/v4/projects/") || !strings.Contains(r.URL.RawQuery, "statistics=true") {
			t.Fatalf("unexpected url path=%s query=%s", r.URL.Path, r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"statistics": map[string]interface{}{"repository_size": 4096},
		})
	}))
	defer gitlabSrv.Close()
	host := strings.TrimPrefix(gitlabSrv.URL, "http://")

	oldAPI := cfg.GitlabAPIBase
	cfg.GitlabAPIBase = gitlabSrv.URL
	t.Cleanup(func() { cfg.GitlabAPIBase = oldAPI })

	oldResolver := providerResolver
	providerResolver = &ProviderResolver{
		entries: []providerEntry{
			{
				Provider:        "gitlab",
				ServiceProvider: "gitlab-local",
				ProviderKey:     "gitlab:gitlab-local",
				Host:            "gitlab.daydaymoney.com",
				Netloc:          "gitlab.daydaymoney.com",
			},
		},
	}
	t.Cleanup(func() { providerResolver = oldResolver })

	_, _ = db.Exec(`INSERT INTO project_entries(id,name,company_id) VALUES('p1','n','tenant-b')`)
	repoURL := "https://gitlab.daydaymoney.com/g/r.git"
	_, _ = db.Exec(`INSERT INTO project_repos(id,project_id,repo_url,created_at,updated_at) VALUES(?,?,?,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`,
		"r1", "p1", repoURL)

	got, err := measureTenantGitlabLocalDiskUsage("tenant-b", "admin-tok")
	if err != nil {
		t.Fatalf("measure: %v", err)
	}
	if got.DiskUsedBytes != 4096 || got.MeasuredCount != 1 || got.RepoCount != 1 {
		t.Fatalf("got=%+v host=%s", got, host)
	}
}

func TestHandleInternalTenantGitlabLocalDiskUsage(t *testing.T) {
	setupTestDB(t)
	oldSecret := cfg.InternalSecret
	cfg.InternalSecret = "sec"
	t.Cleanup(func() { cfg.InternalSecret = oldSecret })

	req := httptest.NewRequest(http.MethodGet, "/api/internal/tenants/tenant-x/gitlab-local-disk-usage/", nil)
	rec := httptest.NewRecorder()
	handleInternalTenantGitlabLocalDiskUsage(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("want 403 without secret, got %d", rec.Code)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/api/internal/tenants/tenant-x/gitlab-local-disk-usage/", nil)
	req2.Header.Set("X-Internal-Secret", "sec")
	rec2 := httptest.NewRecorder()
	handleInternalTenantGitlabLocalDiskUsage(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rec2.Code, rec2.Body.String())
	}
	var body TenantDiskUsageResult
	if err := json.NewDecoder(rec2.Body).Decode(&body); err != nil {
		t.Fatalf("json: %v", err)
	}
	if body.TenantID != "tenant-x" || body.DiskUsedBytes != 0 {
		t.Fatalf("body=%+v", body)
	}
	_ = time.Second
}
