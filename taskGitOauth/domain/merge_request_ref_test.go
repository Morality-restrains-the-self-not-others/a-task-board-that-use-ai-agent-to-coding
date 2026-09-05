package domain

import "testing"

func TestParseMergeRequestURL_GitHub(t *testing.T) {
	ref, err := ParseMergeRequestURL("https://github.com/acme/repo/pull/42")
	if err != nil {
		t.Fatal(err)
	}
	if ref.Provider != MergeProviderGitHub || ref.Host != "github.com" {
		t.Fatalf("provider/host=%s %s", ref.Provider, ref.Host)
	}
	if ref.ProjectPath != "acme/repo" || ref.Number != 42 {
		t.Fatalf("project=%s n=%d", ref.ProjectPath, ref.Number)
	}
}

func TestParseMergeRequestURL_GitLab(t *testing.T) {
	raw := "https://gitlab-tencent-sh-1.daydaymoney.com/example-user/somanyad/-/merge_requests/12"
	ref, err := ParseMergeRequestURL(raw)
	if err != nil {
		t.Fatal(err)
	}
	if ref.Provider != MergeProviderGitLab {
		t.Fatalf("provider=%s", ref.Provider)
	}
	if ref.Host != "gitlab-tencent-sh-1.daydaymoney.com" {
		t.Fatalf("host=%s", ref.Host)
	}
	if ref.ProjectPath != "example-user/somanyad" || ref.Number != 12 {
		t.Fatalf("project=%s n=%d", ref.ProjectPath, ref.Number)
	}
	if ref.Origin() != "https://gitlab-tencent-sh-1.daydaymoney.com" {
		t.Fatalf("origin=%s", ref.Origin())
	}
}

func TestParseMergeRequestURL_GitLabHTTPOrigin(t *testing.T) {
	raw := "http://115.29.110.74/example-user/somanyad/-/merge_requests/1"
	ref, err := ParseMergeRequestURL(raw)
	if err != nil {
		t.Fatal(err)
	}
	if ref.Scheme != "http" {
		t.Fatalf("scheme=%s", ref.Scheme)
	}
	if ref.Host != "115.29.110.74" {
		t.Fatalf("host=%s", ref.Host)
	}
	if got := ref.Origin(); got != "http://115.29.110.74" {
		t.Fatalf("origin=%s want http (not https:443)", got)
	}
}

func TestParseMergeRequestURL_GitLabPreservesPort(t *testing.T) {
	raw := "http://127.0.0.1:8012/group/repo/-/merge_requests/3"
	ref, err := ParseMergeRequestURL(raw)
	if err != nil {
		t.Fatal(err)
	}
	if ref.Port != "8012" {
		t.Fatalf("port=%s", ref.Port)
	}
	if got := ref.Origin(); got != "http://127.0.0.1:8012" {
		t.Fatalf("origin=%s", got)
	}
}

func TestMergeRequestRefOriginPrefersAPIOrigin(t *testing.T) {
	ref, err := ParseMergeRequestURL("https://115.29.110.74/g/r/-/merge_requests/1")
	if err != nil {
		t.Fatal(err)
	}
	ref.APIOrigin = "http://115.29.110.74"
	if got := ref.Origin(); got != "http://115.29.110.74" {
		t.Fatalf("origin=%s", got)
	}
}

func TestParseMergeRequestURL_RejectsBad(t *testing.T) {
	cases := []string{
		"",
		"not-a-url",
		"ftp://github.com/acme/repo/pull/1",
		"https://github.com/acme/repo/issues/1",
		"https://example.com/foo",
	}
	for _, c := range cases {
		if _, err := ParseMergeRequestURL(c); err == nil {
			t.Fatalf("expected error for %q", c)
		}
	}
}
