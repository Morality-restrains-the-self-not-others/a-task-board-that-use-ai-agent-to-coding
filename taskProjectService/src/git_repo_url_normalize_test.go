package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNormalizeGitRepoURLForBranchLookupSSHToProviderOrigin(t *testing.T) {
	initProviderTestConfig(t)
	providerResolver = &ProviderResolver{
		entries: []providerEntry{{
			Provider:        "gitlab",
			ServiceProvider: "gitlab-local",
			ProviderKey:     "gitlab:gitlab-local",
			Host:            "gitlab.daydaymoney.com",
			Netloc:          "gitlab.daydaymoney.com",
			WebsiteOrigin:   "https://gitlab.daydaymoney.com",
		}},
	}

	got, ok := normalizeGitRepoURLForBranchLookup("git@gitlab.daydaymoney.com:example-user/task2app.git")
	if !ok {
		t.Fatal("expected normalization success")
	}
	want := "https://gitlab.daydaymoney.com/example-user/task2app"
	if got != want {
		t.Fatalf("normalized = %q, want %q", got, want)
	}
}

func TestNormalizeGitRepoURLForBranchLookupPreservesHTTPS(t *testing.T) {
	input := "http://127.0.0.1:8012/example-user/task2app.git"
	got, ok := normalizeGitRepoURLForBranchLookup(input)
	if !ok {
		t.Fatal("expected normalization success")
	}
	if got != input {
		t.Fatalf("normalized = %q, want %q", got, input)
	}
}

// OPT-20260827-040：SSH 主机命中 provider website origin（http scheme）时，
// 规范化必须继承 website 的 scheme，不得兜底到 https://host。
func TestNormalizeGitRepoURLForBranchLookupSSHToHTTPOrigin(t *testing.T) {
	initProviderTestConfig(t)
	providerResolver = &ProviderResolver{
		entries: []providerEntry{{
			Provider:        "gitlab",
			ServiceProvider: "gitlab-local",
			ProviderKey:     "gitlab:gitlab-local",
			Host:            "127.0.0.1",
			Netloc:          "127.0.0.1:8012",
			WebsiteOrigin:   "http://127.0.0.1:8012",
		}},
	}

	got, ok := normalizeGitRepoURLForBranchLookup("git@127.0.0.1:example-user/task2app.git")
	if !ok {
		t.Fatal("expected normalization success")
	}
	want := "http://127.0.0.1:8012/example-user/task2app"
	if got != want {
		t.Fatalf("normalized = %q, want %q (HTTP website origin scheme must be preserved)", got, want)
	}
}

func TestNormalizeGitRepoURLForBranchLookupGitHubSSH(t *testing.T) {
	got, ok := normalizeGitRepoURLForBranchLookup("git@github.com:org/repo.git")
	if !ok {
		t.Fatal("expected normalization success")
	}
	want := "https://github.com/org/repo"
	if got != want {
		t.Fatalf("normalized = %q, want %q", got, want)
	}
}

func TestListProjectRepoBranchesSSHUsesOAuthNotSSHError(t *testing.T) {
	initProviderTestConfig(t)
	gitoauthSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"detail":"not bound"}`))
	}))
	defer gitoauthSrv.Close()

	oldBase := cfg.GitoauthBaseURL
	cfg.GitoauthBaseURL = gitoauthSrv.URL
	t.Cleanup(func() { cfg.GitoauthBaseURL = oldBase })

	providerResolver = &ProviderResolver{
		entries: []providerEntry{{
			Provider:        "gitlab",
			ServiceProvider: "gitlab-local",
			ProviderKey:     "gitlab:gitlab-local",
			Host:            "gitlab.daydaymoney.com",
			Netloc:          "gitlab.daydaymoney.com",
			WebsiteOrigin:   "https://gitlab.daydaymoney.com",
			GitoauthBase:    gitoauthSrv.URL,
		}},
	}

	payload := listProjectRepoBranches("1001", "git@gitlab.daydaymoney.com:example-user/task2app.git", "", "")
	if strings.Contains(payload.Error, "SSH Git URLs cannot be checked for branches without credentials") {
		t.Fatalf("unexpected SSH error: %q", payload.Error)
	}
	if !strings.Contains(payload.Error, "未检测到可用授权") {
		t.Fatalf("expected auth hint, got %q", payload.Error)
	}
}

func TestExtractHostNetlocFromSSHGitURL(t *testing.T) {
	host, netloc := extractHostNetloc("git@gitlab.daydaymoney.com:example-user/task2app.git")
	if host != "gitlab.daydaymoney.com" || netloc != "gitlab.daydaymoney.com" {
		t.Fatalf("host=%q netloc=%q", host, netloc)
	}
}

func TestNormalizeGitRepoURLForBranchLookupSSHURI(t *testing.T) {
	got, ok := normalizeGitRepoURLForBranchLookup("ssh://git@github.com/org/repo.git")
	if !ok {
		t.Fatal("expected normalization success")
	}
	want := "https://github.com/org/repo"
	if got != want {
		t.Fatalf("normalized = %q, want %q", got, want)
	}
}

func TestNormalizeGitRepoURLForBranchLookupSSHURICustomPortToHTTPOrigin(t *testing.T) {
	initProviderTestConfig(t)
	providerResolver = &ProviderResolver{
		entries: []providerEntry{{
			Provider:        "gitlab",
			ServiceProvider: "gitlab-local",
			ProviderKey:     "gitlab:gitlab-local",
			Host:            "127.0.0.1",
			Netloc:          "127.0.0.1:8012",
			WebsiteOrigin:   "http://127.0.0.1:8012",
		}},
	}

	got, ok := normalizeGitRepoURLForBranchLookup("ssh://git@127.0.0.1:2222/example-user/task2app.git")
	if !ok {
		t.Fatal("expected normalization success")
	}
	want := "http://127.0.0.1:8012/example-user/task2app"
	if got != want {
		t.Fatalf("normalized = %q, want %q", got, want)
	}
}

func TestNormalizeGitRepoURLForBranchLookupSSHURIRejectsEmptyPath(t *testing.T) {
	if _, ok := normalizeGitRepoURLForBranchLookup("ssh://git@github.com"); ok {
		t.Fatal("expected empty-path ssh:// to fail")
	}
}

func TestExtractHostNetlocFromSSHURI(t *testing.T) {
	host, netloc := extractHostNetloc("ssh://git@gitlab.daydaymoney.com:2222/group/repo.git")
	if host != "gitlab.daydaymoney.com" {
		t.Fatalf("host=%q", host)
	}
	if netloc != "gitlab.daydaymoney.com:2222" {
		t.Fatalf("netloc=%q", netloc)
	}
}
