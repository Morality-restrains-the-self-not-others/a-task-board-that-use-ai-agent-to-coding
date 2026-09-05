package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTranslateBranchTitleChineseUsesFanyiAgent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			t.Fatalf("missing bearer auth")
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]string{"content": "Fix Login Page Styles"}},
			},
		})
	}))
	defer srv.Close()

	old := fanyiAgentCfg
	fanyiAgentCfg = FanyiAgentConfig{
		APIKey: "test-key", BaseURL: srv.URL, Model: "test-model", MaxTokens: 256, Temperature: 0.1, TopP: 1,
	}
	t.Cleanup(func() { fanyiAgentCfg = old })

	body := `{"title":"修复登录页样式"}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/projects/translate-branch-title/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	handleProjectsRoute(rec, req, "t1", []string{"translate-branch-title"})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var payload map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload["translated_title"] != "fix-login-page-styles" {
		t.Fatalf("translated_title=%v", payload["translated_title"])
	}
	if payload["used_ai"] != true {
		t.Fatalf("used_ai=%v", payload["used_ai"])
	}
}

func TestTranslateBranchTitleChineseConfigIncompleteIs502(t *testing.T) {
	old := fanyiAgentCfg
	fanyiAgentCfg = FanyiAgentConfig{}
	t.Cleanup(func() { fanyiAgentCfg = old })

	body := `{"title":"修复登录页样式"}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/projects/translate-branch-title/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleProjectsRoute(rec, req, "t1", []string{"translate-branch-title"})
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "翻译服务暂未就绪") {
		t.Fatalf("body=%s", rec.Body.String())
	}
}

func TestTranslateBranchTitleChineseUpstreamErrorIs502(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"boom"}`))
	}))
	defer srv.Close()

	old := fanyiAgentCfg
	fanyiAgentCfg = FanyiAgentConfig{
		APIKey: "k", BaseURL: srv.URL, Model: "m", MaxTokens: 64,
	}
	t.Cleanup(func() { fanyiAgentCfg = old })

	body := `{"title":"中文标题"}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/projects/translate-branch-title/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleProjectsRoute(rec, req, "t1", []string{"translate-branch-title"})
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestTranslateBranchTitleEmptyUpstreamBodyIsUserSafe502(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	old := fanyiAgentCfg
	fanyiAgentCfg = FanyiAgentConfig{APIKey: "k", BaseURL: srv.URL, Model: "m", MaxTokens: 64}
	t.Cleanup(func() { fanyiAgentCfg = old })

	body := `{"title":"中文标题"}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/projects/translate-branch-title/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Trace-Id", "tid-empty-json")
	rec := httptest.NewRecorder()
	handleProjectsRoute(rec, req, "t1", []string{"translate-branch-title"})
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d: %s", rec.Code, rec.Body.String())
	}
	got := rec.Body.String()
	if strings.Contains(got, "unexpected end of JSON input") {
		t.Fatalf("user-facing body leaked json parse error: %s", got)
	}
	if strings.Contains(got, "fanyi_agent") {
		t.Fatalf("user-facing body leaked fanyi_agent internals: %s", got)
	}
	if !strings.Contains(got, "翻译服务响应异常") {
		t.Fatalf("want categorized 响应异常 reason, got %s", got)
	}
	if !strings.Contains(got, "tid-empty-json") {
		t.Fatalf("want trace_id in error body, got %s", got)
	}
}

func TestTranslateBranchTitleEmptyContentExplainsWhy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"finish_reason": "length",
					"message":       map[string]string{"content": "", "reasoning_content": "long thinking"},
				},
			},
		})
	}))
	t.Cleanup(srv.Close)

	old := fanyiAgentCfg
	fanyiAgentCfg = FanyiAgentConfig{APIKey: "k", BaseURL: srv.URL, Model: "m", MaxTokens: 64}
	t.Cleanup(func() { fanyiAgentCfg = old })

	body := `{"title":"中文标题"}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/projects/translate-branch-title/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleProjectsRoute(rec, req, "t1", []string{"translate-branch-title"})
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d: %s", rec.Code, rec.Body.String())
	}
	got := rec.Body.String()
	if !strings.Contains(got, "未返回可用译文") {
		t.Fatalf("want why=未返回可用译文, got %s", got)
	}
	if strings.Contains(got, "finish_reason") || strings.Contains(got, "reasoning_content") {
		t.Fatalf("user-facing must not leak finish_reason/reasoning: %s", got)
	}
}

func TestTranslateTitleUserFacingErrorCategories(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		{"fanyi_agent 请求超时", "响应超时"},
		{"fanyi_agent 返回内容为空: status=200 bytes=1142 finish_reason=length", "未返回可用译文"},
		{"fanyi_agent 响应无效: status=200", "响应异常"},
		{"AI_AGENT_CONFIG.fanyi_agent 配置不完整", "暂未就绪"},
		{"fanyi_agent 请求失败: status=500", "暂时不可用"},
	}
	for _, tc := range cases {
		got := translateTitleUserFacingError(fmt.Errorf("%s", tc.raw))
		if !strings.Contains(got, tc.want) {
			t.Fatalf("raw=%q got=%q want substring %q", tc.raw, got, tc.want)
		}
		if strings.Contains(got, "fanyi_agent") {
			t.Fatalf("leaked fanyi_agent: %s", got)
		}
	}
}

func TestSanitizeBranchTitleSegment(t *testing.T) {
	if got := sanitizeBranchTitleSegment("Fix Login!!!"); got != "fix-login" {
		t.Fatalf("got=%s", got)
	}
	if got := sanitizeBranchTitleSegment("   "); got != "task" {
		t.Fatalf("empty got=%s", got)
	}
}
