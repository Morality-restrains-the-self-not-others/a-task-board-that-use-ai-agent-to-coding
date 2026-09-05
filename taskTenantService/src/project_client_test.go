package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"tracelog"
)

// TestWorkspaceNameUsesNewRouteConvention 回归测试：
// taskProjectService 删除 /api/tenant/{tid}/workspaces/{wid}/ 旧路由后，
// workspaceName 必须使用 /api/projects/workspaces/tenant_id/{tid}/{wid}，
// 否则 project-service 返回 404 导致 workspace 名称取不到。
func TestWorkspaceNameUsesNewRouteConvention(t *testing.T) {
	oldURL, oldClient := cfg.TaskProjectServiceURL, httpClient
	defer func() {
		cfg.TaskProjectServiceURL, httpClient = oldURL, oldClient
	}()

	var gotPath, gotUser string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotUser = r.Header.Get("X-Auth-User-Id")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"name": "WS 1"})
	}))
	defer srv.Close()
	cfg.TaskProjectServiceURL = srv.URL
	httpClient = srv.Client()

	if name := workspaceName(context.Background(), "t1", "ws1"); name != "WS 1" {
		t.Fatalf("expected workspace name %q, got %q", "WS 1", name)
	}
	if gotPath != "/api/projects/workspaces/tenant_id/t1/ws1" {
		t.Fatalf("expected new route /api/projects/workspaces/tenant_id/t1/ws1, got %s", gotPath)
	}
	if gotUser != "internal" {
		t.Fatalf("X-Auth-User-Id=%q want internal (OPT-20260820-015 requireTenantMember bypass)", gotUser)
	}
}

func TestProjectPOSTSetsInternalIdentity(t *testing.T) {
	oldURL, oldClient := cfg.TaskProjectServiceURL, httpClient
	defer func() {
		cfg.TaskProjectServiceURL, httpClient = oldURL, oldClient
	}()

	var gotUser string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUser = r.Header.Get("X-Auth-User-Id")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	cfg.TaskProjectServiceURL = srv.URL
	httpClient = srv.Client()

	status, err := projectPOST(context.Background(), "/api/projects/workspace-access/tenant_id/t1/set-permission", map[string]interface{}{
		"workspace_id": "ws1",
	})
	if err != nil {
		t.Fatalf("projectPOST: %v", err)
	}
	if status != 200 {
		t.Fatalf("status=%d want 200", status)
	}
	if gotUser != "internal" {
		t.Fatalf("X-Auth-User-Id=%q want internal", gotUser)
	}
}

// TestProjectPOSTPropagatesTraceToOutbound verifies OPT-20260821-012:
// outbound project-service requests carry the inbound X-Trace-Id.
func TestProjectPOSTPropagatesTraceToOutbound(t *testing.T) {
	oldURL, oldClient := cfg.TaskProjectServiceURL, httpClient
	defer func() {
		cfg.TaskProjectServiceURL, httpClient = oldURL, oldClient
	}()

	var gotTraceID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotTraceID = r.Header.Get(tracelog.Header)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	cfg.TaskProjectServiceURL = srv.URL
	httpClient = srv.Client()

	ctx := tracelog.ContextWithCorrelation(context.Background(), tracelog.Correlation{
		TraceID: "trace-tenant-abc",
		SpanID:  "1122334455667788",
	})
	if _, err := projectPOST(ctx, "/api/projects/workspace-access/tenant_id/t1/set-permission", map[string]interface{}{
		"workspace_id": "ws1",
	}); err != nil {
		t.Fatalf("projectPOST: %v", err)
	}
	if gotTraceID != "trace-tenant-abc" {
		t.Fatalf("X-Trace-Id=%q want trace-tenant-abc", gotTraceID)
	}
}
