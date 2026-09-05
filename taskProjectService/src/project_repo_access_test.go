package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClassifyRepoAccessAfterProbeNeedsAuthWithoutToken(t *testing.T) {
	got := classifyRepoAccessAfterProbe(
		false,
		"GitHub repository not found",
		true,
		false,
		"",
		tokenStatusNotBound,
		"github",
		"github-official",
	)
	if got.AccessStatus != accessStatusNeedsAuth {
		t.Fatalf("access_status=%q want needs_auth", got.AccessStatus)
	}
	if got.IsAccessible {
		t.Fatal("expected not accessible")
	}
}

func TestClassifyRepoAccessAfterProbeNotAccessibleWithToken(t *testing.T) {
	got := classifyRepoAccessAfterProbe(
		false,
		"GitHub repository not found",
		true,
		true,
		"",
		tokenStatusAvailable,
		"github",
		"github-official",
	)
	if got.AccessStatus != accessStatusNotAccessible {
		t.Fatalf("access_status=%q want not_accessible", got.AccessStatus)
	}
}

func TestCheckProjectPrimaryRepoAccessEmpty(t *testing.T) {
	got := checkProjectPrimaryRepoAccess("1", nil, "", "")
	if got.AccessStatus != accessStatusNotConfigured {
		t.Fatalf("access_status=%q", got.AccessStatus)
	}
}

func TestRepoAccessCheckGoNative(t *testing.T) {
	setupTestDB(t)

	githubAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/o/private" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer githubAPI.Close()
	oldBase := githubAPIBase
	githubAPIBase = githubAPI.URL
	t.Cleanup(func() { githubAPIBase = oldBase })

	gitoauthSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// not bound
		writeJSON(w, 404, map[string]interface{}{"error": "not bound"})
	}))
	defer gitoauthSrv.Close()
	oldGitoauth := cfg.GitoauthBaseURL
	cfg.GitoauthBaseURL = gitoauthSrv.URL
	t.Cleanup(func() { cfg.GitoauthBaseURL = oldGitoauth })

	oldResolver := providerResolver
	providerResolver = &ProviderResolver{}
	t.Cleanup(func() { providerResolver = oldResolver })

	body := `{"name":"repo-access-go","git_repos":["https://github.com/o/private.git"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/projects/", strings.NewReader(body))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	rec := httptest.NewRecorder()
	handleCreateProject(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var created map[string]interface{}
	_ = json.NewDecoder(rec.Body).Decode(&created)
	pid, _ := created["id"].(string)
	if pid == "" {
		t.Fatalf("missing project id: %#v", created)
	}

	reqCheck := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/projects/"+pid+"/repo-access-check/", nil)
	reqCheck.Header.Set("X-Auth-Tenant-Id", "t1")
	reqCheck.Header.Set("X-Auth-User-Id", "1001")
	recCheck := httptest.NewRecorder()
	handleRepoAccessCheck(recCheck, reqCheck, "t1", pid)

	if recCheck.Code != http.StatusOK {
		t.Fatalf("repo-access-check: expected 200, got %d: %s", recCheck.Code, recCheck.Body.String())
	}
	var resp map[string]interface{}
	_ = json.NewDecoder(recCheck.Body).Decode(&resp)
	if resp["access_status"] != accessStatusNeedsAuth {
		t.Fatalf("access_status = %#v, want needs_auth; body=%s", resp["access_status"], recCheck.Body.String())
	}
	if resp["is_accessible"] != false {
		t.Fatalf("is_accessible=%#v", resp["is_accessible"])
	}
	if resp["token_status"] != tokenStatusNotBound && resp["token_status"] != tokenStatusTokenError {
		// 404 from gitoauth may surface as token_error depending on client
		t.Fatalf("token_status=%#v", resp["token_status"])
	}
	if resp["oauth_provider"] != "github" {
		t.Fatalf("oauth_provider=%#v", resp["oauth_provider"])
	}
}

func TestRepoAccessCheckGoNativeAccessible(t *testing.T) {
	setupTestDB(t)

	githubAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":1}`))
	}))
	defer githubAPI.Close()
	oldBase := githubAPIBase
	githubAPIBase = githubAPI.URL
	t.Cleanup(func() { githubAPIBase = oldBase })

	gitoauthSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]interface{}{"access_token": "tok"})
	}))
	defer gitoauthSrv.Close()
	oldGitoauth := cfg.GitoauthBaseURL
	cfg.GitoauthBaseURL = gitoauthSrv.URL
	t.Cleanup(func() { cfg.GitoauthBaseURL = oldGitoauth })

	oldResolver := providerResolver
	providerResolver = &ProviderResolver{}
	t.Cleanup(func() { providerResolver = oldResolver })

	body := `{"name":"repo-access-ok","git_repos":["https://github.com/o/public.git"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/projects/", strings.NewReader(body))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	rec := httptest.NewRecorder()
	handleCreateProject(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rec.Code, rec.Body.String())
	}
	var created map[string]interface{}
	_ = json.NewDecoder(rec.Body).Decode(&created)
	pid, _ := created["id"].(string)

	reqCheck := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/projects/"+pid+"/repo-access-check/", nil)
	reqCheck.Header.Set("X-Auth-Tenant-Id", "t1")
	reqCheck.Header.Set("X-Auth-User-Id", "1001")
	recCheck := httptest.NewRecorder()
	handleRepoAccessCheck(recCheck, reqCheck, "t1", pid)
	if recCheck.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recCheck.Code, recCheck.Body.String())
	}
	var resp map[string]interface{}
	_ = json.NewDecoder(recCheck.Body).Decode(&resp)
	if resp["access_status"] != accessStatusAccessible || resp["is_accessible"] != true {
		t.Fatalf("resp=%#v", resp)
	}
}
