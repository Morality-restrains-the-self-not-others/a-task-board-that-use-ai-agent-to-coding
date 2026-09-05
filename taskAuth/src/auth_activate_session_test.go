package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleActivateSessionOK(t *testing.T) {
	userID, tokenKey := setupForwardAuthTest(t)

	req := httptest.NewRequest(http.MethodPost, "/api/accounts/users/activate-session/", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Token "+tokenKey)
	rec := httptest.NewRecorder()
	handleActivateSession(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload["token"] != tokenKey {
		t.Fatalf("expected token echo, got %+v", payload["token"])
	}
	if _, leaked := payload["session_key"]; leaked {
		t.Fatalf("session_key must not appear in public JSON body")
	}
	user, _ := payload["user"].(map[string]interface{})
	if user == nil || user["id"] != userID {
		t.Fatalf("expected user.id=%s, got %+v", userID, payload["user"])
	}
}

func TestHandleActivateSessionInvalidToken(t *testing.T) {
	setupForwardAuthTest(t)
	req := httptest.NewRequest(http.MethodPost, "/api/accounts/users/activate-session/", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Token not-a-real-token")
	rec := httptest.NewRecorder()
	handleActivateSession(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleActivateSessionMissingToken(t *testing.T) {
	setupForwardAuthTest(t)
	req := httptest.NewRequest(http.MethodPost, "/api/accounts/users/activate-session/", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleActivateSession(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestHandleActivateSessionUserIDMismatch(t *testing.T) {
	userID, tokenKey := setupForwardAuthTest(t)
	_ = userID
	body := `{"user_id":"999999999999999999"}`
	req := httptest.NewRequest(http.MethodPost, "/api/accounts/users/activate-session/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Token "+tokenKey)
	rec := httptest.NewRecorder()
	handleActivateSession(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleActivateSessionUserIDMatch(t *testing.T) {
	userID, tokenKey := setupForwardAuthTest(t)
	body := `{"user_id":"` + userID + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/accounts/users/activate-session/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Token "+tokenKey)
	rec := httptest.NewRecorder()
	handleActivateSession(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
}

// OPT-20260807-071 回归：activate-session Set-Cookie 必须带父域 Domain（.daydaymoney.com）。
// 修复前无 Domain → cookie 只落 www 子域，callback 落地域 daydaymoney.com 会话失效 → 401 跳登录。
// 注意：Go 1.26 标准库 http.SetCookie 序列化会剥离 Domain 前导点（输出 Domain=daydaymoney.com），
// 与 .daydaymoney.com 浏览器语义等价（RFC 6265bis，均匹配父域及全部子域），断言接受无点形式。
func TestHandleActivateSessionCookieHasParentDomain(t *testing.T) {
	_, tokenKey := setupForwardAuthTest(t)
	oldBase := cfg.GatewayPublicBase
	cfg.GatewayPublicBase = "https://www.daydaymoney.com"
	defer func() { cfg.GatewayPublicBase = oldBase }()

	req := httptest.NewRequest(http.MethodPost, "/api/accounts/users/activate-session/", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Token "+tokenKey)
	rec := httptest.NewRecorder()
	handleActivateSession(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	sc := rec.Header().Values("Set-Cookie")
	if len(sc) < 4 {
		t.Fatalf("expected userId+token Set-Cookie headers, got %v", sc)
	}
	// 种 cookie：userId/token 各一条必须带父域 Domain（host-only 影子清除头无 Domain，
	// 由 TestHandleActivateSessionClearsLegacyHostOnlyShadows 单独断言）
	for _, name := range []string{"userId", "token"} {
		found := false
		for _, c := range sc {
			if strings.Contains(c, name+"=") && strings.Contains(c, "Domain=daydaymoney.com") {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("Set-Cookie for %s missing parent-domain Domain=daydaymoney.com（跨子域会话必需）; got %v", name, sc)
		}
	}
}

// 影子 cookie 清除：activate-session 必须在种域 cookie 的同时，双发 host-only 清除
// （071 修复前旧登录遗留的影子）。影子 token 在 www 上优先于域 cookie 被发送，
// 不除则旧 token 失效后 www 永久 401 —— 域 cookie 在任意子域可用的前提。
func TestHandleActivateSessionClearsLegacyHostOnlyShadows(t *testing.T) {
	_, tokenKey := setupForwardAuthTest(t)
	oldBase := cfg.GatewayPublicBase
	cfg.GatewayPublicBase = "https://www.daydaymoney.com"
	defer func() { cfg.GatewayPublicBase = oldBase }()

	req := httptest.NewRequest(http.MethodPost, "/api/accounts/users/activate-session/", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Token "+tokenKey)
	rec := httptest.NewRecorder()
	handleActivateSession(rec, req)

	sc := rec.Header().Values("Set-Cookie")
	if len(sc) != 4 {
		t.Fatalf("expected 4 Set-Cookie (userId/token 域种 + host-only 影子清除), got %d: %v", len(sc), sc)
	}
	// 每个名字必须有一个不带 Domain 的 Max-Age=0 清除头
	for _, name := range []string{"userId", "token"} {
		found := false
		for _, c := range sc {
			if strings.Contains(c, name+"=") &&
				!strings.Contains(c, "Domain=") &&
				strings.Contains(c, "Max-Age=0") {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected host-only clearing Set-Cookie for %s (no Domain, Max-Age=0), got %v", name, sc)
		}
	}
}
