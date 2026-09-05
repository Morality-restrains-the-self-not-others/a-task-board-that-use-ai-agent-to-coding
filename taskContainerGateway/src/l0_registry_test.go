package main

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestParseContainerComputePathCloneLog(t *testing.T) {
	path := "/api/tenant/t1/workspace/w1/task/task1/cloud/compute/container-clone-log/"
	match, ok := parseContainerComputePath(path)
	if !ok {
		t.Fatal("expected path to match")
	}
	if match.Action != "container-clone-log" {
		t.Fatalf("action=%q", match.Action)
	}
}

func TestL0RegistryLayerDelete(t *testing.T) {
	spec, ok := lookupL0Action("container-layer-delete")
	if !ok {
		t.Fatal("expected registry entry")
	}
	plan := spec.Build(forwardRequest{
		BaseURL: "http://127.0.0.1:8765/",
		Scope:   scope{TenantID: "t1", WorkspaceID: "w1", TaskID: "task1"},
		Method:  http.MethodPost,
		Body:    []byte(`{"layer_id":"layer-abc"}`),
	})
	if plan.ErrDetail != "" {
		t.Fatalf("unexpected err: %s", plan.ErrDetail)
	}
	if plan.Method != http.MethodDelete {
		t.Fatalf("method=%q", plan.Method)
	}
	if plan.URL != "http://127.0.0.1:8765/api/tenant/t1/workspace/w1/task/task1/layers/layer-abc" {
		t.Fatalf("url=%q", plan.URL)
	}
}

func TestL0RegistryLayerFilesQuery(t *testing.T) {
	spec, ok := lookupL0Action("container-layer-files")
	if !ok {
		t.Fatal("expected registry entry")
	}
	q := url.Values{}
	q.Set("layer_id", "L1")
	q.Set("max_files", "100")
	plan := spec.Build(forwardRequest{
		BaseURL: "http://127.0.0.1:8765",
		Scope:   scope{TenantID: "t1", WorkspaceID: "w1", TaskID: "task1"},
		Method:  http.MethodGet,
		Query:   q,
	})
	if plan.ErrDetail != "" {
		t.Fatalf("unexpected err: %s", plan.ErrDetail)
	}
	if plan.URL != "http://127.0.0.1:8765/api/tenant/t1/workspace/w1/task/task1/layers/L1/files?max_files=100" {
		t.Fatalf("url=%q", plan.URL)
	}
}

func TestL0RegistryLayerDiffParentFilesQuery(t *testing.T) {
	spec, ok := lookupL0Action("container-layer-diff-parent-files")
	if !ok {
		t.Fatal("expected registry entry")
	}
	q := url.Values{}
	q.Set("layer_id", "L1")
	q.Set("offset", "100")
	q.Set("limit", "50")
	plan := spec.Build(forwardRequest{
		BaseURL: "http://127.0.0.1:8765",
		Scope:   scope{TenantID: "t1", WorkspaceID: "w1", TaskID: "task1"},
		Method:  http.MethodGet,
		Query:   q,
	})
	if plan.ErrDetail != "" {
		t.Fatalf("unexpected err: %s", plan.ErrDetail)
	}
	if !strings.Contains(plan.URL, "/layers/L1/diff/parent/files?") {
		t.Fatalf("url=%q", plan.URL)
	}
	if !strings.Contains(plan.URL, "offset=100") || !strings.Contains(plan.URL, "limit=50") {
		t.Fatalf("url=%q", plan.URL)
	}
}

func TestParseRelayToTraePathHealth(t *testing.T) {
	path := "/api/tenant/t1/workspace/w1/task/task1/cloud/compute/relay-to-trae/health/"
	match, ok := parseRelayToTraePath(path)
	if !ok {
		t.Fatal("expected path to match")
	}
	if match.SubAction != "health" {
		t.Fatalf("subAction=%q", match.SubAction)
	}
	method, goPath, supported := relayGoPath(match.SubAction)
	if !supported || method != http.MethodGet || goPath != "/health" {
		t.Fatalf("relay mapping method=%s path=%s supported=%v", method, goPath, supported)
	}
}

func TestL0RegistryJobContinueRequiresQuery(t *testing.T) {
	spec, ok := lookupL0Action("container-job-continue")
	if !ok {
		t.Fatal("expected registry entry")
	}
	plan := spec.Build(forwardRequest{BaseURL: "http://127.0.0.1:8765"})
	if plan.ErrDetail == "" {
		t.Fatal("expected missing job_id error")
	}
}

func TestL0RegistryLayerGitMerge(t *testing.T) {
	spec, ok := lookupL0Action("container-layer-git-merge")
	if !ok {
		t.Fatal("expected registry entry")
	}
	plan := spec.Build(forwardRequest{
		BaseURL: "http://127.0.0.1:8765/",
		Scope:   scope{TenantID: "t1", WorkspaceID: "w1", TaskID: "task1"},
		Method:  http.MethodPost,
		Body:    []byte(`{"layer_id":"layer-1","target_branch":"develop","source_ref":"feature/x"}`),
	})
	if plan.ErrDetail != "" {
		t.Fatalf("unexpected err: %s", plan.ErrDetail)
	}
	if plan.Method != http.MethodPost {
		t.Fatalf("method=%q", plan.Method)
	}
	wantURL := "http://127.0.0.1:8765/api/tenant/t1/workspace/w1/task/task1/layers/layer-1/git/merge"
	if plan.URL != wantURL {
		t.Fatalf("url=%q want %q", plan.URL, wantURL)
	}
	body := string(plan.Body)
	if !strings.Contains(body, `"target_branch":"develop"`) {
		t.Fatalf("body missing target_branch: %s", body)
	}
	if !strings.Contains(body, `"source_ref":"feature/x"`) {
		t.Fatalf("body missing source_ref: %s", body)
	}

	missing := spec.Build(forwardRequest{
		BaseURL: "http://127.0.0.1:8765/",
		Scope:   scope{TenantID: "t1", WorkspaceID: "w1", TaskID: "task1"},
		Body:    []byte(`{"layer_id":"layer-1"}`),
	})
	if missing.ErrDetail == "" {
		t.Fatal("expected target_branch required")
	}
}

func TestL0RegistryLayerGitLogForwardsPathAndLimit(t *testing.T) {
	spec, ok := lookupL0Action("container-layer-git-log")
	if !ok {
		t.Fatal("expected registry entry")
	}
	q := url.Values{}
	q.Set("layer_id", "L1")
	q.Set("path", "repo-a")
	q.Set("limit", "30")
	plan := spec.Build(forwardRequest{
		BaseURL: "http://127.0.0.1:8765",
		Scope:   scope{TenantID: "t1", WorkspaceID: "w1", TaskID: "task1"},
		Method:  http.MethodGet,
		Query:   q,
	})
	if plan.ErrDetail != "" {
		t.Fatalf("unexpected err: %s", plan.ErrDetail)
	}
	if plan.Method != http.MethodGet {
		t.Fatalf("method=%q", plan.Method)
	}
	want := "http://127.0.0.1:8765/api/tenant/t1/workspace/w1/task/task1/layers/L1/git/log?limit=30&path=repo-a"
	if plan.URL != want {
		t.Fatalf("url=%q want %q", plan.URL, want)
	}
}

func TestL0RegistryAutoRunSteps(t *testing.T) {
	spec, ok := lookupL0Action("container-auto-run-steps")
	if !ok {
		t.Fatal("expected container-auto-run-steps in registry")
	}
	if !methodAllowed(spec, http.MethodGet) {
		t.Fatal("GET should be allowed")
	}
	plan := spec.Build(forwardRequest{
		BaseURL: "http://container:8765",
		Scope:   scope{TenantID: "t1", WorkspaceID: "w1", TaskID: "task1"},
		Method:  http.MethodGet,
	})
	if plan.ErrDetail != "" {
		t.Fatalf("unexpected err: %s", plan.ErrDetail)
	}
	if plan.Method != http.MethodGet {
		t.Fatalf("method=%s", plan.Method)
	}
	if !strings.HasSuffix(plan.URL, "/auto-run-steps") {
		t.Fatalf("url=%s", plan.URL)
	}
}

func TestL0RegistryLayerFileContentKeepsPathSeparators(t *testing.T) {
	spec, ok := lookupL0Action("container-layer-file-content")
	if !ok {
		t.Fatal("expected registry entry")
	}
	q := url.Values{}
	q.Set("layer_id", "L1")
	q.Set("path", "ram-work/docs/a.md")
	plan := spec.Build(forwardRequest{
		BaseURL: "http://127.0.0.1:8765",
		Scope:   scope{TenantID: "t1", WorkspaceID: "w1", TaskID: "task1"},
		Method:  http.MethodGet,
		Query:   q,
	})
	if plan.ErrDetail != "" {
		t.Fatalf("unexpected err: %s", plan.ErrDetail)
	}
	want := "http://127.0.0.1:8765/api/tenant/t1/workspace/w1/task/task1/layers/L1/files/ram-work/docs/a.md"
	if plan.URL != want {
		t.Fatalf("url=%q want %q", plan.URL, want)
	}
	if strings.Contains(plan.URL, "%2F") {
		t.Fatalf("path separators must not be escaped as %%2F: %s", plan.URL)
	}

	bad := url.Values{}
	bad.Set("layer_id", "L1")
	bad.Set("path", "../etc/passwd")
	badPlan := spec.Build(forwardRequest{
		BaseURL: "http://127.0.0.1:8765",
		Scope:   scope{TenantID: "t1", WorkspaceID: "w1", TaskID: "task1"},
		Method:  http.MethodGet,
		Query:   bad,
	})
	if badPlan.ErrDetail == "" {
		t.Fatal("expected invalid path error")
	}
}
