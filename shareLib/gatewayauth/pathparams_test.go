package gatewayauth

import (
	"net/http/httptest"
	"testing"
)

func parse(t *testing.T, path string) *httptest.ResponseRecorder {
	t.Helper()
	return nil
}

// ---------- ParseConventionPath -------------------------------------------------------

func TestParseConventionPath_NoKvPairs(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	if got := ParseConventionPath(req, "server-images/vpcs"); got != "server-images/vpcs" {
		t.Fatalf("expected unchanged path, got %q", got)
	}
	if req.Header.Get(HeaderAuthTenantID) != "" {
		t.Fatalf("tenant header must stay empty, got %q", req.Header.Get(HeaderAuthTenantID))
	}
}

func TestParseConventionPath_FuncNameWithTrailingPair(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	got := ParseConventionPath(req, "server-images/vpcs/tenant_id/123")
	if got != "server-images/vpcs" {
		t.Fatalf("expected prefix, got %q", got)
	}
	if tid := req.Header.Get(HeaderAuthTenantID); tid != "123" {
		t.Fatalf("expected tenant 123, got %q", tid)
	}
}

func TestParseConventionPath_TenantFirstCollection(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	got := ParseConventionPath(req, "tenant_id/123")
	if got != "" {
		t.Fatalf("expected empty remainder for collection URL, got %q", got)
	}
	if tid := req.Header.Get(HeaderAuthTenantID); tid != "123" {
		t.Fatalf("expected tenant 123, got %q", tid)
	}
}

// Regression: PATCH /api/projects/tenant_id/{tid}/{pid}/ 曾因位置段被吞返回 405。
func TestParseConventionPath_TenantFirstWithPositionalSuffix(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	got := ParseConventionPath(req, "tenant_id/123/proj_1")
	if got != "proj_1" {
		t.Fatalf("expected positional suffix preserved, got %q", got)
	}
	if tid := req.Header.Get(HeaderAuthTenantID); tid != "123" {
		t.Fatalf("expected tenant 123, got %q", tid)
	}
}

func TestParseConventionPath_ResourceFirstWithTrailingPair(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	got := ParseConventionPath(req, "proj_1/tenant_id/123")
	if got != "proj_1" {
		t.Fatalf("expected resource prefix, got %q", got)
	}
	if tid := req.Header.Get(HeaderAuthTenantID); tid != "123" {
		t.Fatalf("expected tenant 123, got %q", tid)
	}
}

func TestParseConventionPath_FuncWithPositionalSuffix(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	got := ParseConventionPath(req, "func/tenant_id/123/rest")
	if got != "func/rest" {
		t.Fatalf("expected func/rest, got %q", got)
	}
}

func TestParseConventionPath_WorkspacesWithWsID(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	got := ParseConventionPath(req, "workspaces/tenant_id/123/ws_x")
	if got != "workspaces/ws_x" {
		t.Fatalf("expected workspaces/ws_x, got %q", got)
	}
}

func TestParseConventionPath_MultiplePairsAndSuffix(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	got := ParseConventionPath(req, "tenant_id/123/workspace_id/456/task_id/789/rest")
	if got != "rest" {
		t.Fatalf("expected rest, got %q", got)
	}
	if tid := req.Header.Get(HeaderAuthTenantID); tid != "123" {
		t.Fatalf("tenant: got %q", tid)
	}
	if wid := req.Header.Get("X-Workspace-Id"); wid != "456" {
		t.Fatalf("workspace: got %q", wid)
	}
	if tk := req.Header.Get("X-Task-Id"); tk != "789" {
		t.Fatalf("task: got %q", tk)
	}
}

func TestParseConventionPath_CommentIDKv(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	got := ParseConventionPath(req, "compute/container-clone-log/tenant_id/t1/workspace_id/w1/task_id/task1/comment_id/cmt-exec")
	if got != "compute/container-clone-log" {
		t.Fatalf("expected compute/container-clone-log, got %q", got)
	}
	if cid := req.Header.Get(HeaderCommentID); cid != "cmt-exec" {
		t.Fatalf("comment: got %q", cid)
	}
	if tk := req.Header.Get("X-Task-Id"); tk != "task1" {
		t.Fatalf("task: got %q", tk)
	}
	req = httptest.NewRequest("GET", "/", nil)
	got = ParseConventionPath(req, "compute/tenant_id/t1/workspace_id/w1/task_id/task1/comment_id/cmt-b/container-job-execution-log")
	if got != "compute/container-job-execution-log" {
		t.Fatalf("kv-last with comment_id: got %q", got)
	}
	if cid := req.Header.Get(HeaderCommentID); cid != "cmt-b" {
		t.Fatalf("comment: got %q", cid)
	}
}

func TestParseConventionPath_UnrecognizedKeyStopsParsing(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	got := ParseConventionPath(req, "tenant_id/123/foo/bar")
	if got != "foo/bar" {
		t.Fatalf("expected remainder foo/bar, got %q", got)
	}
	if tid := req.Header.Get(HeaderAuthTenantID); tid != "123" {
		t.Fatalf("tenant: got %q", tid)
	}
	if wid := req.Header.Get("X-Workspace-Id"); wid != "" {
		t.Fatalf("workspace must stay empty, got %q", wid)
	}
}

func TestParseConventionPath_EmptyPath(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	if got := ParseConventionPath(req, ""); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
	// 空白串非合法路径：与历史行为一致，原样返回（不 trim 空白）
}

// 回归：taskFE ai-comments 列表 URL（kv-last + 单段资源后缀）。
// 若吞掉末尾 ai-comments，taskAIComment 网关分发会 404 → 页面「ai_comments：HTTP 404」。
func TestParseConventionPath_TaskDetailAICommentsSuffix(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	got := ParseConventionPath(req, "task-detail/tenant_id/t1/workspace_id/ws1/task_id/task1/ai-comments")
	if got != "task-detail/ai-comments" {
		t.Fatalf("expected task-detail/ai-comments, got %q", got)
	}
	if tid := req.Header.Get(HeaderAuthTenantID); tid != "t1" {
		t.Fatalf("tenant: got %q", tid)
	}
	if wid := req.Header.Get("X-Workspace-Id"); wid != "ws1" {
		t.Fatalf("workspace: got %q", wid)
	}
	if tk := req.Header.Get("X-Task-Id"); tk != "task1" {
		t.Fatalf("task: got %q", tk)
	}
}
