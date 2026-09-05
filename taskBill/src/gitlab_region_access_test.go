package main

import (
	"net/http"
	"testing"
)

func TestCanUseGitlabRegion(t *testing.T) {
	cases := []struct {
		mode   string
		tester bool
		want   bool
		name   string
	}{
		{name: "release non-tester", mode: "release", tester: false, want: true},
		{name: "release tester", mode: "release", tester: true, want: true},
		{name: "empty is release", mode: "", tester: false, want: true},
		{name: "development tester", mode: "development", tester: true, want: true},
		{name: "development non-tester", mode: "development", tester: false, want: false},
		{name: "DEVELOPMENT mixed case", mode: "Development", tester: false, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := CanUseGitlabRegion(tc.mode, tc.tester); got != tc.want {
				t.Fatalf("CanUseGitlabRegion(%q,%v)=%v want %v", tc.mode, tc.tester, got, tc.want)
			}
		})
	}
}

func TestParseGitlabRegionAccessMode(t *testing.T) {
	got, err := ParseGitlabRegionAccessMode("")
	if err != nil || got != gitlabRegionAccessRelease {
		t.Fatalf("empty: got %q err=%v", got, err)
	}
	got, err = ParseGitlabRegionAccessMode("development")
	if err != nil || got != gitlabRegionAccessDevelopment {
		t.Fatalf("dev: got %q err=%v", got, err)
	}
	if _, err = ParseGitlabRegionAccessMode("staging"); err == nil {
		t.Fatal("staging must be rejected")
	}
}

func TestRequestIsTester(t *testing.T) {
	req := httptestRequestWithTester("1")
	if !requestIsTester(req) {
		t.Fatal("want tester")
	}
	if requestIsTester(httptestRequestWithTester("")) {
		t.Fatal("empty header is not tester")
	}
}

func httptestRequestWithTester(v string) *http.Request {
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	if v != "" {
		req.Header.Set("X-User-Is-Tester", v)
	}
	return req
}

func TestFilterUsableGitlabRegionsHidesDevelopment(t *testing.T) {
	regions := []GitlabRegion{
		{Slug: "prod", AccessMode: "release"},
		{Slug: "dev", AccessMode: "development"},
	}
	got := filterUsableGitlabRegions(regions, false)
	if len(got) != 1 || got[0].Slug != "prod" {
		t.Fatalf("non-tester got %#v", got)
	}
	got = filterUsableGitlabRegions(regions, true)
	if len(got) != 2 {
		t.Fatalf("tester should see both, got %#v", got)
	}
}
