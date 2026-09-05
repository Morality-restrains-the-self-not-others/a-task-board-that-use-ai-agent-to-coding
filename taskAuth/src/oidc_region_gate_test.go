package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// startOidcRegionGateBillServer 模拟 taskBill 内部只读 API gitlab-regions-admin。
func startOidcRegionGateBillServer(t *testing.T, regions []map[string]string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/internal/taskbill/gitlab-regions-admin/" {
			http.NotFound(w, r)
			return
		}
		out := make([]map[string]string, 0, len(regions))
		out = append(out, regions...)
		writeJSON(w, http.StatusOK, map[string]interface{}{"regions": out, "total": len(out)})
	}))
	t.Cleanup(srv.Close)
	return srv
}

// startOidcRegionGateBillServer500 模拟 bill 内部 API 故障（500）。
func startOidcRegionGateBillServer500(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "billing unavailable", http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// oidcRegionGateTestSetup 注册一个 GitLab 区域 OIDC client，并把 bill 内部 API 指向 mock。
func oidcRegionGateTestSetup(t *testing.T, clientID, redirectURI string, regions []map[string]string) {
	t.Helper()
	if err := ensureOidcClient(clientID, "oidc-region-gate-test-secret", clientID,
		`["`+redirectURI+`"]`); err != nil {
		t.Fatalf("ensureOidcClient: %v", err)
	}
	cfg.BillServiceURL = startOidcRegionGateBillServer(t, regions).URL
}

// oidcAuthorizeAsUser 以指定 token 请求 OIDC authorize。
func oidcAuthorizeAsUser(t *testing.T, token, clientID, redirectURI string) *httptest.ResponseRecorder {
	t.Helper()
	params := url.Values{}
	params.Set("client_id", clientID)
	params.Set("redirect_uri", redirectURI)
	params.Set("response_type", "code")
	params.Set("scope", "openid profile email")
	params.Set("state", "s1")
	params.Set("nonce", "n1")

	req := httptest.NewRequest(http.MethodGet, "/api/oidc/authorize?"+params.Encode(), nil)
	req.Header.Set("Authorization", "Token "+token)
	rec := httptest.NewRecorder()
	handleOidcAuthorize(rec, req)
	return rec
}

func assertOidcCodeIssued(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d body=%s", rec.Code, rec.Body.String())
	}
	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "code=") {
		t.Fatalf("expected code issued, location=%q", loc)
	}
}

func assertOidcAccessDenied(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302 redirect, got %d body=%s", rec.Code, rec.Body.String())
	}
	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "error=access_denied") {
		t.Fatalf("expected error=access_denied, location=%q", loc)
	}
	if strings.Contains(loc, "code=") {
		t.Fatalf("must not issue code, location=%q", loc)
	}
}

// OPT-20260823-051 三个验收场景：
//  1. development 区域 + 非测试角色 → 拒绝签发授权码
//  2. development 区域 + 测试角色 → 放行
//  3. release 区域 + 非测试角色 → 放行
func TestOidcAuthorizeDevRegionBlocksNonTester(t *testing.T) {
	_, cleanup := oidcTestSetup(t)
	defer cleanup()
	t.Setenv("TASKAUTH_OIDC_REGION_MODE_CACHE_TTL_MS", "0")
	resetOidcRegionModeCacheForTest()

	const (
		clientID    = "gitlab-git-service-tencent-sh-1"
		redirectURI = "https://gitlab-tencent-sh-1.daydaymoney.com/users/auth/openid_connect/callback"
	)
	oidcRegionGateTestSetup(t, clientID, redirectURI, []map[string]string{
		{"gitlab_web_url": "https://gitlab-tencent-sh-1.daydaymoney.com", "access_mode": "development"},
		{"gitlab_web_url": "https://gitlab.daydaymoney.com", "access_mode": "release"},
	})

	token := oidcTestToken(t) // test@example.com，默认 is_tester=0
	rec := oidcAuthorizeAsUser(t, token, clientID, redirectURI)
	assertOidcAccessDenied(t, rec)
}

func TestOidcAuthorizeDevRegionAllowsTester(t *testing.T) {
	_, cleanup := oidcTestSetup(t)
	defer cleanup()
	t.Setenv("TASKAUTH_OIDC_REGION_MODE_CACHE_TTL_MS", "0")
	resetOidcRegionModeCacheForTest()

	const (
		clientID    = "gitlab-git-service-tencent-sh-1"
		redirectURI = "https://gitlab-tencent-sh-1.daydaymoney.com/users/auth/openid_connect/callback"
	)
	oidcRegionGateTestSetup(t, clientID, redirectURI, []map[string]string{
		{"gitlab_web_url": "https://gitlab-tencent-sh-1.daydaymoney.com", "access_mode": "development"},
	})

	// 将 test@example.com 标记为测试角色（测试等同租户，setUserTesterFlag 同时置 is_tenant）
	lm, err := findLoginMethodByEmail("test@example.com")
	if err != nil || lm == nil {
		t.Fatalf("findLoginMethodByEmail: err=%v row=%v", err, lm)
	}
	if err := setUserTesterFlag(context.Background(), lm.ObjectID, true); err != nil {
		t.Fatalf("setUserTesterFlag: %v", err)
	}

	token := oidcTestToken(t)
	rec := oidcAuthorizeAsUser(t, token, clientID, redirectURI)
	assertOidcCodeIssued(t, rec)
}

func TestOidcAuthorizeReleaseRegionAllowsNonTester(t *testing.T) {
	_, cleanup := oidcTestSetup(t)
	defer cleanup()
	t.Setenv("TASKAUTH_OIDC_REGION_MODE_CACHE_TTL_MS", "0")
	resetOidcRegionModeCacheForTest()

	const (
		clientID    = "gitlab-git-service" // 默认实例 → tencent-shanghai-5
		redirectURI = "https://gitlab.daydaymoney.com/users/auth/openid_connect/callback"
	)
	oidcRegionGateTestSetup(t, clientID, redirectURI, []map[string]string{
		{"gitlab_web_url": "https://gitlab.daydaymoney.com", "access_mode": "release"},
		{"gitlab_web_url": "https://gitlab-tencent-sh-1.daydaymoney.com", "access_mode": "development"},
	})

	token := oidcTestToken(t) // is_tester=0
	rec := oidcAuthorizeAsUser(t, token, clientID, redirectURI)
	assertOidcCodeIssued(t, rec)
}

// 非 GitLab 区域 client（如 ai-provider / chrome-extension）不参与区域模式判定。
func TestOidcAuthorizeNonGitlabClientUnaffectedByDevRegion(t *testing.T) {
	_, cleanup := oidcTestSetup(t)
	defer cleanup()
	t.Setenv("TASKAUTH_OIDC_REGION_MODE_CACHE_TTL_MS", "0")
	resetOidcRegionModeCacheForTest()

	const (
		clientID    = "ai-provider"
		redirectURI = "https://provider.daydaymoney.com/api/auth/oidc/callback/"
	)
	oidcRegionGateTestSetup(t, clientID, redirectURI, []map[string]string{
		{"gitlab_web_url": "https://gitlab-tencent-sh-1.daydaymoney.com", "access_mode": "development"},
	})

	token := oidcTestToken(t)
	rec := oidcAuthorizeAsUser(t, token, clientID, redirectURI)
	assertOidcCodeIssued(t, rec)
}

// 区域模式查询失败（bill 内部 API 不可达）→ fail-open 放行并告警（缺省 release）。
func TestOidcAuthorizeDevRegionFailOpenWhenBillUnavailable(t *testing.T) {
	_, cleanup := oidcTestSetup(t)
	defer cleanup()
	t.Setenv("TASKAUTH_OIDC_REGION_MODE_CACHE_TTL_MS", "0")
	resetOidcRegionModeCacheForTest()

	const (
		clientID    = "gitlab-git-service-tencent-sh-1"
		redirectURI = "https://gitlab-tencent-sh-1.daydaymoney.com/users/auth/openid_connect/callback"
	)
	if err := ensureOidcClient(clientID, "oidc-region-gate-test-secret", clientID,
		`["`+redirectURI+`"]`); err != nil {
		t.Fatalf("ensureOidcClient: %v", err)
	}
	// bill 内部 API 返回 500（服务不可达等价路径）
	cfg.BillServiceURL = startOidcRegionGateBillServer500(t).URL

	token := oidcTestToken(t)
	rec := oidcAuthorizeAsUser(t, token, clientID, redirectURI)
	assertOidcCodeIssued(t, rec)
}
