package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseContainerComputePathGitCommit(t *testing.T) {
	path := "/api/tenant/827923618468040704/workspace/827923618602258432/task/847744505890045952/cloud/compute/container-layer-git-commit/"
	match, ok := parseContainerComputePath(path)
	if !ok {
		t.Fatal("expected path to match")
	}
	if match.Action != "container-layer-git-commit" {
		t.Fatalf("action=%q want container-layer-git-commit", match.Action)
	}
	if match.Scope.TenantID != "827923618468040704" || match.Scope.WorkspaceID != "827923618602258432" || match.Scope.TaskID != "847744505890045952" {
		t.Fatalf("unexpected scope: %+v", match.Scope)
	}
}

func TestParseContainerComputePathRejectsNonContainerAction(t *testing.T) {
	path := "/api/tenant/t1/workspace/w1/task/task1/cloud/compute/server-startup/"
	if _, ok := parseContainerComputePath(path); ok {
		t.Fatal("expected non container-* action to be rejected")
	}
}

// 新约定形态（funcName-first kv-last）：taskCloudService /api/cloud/* 原样代理转发，
// 网关须与旧形态等价解析，否则前端文件树等全部 404。
func TestParseContainerComputePathFuncNameFirstKvLast(t *testing.T) {
	path := "/api/cloud/compute/container-layer-children/tenant_id/874176608758427648/workspace_id/ws_-3077429177147314749/task_id/task_15389083077187235865/"
	match, ok := parseContainerComputePath(path)
	if !ok {
		t.Fatal("expected funcName-first kv-last path to match")
	}
	if match.Action != "container-layer-children" {
		t.Fatalf("action=%q want container-layer-children", match.Action)
	}
	if match.Scope.TenantID != "874176608758427648" || match.Scope.WorkspaceID != "ws_-3077429177147314749" || match.Scope.TaskID != "task_15389083077187235865" {
		t.Fatalf("unexpected scope: %+v", match.Scope)
	}
}

func TestParseContainerComputePathCommentIDKv(t *testing.T) {
	path := "/api/cloud/compute/container-clone-log/tenant_id/t1/workspace_id/w1/task_id/task1/comment_id/cmt-exec/"
	match, ok := parseContainerComputePath(path)
	if !ok {
		t.Fatal("expected path with comment_id kv to match")
	}
	if match.Action != "container-clone-log" {
		t.Fatalf("action=%q", match.Action)
	}
	if match.Scope.CommentID != "cmt-exec" || match.Scope.TaskID != "task1" {
		t.Fatalf("unexpected scope: %+v", match.Scope)
	}
	kvLast := "/api/cloud/compute/tenant_id/t1/workspace_id/w1/task_id/task1/comment_id/cmt-b/container-job-execution-log/"
	match, ok = parseContainerComputePath(kvLast)
	if !ok {
		t.Fatal("expected kv-last with comment_id to match")
	}
	if match.Action != "container-job-execution-log" || match.Scope.CommentID != "cmt-b" {
		t.Fatalf("unexpected match: %+v", match)
	}
}

// 新约定形态（kv-last，apiBase 拼接）：funcName 在 kv 之后，同样须解析。
func TestParseContainerComputePathKvLast(t *testing.T) {
	path := "/api/cloud/compute/tenant_id/t1/workspace_id/w1/task_id/task1/container-job-execution-log/"
	match, ok := parseContainerComputePath(path)
	if !ok {
		t.Fatal("expected kv-last path to match")
	}
	if match.Action != "container-job-execution-log" {
		t.Fatalf("action=%q want container-job-execution-log", match.Action)
	}
	if match.Scope.TenantID != "t1" || match.Scope.WorkspaceID != "w1" || match.Scope.TaskID != "task1" {
		t.Fatalf("unexpected scope: %+v", match.Scope)
	}
}

func TestParseContainerComputePathKvLastRejectsMissing(t *testing.T) {
	for _, path := range []string{
		"/api/cloud/compute/tenant_id/t1/workspace_id/w1/container-job-execution-log/",         // 缺 task_id
		"/api/cloud/compute/tenant_id/t1/workspace_id/w1/task_id/container-job-execution-log/", // task_id 缺值
		"/api/cloud/compute/container-layer-children/unknown_key/x/tenant_id/t1/workspace_id/w1/task_id/task1/",
		"/api/cloud/compute/server-startup/tenant_id/t1/workspace_id/w1/task_id/task1/", // 非 container-* 动作
	} {
		if _, ok := parseContainerComputePath(path); ok {
			t.Fatalf("expected %q to be rejected", path)
		}
	}
}

// 回归（OPT-20260809-030）：前端残缺形态 `task_id/{值}{funcName}`（funcName 缺 /
// 分隔粘在 task_id 值后）曾造成线上「读取文件树失败（HTTP 404）」。funcName-first
// 形态的粘尾（container-layer-children 被当 action、粘尾被当 task_id 值）必须拒绝。
func TestParseContainerComputePathRejectsFuncNameStuckToTaskId(t *testing.T) {
	for _, path := range []string{
		// funcName-first 形态粘尾：粘尾被吞成 task_id 值
		"/api/cloud/compute/container-layer-graph/tenant_id/t1/workspace_id/w1/task_id/task1container-layer-graph/",
		"/api/cloud/compute/container-layer-children/tenant_id/t1/workspace_id/w1/task_id/task1container-layer-children/",
		"/api/cloud/compute/container-bootstrap-clone-log/tenant_id/t1/workspace_id/w1/task_id/task1container-bootstrap-clone-log/",
		// kv-last 形态粘尾：粘尾被吞成 task_id 值（无独立 action 段）
		"/api/cloud/compute/tenant_id/t1/workspace_id/w1/task_id/task1container-layer-graph/",
		// 旧形态粘尾：action 段粘在 task 值后
		"/api/tenant/t1/workspace/w1/task/task1container-layer-graph/cloud/compute/container-layer-graph/",
	} {
		if _, ok := parseContainerComputePath(path); ok {
			t.Fatalf("expected %q (funcName stuck to task_id) to be rejected", path)
		}
	}
	// 对照组：合法 task_id 值（含 task_ 前缀、无 container-）不受影响
	for _, path := range []string{
		"/api/cloud/compute/container-layer-graph/tenant_id/t1/workspace_id/w1/task_id/task_15389083077187235865/",
		"/api/cloud/compute/tenant_id/t1/workspace_id/w1/task_id/task1/container-job-execution-log/",
	} {
		if _, ok := parseContainerComputePath(path); !ok {
			t.Fatalf("expected %q to be accepted", path)
		}
	}
}

func TestBuildGitCommitForward(t *testing.T) {
	raw := []byte(`{"layer_id":"layer-1","message":" init ","stage_all":true}`)
	sc := scope{TenantID: "t1", WorkspaceID: "w1", TaskID: "task1"}
	up, body, errDetail := buildGitCommitForward("http://127.0.0.1:8765/", sc, raw)
	if errDetail != "" {
		t.Fatalf("unexpected err: %s", errDetail)
	}
	if up != "http://127.0.0.1:8765/api/tenant/t1/workspace/w1/task/task1/layers/layer-1/git/commit" {
		t.Fatalf("upstream=%q", up)
	}
	var parsed map[string]any
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed["message"] != "init" {
		t.Fatalf("message=%v", parsed["message"])
	}
	if parsed["stage_all"] != true {
		t.Fatalf("stage_all=%v", parsed["stage_all"])
	}
}

func TestParseContainerComputePathLayerGraph(t *testing.T) {
	path := "/api/tenant/850256677331562496/workspace/857903329669984256/task/859354231702982656/cloud/compute/container-layer-graph/"
	match, ok := parseContainerComputePath(path)
	if !ok {
		t.Fatal("expected path to match")
	}
	if match.Action != "container-layer-graph" {
		t.Fatalf("action=%q want container-layer-graph", match.Action)
	}
	if match.Scope.TenantID != "850256677331562496" {
		t.Fatalf("tenant=%q", match.Scope.TenantID)
	}
}

func TestCheckOnlineServiceReady_InvalidURL(t *testing.T) {
	ok, msg := checkOnlineServiceReady(context.Background(), "://invalid", "")
	if ok {
		t.Fatal("expected false for invalid url")
	}
	if msg == "" {
		t.Fatal("expected diagnosis message")
	}
}

func TestCheckOnlineServiceReady_EmptyURL(t *testing.T) {
	ok, msg := checkOnlineServiceReady(context.Background(), "", "")
	if ok {
		t.Fatal("expected false for empty url")
	}
	if msg == "" {
		t.Fatal("expected diagnosis message")
	}
}

func TestCommentIDFromRequestPrefersPathThenQueryThenBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/cloud/compute/container-layer-children/comment_id/cmt-path/?comment_id=cmt-q", nil)
	if got := commentIDFromRequest(req, []byte(`{"comment_id":"cmt-body"}`)); got != "cmt-path" {
		t.Fatalf("got %q want cmt-path", got)
	}
	req = httptest.NewRequest(http.MethodGet, "/api/cloud/compute/container-layer-children/?comment_id=cmt-q", nil)
	if got := commentIDFromRequest(req, []byte(`{"comment_id":"cmt-body"}`)); got != "cmt-q" {
		t.Fatalf("got %q want cmt-q", got)
	}
	req = httptest.NewRequest(http.MethodPost, "/api/cloud/compute/container-layer-children/", nil)
	if got := commentIDFromRequest(req, []byte(`{"comment_id":"cmt-body"}`)); got != "cmt-body" {
		t.Fatalf("got %q want cmt-body", got)
	}
}

func TestLookupCommentIDPrefersScopePathOverEmptyQuery(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/cloud/compute/container-layer-command/tenant_id/t/workspace_id/w/task_id/k/", nil)
	got := lookupCommentID(req, scope{CommentID: "cmt-path"})
	if got != "cmt-path" {
		t.Fatalf("got %q want cmt-path from scope", got)
	}
	req = httptest.NewRequest(http.MethodPost, "/api/cloud/compute/container-layer-command/tenant_id/t/workspace_id/w/task_id/k/comment_id/cmt-url/", nil)
	got = lookupCommentID(req, scope{})
	if got != "cmt-url" {
		t.Fatalf("got %q want cmt-url from path", got)
	}
}

func TestContainerPageURLFromRequestUsesQueryOnGET(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/x/?container_page_url=http://10.0.0.1:8765/ui/tok", nil)
	if got := containerPageURLFromRequest(req, nil); got != "http://10.0.0.1:8765/ui/tok" {
		t.Fatalf("got %q", got)
	}
}

// 前端 apiBase 拼接是 kv-last：/api/cloud/compute/tenant_id/.../task_id/{task}/{action}/
// 旧 mux 用 Contains("/cloud/compute/container-")，该子串只出现在 funcName-first，
// kv-last 会落到 http.NotFound → 页面克隆日志/任务日志只显示裸「404」。
func TestMountRoutesDispatchesKvLastContainerCloneAndJobLog(t *testing.T) {
	mux := http.NewServeMux()
	mountRoutes(mux)
	paths := []string{
		"/api/cloud/compute/tenant_id/t1/workspace_id/w1/task_id/task1/container-clone-log/?layer_id=L1",
		"/api/cloud/compute/tenant_id/t1/workspace_id/w1/task_id/task1/container-job-execution-log/?job_id=J1",
		"/api/cloud/compute/container-clone-log/tenant_id/t1/workspace_id/w1/task_id/task1/?layer_id=L1",
		"/api/tenant/t1/workspace/w1/task/task1/cloud/compute/container-clone-log/?layer_id=L1",
	}
	for _, path := range paths {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code == http.StatusNotFound && strings.Contains(rec.Body.String(), "404 page not found") {
			t.Fatalf("path %s hit net/http NotFound (mux missed kv-last container action)", path)
		}
	}
}

func TestHandleContainerComputeUnknownPathReturnsJSONDetail(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/cloud/compute/server-startup/tenant_id/t1/workspace_id/w1/task_id/task1/", nil)
	rec := httptest.NewRecorder()
	handleContainerCompute(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d want 404", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "404 page not found") {
		t.Fatal("plain net/http NotFound body")
	}
	var payload map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("body is not JSON: %v %q", err, rec.Body.String())
	}
	if payload["detail"] == "" {
		t.Fatalf("expected JSON detail, got %#v", payload)
	}
}

func TestMountRoutesDispatchesRelayBeforeContainerCompute(t *testing.T) {
	mux := http.NewServeMux()
	mountRoutes(mux)
	req := httptest.NewRequest(http.MethodGet, "/api/cloud/compute/relay-to-trae/tenant_id/t1/workspace_id/w1/task_id/task1/status/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if strings.Contains(rec.Body.String(), "unknown container compute path") {
		t.Fatal("relay-to-trae path was dispatched to handleContainerCompute")
	}
}
