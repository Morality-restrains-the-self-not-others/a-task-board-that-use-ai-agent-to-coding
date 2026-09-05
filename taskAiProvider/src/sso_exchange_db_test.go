package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"taskAiProvider/infrastructure"
)

// SSO exchange 关键链路测试（2026-08-06「无效的 bridge」回归防护）。
// 命名统一含 "SSO" 前缀 — taskAiProvider/.githooks/pre-commit 的必跑门禁
// 以 `go test -run 'SSO'` 在每次提交前执行本文件与 infrastructure 的 SSO 测试，
// 新 SSO 相关测试请保持 TestSSO... 命名。
//
// 成功路径依赖真实 MySQL（dbload.OpenTestMySQL 每测试独立建库），
// 与 taskAuth/src/db_test.go 同模式：MySQL 不可用时 t.Skip，门禁不误报。
// 测试库初始化/迁移应用共享 helper 见 testdb_test.go（OPT-20260806-063 合并）。

// signSSOBridge 以 App 解析到的真源 SSOJwtSecret 签发 taskAuth 风格 bridge JWT。
func signSSOBridge(t *testing.T, app *App, claims map[string]any) string {
	t.Helper()
	bridge, err := infrastructure.SignHS256(app.Cfg.SSOJwtSecret, claims)
	if err != nil {
		t.Fatalf("SignHS256: %v", err)
	}
	return bridge
}

func taskAuthStyleClaims(app *App, sub, typ string, extra map[string]any) map[string]any {
	now := time.Now().Unix()
	claims := map[string]any{
		"iss": app.Cfg.SSOJwtIssuer,
		"aud": app.Cfg.SSOAudience,
		"sub": sub,
		"typ": typ,
		"iat": now,
		"exp": now + 300,
	}
	for k, v := range extra {
		claims[k] = v
	}
	return claims
}

// ── 成功路径（ExchangeBridge + 真实 MySQL） ──

func TestSSOExchangeGetValidStaffBridge(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	bridge := signSSOBridge(t, app, taskAuthStyleClaims(app, "90001", "staff_bridge", nil))
	req := httptest.NewRequest(http.MethodGet, "/api/auth/sso/exchange/?bridge="+bridge, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d body=%s", rec.Code, rec.Body.String())
	}
	loc := rec.Header().Get("Location")
	if !strings.HasPrefix(loc, "/admin#token=") {
		t.Fatalf("expected /admin#token= redirect for staff, got %s", loc)
	}
	tok := strings.TrimPrefix(loc, "/admin#token=")
	claims, err := infrastructure.ParseHS256JWT(tok, app.Cfg.SecretKey, infrastructure.JWTIssuer, "", "staff")
	if err != nil {
		t.Fatalf("exchanged staff token invalid: %v", err)
	}
	if infrastructure.ClaimString(claims, "sub") == "" {
		t.Fatal("exchanged staff token missing sub")
	}
}

func TestSSOExchangeGetValidVendorBridge(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	// OPT-20260806-065 审核流：自动建号已移除，先经申请/审核建档（is_active=1）再兑换
	if _, err := app.DB.SQL.Exec(`INSERT INTO ai_provider_vendor
		(saas_user_id, email, password_hash, company_name, contact_name, is_active, created_at, updated_at)
		VALUES (90002, 'sso-vendor-test@example.com', 'x', 'VendorCo', 'Contact', 1, NOW(), NOW())`); err != nil {
		t.Fatalf("seed vendor: %v", err)
	}
	bridge := signSSOBridge(t, app, taskAuthStyleClaims(app, "90002", "vendor_bridge",
		map[string]any{"email": "sso-vendor-test@example.com"}))
	req := httptest.NewRequest(http.MethodGet, "/api/auth/sso/exchange/?bridge="+bridge, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d body=%s", rec.Code, rec.Body.String())
	}
	loc := rec.Header().Get("Location")
	if !strings.HasPrefix(loc, "/#token=") {
		t.Fatalf("expected /#token= redirect for vendor, got %s", loc)
	}
	tok := strings.TrimPrefix(loc, "/#token=")
	claims, err := infrastructure.ParseHS256JWT(tok, app.Cfg.SecretKey, infrastructure.JWTIssuer, "", "vendor")
	if err != nil {
		t.Fatalf("exchanged vendor token invalid: %v", err)
	}
	if infrastructure.ClaimString(claims, "sub") == "" {
		t.Fatal("exchanged vendor token missing sub")
	}
}

// TestSSOExchangeRejectsSyntheticEmailVendorBridge（OPT-20260806-065 纵深防御）：
// vendor_bridge 携带合成邮箱（sso-<id>@sso.invalid，无邮箱登录方式用户的确定性
// 兜底值）时不得建号/兑换 —— 即使旧版 taskAuth 仍签发或 bridge 被手工构造，
// provider 侧也拒绝自动建号，强制用户先绑定邮箱。
func TestSSOExchangeRejectsSyntheticEmailVendorBridge(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	bridge := signSSOBridge(t, app, taskAuthStyleClaims(app, "90009", "vendor_bridge",
		map[string]any{"email": "sso-90009@sso.invalid"}))
	req := httptest.NewRequest(http.MethodPost, "/api/auth/sso/exchange/",
		strings.NewReader(`{"bridge":"`+bridge+`"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for synthetic email, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	detail, _ := out["detail"].(string)
	if !strings.Contains(detail, "绑定邮箱") {
		t.Fatalf("expected detail mentioning 绑定邮箱, got %q", detail)
	}
	// 未建号：ai_provider_vendor 无该用户记录
	var n int
	if err := app.DB.SQL.QueryRow(`SELECT COUNT(1) FROM ai_provider_vendor WHERE saas_user_id=90009`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("expected no vendor auto-created for synthetic email (n=%d err=%v)", n, err)
	}
}

func TestSSOExchangePostValidStaffBridge(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	bridge := signSSOBridge(t, app, taskAuthStyleClaims(app, "90003", "staff_bridge", nil))
	body := `{"bridge":"` + bridge + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/sso/exchange/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out["role"] != "staff" {
		t.Fatalf("expected role=staff, got %v", out["role"])
	}
	if tok, _ := out["access"].(string); tok == "" {
		t.Fatal("expected access token in response")
	}
}

// ── 回归防护：错误密钥（本次生产故障模式） ──

// TestSSOExchangeRejectsWrongSecretBridge 复现 2026-08-06 生产故障：
// taskAuth 签发密钥与 provider 验证密钥不一致（旧回退链解析出 DefaultSecretKey）
// 时，合法结构的 bridge 必须被拒绝并返回「无效的 bridge」而非静默通过。
func TestSSOExchangeRejectsWrongSecretBridge(t *testing.T) {
	app := testAppMinimal(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	claims := taskAuthStyleClaims(app, "42", "staff_bridge", nil)
	// 用 DefaultSecretKey（旧 config_jwt.go 回退链的错误值）签名
	wrongBridge, err := infrastructure.SignHS256(infrastructure.DefaultSecretKey, claims)
	if err != nil {
		t.Fatal(err)
	}

	// GET：浏览器路径 → 302 /?error=无效的 bridge
	req := httptest.NewRequest(http.MethodGet, "/api/auth/sso/exchange/?bridge="+wrongBridge, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "error=") {
		t.Fatalf("expected error redirect, got %s", loc)
	}

	// POST：API 路径 → 401 detail=无效的 bridge
	body := `{"bridge":"` + wrongBridge + `"}`
	req2 := httptest.NewRequest(http.MethodPost, "/api/auth/sso/exchange/", strings.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rec2.Code, rec2.Body.String())
	}
	var out map[string]any
	_ = json.Unmarshal(rec2.Body.Bytes(), &out)
	if out["detail"] != "无效的 bridge" {
		t.Fatalf("expected 无效的 bridge, got %v", out["detail"])
	}
}
