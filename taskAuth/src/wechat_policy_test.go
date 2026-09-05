package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// 微信登录策略开关（auth_system_feature_policy.enable_wechat_login）回归：
// 管理员未开启时，登录发起与回调都必须拒绝；绑定流程（登录态账号管理）不受影响。

// setWechatPolicyEnabled 直接写入策略行（保留其余字段默认值），测试结束不恢复
// （setupAuthTestDB 每个用例独立建库，天然隔离）。
func setWechatPolicyEnabled(t *testing.T, enabled bool) {
	t.Helper()
	if err := saveFeaturePolicy(&systemFeaturePolicy{
		EnablePhoneLogin:                true,
		EnableRechargePhoneVerification: false,
		EnableEmailRegister:             true,
		EnableWechatLogin:               enabled,
		AllowedPhoneCountryCodes:        []string{},
	}); err != nil {
		t.Fatalf("saveFeaturePolicy: %v", err)
	}
}

// TestWeChatLoginRejectedWhenPolicyDisabled — 修复回归：管理员未开启微信登录时，
// 发起扫码登录必须被拒绝（此前仅检查应用配置，开关形同虚设）。
func TestWeChatLoginRejectedWhenPolicyDisabled(t *testing.T) {
	setupAuthTestDB(t)
	setTestWechatApp(t)
	setWechatPolicyEnabled(t, false)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/wechat/login/", nil)
	rec := httptest.NewRecorder()
	handleWeChatLogin(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("policy off: status = %d, want 403 (body: %s)", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "未开启") {
		t.Fatalf("policy off: body should mention 未开启, got %s", rec.Body.String())
	}
}

// TestWeChatLoginAllowedWhenPolicyEnabled — 管理员开启后正常跳转微信授权。
func TestWeChatLoginAllowedWhenPolicyEnabled(t *testing.T) {
	setupAuthTestDB(t)
	setTestWechatApp(t)
	setWechatPolicyEnabled(t, true)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/wechat/login/", nil)
	rec := httptest.NewRecorder()
	handleWeChatLogin(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("policy on: status = %d, want 302 (body: %s)", rec.Code, rec.Body.String())
	}
	if loc := rec.Header().Get("Location"); !strings.Contains(loc, "open.weixin.qq.com/connect/qrconnect") {
		t.Fatalf("policy on: location should point to wechat qrconnect, got %s", loc)
	}
}

// TestWeChatCallbackLoginRejectedWhenPolicyDisabled — 纵深防御：用户在扫码途中
// 开关被关闭，回调也必须拒绝发 token（重定向到登录页并带错误提示）。
func TestWeChatCallbackLoginRejectedWhenPolicyDisabled(t *testing.T) {
	setupAuthTestDB(t)
	wechatCallbackTestCfg(t)
	setTestWechatApp(t)
	setWechatPolicyEnabled(t, false)
	mockWechatHTTP(t,
		`{"access_token":"at-pd-1","expires_in":7200,"openid":"o-pd-1","unionid":"u-pd-1"}`,
		`{"openid":"o-pd-1","unionid":"u-pd-1","nickname":"wx-user"}`,
	)

	state, err := generateWeChatState("web", "", "/projects/")
	if err != nil {
		t.Fatalf("generate state: %v", err)
	}
	loc := runWeChatCallback(t, state)
	if strings.Contains(loc, "wechat_token=") {
		t.Fatalf("policy off: callback must not issue token, got %s", loc)
	}
	if !strings.Contains(loc, "wechat_error=") {
		t.Fatalf("policy off: callback should redirect with wechat_error, got %s", loc)
	}
	// 错误信息经 URL 编码，解码后应包含「未开启」
	msg, err := url.QueryUnescape(loc[strings.Index(loc, "wechat_error=")+len("wechat_error="):])
	if err != nil || !strings.Contains(msg, "未开启") {
		t.Fatalf("policy off: wechat_error should mention 未开启, got %s", loc)
	}
}

// TestWeChatCallbackBindFlowUnchangedWhenPolicyDisabled — 绑定流程（登录态用户
// 账号管理）不受登录开关影响，仍正常完成绑定回跳。
func TestWeChatCallbackBindFlowUnchangedWhenPolicyDisabled(t *testing.T) {
	setupAuthTestDB(t)
	wechatCallbackTestCfg(t)
	setTestWechatApp(t)
	setWechatPolicyEnabled(t, false)
	mockWechatHTTP(t,
		`{"access_token":"at-pb-1","expires_in":7200,"openid":"o-pb-1","unionid":"u-pb-1"}`,
		`{"openid":"o-pb-1","unionid":"u-pb-1","nickname":"wx-user"}`,
	)
	bindUserID := mustCreateWechatTestUser(t, "u-bind-policy-off")

	state, err := generateWeChatState("web", bindUserID, "/projects/")
	if err != nil {
		t.Fatalf("generate state: %v", err)
	}
	loc := runWeChatCallback(t, state)
	if !strings.Contains(loc, "wechat_bound=1") {
		t.Fatalf("bind flow must stay functional when login policy is off: %s", loc)
	}
}

// TestPublicFeaturePolicyWechatAvailability — 公共策略端点按「策略开启 && 应用已配置」
// 计算 wechat_login_available（前端据此展示/隐藏扫码入口）。
func TestPublicFeaturePolicyWechatAvailability(t *testing.T) {
	setupAuthTestDB(t)
	setTestWechatApp(t)
	setWechatPolicyEnabled(t, true)

	req := httptest.NewRequest(http.MethodGet, "/api/public/system-feature-policy/", nil)
	rec := httptest.NewRecorder()
	handlePublicSystemFeaturePolicy(rec, req)
	var payload map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload["wechat_login_available"] != true {
		t.Fatalf("wechat_login_available = %v, want true", payload["wechat_login_available"])
	}
}

// TestPublicFeaturePolicyWechatUnavailableWhenPolicyOff — 策略关闭 → 不可用。
func TestPublicFeaturePolicyWechatUnavailableWhenPolicyOff(t *testing.T) {
	setupAuthTestDB(t)
	setTestWechatApp(t)
	setWechatPolicyEnabled(t, false)

	req := httptest.NewRequest(http.MethodGet, "/api/public/system-feature-policy/", nil)
	rec := httptest.NewRecorder()
	handlePublicSystemFeaturePolicy(rec, req)
	var payload map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload["wechat_login_available"] != false {
		t.Fatalf("wechat_login_available = %v, want false", payload["wechat_login_available"])
	}
}

// TestPublicFeaturePolicyWechatUnavailableWhenNotConfigured — 策略开启但应用未配置 → 不可用。
func TestPublicFeaturePolicyWechatUnavailableWhenNotConfigured(t *testing.T) {
	setupAuthTestDB(t)
	setWechatPolicyEnabled(t, true)
	old := wechatApps
	wechatApps = nil
	t.Cleanup(func() { wechatApps = old })

	req := httptest.NewRequest(http.MethodGet, "/api/public/system-feature-policy/", nil)
	rec := httptest.NewRecorder()
	handlePublicSystemFeaturePolicy(rec, req)
	var payload map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload["wechat_login_available"] != false {
		t.Fatalf("wechat_login_available = %v, want false (not configured)", payload["wechat_login_available"])
	}
}

// ── OPT-20260806-029: 管理后台 wechat_app_configured 字段 ────────────────
// 开关开启但应用未配置时，admin 端点必须暴露配置状态供管理页提示
// （公共端点 wechat_login_available 无法区分「开关未开」与「应用未配置」）。

func TestAdminFeaturePolicyWechatAppConfigured(t *testing.T) {
	setupAuthTestDB(t)
	setWechatPolicyEnabled(t, true)
	setTestWechatApp(t)

	req := httptest.NewRequest(http.MethodGet, "/api/system-admin/system-feature-policy/", nil)
	rec := httptest.NewRecorder()
	handleAdminGetFeaturePolicy(rec, req)
	var payload map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload["wechat_app_configured"] != true {
		t.Fatalf("wechat_app_configured = %v, want true (app configured)", payload["wechat_app_configured"])
	}
}

func TestAdminFeaturePolicyWechatAppNotConfigured(t *testing.T) {
	setupAuthTestDB(t)
	setWechatPolicyEnabled(t, true)
	old := wechatApps
	wechatApps = nil
	t.Cleanup(func() { wechatApps = old })

	req := httptest.NewRequest(http.MethodGet, "/api/system-admin/system-feature-policy/", nil)
	rec := httptest.NewRecorder()
	handleAdminGetFeaturePolicy(rec, req)
	var payload map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload["wechat_app_configured"] != false {
		t.Fatalf("wechat_app_configured = %v, want false (app not configured)", payload["wechat_app_configured"])
	}
}
