package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"tracelog"
)

func TestProjectRequestWithClientUsesInjectedClient(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)
	prev := cfg.ProjectServiceURL
	cfg.ProjectServiceURL = srv.URL
	t.Cleanup(func() { cfg.ProjectServiceURL = prev })

	status, _, err := projectRequestWithClient(projectGitProbeHTTP, context.Background(), http.MethodGet, "/api/projects/ping", "t1", "u1", nil)
	if err != nil {
		t.Fatalf("projectRequestWithClient: %v", err)
	}
	if status != 200 {
		t.Fatalf("status=%d want 200", status)
	}
	if hits != 1 {
		t.Fatalf("hits=%d", hits)
	}
}

func TestProjectRequestEmptyUserSetsInternalIdentity(t *testing.T) {
	var gotUser string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUser = r.Header.Get("X-Auth-User-Id")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"ws1"}`))
	}))
	t.Cleanup(srv.Close)
	prev := cfg.ProjectServiceURL
	cfg.ProjectServiceURL = srv.URL
	t.Cleanup(func() { cfg.ProjectServiceURL = prev })

	status, _, err := projectRequest(context.Background(), http.MethodGet, "/api/projects/workspaces/tenant_id/t1/ws1", "t1", nil)
	if err != nil {
		t.Fatalf("projectRequest: %v", err)
	}
	if status != 200 {
		t.Fatalf("status=%d want 200", status)
	}
	if gotUser != "internal" {
		t.Fatalf("X-Auth-User-Id=%q want internal (OPT-20260820-015 requireTenantMember bypass)", gotUser)
	}
}

func TestProjectRequestWithUserKeepsCallerIdentity(t *testing.T) {
	var gotUser string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUser = r.Header.Get("X-Auth-User-Id")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
	t.Cleanup(srv.Close)
	prev := cfg.ProjectServiceURL
	cfg.ProjectServiceURL = srv.URL
	t.Cleanup(func() { cfg.ProjectServiceURL = prev })

	_, _, err := projectRequestWithUser(context.Background(), http.MethodGet, "/api/projects/workspaces/tenant_id/t1?mine=1", "t1", "u-real", nil)
	if err != nil {
		t.Fatalf("projectRequestWithUser: %v", err)
	}
	if gotUser != "u-real" {
		t.Fatalf("X-Auth-User-Id=%q want u-real", gotUser)
	}
}

func TestVerifyWorkspaceHyphenIDHitsProjectGetWithInternalUser(t *testing.T) {
	const hyphenWS = "ws_-2309487803472456748"
	var gotPath, gotUser, gotTenant string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotUser = r.Header.Get("X-Auth-User-Id")
		gotTenant = r.Header.Get("X-Auth-Tenant-Id")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"id": hyphenWS, "company_id": "t1"})
	}))
	t.Cleanup(srv.Close)
	prev := cfg.ProjectServiceURL
	cfg.ProjectServiceURL = srv.URL
	t.Cleanup(func() { cfg.ProjectServiceURL = prev })

	ws, err := verifyWorkspace(context.Background(), "t1", hyphenWS)
	if err != nil {
		t.Fatalf("verifyWorkspace: %v", err)
	}
	if ws["id"] != hyphenWS {
		t.Fatalf("id=%v want %s", ws["id"], hyphenWS)
	}
	if gotUser != "internal" {
		t.Fatalf("X-Auth-User-Id=%q want internal", gotUser)
	}
	if gotTenant != "t1" {
		t.Fatalf("X-Auth-Tenant-Id=%q want t1", gotTenant)
	}
	if !strings.Contains(gotPath, hyphenWS) {
		t.Fatalf("path %q should contain hyphen workspace id", gotPath)
	}
}

func TestVerifyWorkspaceNon200IsNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"请先登录"}`))
	}))
	t.Cleanup(srv.Close)
	prev := cfg.ProjectServiceURL
	cfg.ProjectServiceURL = srv.URL
	t.Cleanup(func() { cfg.ProjectServiceURL = prev })

	_, err := verifyWorkspace(context.Background(), "t1", "ws1")
	if err == nil || err.Error() != "workspace not found" {
		t.Fatalf("err=%v want workspace not found", err)
	}
}

// TestProjectRequestPropagatesTraceToOutbound verifies OPT-20260821-012:
// the outbound project-service request carries the inbound X-Trace-Id and
// parent span, so a single trace key resolves across task-task-service →
// task-project-service in Loki.
func TestProjectRequestPropagatesTraceToOutbound(t *testing.T) {
	var gotTraceID, gotParentSpan string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotTraceID = r.Header.Get(tracelog.Header)
		gotParentSpan = r.Header.Get(tracelog.ParentSpanHeader)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)
	prev := cfg.ProjectServiceURL
	cfg.ProjectServiceURL = srv.URL
	t.Cleanup(func() { cfg.ProjectServiceURL = prev })

	ctx := tracelog.ContextWithCorrelation(context.Background(), tracelog.Correlation{
		TraceID: "trace-0123abc",
		SpanID:  "aabbccddeeff0011",
	})
	if _, _, err := projectRequest(ctx, http.MethodGet, "/api/projects/workspaces/tenant_id/t1/ws1", "t1", nil); err != nil {
		t.Fatalf("projectRequest: %v", err)
	}
	if gotTraceID != "trace-0123abc" {
		t.Fatalf("X-Trace-Id=%q want trace-0123abc", gotTraceID)
	}
	if gotParentSpan != "aabbccddeeff0011" {
		t.Fatalf("X-Parent-Span-Id=%q want aabbccddeeff0011", gotParentSpan)
	}
}

func TestLoadProjectRepoMetaPropagatesTraceToOutbound(t *testing.T) {
	var gotTraceID, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotTraceID = r.Header.Get(tracelog.Header)
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"git_repos":["https://example.test/a.git"],"auto_clone_nested_repos":true}`))
	}))
	t.Cleanup(srv.Close)
	prev := cfg.ProjectServiceURL
	cfg.ProjectServiceURL = srv.URL
	t.Cleanup(func() { cfg.ProjectServiceURL = prev })

	ctx := tracelog.ContextWithCorrelation(context.Background(), tracelog.Correlation{
		TraceID: "trace-create-task",
		SpanID:  "1122334455667788",
	})
	meta, err := loadProjectRepoMeta(ctx, "t1", "p1")
	if err != nil {
		t.Fatalf("loadProjectRepoMeta: %v", err)
	}
	if gotTraceID != "trace-create-task" {
		t.Fatalf("X-Trace-Id=%q want trace-create-task", gotTraceID)
	}
	if !strings.Contains(gotPath, "/t1/p1") {
		t.Fatalf("path=%q want project GET", gotPath)
	}
	if len(meta.Entries) != 1 || meta.Entries[0].URL != "https://example.test/a.git" {
		t.Fatalf("meta=%+v", meta)
	}
}

func TestGetProjectRepoURLsUsesRequestContext(t *testing.T) {
	var gotTraceID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotTraceID = r.Header.Get(tracelog.Header)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"git_repos":["https://example.test/b.git"]}`))
	}))
	t.Cleanup(srv.Close)
	prev := cfg.ProjectServiceURL
	cfg.ProjectServiceURL = srv.URL
	t.Cleanup(func() { cfg.ProjectServiceURL = prev })

	ctx := tracelog.ContextWithCorrelation(context.Background(), tracelog.Correlation{
		TraceID: "trace-urls",
		SpanID:  "aabbccddeeff0011",
	})
	urls, err := getProjectRepoURLs(ctx, "t1", "p2")
	if err != nil {
		t.Fatalf("getProjectRepoURLs: %v", err)
	}
	if gotTraceID != "trace-urls" {
		t.Fatalf("X-Trace-Id=%q want trace-urls", gotTraceID)
	}
	if len(urls) != 1 || urls[0] != "https://example.test/b.git" {
		t.Fatalf("urls=%v", urls)
	}
}
