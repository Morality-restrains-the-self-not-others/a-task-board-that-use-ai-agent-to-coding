package infrastructure

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNestedGitReposHTTPClient_FetchWhenUserIDZero(t *testing.T) {
	var gotUser string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/internal/nested-git-repos/" {
			http.NotFound(w, r)
			return
		}
		gotUser = r.URL.Query().Get("user_id")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"nested_repos": []map[string]string{
				{"path": "AiMonitor", "url": "https://github.com/task2money/AiMonitor.git", "source": "gitmodules"},
			},
		})
	}))
	t.Cleanup(srv.Close)

	client := NewNestedGitReposHTTPClient(srv.URL, "", 5)
	items, err := client.FetchNestedGitRepos(0, "", "https://github.com/task2money/ram-work.git")
	if err != nil {
		t.Fatalf("FetchNestedGitRepos: %v", err)
	}
	if gotUser != "" {
		t.Fatalf("user_id query must be omitted when userID=0, got %q", gotUser)
	}
	if len(items) != 1 || items[0].Path != "AiMonitor" {
		t.Fatalf("items=%+v", items)
	}
}

// TestNestedGitReposHTTPClient_PassesTenantID（OPT-20260827-019）：
// 子仓发现请求必须把 tenant_id 放进 query 与 X-Auth-Tenant-Id 头，
// 供 taskProjectService 匹配 Path A GitLab / 区域 GitLab 的 OAuth provider。
func TestNestedGitReposHTTPClient_PassesTenantID(t *testing.T) {
	var gotQueryTenant, gotHeaderTenant, gotUser string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQueryTenant = r.URL.Query().Get("tenant_id")
		gotHeaderTenant = r.Header.Get("X-Auth-Tenant-Id")
		gotUser = r.URL.Query().Get("user_id")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"nested_repos": []map[string]string{}})
	}))
	t.Cleanup(srv.Close)

	client := NewNestedGitReposHTTPClient(srv.URL, "secret", 5)
	items, err := client.FetchNestedGitRepos(42, "877397588196749312", "https://gitlab.example/g/ram-work.git")
	if err != nil {
		t.Fatalf("FetchNestedGitRepos: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("items=%+v want empty", items)
	}
	if gotQueryTenant != "877397588196749312" {
		t.Fatalf("tenant_id query = %q, want 877397588196749312", gotQueryTenant)
	}
	if gotHeaderTenant != "877397588196749312" {
		t.Fatalf("X-Auth-Tenant-Id header = %q, want 877397588196749312", gotHeaderTenant)
	}
	if gotUser != "42" {
		t.Fatalf("user_id query = %q, want 42", gotUser)
	}
}

// TestNestedGitReposHTTPClient_OmitsTenantWhenEmpty：租户为空时不带
// tenant_id query / X-Auth-Tenant-Id 头（公开 GitHub 匿名发现兼容）。
func TestNestedGitReposHTTPClient_OmitsTenantWhenEmpty(t *testing.T) {
	var gotQueryTenant, gotHeaderTenant string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQueryTenant = r.URL.Query().Get("tenant_id")
		gotHeaderTenant = r.Header.Get("X-Auth-Tenant-Id")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"nested_repos": []map[string]string{}})
	}))
	t.Cleanup(srv.Close)

	client := NewNestedGitReposHTTPClient(srv.URL, "", 5)
	if _, err := client.FetchNestedGitRepos(0, "", "https://github.com/task2money/ram-work.git"); err != nil {
		t.Fatalf("FetchNestedGitRepos: %v", err)
	}
	if gotQueryTenant != "" {
		t.Fatalf("tenant_id query should be omitted, got %q", gotQueryTenant)
	}
	if gotHeaderTenant != "" {
		t.Fatalf("X-Auth-Tenant-Id header should be omitted, got %q", gotHeaderTenant)
	}
}
