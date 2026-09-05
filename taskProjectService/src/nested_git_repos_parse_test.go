package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseGitmodulesRamWorkFixture(t *testing.T) {
	root, err := findMonorepoRoot()
	if err != nil {
		t.Skip(err)
	}
	raw, err := os.ReadFile(filepath.Join(root, ".gitmodules"))
	if err != nil {
		t.Fatal(err)
	}
	got := parseGitmodules(string(raw))
	if len(got) < 10 {
		t.Fatalf("expected many nested repos from ram-work .gitmodules, got %d", len(got))
	}
	found := map[string]nestedRepoCandidate{}
	for _, c := range got {
		found[c.Path] = c
	}
	for _, need := range []string{"docs", "runAll", "taskProjectService"} {
		c, ok := found[need]
		if !ok {
			t.Errorf("missing %s", need)
			continue
		}
		want := "git@github.com:task2money/" + need + ".git"
		if c.URL != want {
			t.Errorf("%s url=%q want %s", need, c.URL, want)
		}
		if c.Source != nestedSourceGitmodules {
			t.Errorf("%s source=%q", need, c.Source)
		}
	}
}

func TestParseGitmodules(t *testing.T) {
	content := `[submodule "libs/foo"]
	path = libs/foo
	url = https://gitlab.daydaymoney.com/org/foo.git
[submodule "bar"]
	path = bar
	url = ../bar.git
`
	got := parseGitmodules(content)
	if len(got) != 2 {
		t.Fatalf("len=%d %#v", len(got), got)
	}
	if got[0].Path != "libs/foo" || !strings.Contains(got[0].URL, "foo.git") {
		t.Errorf("first=%#v", got[0])
	}
	if got[1].Path != "bar" || got[1].URL != "../bar.git" {
		t.Errorf("second=%#v", got[1])
	}
}

func TestParseGitmodulesEmpty(t *testing.T) {
	if got := parseGitmodules(""); got != nil {
		t.Fatalf("empty content: %#v", got)
	}
}

func TestResolveSubmoduleURL(t *testing.T) {
	parent := "https://gitlab.daydaymoney.com/example-user/ram-work.git"

	// Git docs: sibling of bar.git is ../foo.git (parent remote treated as directory).
	url, errMsg := resolveSubmoduleURL(parent, "../task2app.git")
	if errMsg != "" {
		t.Fatal(errMsg)
	}
	want := "https://gitlab.daydaymoney.com/example-user/task2app.git"
	if url != want {
		t.Fatalf("relative sibling: got %q want %q", url, want)
	}

	url, errMsg = resolveSubmoduleURL(parent, "https://gitlab.daydaymoney.com/other/custom.git")
	if errMsg != "" || url != "https://gitlab.daydaymoney.com/other/custom.git" {
		t.Fatalf("absolute: url=%q err=%q", url, errMsg)
	}

	url, errMsg = resolveSubmoduleURL("git@gitlab.daydaymoney.com:example-user/ram-work.git", "../docs.git")
	if errMsg != "" {
		t.Fatal(errMsg)
	}
	if url != "git@gitlab.daydaymoney.com:example-user/docs.git" {
		t.Fatalf("scp relative: got %q", url)
	}

	url, errMsg = resolveSubmoduleURL(parent, "")
	if url != "" || errMsg == "" {
		t.Fatalf("empty url: url=%q err=%q", url, errMsg)
	}
}

func TestMergeNestedRepos(t *testing.T) {
	parent := "https://gitlab.daydaymoney.com/g/ram-work.git"
	gm := []nestedRepoCandidate{
		{Path: "task2app", URL: "../custom-task2app.git", Source: nestedSourceGitmodules},
		{Path: "docs", URL: "../docs.git", Source: nestedSourceGitmodules},
	}
	merged := mergeNestedRepos(gm, parent)
	if len(merged) != 2 {
		t.Fatalf("len=%d", len(merged))
	}
	if merged[0].Path != "docs" || merged[0].URL != "https://gitlab.daydaymoney.com/g/docs.git" {
		t.Errorf("docs row=%#v", merged[0])
	}
	if merged[1].Path != "task2app" || merged[1].Source != nestedSourceGitmodules {
		t.Errorf("task2app row=%#v", merged[1])
	}
	if merged[1].URL != "https://gitlab.daydaymoney.com/g/custom-task2app.git" {
		t.Errorf("url=%q", merged[1].URL)
	}
}

func TestMergeNestedReposIgnoresAbsentGitmodules(t *testing.T) {
	merged := mergeNestedRepos(nil, "https://gitlab.daydaymoney.com/g/ram-work.git")
	if len(merged) != 0 {
		t.Fatalf("want empty without .gitmodules, got %#v", merged)
	}
}
