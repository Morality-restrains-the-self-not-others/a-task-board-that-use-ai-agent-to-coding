package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseGitHubAPIMessageFromJSON(t *testing.T) {
	msg := parseGitHubAPIMessage([]byte(`{"message":"Resource protected by organization SAML enforcement. You must grant your OAuth token access to this organization.","documentation_url":"https://docs.github.com/"}`))
	if !strings.Contains(msg, "SAML enforcement") {
		t.Fatalf("msg=%q", msg)
	}
}

func TestParseGitHubAPIMessagePlainTextFallback(t *testing.T) {
	msg := parseGitHubAPIMessage([]byte("  plain failure text  "))
	if msg != "plain failure text" {
		t.Fatalf("msg=%q", msg)
	}
}

func TestFormatGitHubAPIErrorForbiddenIncludesProviderMessage(t *testing.T) {
	body := []byte(`{"message":"Resource protected by organization SAML enforcement. You must grant your OAuth token access to this organization."}`)
	got := formatGitHubAPIError(http.StatusForbidden, body)
	if !strings.Contains(got, "SAML enforcement") {
		t.Fatalf("missing GitHub message: %q", got)
	}
	if !strings.Contains(got, "403") {
		t.Fatalf("missing status: %q", got)
	}
	if !strings.Contains(got, "SSO") && !strings.Contains(got, "组织") {
		t.Fatalf("missing SSO/org guidance: %q", got)
	}
}

func TestFormatGitHubAPIErrorForbiddenRateLimit(t *testing.T) {
	body := []byte(`{"message":"API rate limit exceeded for user ID 1."}`)
	got := formatGitHubAPIError(http.StatusForbidden, body)
	if !strings.Contains(got, "rate limit exceeded") {
		t.Fatalf("missing rate limit message: %q", got)
	}
	if !strings.Contains(got, "限流") {
		t.Fatalf("missing rate-limit framing: %q", got)
	}
}

func TestFormatGitHubAPIErrorForbiddenGenericProviderMessage(t *testing.T) {
	body := []byte(`{"message":"Resource not accessible by integration"}`)
	got := formatGitHubAPIError(http.StatusForbidden, body)
	if !strings.Contains(got, "Resource not accessible by integration") {
		t.Fatalf("missing GitHub message: %q", got)
	}
	if strings.Contains(got, "若为组织仓库可能需要 SSO") {
		t.Fatalf("should not append SSO hint for non-SSO errors: %q", got)
	}
}

func TestFormatGitHubAPIErrorUnauthorizedIncludesProviderMessage(t *testing.T) {
	body := []byte(`{"message":"Bad credentials"}`)
	got := formatGitHubAPIError(http.StatusUnauthorized, body)
	if !strings.Contains(got, "Bad credentials") {
		t.Fatalf("missing GitHub message: %q", got)
	}
	if !strings.Contains(got, "401") {
		t.Fatalf("missing status: %q", got)
	}
}

func TestFormatGitHubAPIErrorDefaultIncludesProviderMessage(t *testing.T) {
	body := []byte(`{"message":"Server Error"}`)
	got := formatGitHubAPIError(http.StatusInternalServerError, body)
	if got != "GitHub API error: 500 — Server Error" {
		t.Fatalf("got=%q", got)
	}
}

func TestFetchGitHubBranchesForbiddenSurfacesProviderMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message":"Resource protected by organization SAML enforcement. You must grant your OAuth token access to this organization."}`))
	}))
	defer srv.Close()
	old := githubAPIBase
	githubAPIBase = srv.URL
	t.Cleanup(func() { githubAPIBase = old })

	_, errMsg := fetchGitHubBranches("https://github.com/task2money/ram-work", "tok_test")
	if !strings.Contains(errMsg, "SAML enforcement") {
		t.Fatalf("err=%q want GitHub SAML message", errMsg)
	}
	if strings.HasPrefix(errMsg, "无法获取") {
		t.Fatalf("fetchGitHubBranches should return raw provider err without 无法获取 prefix: %q", errMsg)
	}
}

func TestGithubRepoResponseAllowsPush(t *testing.T) {
	if githubRepoResponseAllowsPush([]byte(`{"permissions":{"push":true}}`)) != true {
		t.Fatal("push true")
	}
	if githubRepoResponseAllowsPush([]byte(`{"permissions":{"admin":true,"push":false}}`)) != true {
		t.Fatal("admin true")
	}
	if githubRepoResponseAllowsPush([]byte(`{"permissions":{"maintain":true}}`)) != true {
		t.Fatal("maintain true")
	}
	if githubRepoResponseAllowsPush([]byte(`{"permissions":{"admin":false,"push":false,"pull":true}}`)) != false {
		t.Fatal("pull-only must not allow push")
	}
	if githubRepoResponseAllowsPush([]byte(`{"id":1}`)) != true {
		t.Fatal("missing permissions is unknown, do not deny")
	}
	if githubRepoResponseAllowsPush([]byte(`not-json`)) != true {
		t.Fatal("invalid json is unknown, do not deny")
	}
}
