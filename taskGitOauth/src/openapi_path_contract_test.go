package main

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"taskGitOauth/infrastructure"
)

// pathParamRe 匹配 OpenAPI 路径模板中的 {param} 占位符。
var pathParamRe = regexp.MustCompile(`\{[^}]+\}`)

// TestOpenAPIPathsMatchRegisteredRoutes 回归 OPT-20260807-022：
// openapi.go 文档登记的路径必须与 RegisterRoutes 实际注册的路由一致（防再度漂移）。
// 对每个文档路径，将 {param} 占位符替换为样本值后用 mux.Handler 校验能命中注册模式；
// 反向：RegisterRoutes 中 /api/git-oauth/ 与 /redirect/ 前缀路由必须出现在文档中。
// TestUserAppConnectionServiceProviderOptional 回归 OPT-20260809-008：
// /api/git-oauth/user-app-connection/ 的 GET/DELETE service_provider 契约必须非必填
// （handler 按缺省 "default" 解析，任务详情页只传 repo_url）。
func TestUserAppConnectionServiceProviderOptional(t *testing.T) {
	doc := openAPIDocument()
	paths, ok := doc["paths"].(map[string]any)
	if !ok {
		t.Fatal("missing paths")
	}
	entry, ok := paths["/api/git-oauth/user-app-connection/"].(map[string]any)
	if !ok {
		t.Fatal("missing user-app-connection path")
	}
	for _, method := range []string{"get", "delete"} {
		op, ok := entry[method].(map[string]any)
		if !ok {
			t.Fatalf("missing %s operation", method)
		}
		params, ok := op["parameters"].([]any)
		if !ok {
			t.Fatalf("%s missing parameters", method)
		}
		found := false
		for _, p := range params {
			pm, ok := p.(map[string]any)
			if !ok {
				continue
			}
			if pm["name"] == "service_provider" {
				found = true
				if req, _ := pm["required"].(bool); req {
					t.Errorf("%s service_provider must be optional", method)
				}
			}
		}
		if !found {
			t.Errorf("%s missing service_provider parameter", method)
		}
	}
}

func TestOpenAPIPathsMatchRegisteredRoutes(t *testing.T) {
	cfg := &infrastructure.Config{
		SecretKey:           infrastructure.DefaultSecretKey,
		Port:                8002,
		Host:                "127.0.0.1",
		PublicBaseURL:       "http://localhost:8002",
		BridgeJWTSecret:     "test-secret",
		BridgeJWTIssuer:     "task2app",
		BridgeJWTAudienceGH: "gitOauth-github-oauth",
		BridgeJWTAudienceGL: "gitOauth-gitlab-oauth",
		Task2appInternalAPI: "http://127.0.0.1:8001",
		Providers:           map[string][]infrastructure.ProviderConfig{},
	}
	app := NewApp(cfg, nil)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	doc := openAPIDocument()
	pathsMap, ok := doc["paths"].(map[string]any)
	if !ok || len(pathsMap) == 0 {
		t.Fatal("openapi document missing paths")
	}

	sample := map[string]string{
		"service_provider": "github",
		"gitsite":          "github.com",
		"tid":              "1",
	}
	concretize := func(p string) string {
		return pathParamRe.ReplaceAllStringFunc(p, func(m string) string {
			name := m[1 : len(m)-1]
			if v, ok := sample[name]; ok {
				return v
			}
			return "x"
		})
	}

	for p := range pathsMap {
		if p == "/api/health/" || p == "/api/swagger/" || p == "/api/schema/" {
			continue
		}
		concrete := concretize(p)
		req := httptest.NewRequest(http.MethodGet, concrete, nil)
		h, pattern := mux.Handler(req)
		if h == nil || pattern == "" {
			t.Errorf("documented path %q (→ %q) has no registered route", p, concrete)
		}
	}

	// 反向校验：实际注册的关键前缀路由必须都有文档条目
	expected := []string{
		"/api/git-oauth/providers/",
		"/api/git-oauth/github-start/",
		"/api/git-oauth/github-start-from-gateway/",
		"/api/git-oauth/github-callback/",
		"/api/git-oauth/gitlab-start/",
		"/api/git-oauth/gitlab-start-from-gateway/",
		"/api/git-oauth/gitlab-callback/",
		"/api/accounts/{service_provider}/oauth/callback/",
		"/redirect/gitsite/{gitsite}/oauth/callback/",
		"/api/git-oauth/user-app-connection/",
		"/api/git-oauth/tenant-connection/tenant_id/{tid}/",
		"/api/git-oauth/tenant-connection/tenant_id/{tid}/reachability/",
		"/api/git-oauth/tenant-connection/{tid}/gitlab-oauth-connection/",
		"/api/git-oauth/merge-request-status/tenant_id/{tid}/",
		"/api/git-oauth/merge-request-merge/tenant_id/{tid}/",
	}
	for _, p := range expected {
		if _, ok := pathsMap[p]; !ok {
			t.Errorf("registered route %q missing from openapi document", p)
		}
	}
}
