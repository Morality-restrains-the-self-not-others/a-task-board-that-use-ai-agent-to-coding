package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// ---- 测试工具 ----

const testClientID = "test-oidc-client"
const testClientSecret = "test-oidc-secret"
const testRedirectURI = "http://127.0.0.1:8012/users/auth/openid_connect/callback"

// oidcAuthorizeWithScope 走完 authorize → 返回授权码。
func oidcAuthorizeWithScope(t *testing.T, scope string) string {
	t.Helper()
	token := oidcTestToken(t)

	params := url.Values{}
	params.Set("client_id", testClientID)
	params.Set("redirect_uri", testRedirectURI)
	params.Set("response_type", "code")
	params.Set("scope", scope)
	params.Set("state", "rt-test-state")
	params.Set("nonce", "rt-test-nonce")

	req := httptest.NewRequest(http.MethodGet, "/api/oidc/authorize?"+params.Encode(), nil)
	req.Header.Set("Authorization", "Token "+token)
	rec := httptest.NewRecorder()
	handleOidcAuthorize(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("authorize: expected 302, got %d body=%s", rec.Code, rec.Body.String())
	}
	location, err := url.Parse(rec.Header().Get("Location"))
	if err != nil {
		t.Fatalf("parse redirect: %v", err)
	}
	code := location.Query().Get("code")
	if code == "" {
		t.Fatalf("no code in redirect: %s", location)
	}
	return code
}

// oidcExchangeForm 以 form 编码 POST /api/oidc/token，返回响应（code/body）。
func oidcExchangeForm(t *testing.T, form url.Values) (int, map[string]interface{}) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/oidc/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	handleOidcToken(rec, req)

	var resp map[string]interface{}
	if rec.Body.Len() > 0 {
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	}
	return rec.Code, resp
}

// oidcExchangeCode 用授权码换 token（不带 PKCE，走 code_verifier 可选路径）。
func oidcExchangeCode(t *testing.T, code string) (int, map[string]interface{}) {
	t.Helper()
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("client_id", testClientID)
	form.Set("client_secret", testClientSecret)
	form.Set("redirect_uri", testRedirectURI)
	return oidcExchangeForm(t, form)
}

// oidcExchangeRefresh 用 refresh token 换新 token。
func oidcExchangeRefresh(t *testing.T, refreshToken, secret string) (int, map[string]interface{}) {
	t.Helper()
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)
	form.Set("client_id", testClientID)
	if secret == "" {
		secret = testClientSecret
	}
	form.Set("client_secret", secret)
	return oidcExchangeForm(t, form)
}

// ---- 测试 ----

func TestOidcTokenExchangeIssuesRefreshTokenWithOfflineAccess(t *testing.T) {
	_, cleanup := oidcTestSetup(t)
	defer cleanup()

	code := oidcAuthorizeWithScope(t, "openid profile email offline_access")
	status, resp := oidcExchangeCode(t, code)

	if status != http.StatusOK {
		t.Fatalf("token: expected 200, got %d body=%v", status, resp)
	}
	accessToken, _ := resp["access_token"].(string)
	refreshToken, _ := resp["refresh_token"].(string)
	if accessToken == "" {
		t.Fatal("access_token missing")
	}
	if refreshToken == "" {
		t.Fatal("refresh_token missing (offline_access requested)")
	}
	if !strings.Contains(refreshToken, "invalid") && len(refreshToken) < 32 {
		t.Fatalf("refresh_token looks suspicious: %q", refreshToken)
	}
	// token_type / expires_in 保持既有契约
	if tt, _ := resp["token_type"].(string); tt != "Bearer" {
		t.Fatalf("token_type: %q", tt)
	}
	if exp, _ := resp["expires_in"].(float64); exp != 3600 {
		t.Fatalf("expires_in: %v", exp)
	}
}

func TestOidcTokenExchangeNoRefreshTokenWithoutOfflineAccess(t *testing.T) {
	_, cleanup := oidcTestSetup(t)
	defer cleanup()

	// 不请求 offline_access（gitService OmniAuth 现状）→ 行为不变：无 refresh_token
	code := oidcAuthorizeWithScope(t, "openid profile email")
	status, resp := oidcExchangeCode(t, code)

	if status != http.StatusOK {
		t.Fatalf("token: expected 200, got %d body=%v", status, resp)
	}
	if _, ok := resp["refresh_token"]; ok {
		t.Fatalf("refresh_token should NOT be issued without offline_access: %v", resp)
	}
	if _, ok := resp["access_token"]; !ok {
		t.Fatal("access_token missing")
	}
}

func TestOidcRefreshTokenGrantRotatesAndOldTokenReplayFails(t *testing.T) {
	_, cleanup := oidcTestSetup(t)
	defer cleanup()

	code := oidcAuthorizeWithScope(t, "openid offline_access")
	status, resp := oidcExchangeCode(t, code)
	if status != http.StatusOK {
		t.Fatalf("exchange: %d %v", status, resp)
	}
	oldRefresh, _ := resp["refresh_token"].(string)
	if oldRefresh == "" {
		t.Fatal("initial refresh_token missing")
	}

	// 刷新：返回新 access_token + 轮换后的新 refresh_token
	status2, resp2 := oidcExchangeRefresh(t, oldRefresh, "")
	if status2 != http.StatusOK {
		t.Fatalf("refresh: expected 200, got %d body=%v", status2, resp2)
	}
	newAccess, _ := resp2["access_token"].(string)
	newRefresh, _ := resp2["refresh_token"].(string)
	if newAccess == "" {
		t.Fatal("refreshed access_token missing")
	}
	if newRefresh == "" {
		t.Fatal("rotated refresh_token missing")
	}
	if newRefresh == oldRefresh {
		t.Fatal("refresh token must rotate (same value reused)")
	}
	// id_token 随 openid scope 补发
	if idt, _ := resp2["id_token"].(string); idt == "" {
		t.Fatal("id_token missing on refresh grant (openid scope)")
	}

	// 新 access token 可直接访问 userinfo
	ur := httptest.NewRequest(http.MethodGet, "/api/oidc/userinfo", nil)
	ur.Header.Set("Authorization", "Bearer "+newAccess)
	urec := httptest.NewRecorder()
	handleOidcUserInfo(urec, ur)
	if urec.Code != http.StatusOK {
		t.Fatalf("userinfo with refreshed access_token: %d body=%s", urec.Code, urec.Body.String())
	}

	// 旧 refresh token 重放 → 轮换后已撤销 → invalid_grant
	status3, resp3 := oidcExchangeRefresh(t, oldRefresh, "")
	if status3 != http.StatusBadRequest {
		t.Fatalf("replay: expected 400, got %d body=%v", status3, resp3)
	}
	if errCode, _ := resp3["error"].(string); errCode != "invalid_grant" {
		t.Fatalf("replay error code: %q (want invalid_grant)", errCode)
	}

	// 新 refresh token 仍可再次轮换（正常续期链路）
	status4, resp4 := oidcExchangeRefresh(t, newRefresh, "")
	if status4 != http.StatusOK {
		t.Fatalf("second refresh: expected 200, got %d body=%v", status4, resp4)
	}
	if rt, _ := resp4["refresh_token"].(string); rt == "" || rt == newRefresh {
		t.Fatal("second rotation must issue another distinct refresh_token")
	}
}

func TestOidcRefreshTokenExpired(t *testing.T) {
	_, cleanup := oidcTestSetup(t)
	defer cleanup()

	code := oidcAuthorizeWithScope(t, "openid offline_access")
	status, resp := oidcExchangeCode(t, code)
	if status != http.StatusOK {
		t.Fatalf("exchange: %d %v", status, resp)
	}
	refreshToken, _ := resp["refresh_token"].(string)
	if refreshToken == "" {
		t.Fatal("refresh_token missing")
	}

	// 直接改库把 expires_at 拨到过去 → refresh 应 invalid_grant
	past := time.Now().UTC().Add(-time.Hour).Format("2006-01-02 15:04:05.000000")
	if _, err := db.Exec(`UPDATE auth_oidc_refresh_token SET expires_at = ?`, past); err != nil {
		t.Fatalf("force expire: %v", err)
	}

	status2, resp2 := oidcExchangeRefresh(t, refreshToken, "")
	if status2 != http.StatusBadRequest {
		t.Fatalf("expired refresh: expected 400, got %d body=%v", status2, resp2)
	}
	if errCode, _ := resp2["error"].(string); errCode != "invalid_grant" {
		t.Fatalf("error code: %q (want invalid_grant)", errCode)
	}
}

func TestOidcRefreshTokenUnknownToken(t *testing.T) {
	_, cleanup := oidcTestSetup(t)
	defer cleanup()

	status, resp := oidcExchangeRefresh(t, "deadbeefdeadbeefdeadbeef", "")
	if status != http.StatusBadRequest {
		t.Fatalf("unknown refresh: expected 400, got %d body=%v", status, resp)
	}
	if errCode, _ := resp["error"].(string); errCode != "invalid_grant" {
		t.Fatalf("error code: %q (want invalid_grant)", errCode)
	}
}

func TestOidcRefreshTokenWrongClientSecret(t *testing.T) {
	_, cleanup := oidcTestSetup(t)
	defer cleanup()

	code := oidcAuthorizeWithScope(t, "openid offline_access")
	status, resp := oidcExchangeCode(t, code)
	if status != http.StatusOK {
		t.Fatalf("exchange: %d %v", status, resp)
	}
	refreshToken, _ := resp["refresh_token"].(string)

	status2, resp2 := oidcExchangeRefresh(t, refreshToken, "wrong-secret")
	if status2 != http.StatusUnauthorized {
		t.Fatalf("wrong secret: expected 401, got %d body=%v", status2, resp2)
	}
	if errCode, _ := resp2["error"].(string); errCode != "invalid_client" {
		t.Fatalf("error code: %q (want invalid_client)", errCode)
	}
}

func TestOidcRefreshTokenUserInactive(t *testing.T) {
	_, cleanup := oidcTestSetup(t)
	defer cleanup()

	code := oidcAuthorizeWithScope(t, "openid offline_access")
	status, resp := oidcExchangeCode(t, code)
	if status != http.StatusOK {
		t.Fatalf("exchange: %d %v", status, resp)
	}
	refreshToken, _ := resp["refresh_token"].(string)
	if refreshToken == "" {
		t.Fatal("refresh_token missing")
	}

	// 停用用户 → refresh 应 invalid_grant（fail-closed：不续期给已停用账号）
	// auth_user 无 identifier 列：经 auth_login_method.email 反查 user id 再停用
	var uid string
	err := db.QueryRow(`
		SELECT object_id FROM auth_login_method
		WHERE identifier = 'test@example.com' AND method_type = 'email'
		  AND binding_voided_at IS NULL ORDER BY created_at LIMIT 1`).Scan(&uid)
	if err != nil || uid == "" {
		t.Fatalf("locate test user: %v", err)
	}
	if _, err := db.Exec(`UPDATE auth_user SET is_active = 0 WHERE id = ?`, uid); err != nil {
		t.Fatalf("deactivate user: %v", err)
	}

	status2, resp2 := oidcExchangeRefresh(t, refreshToken, "")
	if status2 != http.StatusBadRequest {
		t.Fatalf("inactive user refresh: expected 400, got %d body=%v", status2, resp2)
	}
	if errCode, _ := resp2["error"].(string); errCode != "invalid_grant" {
		t.Fatalf("error code: %q (want invalid_grant)", errCode)
	}
}

func TestOidcRefreshTokenMissingToken(t *testing.T) {
	_, cleanup := oidcTestSetup(t)
	defer cleanup()

	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("client_id", testClientID)
	form.Set("client_secret", testClientSecret)
	status, resp := oidcExchangeForm(t, form)
	if status != http.StatusBadRequest {
		t.Fatalf("missing refresh_token: expected 400, got %d body=%v", status, resp)
	}
	if errCode, _ := resp["error"].(string); errCode != "invalid_request" {
		t.Fatalf("error code: %q (want invalid_request)", errCode)
	}
}

func TestOidcDiscoveryAdvertisesOfflineAccessAndRefreshGrant(t *testing.T) {
	_, cleanup := oidcTestSetup(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/.well-known/openid-configuration", nil)
	rec := httptest.NewRecorder()
	handleOidcDiscovery(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("discovery: %d", rec.Code)
	}
	var doc map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	scopes, _ := json.Marshal(doc["scopes_supported"])
	if !strings.Contains(string(scopes), "offline_access") {
		t.Fatalf("scopes_supported missing offline_access: %s", scopes)
	}
	grants, _ := json.Marshal(doc["grant_types_supported"])
	if !strings.Contains(string(grants), "refresh_token") {
		t.Fatalf("grant_types_supported missing refresh_token: %s", grants)
	}
}

func TestOidcRefreshTokenStoredHashedNotPlaintext(t *testing.T) {
	_, cleanup := oidcTestSetup(t)
	defer cleanup()

	code := oidcAuthorizeWithScope(t, "openid offline_access")
	status, resp := oidcExchangeCode(t, code)
	if status != http.StatusOK {
		t.Fatalf("exchange: %d %v", status, resp)
	}
	refreshToken, _ := resp["refresh_token"].(string)
	if refreshToken == "" {
		t.Fatal("refresh_token missing")
	}

	var storedHash string
	if err := db.QueryRow(`SELECT token_hash FROM auth_oidc_refresh_token ORDER BY created_at DESC LIMIT 1`).Scan(&storedHash); err != nil {
		t.Fatalf("query stored hash: %v", err)
	}
	if storedHash == refreshToken {
		t.Fatal("refresh token stored in plaintext — must be hashed")
	}
	expected := sha256Hash(refreshToken)
	if storedHash != expected {
		t.Fatalf("stored hash mismatch: %q != sha256(token)", storedHash)
	}
	_ = fmt.Sprint()
}
