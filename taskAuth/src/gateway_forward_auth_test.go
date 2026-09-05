package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// setupForwardAuthTest initialises a MySQL test DB and returns the userID and token.
func setupForwardAuthTest(t *testing.T) (userID string, tokenKey string) {
	t.Helper()
	cfg.InternalSecret = "test-secret"
	setupAuthTestDB(t)

	uid, _, err := createUserWithEmailLogin("fwd-auth@test.com", "hash")
	if err != nil {
		t.Fatalf("createUserWithEmailLogin: %v", err)
	}
	tok, err := getOrCreateToken(uid, cfg.UserContentTypeID, "")
	if err != nil {
		t.Fatalf("getOrCreateToken: %v", err)
	}
	return uid, tok
}

func TestGatewayForwardAuthRejectsMissingSecret(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/internal/gateway/forward-auth/", nil)
	rec := httptest.NewRecorder()
	handleGatewayForwardAuth(rec, req)
	if rec.Code != http.StatusUnauthorized && rec.Code != http.StatusForbidden {
		t.Fatalf("expected 401 or 403 without credentials, got %d", rec.Code)
	}
}

func TestGatewayForwardAuthTokenHeader(t *testing.T) {
	userID, tokenKey := setupForwardAuthTest(t)
	req := httptest.NewRequest(http.MethodGet, "/api/internal/gateway/forward-auth/", nil)
	req.Header.Set("X-TaskAuth-Internal-Secret", cfg.InternalSecret)
	req.Header.Set("Authorization", "Token "+tokenKey)
	rec := httptest.NewRecorder()
	handleGatewayForwardAuth(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for valid token, got %d", rec.Code)
	}
	if rec.Header().Get("X-User-Id") != userID {
		t.Fatalf("expected X-User-Id=%s, got %s", userID, rec.Header().Get("X-User-Id"))
	}
	// X-User-Email：主邮箱随 forward-auth 注入（resolveUserEmail，供下游厂商资格匹配）
	if got := rec.Header().Get("X-User-Email"); got != "fwd-auth@test.com" {
		t.Fatalf("expected X-User-Email=fwd-auth@test.com, got %q", got)
	}
}

func TestGatewayForwardAuthUserIdCookie(t *testing.T) {
	userID, _ := setupForwardAuthTest(t)

	// OPT-20260807-004: 裸 userId cookie 必须被拒绝（同域侧可伪造裸 ID）。
	req := httptest.NewRequest(http.MethodGet, "/api/internal/gateway/forward-auth/", nil)
	req.Header.Set("X-TaskAuth-Internal-Secret", cfg.InternalSecret)
	req.AddCookie(&http.Cookie{Name: "userId", Value: userID})
	rec := httptest.NewRecorder()
	handleGatewayForwardAuth(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("bare userId cookie must be rejected (OPT-20260807-004), got %d", rec.Code)
	}

	// 签名形态 <id>.<ts>.<hmac> + 活跃用户 + 活 token 行 → 200（SSO 桥浏览器导航兜底）。
	signed, err := signUserIDCookieValue(userID)
	if err != nil {
		t.Fatalf("signUserIDCookieValue: %v", err)
	}
	req = httptest.NewRequest(http.MethodGet, "/api/internal/gateway/forward-auth/", nil)
	req.Header.Set("X-TaskAuth-Internal-Secret", cfg.InternalSecret)
	req.AddCookie(&http.Cookie{Name: "userId", Value: signed})
	rec = httptest.NewRecorder()
	handleGatewayForwardAuth(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for signed active userId cookie (SSO bridge fallback), got %d", rec.Code)
	}
}

func TestGatewayForwardAuthUserIdCookieUnknownUser(t *testing.T) {
	setupForwardAuthTest(t)
	// 签名合法但用户不存在 → 401（userIsActive 校验）。
	signed, err := signUserIDCookieValue("999999999999999999")
	if err != nil {
		t.Fatalf("signUserIDCookieValue: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/internal/gateway/forward-auth/", nil)
	req.Header.Set("X-TaskAuth-Internal-Secret", cfg.InternalSecret)
	req.AddCookie(&http.Cookie{Name: "userId", Value: signed})
	rec := httptest.NewRecorder()
	handleGatewayForwardAuth(rec, req)
	// userId cookie fallback 需用户存在且活跃（userIsActive 校验）。
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unknown userId cookie, got %d", rec.Code)
	}
}

// TestGatewayForwardAuthUserIdCookieRequiresLiveToken reproduces the reported
// bug: after clearing + re-initialising the database, the user stayed logged in.
//
// Chain: login creates a row in auth_customtoken; a DB reset wipes that table
// but re-seeds auth_user with deterministic IDs (e.g. 'bootstrap-admin').
// A browser still holding the 30-day HttpOnly userId cookie must NOT be
// authenticated by it alone — the cookie must prove a live session token
// (OPT-20260807-002).
func TestGatewayForwardAuthUserIdCookieRequiresLiveToken(t *testing.T) {
	userID, tokenKey := setupForwardAuthTest(t)
	// OPT-20260807-004: cookie 必须为签名形态；裸 ID 直接拒绝（见
	// TestGatewayForwardAuthUserIdCookie），此处用签名 cookie 验证「活 token」约束。
	signed, err := signUserIDCookieValue(userID)
	if err != nil {
		t.Fatalf("signUserIDCookieValue: %v", err)
	}

	// Sanity: with a live token row, the userId cookie fallback works.
	req := httptest.NewRequest(http.MethodGet, "/api/internal/gateway/forward-auth/", nil)
	req.Header.Set("X-TaskAuth-Internal-Secret", cfg.InternalSecret)
	req.AddCookie(&http.Cookie{Name: "userId", Value: signed})
	rec := httptest.NewRecorder()
	handleGatewayForwardAuth(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("sanity: expected 200 with live token row, got %d", rec.Code)
	}

	// Simulate DB reset: the auth_customtoken row is gone (re-init wipes it).
	if err := deleteTokenByKey(tokenKey); err != nil {
		t.Fatalf("deleteTokenByKey: %v", err)
	}
	// The sanity call cached a 200 entry — flush so the reset is observable.
	clearForwardAuthCacheForTest()

	rec2 := httptest.NewRecorder()
	handleGatewayForwardAuth(rec2, req)
	if rec2.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 after token wiped (DB reset) with userId cookie only, got %d", rec2.Code)
	}
}

// TestForwardAuthResolveUserIDFromRequestUserIdCookieRequiresLiveToken covers
// the OIDC-path resolver (resolveTokenUserIDFromRequest): same rule — a bare
// userId cookie must not authenticate a user without a live token row.
func TestForwardAuthResolveUserIDFromRequestUserIdCookieRequiresLiveToken(t *testing.T) {
	userID, tokenKey := setupForwardAuthTest(t)
	signed, err := signUserIDCookieValue(userID)
	if err != nil {
		t.Fatalf("signUserIDCookieValue: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/internal/oidc/authorize", nil)
	req.AddCookie(&http.Cookie{Name: "userId", Value: signed})
	if uid, err := resolveTokenUserIDFromRequest(req); err != nil || uid != userID {
		t.Fatalf("sanity: expected userId cookie to resolve with live token, got uid=%q err=%v", uid, err)
	}

	if err := deleteTokenByKey(tokenKey); err != nil {
		t.Fatalf("deleteTokenByKey: %v", err)
	}
	if uid, err := resolveTokenUserIDFromRequest(req); err == nil {
		t.Fatalf("expected error after token wiped (DB reset) with userId cookie only, got uid=%q", uid)
	}
}

func TestGatewayForwardAuthNoCredentials(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/internal/gateway/forward-auth/", nil)
	req.Header.Set("X-TaskAuth-Internal-Secret", cfg.InternalSecret)
	rec := httptest.NewRecorder()
	handleGatewayForwardAuth(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with no credentials, got %d", rec.Code)
	}
}

func TestGatewayForwardAuthTokenPriorityOverCookie(t *testing.T) {
	userID, tokenKey := setupForwardAuthTest(t)

	// Set a different userId cookie to verify token takes priority
	req := httptest.NewRequest(http.MethodGet, "/api/internal/gateway/forward-auth/", nil)
	req.Header.Set("X-TaskAuth-Internal-Secret", cfg.InternalSecret)
	req.Header.Set("Authorization", "Token "+tokenKey)
	req.AddCookie(&http.Cookie{Name: "userId", Value: "999999999999999998"})
	rec := httptest.NewRecorder()
	handleGatewayForwardAuth(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	// Token header (method 1) takes priority over userId cookie (method 4)
	if rec.Header().Get("X-User-Id") != userID {
		t.Fatalf("expected token user %s to take priority, got %s", userID, rec.Header().Get("X-User-Id"))
	}
}

// Chrome 插件带 Authorization: Bearer（OAuth JWT）时，网页会话 Cookie token
// 不得抢身份——否则浮窗会用网页账号去拉插件账号的租户工作空间，403「您不是该公司的成员」。
func TestGatewayForwardAuthBearerPriorityOverCookieToken(t *testing.T) {
	cookieUserID, cookieToken := setupForwardAuthTest(t)
	bearerUserID, _, err := createUserWithEmailLogin("fwd-bearer-plugin@test.com", "hash")
	if err != nil {
		t.Fatalf("createUserWithEmailLogin bearer user: %v", err)
	}

	oldNow := cfg.nowUnix
	oldTTL := cfg.OidcAccessTokenTTL
	t.Cleanup(func() {
		cfg.nowUnix = oldNow
		cfg.OidcAccessTokenTTL = oldTTL
	})
	cfg.nowUnix = func() int64 { return time.Now().Unix() }
	cfg.OidcAccessTokenTTL = 3600
	if err := initOidcSigningKey(""); err != nil {
		t.Fatalf("initOidcSigningKey: %v", err)
	}
	bearer, err := makeAccessToken(bearerUserID, "http://127.0.0.1:8003", "chrome-extension", nil)
	if err != nil {
		t.Fatalf("makeAccessToken: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/internal/gateway/forward-auth/", nil)
	req.Header.Set("X-TaskAuth-Internal-Secret", cfg.InternalSecret)
	req.Header.Set("Authorization", "Bearer "+bearer)
	req.AddCookie(&http.Cookie{Name: "token", Value: cookieToken})
	rec := httptest.NewRecorder()
	handleGatewayForwardAuth(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	got := rec.Header().Get("X-User-Id")
	if got != bearerUserID {
		t.Fatalf("expected Bearer user %s to beat Cookie token user %s, got %s", bearerUserID, cookieUserID, got)
	}
}

func TestGatewayForwardAuthCacheHit(t *testing.T) {
	clearForwardAuthCacheForTest()
	t.Setenv("TASKAUTH_FORWARD_AUTH_CACHE_TTL_MS", "5000")
	userID, tokenKey := setupForwardAuthTest(t)

	req1 := httptest.NewRequest(http.MethodGet, "/api/internal/gateway/forward-auth/", nil)
	req1.Header.Set("X-TaskAuth-Internal-Secret", cfg.InternalSecret)
	req1.Header.Set("Authorization", "Token "+tokenKey)
	rec1 := httptest.NewRecorder()
	handleGatewayForwardAuth(rec1, req1)
	if rec1.Code != http.StatusOK {
		t.Fatalf("miss: expected 200, got %d", rec1.Code)
	}
	if rec1.Header().Get("X-Auth-Cache") != "miss" {
		t.Fatalf("expected cache miss, got %q", rec1.Header().Get("X-Auth-Cache"))
	}

	req2 := httptest.NewRequest(http.MethodGet, "/api/internal/gateway/forward-auth/", nil)
	req2.Header.Set("X-TaskAuth-Internal-Secret", cfg.InternalSecret)
	req2.Header.Set("Authorization", "Token "+tokenKey)
	rec2 := httptest.NewRecorder()
	handleGatewayForwardAuth(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("hit: expected 200, got %d", rec2.Code)
	}
	if rec2.Header().Get("X-Auth-Cache") != "hit" {
		t.Fatalf("expected cache hit, got %q", rec2.Header().Get("X-Auth-Cache"))
	}
	if rec2.Header().Get("X-User-Id") != userID {
		t.Fatalf("user id mismatch on hit")
	}
	// 缓存命中路径同样注入 X-User-Email（entry 缓存了 userEmail）
	if got := rec2.Header().Get("X-User-Email"); got != "fwd-auth@test.com" {
		t.Fatalf("expected X-User-Email=fwd-auth@test.com on cache hit, got %q", got)
	}
}

// ---- OPT-20260825-005 未验证手机号业务写门禁回归测 ----

// forwardAuthWriteRequest 构造带原始方法/路径头的 forward-auth 请求
// （APISIX forward-auth 子请求固定 GET，原始方法/路径经 X-Forwarded-Method/Uri 注入）。
func forwardAuthWriteRequest(tokenKey, method, uri string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/api/internal/gateway/forward-auth/", nil)
	req.Header.Set("X-TaskAuth-Internal-Secret", cfg.InternalSecret)
	req.Header.Set("Authorization", "Token "+tokenKey)
	req.Header.Set("X-Forwarded-Method", method)
	req.Header.Set("X-Forwarded-Uri", uri)
	return req
}

func TestGatewayForwardAuthPhoneGateBlocksUnverifiedWrite(t *testing.T) {
	clearForwardAuthCacheForTest()
	userID, tokenKey := setupForwardAuthTest(t)
	// email 登录用户：无已验证 phone login_method → 客户业务写路径应 403
	req := forwardAuthWriteRequest(tokenKey, "POST", "/api/tenant/123/accounts/groups/")
	rec := httptest.NewRecorder()
	handleGatewayForwardAuth(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for unverified phone write, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "phone_verification_required") {
		t.Fatalf("expected error code phone_verification_required, got body=%s", rec.Body.String())
	}
	_ = userID
}

func TestGatewayForwardAuthPhoneGateAllowsVerifiedPhoneWrite(t *testing.T) {
	clearForwardAuthCacheForTest()
	cfg.InternalSecret = "test-secret"
	setupAuthTestDB(t)
	uid, err := createUserWithPhoneLogin("+86", "13800138000", "hash", "")
	if err != nil {
		t.Fatalf("createUserWithPhoneLogin: %v", err)
	}
	tok, err := getOrCreateToken(uid, cfg.UserContentTypeID, "")
	if err != nil {
		t.Fatalf("getOrCreateToken: %v", err)
	}
	req := forwardAuthWriteRequest(tok, "POST", "/api/tenant/123/accounts/groups/")
	rec := httptest.NewRecorder()
	handleGatewayForwardAuth(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for verified phone write, got %d body=%s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("X-User-Phone-Verified") != "1" {
		t.Fatalf("expected X-User-Phone-Verified=1, got %q", rec.Header().Get("X-User-Phone-Verified"))
	}
}

func TestGatewayForwardAuthPhoneGateAllowsStaffWrite(t *testing.T) {
	clearForwardAuthCacheForTest()
	userID, tokenKey := setupForwardAuthTest(t)
	if _, err := db.Exec(`UPDATE auth_user SET is_staff = 1, is_superuser = 1 WHERE id = ?`, userID); err != nil {
		t.Fatalf("mark staff: %v", err)
	}
	req := forwardAuthWriteRequest(tokenKey, "POST", "/api/tenant/123/accounts/groups/")
	rec := httptest.NewRecorder()
	handleGatewayForwardAuth(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for staff write, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestGatewayForwardAuthPhoneGateAllowsRead(t *testing.T) {
	clearForwardAuthCacheForTest()
	_, tokenKey := setupForwardAuthTest(t)
	// 未验证手机用户读请求不受门禁影响
	req := forwardAuthWriteRequest(tokenKey, "GET", "/api/tenant/123/accounts/groups/")
	rec := httptest.NewRecorder()
	handleGatewayForwardAuth(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for GET, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestGatewayForwardAuthPhoneGateAllowsJoinPath(t *testing.T) {
	clearForwardAuthCacheForTest()
	_, tokenKey := setupForwardAuthTest(t)
	// 成员加入（invite/join）是 SPA 对未验证手机豁免的写路径
	req := forwardAuthWriteRequest(tokenKey, "POST", "/api/tenant/123/accounts/members/join/")
	rec := httptest.NewRecorder()
	handleGatewayForwardAuth(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for members/join write, got %d body=%s", rec.Code, rec.Body.String())
	}
	req = forwardAuthWriteRequest(tokenKey, "POST", "/api/tenant/123/accounts/members/invite/")
	rec = httptest.NewRecorder()
	handleGatewayForwardAuth(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for members/invite write, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestGatewayForwardAuthPhoneGateAllowsNonBusinessPath(t *testing.T) {
	clearForwardAuthCacheForTest()
	_, tokenKey := setupForwardAuthTest(t)
	// 认证/资料路径（/api/accounts/）不受客户业务写门禁影响（手机号绑定依赖这些路径）
	req := forwardAuthWriteRequest(tokenKey, "POST", "/api/accounts/users/bind_phone/")
	rec := httptest.NewRecorder()
	handleGatewayForwardAuth(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for /api/accounts/ write, got %d body=%s", rec.Code, rec.Body.String())
	}
}

// TestPhoneWriteGateBlocksUnit 覆盖 phoneWriteGateBlocks 的豁免/回退分支：
// 模拟登录豁免、无头回退、非业务路径、已认证员工。
func TestPhoneWriteGateBlocksUnit(t *testing.T) {
	base := forwardAuthCacheEntry{userID: "u1"}
	impersonated := forwardAuthCacheEntry{userID: "u1", impersonatorID: "actor-1"}
	staffRole := forwardAuthCacheEntry{userID: "u1", platformRoles: []string{"employee"}}
	verified := forwardAuthCacheEntry{userID: "u1", phoneVerified: true}

	writeReq := func(method, uri string) *http.Request {
		req := httptest.NewRequest(http.MethodGet, "/api/internal/gateway/forward-auth/", nil)
		if method != "" {
			req.Header.Set("X-Forwarded-Method", method)
		}
		if uri != "" {
			req.Header.Set("X-Forwarded-Uri", uri)
		}
		return req
	}

	cases := []struct {
		name string
		ent  forwardAuthCacheEntry
		req  *http.Request
		want bool
	}{
		{"unverified tenant POST blocks", base, writeReq("POST", "/api/tenant/123/accounts/groups/"), true},
		{"PUT blocks", base, writeReq("PUT", "/api/tenant/123/accounts/groups/"), true},
		{"DELETE blocks", base, writeReq("DELETE", "/api/tenant/123/accounts/groups/"), true},
		{"GET read allowed", base, writeReq("GET", "/api/tenant/123/accounts/groups/"), false},
		{"non-business path allowed", base, writeReq("POST", "/api/accounts/users/bind_phone/"), false},
		{"business prefix projects allowed? no blocks", base, writeReq("POST", "/api/projects/123/tasks/"), true},
		{"members/join exempt", base, writeReq("POST", "/api/tenant/123/accounts/members/join/"), false},
		{"members/invite exempt", base, writeReq("POST", "/api/tenant/123/accounts/members/invite/"), false},
		{"impersonation bypass", impersonated, writeReq("POST", "/api/tenant/123/accounts/groups/"), false},
		{"staff role bypass", staffRole, writeReq("POST", "/api/tenant/123/accounts/groups/"), false},
		{"verified phone allowed", verified, writeReq("POST", "/api/tenant/123/accounts/groups/"), false},
		{"header fallback to r.Method", base, func() *http.Request {
			req := httptest.NewRequest(http.MethodPost, "/api/internal/gateway/forward-auth/", nil)
			req.Header.Set("X-Forwarded-Uri", "/api/tenant/123/accounts/groups/")
			return req
		}(), true},
		{"header fallback to r.URL.Path", base, func() *http.Request {
			req := httptest.NewRequest(http.MethodGet, "/api/internal/gateway/forward-auth/", nil)
			req.Header.Set("X-Forwarded-Method", "POST")
			req.URL.Path = "/api/tenant/123/accounts/groups/"
			return req
		}(), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := phoneWriteGateBlocks(tc.req, tc.ent); got != tc.want {
				t.Fatalf("phoneWriteGateBlocks()=%v want %v", got, tc.want)
			}
		})
	}
}

func TestGatewayForwardAuthTesterHeaderNotInRoles(t *testing.T) {
	userID, tokenKey := setupForwardAuthTest(t)
	if _, err := db.Exec(`UPDATE auth_user SET is_tester = 1, is_tenant = 1 WHERE id = ?`, userID); err != nil {
		t.Fatalf("mark tester: %v", err)
	}
	clearForwardAuthCacheForTest()
	req := httptest.NewRequest(http.MethodGet, "/api/internal/gateway/forward-auth/", nil)
	req.Header.Set("X-TaskAuth-Internal-Secret", cfg.InternalSecret)
	req.Header.Set("Authorization", "Token "+tokenKey)
	rec := httptest.NewRecorder()
	handleGatewayForwardAuth(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("X-User-Is-Tester") != "1" {
		t.Fatalf("X-User-Is-Tester=%q", rec.Header().Get("X-User-Is-Tester"))
	}
	roles := rec.Header().Get("X-User-Roles")
	if strings.Contains(roles, "tester") {
		t.Fatalf("X-User-Roles must not contain tester, got %q", roles)
	}
}
