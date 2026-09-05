package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// setTokenClientIP re-binds an existing custom token's client_ip (test helper).
func setTokenClientIP(t *testing.T, tokenKey, ip string) {
	t.Helper()
	if _, err := db.Exec(`UPDATE auth_customtoken SET client_ip = ? WHERE `+"`"+`key`+"`"+` = ?`, ip, tokenKey); err != nil {
		t.Fatalf("set token client_ip=%s: %v", ip, err)
	}
}

// OPT-20260904-006 回归：token 绑定的 client_ip 若为 docker bridge 私有地址
// （APISIX 发布端口视角 172.25.0.1），而激活请求经本地 :4000 nginx 代理到达时
// 解析为 loopback 127.0.0.1 —— 同机两种视图。此前 resolveTokenUserIDWithIP 将其
// 判为 "token ip mismatch" → activate-session 500 db error。任一侧为 loopback 应
// 视为同主机放行并把 client_ip promote 到当前地址。
func TestResolveTokenUserID_LoopbackBridgeTolerance(t *testing.T) {
	_, tokenKey := setupForwardAuthTest(t)
	setTokenClientIP(t, tokenKey, "172.25.0.1")

	uid, err := resolveTokenUserIDWithIP(tokenKey, "127.0.0.1")
	if err != nil {
		t.Fatalf("loopback↔bridge same-host should be accepted, got err=%v", err)
	}
	if uid == "" {
		t.Fatalf("expected non-empty user id")
	}
	var promoted string
	if err := db.QueryRow(`SELECT client_ip FROM auth_customtoken WHERE `+"`"+`key`+"`"+` = ?`, tokenKey).Scan(&promoted); err != nil {
		t.Fatalf("read client_ip: %v", err)
	}
	if promoted != "127.0.0.1" {
		t.Fatalf("expected client_ip promoted to 127.0.0.1, got %q", promoted)
	}
}

// 真跨网络（公网 IP 变更）仍须拒绝，并返回 errTokenIPMismatch（不是普通 DB 错误）。
func TestResolveTokenUserID_PublicIPMismatch(t *testing.T) {
	_, tokenKey := setupForwardAuthTest(t)
	setTokenClientIP(t, tokenKey, "1.2.3.4")

	if _, err := resolveTokenUserIDWithIP(tokenKey, "5.6.7.8"); !errors.Is(err, errTokenIPMismatch) {
		t.Fatalf("expected errTokenIPMismatch, got %v", err)
	}
}

// 公网 IP 一致仍正常放行。
func TestResolveTokenUserID_PublicIPMatch(t *testing.T) {
	_, tokenKey := setupForwardAuthTest(t)
	setTokenClientIP(t, tokenKey, "1.2.3.4")

	uid, err := resolveTokenUserIDWithIP(tokenKey, "1.2.3.4")
	if err != nil {
		t.Fatalf("matching public IP should be accepted, got err=%v", err)
	}
	if uid == "" {
		t.Fatalf("expected non-empty user id")
	}
}

// activate-session 对 IP 绑定不匹配应返回 401（登录环境已变化），而不再是 500 "db error"。
func TestHandleActivateSessionIPMismatchReturns401(t *testing.T) {
	userID, tokenKey := setupForwardAuthTest(t)
	setTokenClientIP(t, tokenKey, "1.2.3.4")

	req := httptest.NewRequest(http.MethodPost, "/api/accounts/users/activate-session/", strings.NewReader(`{"user_id":"`+userID+`"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Token "+tokenKey)
	req.RemoteAddr = "5.6.7.8:12345"
	rec := httptest.NewRecorder()
	handleActivateSession(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 on public-IP mismatch, got %d body=%s", rec.Code, rec.Body.String())
	}
}

// activate-session 对 loopback↔bridge 同机视图应 200（回归 OPT-20260904-006 本地 :4000 场景）。
func TestHandleActivateSessionLoopbackBridgeAccepted(t *testing.T) {
	userID, tokenKey := setupForwardAuthTest(t)
	setTokenClientIP(t, tokenKey, "172.25.0.1")

	req := httptest.NewRequest(http.MethodPost, "/api/accounts/users/activate-session/", strings.NewReader(`{"user_id":"`+userID+`"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Token "+tokenKey)
	req.RemoteAddr = "127.0.0.1:54321"
	rec := httptest.NewRecorder()
	handleActivateSession(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for same-host loopback activate, got %d body=%s", rec.Code, rec.Body.String())
	}
}
