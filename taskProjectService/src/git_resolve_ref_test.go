package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestIsCommitHashLike(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"", false},
		{"main", false},
		{"develop", false},
		{"abc123", false}, // 6 chars
		{"abc1234", true},
		{"ABCDEF1", true},
		{"0123456789abcdef0123456789abcdef01234567", true},   // 40
		{"0123456789abcdef0123456789abcdef012345678", false}, // 41
		{"gbc1234", false},
		{" abc1234 ", true},
	}
	for _, tc := range cases {
		if got := isCommitHashLike(tc.in); got != tc.want {
			t.Errorf("isCommitHashLike(%q)=%v want %v", tc.in, got, tc.want)
		}
	}
}

func TestResolveGitLabCommitExists(t *testing.T) {
	initProviderTestConfig(t)
	gitlabSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/repository/commits/") {
			t.Errorf("path = %q", r.URL.Path)
		}
		if !strings.Contains(r.URL.Path, "abc1234") {
			t.Errorf("expected sha in path, got %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"id":"abc1234deadbeef0123456789abcdef01234567"}`))
	}))
	defer gitlabSrv.Close()

	gitoauthSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"detail":"not bound"}`))
	}))
	defer gitoauthSrv.Close()

	oldBase := cfg.GitoauthBaseURL
	cfg.GitoauthBaseURL = gitoauthSrv.URL
	t.Cleanup(func() { cfg.GitoauthBaseURL = oldBase })
	providerResolver = &ProviderResolver{}

	repoURL := strings.TrimSuffix(gitlabSrv.URL, "/") + "/group-a/demo-repo.git"
	payload := resolveProjectRepoRef("1001", repoURL, "abc1234", "sess", "")
	if !payload.Exists {
		t.Fatalf("expected exists, got %+v", payload)
	}
	if payload.SHA != "abc1234deadbeef0123456789abcdef01234567" {
		t.Errorf("sha = %q", payload.SHA)
	}
}

func TestResolveGitLabCommitNotFound(t *testing.T) {
	initProviderTestConfig(t)
	gitlabSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"404 Commit Not Found"}`))
	}))
	defer gitlabSrv.Close()

	providerResolver = &ProviderResolver{}
	repoURL := strings.TrimSuffix(gitlabSrv.URL, "/") + "/group-a/demo-repo.git"
	payload := resolveProjectRepoRef("1001", repoURL, "deadbeef", "sess", "")
	if payload.Exists {
		t.Fatalf("expected not exists, got %+v", payload)
	}
	if payload.Error != "" {
		t.Errorf("expected empty error for not_found, got %q", payload.Error)
	}
}

func TestResolveRefRejectsNonHash(t *testing.T) {
	payload := resolveProjectRepoRef("1001", "http://example.com/a/b.git", "main", "", "")
	if payload.Exists {
		t.Fatal("expected not exists")
	}
	if !strings.Contains(payload.Error, "commit hash") {
		t.Errorf("error = %q", payload.Error)
	}
}

func TestHandleProjectResolveRefRoute(t *testing.T) {
	setupTestDB(t)

	body := `{"name":"Resolve Ref Project"}`
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/projects/tenant_id/t1/", strings.NewReader(body))
	reqCreate.Header.Set("X-Auth-Tenant-Id", "t1")
	recCreate := httptest.NewRecorder()
	handleCreateProject(recCreate, reqCreate)
	var created map[string]interface{}
	_ = json.NewDecoder(recCreate.Body).Decode(&created)
	pid := created["id"].(string)

	gitlabSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id":"abcdef0123456789abcdef0123456789abcdef01"}`))
	}))
	defer gitlabSrv.Close()

	providerResolver = &ProviderResolver{}
	repoURL := strings.TrimSuffix(gitlabSrv.URL, "/") + "/g/r.git"

	mux := http.NewServeMux()
	mountRoutes(mux)
	q := url.Values{}
	q.Set("repo_url", repoURL)
	q.Set("ref", "abcdef0")
	req := httptest.NewRequest(http.MethodGet, "/api/projects/"+pid+"/resolve-ref/tenant_id/t1/?"+q.Encode(), nil)
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "1001")
	req.AddCookie(&http.Cookie{Name: "_gitlab_session", Value: "s1"})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var payload resolveRefPayload
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !payload.Exists {
		t.Fatalf("expected exists, got %+v", payload)
	}
}
