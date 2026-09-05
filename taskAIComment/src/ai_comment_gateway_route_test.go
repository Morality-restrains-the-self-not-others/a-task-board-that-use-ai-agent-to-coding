package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// 回归：前端约定 kv-last URL
//
//	/api/ai-comment/task-detail/tenant_id/{tid}/workspace_id/{wid}/task_id/{taskId}/ai-comments/
//
// 旧手写 kv 循环在「kv 对之后仅剩 1 个位置段」时把 rest 丢成空串 → 业务层 HTTP 404
// （页面 comments-feed-error-item：「ai_comments：HTTP 404」）。
func TestParseAICommentGatewayRoute_TaskDetailListKeepsAICommentsRest(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	path := "task-detail/tenant_id/t1/workspace_id/ws1/task_id/task1/ai-comments"
	got, err := parseAICommentGatewayRoute(req, path)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got.FuncName != "task-detail" {
		t.Fatalf("FuncName=%q want task-detail", got.FuncName)
	}
	if got.TenantID != "t1" || got.WorkspaceID != "ws1" || got.TaskID != "task1" {
		t.Fatalf("ids=%+v", got)
	}
	if got.Rest != "ai-comments" {
		t.Fatalf("Rest=%q want ai-comments (trailing resource after kv must be preserved)", got.Rest)
	}
}

func TestParseAICommentGatewayRoute_TaskDetailCommentID(t *testing.T) {
	req := httptest.NewRequest(http.MethodPatch, "/", nil)
	path := "task-detail/tenant_id/t1/workspace_id/ws1/task_id/task1/ai-comments/c9"
	got, err := parseAICommentGatewayRoute(req, path)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got.Rest != "ai-comments/c9" {
		t.Fatalf("Rest=%q want ai-comments/c9", got.Rest)
	}
}

func TestParseAICommentGatewayRoute_ContainerAgentSingleID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	path := "container-agent-comments/tenant_id/t1/workspace_id/ws1/task_id/task1/agent1"
	got, err := parseAICommentGatewayRoute(req, path)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got.FuncName != "container-agent-comments" {
		t.Fatalf("FuncName=%q", got.FuncName)
	}
	if got.Rest != "agent1" {
		t.Fatalf("Rest=%q want agent1 (single trailing id after kv)", got.Rest)
	}
}

func TestParseAICommentGatewayRoute_ContainerAgentStream(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	path := "container-agent-comments/tenant_id/t1/workspace_id/ws1/task_id/task1/agent1/stream"
	got, err := parseAICommentGatewayRoute(req, path)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got.Rest != "agent1/stream" {
		t.Fatalf("Rest=%q want agent1/stream", got.Rest)
	}
}

func TestParseAICommentGatewayRoute_MissingKV(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	_, err := parseAICommentGatewayRoute(req, "task-detail/ai-comments")
	if err == nil {
		t.Fatal("expected error when tenant/workspace/task missing")
	}
}

func TestParseAICommentGatewayRoute_ExactFrontendListURL(t *testing.T) {
	// 与 taskFE fetchAIComments 拼接的 URL 一致（含尾部 /）。
	urlPath := "/api/ai-comment/task-detail/tenant_id/t1/workspace_id/ws1/task_id/task1/ai-comments/"
	path := strings.TrimPrefix(urlPath, "/api/ai-comment/")
	req := httptest.NewRequest(http.MethodGet, urlPath, nil)
	got, err := parseAICommentGatewayRoute(req, path)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got.Rest != "ai-comments" {
		t.Fatalf("Rest=%q want ai-comments — FE list URL must not 404 at gateway dispatch", got.Rest)
	}
}

func TestMountAICommentGateway_ListReturnsComments(t *testing.T) {
	setupTestDB(t)
	startMockTaskService(t)
	startMockAuthService(t)

	mux := http.NewServeMux()
	mountRoutes(mux)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/ai-comment/task-detail/tenant_id/t1/workspace_id/ws1/task_id/task1/ai-comments/",
		nil,
	)
	req.Header.Set("X-Auth-User-Id", "u1")
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s want 200 (route must reach list handler)", rec.Code, rec.Body.String())
	}
}
