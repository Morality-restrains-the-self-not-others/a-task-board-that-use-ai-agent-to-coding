package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSanitizeCloneAlias(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"  my-app  ", "my-app"},
		{"My App!!", "My-App"},
		{"../evil", "evil"},
		{"a/b", "a-b"},
		{"---", ""},
	}
	for _, c := range cases {
		if got := sanitizeCloneAlias(c.in); got != c.want {
			t.Errorf("sanitizeCloneAlias(%q)=%q want %q", c.in, got, c.want)
		}
	}
}

func TestParseGitRepoItemsMixed(t *testing.T) {
	raw := []interface{}{
		"https://example.com/a.git",
		map[string]interface{}{"url": "https://example.com/b.git", "clone_alias": "custom-b"},
		map[string]interface{}{"repo_url": "https://example.com/c.git", "alias": "c-name"},
		"https://example.com/a.git", // dup
		"  ",
	}
	got := parseGitRepoItems(raw)
	if len(got) != 3 {
		t.Fatalf("len=%d want 3: %#v", len(got), got)
	}
	if got[0].URL != "https://example.com/a.git" || got[0].CloneAlias != "" {
		t.Errorf("entry0=%+v", got[0])
	}
	if got[1].CloneAlias != "custom-b" {
		t.Errorf("entry1=%+v", got[1])
	}
	if got[2].CloneAlias != "c-name" {
		t.Errorf("entry2=%+v", got[2])
	}
}

func TestDuplicateCloneAliasError(t *testing.T) {
	if msg := duplicateCloneAliasError([]gitRepoEntry{
		{URL: "https://a", CloneAlias: "web"},
		{URL: "https://b", CloneAlias: "Web"},
	}); msg == "" {
		t.Fatal("expected duplicate alias error")
	}
	if msg := duplicateCloneAliasError([]gitRepoEntry{
		{URL: "https://a", CloneAlias: "web"},
		{URL: "https://b", CloneAlias: ""},
		{URL: "https://c", CloneAlias: "api"},
	}); msg != "" {
		t.Fatalf("unexpected error: %s", msg)
	}
}

func TestCreateProjectRejectsDuplicateCloneAlias(t *testing.T) {
	setupTestDB(t)
	body := `{
		"name":"Dup Alias",
		"git_repos":[
			{"url":"https://github.com/o/a.git","clone_alias":"same"},
			{"url":"https://github.com/o/b.git","clone_alias":"SAME"}
		]
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/projects/", strings.NewReader(body))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	handleCreateProject(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("create: %d %s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&out)
	if !strings.Contains(fmt.Sprint(out["error"]), "别名不能重复") {
		t.Fatalf("body=%#v", out)
	}
	var n int
	_ = db.QueryRow(`SELECT COUNT(*) FROM project_entries WHERE name=?`, "Dup Alias").Scan(&n)
	if n != 0 {
		t.Fatalf("orphan project created: count=%d", n)
	}
}

func TestCreateProjectWithCloneAlias(t *testing.T) {
	setupTestDB(t)

	body := `{
		"name":"Alias Proj",
		"git_repos":[
			"https://github.com/o/plain.git",
			{"url":"https://github.com/o/named.git","clone_alias":"my-named"}
		]
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/projects/", strings.NewReader(body))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	handleCreateProject(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rec.Code, rec.Body.String())
	}
	var created map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	repos, _ := created["git_repos"].([]interface{})
	if len(repos) != 2 {
		t.Fatalf("git_repos=%#v", created["git_repos"])
	}
	entries, _ := created["git_repo_entries"].([]interface{})
	if len(entries) != 2 {
		t.Fatalf("git_repo_entries=%#v", created["git_repo_entries"])
	}
	findEntry := func(entries []interface{}, url string) map[string]interface{} {
		for _, raw := range entries {
			e, _ := raw.(map[string]interface{})
			if e != nil && e["url"] == url {
				return e
			}
		}
		return nil
	}
	e1 := findEntry(entries, "https://github.com/o/named.git")
	if e1 == nil || e1["clone_alias"] != "my-named" {
		t.Fatalf("expected named entry with clone_alias=my-named, entries=%#v", entries)
	}

	pid, _ := created["id"].(string)
	req2 := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/projects/"+pid+"/", nil)
	req2.Header.Set("X-Auth-Tenant-Id", "t1")
	req2.Header.Set("X-Resource-Id", pid)
	rec2 := httptest.NewRecorder()
	handleGetProject(rec2, req2)
	if rec2.Code != 200 {
		t.Fatalf("get: %d %s", rec2.Code, rec2.Body.String())
	}
	var detail map[string]interface{}
	json.NewDecoder(rec2.Body).Decode(&detail)
	entries2, _ := detail["git_repo_entries"].([]interface{})
	e1b := findEntry(entries2, "https://github.com/o/named.git")
	if e1b == nil || e1b["clone_alias"] != "my-named" {
		t.Fatalf("persisted entry missing clone_alias, entries=%#v", entries2)
	}

	// PATCH update alias
	patch := `{"git_repos":[{"url":"https://github.com/o/named.git","clone_alias":"renamed"}]}`
	req3 := httptest.NewRequest(http.MethodPatch, "/api/tenant/t1/projects/"+pid+"/", strings.NewReader(patch))
	req3.Header.Set("X-Auth-Tenant-Id", "t1")
	req3.Header.Set("X-Resource-Id", pid)
	rec3 := httptest.NewRecorder()
	handleUpdateProject(rec3, req3)
	if rec3.Code != 200 {
		t.Fatalf("patch: %d %s", rec3.Code, rec3.Body.String())
	}
	var updated map[string]interface{}
	json.NewDecoder(rec3.Body).Decode(&updated)
	ents, _ := updated["git_repo_entries"].([]interface{})
	if len(ents) != 1 {
		t.Fatalf("after patch entries=%#v", ents)
	}
	eu, _ := ents[0].(map[string]interface{})
	if eu["clone_alias"] != "renamed" {
		t.Fatalf("after patch=%#v", eu)
	}
}
