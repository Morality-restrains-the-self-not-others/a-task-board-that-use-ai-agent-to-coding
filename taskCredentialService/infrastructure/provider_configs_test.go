package infrastructure

import (
	"testing"

	"confload"
)

func TestExtractHostNetlocSCPStyle(t *testing.T) {
	host, netloc := extractHostNetloc("git@gitlab.daydaymoney.com:example-user/somanyad.git")
	if host != "gitlab.daydaymoney.com" {
		t.Fatalf("host=%q", host)
	}
	if netloc != "gitlab.daydaymoney.com" {
		t.Fatalf("netloc=%q", netloc)
	}
}

func TestExtractHostNetlocHTTPS(t *testing.T) {
	host, netloc := extractHostNetloc("https://127.0.0.1:8012/group/repo.git")
	if host != "127.0.0.1" {
		t.Fatalf("host=%q", host)
	}
	if netloc != "127.0.0.1:8012" {
		t.Fatalf("netloc=%q", netloc)
	}
}

func TestResolveProviderGithubOfficial(t *testing.T) {
	resolver, err := LoadProviderConfigs("/tmp/ram-work")
	if err != nil {
		t.Fatal(err)
	}
	key := resolver.ResolveProvider("https://github.com/acme/demo.git")
	if key != "github:github-official-daydaymoney" {
		t.Fatalf("provider=%q want github:github-official-daydaymoney", key)
	}
	if got := resolver.DefaultGithubProviderKey(); got != "github:github-official-daydaymoney" {
		t.Fatalf("DefaultGithubProviderKey=%q", got)
	}
}

func TestResolveProviderSCPStyleLocalGitLab(t *testing.T) {
	resolver, err := LoadProviderConfigs("/tmp/ram-work")
	if err != nil {
		t.Fatal(err)
	}
	key := resolver.ResolveProvider("https://127.0.0.1:8012/group/repo.git")
	if key != "gitlab:gitlab-local" {
		t.Fatalf("provider=%q want gitlab:gitlab-local", key)
	}
}

func TestResolveProviderUnmatchedGitLabHostEmpty(t *testing.T) {
	resolver, err := LoadProviderConfigs("/tmp/ram-work")
	if err != nil {
		t.Fatal(err)
	}
	key := resolver.ResolveProvider("git@gitlab.daydaymoney.com:example-user/somanyad.git")
	if key != "" {
		t.Fatalf("unmatched gitlab host must not collapse to %q", key)
	}
	key = resolver.ResolveProvider("https://gitlab.example.invalid/group/repo.git")
	if key != "" {
		t.Fatalf("unmatched gitlab https must not collapse to %q", key)
	}
}

func TestResolveProviderTencentSh1InterpolatedWebsite(t *testing.T) {
	resolver, err := LoadProviderConfigs("/tmp/ram-work")
	if err != nil {
		t.Fatal(err)
	}
	origin := confload.ResolveTemplate("${scheme}://${subdomains.gitlabTencentSh1}", confload.ResolveBaseYaml("/tmp/ram-work"))
	if origin == "" || origin == "${scheme}://${subdomains.gitlabTencentSh1}" {
		t.Fatalf("expected interpolated gitlabTencentSh1 origin, got %q", origin)
	}
	key := resolver.ResolveProvider(origin + "/group/repo.git")
	if key != "gitlab:tencent-sh-1" {
		t.Fatalf("provider=%q want gitlab:tencent-sh-1 origin=%s", key, origin)
	}
}

func TestResolveHttpsCloneURL_SCPStyleLocalGitLab(t *testing.T) {
	resolver, err := LoadProviderConfigs("/tmp/ram-work")
	if err != nil {
		t.Fatal(err)
	}
	got := resolver.ResolveHttpsCloneURL("git@gitlab.daydaymoney.com:example-user/somanyad.git")
	want := "https://gitlab.daydaymoney.com/example-user/somanyad"
	if got != want {
		t.Fatalf("https_clone_url=%q want %q", got, want)
	}
}

func TestResolveHttpsCloneURL_Github(t *testing.T) {
	resolver, err := LoadProviderConfigs("/tmp/ram-work")
	if err != nil {
		t.Fatal(err)
	}
	got := resolver.ResolveHttpsCloneURL("git@github.com:acme/demo.git")
	if got != "https://github.com/acme/demo" {
		t.Fatalf("got=%q", got)
	}
}

// OPT-20260827-040：SSH 主机命中 provider website 时，clone URL 必须继承
// website 的 scheme（http://127.0.0.1:8012），不得兜底到 https://。
func TestResolveHttpsCloneURL_PreservesWebsiteSchemeForHTTPGitLab(t *testing.T) {
	resolver, err := LoadProviderConfigs("/tmp/ram-work")
	if err != nil {
		t.Fatal(err)
	}
	got := resolver.ResolveHttpsCloneURL("git@127.0.0.1:example-user/somanyad.git")
	want := "http://127.0.0.1:8012/example-user/somanyad"
	if got != want {
		t.Fatalf("https_clone_url=%q want %q (HTTP website scheme must be preserved)", got, want)
	}
}

// OPT-20260827-040：已是 http(s) 的 URL 原样返回，禁止改写 scheme 或追加端口。
func TestResolveHttpsCloneURL_HTTPURLPassthrough(t *testing.T) {
	resolver, err := LoadProviderConfigs("/tmp/ram-work")
	if err != nil {
		t.Fatal(err)
	}
	got := resolver.ResolveHttpsCloneURL("http://127.0.0.1:8012/example-user/somanyad.git")
	want := "http://127.0.0.1:8012/example-user/somanyad.git"
	if got != want {
		t.Fatalf("https_clone_url=%q want %q (existing http URL must pass through unchanged)", got, want)
	}
}

func TestExtractRepoPath(t *testing.T) {
	if got := extractRepoPath("git@gitlab.daydaymoney.com:example-user/somanyad.git"); got != "example-user/somanyad" {
		t.Fatalf("got=%q", got)
	}
}
