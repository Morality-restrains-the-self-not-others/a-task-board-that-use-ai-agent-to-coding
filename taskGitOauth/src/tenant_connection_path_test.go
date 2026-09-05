package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExtractTenantIDFromPath(t *testing.T) {
	cases := []struct {
		name string
		path string
		want string
		ok   bool // isTenantGitlabConnectionPath
	}{
		{
			name: "canonical reachability",
			path: "/api/git-oauth/tenant-connection/tenant_id/874599492341493760/reachability/",
			want: "874599492341493760",
			ok:   true,
		},
		{
			name: "canonical without trailing slash",
			path: "/api/git-oauth/tenant-connection/tenant_id/company-42",
			want: "company-42",
			ok:   true,
		},
		{
			name: "legacy positional",
			path: "/api/git-oauth/tenant-connection/company-42/gitlab-oauth-connection/",
			want: "company-42",
			ok:   true,
		},
		{
			name: "tenant_id segment then gitlab suffix still canonical",
			path: "/api/git-oauth/tenant-connection/tenant_id/874599492341493760/gitlab-oauth-connection/",
			want: "874599492341493760",
			ok:   true,
		},
		{
			name: "missing tid after tenant_id",
			path: "/api/git-oauth/tenant-connection/tenant_id/",
			want: "",
			ok:   false,
		},
		{
			name: "prefix only",
			path: "/api/git-oauth/tenant-connection/",
			want: "",
			ok:   false,
		},
		{
			name: "wrong service",
			path: "/api/other/tenant-connection/tenant_id/1/",
			want: "",
			ok:   false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isTenantGitlabConnectionPath(tc.path); got != tc.ok {
				t.Fatalf("isTenantGitlabConnectionPath=%v want %v", got, tc.ok)
			}
			if got := extractTenantIDFromPath(tc.path); got != tc.want {
				t.Fatalf("extractTenantIDFromPath=%q want %q", got, tc.want)
			}
		})
	}
}

func TestTenantGitlabConnectionCanonicalPathGet(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	tid := "company-kv"
	path := "/api/git-oauth/tenant-connection/tenant_id/" + tid + "/"

	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("X-User-Id", "u1")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("canonical GET status %d body %s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["configured"] != false {
		t.Fatalf("configured=%v", body["configured"])
	}
	if body["company_id"] != tid {
		t.Fatalf("company_id=%v", body["company_id"])
	}
}

func TestIsTenantGitlabReachabilityPath(t *testing.T) {
	if !isTenantGitlabReachabilityPath("/api/git-oauth/tenant-connection/tenant_id/1/reachability/") {
		t.Fatal("canonical reachability")
	}
	if isTenantGitlabReachabilityPath("/api/git-oauth/tenant-connection/tenant_id/1/") {
		t.Fatal("connection CRUD is not reachability")
	}
}
