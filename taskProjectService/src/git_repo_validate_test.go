package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestShouldDowngradeTokenAvailableOnInaccessibleProbe(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name          string
		accessible    bool
		tokenStatus   string
		message       string
		wantDowngrade bool
	}{
		{
			name:          "github 404 other-owner private repo",
			accessible:    false,
			tokenStatus:   tokenStatusAvailable,
			message:       "GitHub repository not found",
			wantDowngrade: true,
		},
		{
			name:          "github 404 with status in message",
			accessible:    false,
			tokenStatus:   tokenStatusAvailable,
			message:       "GitHub repository not found（404）：Not Found",
			wantDowngrade: true,
		},
		{
			name:          "github 403 org/sso",
			accessible:    false,
			tokenStatus:   tokenStatusAvailable,
			message:       "GitHub API 访问被拒绝（403）：Resource protected by organization SAML",
			wantDowngrade: true,
		},
		{
			name:          "github 401 expired token",
			accessible:    false,
			tokenStatus:   tokenStatusAvailable,
			message:       "GitHub API 拒绝访问（401）：Bad credentials",
			wantDowngrade: true,
		},
		{
			name:          "gitlab 404",
			accessible:    false,
			tokenStatus:   tokenStatusAvailable,
			message:       "GitLab repository not found",
			wantDowngrade: true,
		},
		{
			name:          "rate limit must not look like missing repo auth",
			accessible:    false,
			tokenStatus:   tokenStatusAvailable,
			message:       "GitHub API 限流（403）：API rate limit exceeded",
			wantDowngrade: false,
		},
		{
			name:          "github 5xx keeps token_available",
			accessible:    false,
			tokenStatus:   tokenStatusAvailable,
			message:       "GitHub API error: 502",
			wantDowngrade: false,
		},
		{
			name:          "already accessible",
			accessible:    true,
			tokenStatus:   tokenStatusAvailable,
			message:       "",
			wantDowngrade: false,
		},
		{
			name:          "not bound stays not bound",
			accessible:    false,
			tokenStatus:   tokenStatusNotBound,
			message:       "GitHub repository not found",
			wantDowngrade: false,
		},
		{
			name:          "empty probe message is inconclusive",
			accessible:    false,
			tokenStatus:   tokenStatusAvailable,
			message:       "",
			wantDowngrade: false,
		},
		{
			name:          "public other-owner repo without push",
			accessible:    false,
			tokenStatus:   tokenStatusAvailable,
			message:       "当前 GitHub 授权对该仓库没有 push 权限",
			wantDowngrade: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := shouldDowngradeTokenAvailableOnInaccessibleProbe(tc.accessible, tc.tokenStatus, tc.message)
			if got != tc.wantDowngrade {
				t.Fatalf("got %v want %v status=%s msg=%q", got, tc.wantDowngrade, tc.tokenStatus, tc.message)
			}
		})
	}
}

func TestValidateGitRepoForUserProbe404DowngradesTokenAvailable(t *testing.T) {
	oldResolver := providerResolver
	providerResolver = &ProviderResolver{}
	t.Cleanup(func() { providerResolver = oldResolver })

	githubAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/test-ruandao/helloworld" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer githubAPI.Close()
	oldBase := githubAPIBase
	githubAPIBase = githubAPI.URL
	t.Cleanup(func() { githubAPIBase = oldBase })

	gitoauthSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]interface{}{"access_token": "ghu_other_owner"})
	}))
	defer gitoauthSrv.Close()
	oldGitoauth := cfg.GitoauthBaseURL
	cfg.GitoauthBaseURL = gitoauthSrv.URL
	t.Cleanup(func() { cfg.GitoauthBaseURL = oldGitoauth })

	got := validateGitRepoForUser("1001", "https://github.com/test-ruandao/helloworld.git", true, "", "")
	if got.TokenStatus != tokenStatusTokenError {
		t.Fatalf("token_status=%q want token_error (switched to another person's repo)", got.TokenStatus)
	}
	if got.IsAccessible {
		t.Fatal("expected not accessible")
	}
	if !strings.Contains(strings.ToLower(got.Message), "not found") {
		t.Fatalf("message=%q", got.Message)
	}
}

func TestValidateGitRepoForUserPublicRepoWithoutPushDowngrades(t *testing.T) {
	oldResolver := providerResolver
	providerResolver = &ProviderResolver{}
	t.Cleanup(func() { providerResolver = oldResolver })

	githubAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/test-ruandao/helloworld" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer ghu_other_owner" {
			t.Fatalf("Authorization=%q", r.Header.Get("Authorization"))
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":1,"permissions":{"admin":false,"maintain":false,"push":false,"pull":true}}`))
	}))
	defer githubAPI.Close()
	oldBase := githubAPIBase
	githubAPIBase = githubAPI.URL
	t.Cleanup(func() { githubAPIBase = oldBase })

	gitoauthSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]interface{}{"access_token": "ghu_other_owner"})
	}))
	defer gitoauthSrv.Close()
	oldGitoauth := cfg.GitoauthBaseURL
	cfg.GitoauthBaseURL = gitoauthSrv.URL
	t.Cleanup(func() { cfg.GitoauthBaseURL = oldGitoauth })

	got := validateGitRepoForUser("1001", "https://github.com/test-ruandao/helloworld.git", true, "", "")
	if got.TokenStatus != tokenStatusTokenError {
		t.Fatalf("token_status=%q want token_error for other-owner public repo without push", got.TokenStatus)
	}
	if got.IsAccessible {
		t.Fatal("expected not accessible without push permission")
	}
	if !strings.Contains(got.Message, "push") {
		t.Fatalf("message=%q", got.Message)
	}
}

func TestValidateGitRepoForUserPushPermissionKeepsAvailable(t *testing.T) {
	oldResolver := providerResolver
	providerResolver = &ProviderResolver{}
	t.Cleanup(func() { providerResolver = oldResolver })

	githubAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":1,"permissions":{"admin":false,"push":true,"pull":true}}`))
	}))
	defer githubAPI.Close()
	oldBase := githubAPIBase
	githubAPIBase = githubAPI.URL
	t.Cleanup(func() { githubAPIBase = oldBase })

	gitoauthSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]interface{}{"access_token": "ghu_own"})
	}))
	defer gitoauthSrv.Close()
	oldGitoauth := cfg.GitoauthBaseURL
	cfg.GitoauthBaseURL = gitoauthSrv.URL
	t.Cleanup(func() { cfg.GitoauthBaseURL = oldGitoauth })

	got := validateGitRepoForUser("1001", "https://github.com/me/mine.git", true, "", "")
	if got.TokenStatus != tokenStatusAvailable {
		t.Fatalf("token_status=%q want token_available", got.TokenStatus)
	}
	if !got.IsAccessible {
		t.Fatal("expected accessible with push")
	}
}

func TestValidateGitRepoForUserTokenOnlyKeepsAvailableWithoutProbe(t *testing.T) {
	oldResolver := providerResolver
	providerResolver = &ProviderResolver{}
	t.Cleanup(func() { providerResolver = oldResolver })

	gitoauthSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]interface{}{"access_token": "ghu_ok"})
	}))
	defer gitoauthSrv.Close()
	oldGitoauth := cfg.GitoauthBaseURL
	cfg.GitoauthBaseURL = gitoauthSrv.URL
	t.Cleanup(func() { cfg.GitoauthBaseURL = oldGitoauth })

	got := validateGitRepoForUser("1001", "https://github.com/test-ruandao/helloworld.git", false, "", "")
	if got.TokenStatus != tokenStatusAvailable {
		t.Fatalf("token-only path token_status=%q want token_available", got.TokenStatus)
	}
}

func TestEnrichProjectGitReposStatusProbesAndDowngradesInaccessible(t *testing.T) {
	oldResolver := providerResolver
	providerResolver = &ProviderResolver{}
	t.Cleanup(func() { providerResolver = oldResolver })

	githubAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer githubAPI.Close()
	oldBase := githubAPIBase
	githubAPIBase = githubAPI.URL
	t.Cleanup(func() { githubAPIBase = oldBase })

	gitoauthSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]interface{}{"access_token": "ghu_ok"})
	}))
	defer gitoauthSrv.Close()
	oldGitoauth := cfg.GitoauthBaseURL
	cfg.GitoauthBaseURL = gitoauthSrv.URL
	t.Cleanup(func() { cfg.GitoauthBaseURL = oldGitoauth })

	detail := map[string]interface{}{
		"git_repos": []string{"https://github.com/test-ruandao/helloworld.git"},
	}
	enrichProjectGitReposStatus(detail, "1001", "")
	raw, ok := detail["git_repos_status"].([]map[string]interface{})
	if !ok || len(raw) != 1 {
		t.Fatalf("git_repos_status=%#v", detail["git_repos_status"])
	}
	if raw[0]["token_status"] != tokenStatusTokenError {
		t.Fatalf("token_status=%#v want token_error", raw[0]["token_status"])
	}
}
