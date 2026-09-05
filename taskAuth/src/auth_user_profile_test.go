package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestBuildUserProfileJSON_WechatBindAvailable 验证 profile 响应中的
// wechat_bind_available 字段：web 应用凭据（appId/appSecret/redirectUri）
// 齐全时为 true；缺 appSecret 或无任何应用时为 false。
// 前端 UserProfileWechatBindingPanel 据此隐藏「绑定其他微信应用」入口
// （绑定流程固定走 ?app=web，凭据缺失时服务端必然拒绝）。
func TestBuildUserProfileJSON_WechatBindAvailable(t *testing.T) {
	setupAuthTestDB(t)
	userID := mustCreateWechatTestUser(t, "o-profile-bindavail-001")
	upsertWechatIdentity(userID, "web", "wx-test", "o-profile-bindavail-001", "u-profile-bindavail-001", "tester", "", timeNowUTC())

	// 用例 1: web 应用凭据齐全 → true
	setTestWechatApp(t)
	payload, err := buildUserProfileJSON(userID)
	if err != nil {
		t.Fatalf("buildUserProfileJSON (configured): %v", err)
	}
	if got, _ := payload["wechat_bind_available"].(bool); !got {
		t.Errorf("fully-configured web app: wechat_bind_available = %v, want true", payload["wechat_bind_available"])
	}

	// 用例 2: appSecret 未设置 → false
	old := wechatApps
	wechatApps = map[string]*WeChatAppConfig{
		"web": {Key: "web", AppID: "wx-test", AppSecret: "", RedirectURI: "https://example.com/api/auth/wechat/callback/", Type: "qr"},
	}
	t.Cleanup(func() { wechatApps = old })

	payload, err = buildUserProfileJSON(userID)
	if err != nil {
		t.Fatalf("buildUserProfileJSON (no secret): %v", err)
	}
	if got, _ := payload["wechat_bind_available"].(bool); got {
		t.Errorf("missing appSecret: wechat_bind_available = true, want false")
	}

	// 用例 3: 无任何微信应用 → false
	wechatApps = map[string]*WeChatAppConfig{}
	payload, err = buildUserProfileJSON(userID)
	if err != nil {
		t.Fatalf("buildUserProfileJSON (no apps): %v", err)
	}
	if got, _ := payload["wechat_bind_available"].(bool); got {
		t.Errorf("no apps: wechat_bind_available = true, want false")
	}

	// 回归: has_wechat / wechat_apps 不受影响（用户已绑定 web 应用）
	if hw, _ := payload["has_wechat"].(bool); !hw {
		t.Errorf("has_wechat = %v, want true", payload["has_wechat"])
	}
}

// TestGetUserProfile401ClearsResidualCookies（OPT-20260807-006 回归）：
// 未认证 GET profile 必须返回 401，且响应**首段**必须带 userId/token 两个
// Set-Cookie 清理头（Max-Age=-1）。回归目标：清理调用若发生在 writeJSON 401
// 之后（header 已 flush）会成为 no-op，浏览器残留的 30 天 HttpOnly cookie
// 将无法在登录页加载时被清除。
func TestGetUserProfile401ClearsResidualCookies(t *testing.T) {
	// 配置真实域使 ssoCookieDomain() 返回父域，双变体（域 + host-only）才可区分
	oldBase := cfg.GatewayPublicBase
	cfg.GatewayPublicBase = "https://www.daydaymoney.com"
	defer func() { cfg.GatewayPublicBase = oldBase }()

	req := httptest.NewRequest(http.MethodGet, "/api/accounts/users/profile/", nil)
	rec := httptest.NewRecorder()
	handleGetUserProfile(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	cookies := rec.Result().Cookies()
	var gotUserID, gotToken bool
	for _, c := range cookies {
		if c.Name == "userId" && c.MaxAge < 0 {
			gotUserID = true
		}
		if c.Name == "token" && c.MaxAge < 0 {
			gotToken = true
		}
	}
	if !gotUserID || !gotToken {
		got := []string{}
		for _, c := range cookies {
			got = append(got, c.Name+"="+c.String())
		}
		t.Fatalf("401 must carry userId/token clearing cookies, got %v (raw headers: %v)",
			strings.Join(got, ", "), rec.Header()["Set-Cookie"])
	}
	// 双变体断言（OPT-20260808）：每个名字必须同时带「域变体 + host-only 变体」清除，
	// 只清 host-only 时 071 后的父域 cookie 残留 → 裸域/其他子域仍登录。
	sc := rec.Header().Values("Set-Cookie")
	for _, name := range []string{"userId", "token"} {
		var domainVariant, hostOnlyVariant bool
		for _, c := range sc {
			if !strings.Contains(c, name+"=") || !strings.Contains(c, "Max-Age=0") {
				continue
			}
			if strings.Contains(c, "Domain=") {
				domainVariant = true
			} else {
				hostOnlyVariant = true
			}
		}
		if !domainVariant || !hostOnlyVariant {
			t.Fatalf("401 must clear BOTH domain+host-only variants for %s, got %v", name, sc)
		}
	}
}

// TestFetchTenantMembersOnce_PassesIsActive（OPT-20260827-021）：
// 租户成员的 is_active 必须从 taskTenantService 内部 API 一路透传到
// companies（/me 与 profile），插件 uniqueCompanies 据此跳过已退出租户。
func TestFetchTenantMembersOnce_PassesIsActive(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/internal/tenant/members", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[
			{"company_id":"c-1","company_name":"启元","member_name":"甲","member_avatar_url":"","is_admin":true,"is_creator":true,"is_active":true,"workspace_id":"ws-1"},
			{"company_id":"c-2","company_name":"云启","member_name":"乙","member_avatar_url":"","is_admin":false,"is_creator":false,"is_active":false,"workspace_id":"ws-2"}
		]`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	result, err := fetchTenantMembersOnce(srv.URL, "u-test")
	if err != nil {
		t.Fatalf("fetchTenantMembersOnce: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("len(result) = %d, want 2", len(result))
	}
	m0 := result[0].(map[string]interface{})
	if active, _ := m0["is_active"].(bool); !active {
		t.Errorf("member 0 is_active = %v, want true", m0["is_active"])
	}
	m1 := result[1].(map[string]interface{})
	if active, _ := m1["is_active"].(bool); active {
		t.Errorf("member 1 is_active = %v, want false", m1["is_active"])
	}

	companies := buildCompaniesFromNicknames(result)
	if len(companies) != 2 {
		t.Fatalf("len(companies) = %d, want 2", len(companies))
	}
	if c0, _ := companies[0]["is_active"].(bool); !c0 {
		t.Errorf("companies[0] is_active = %v, want true", companies[0]["is_active"])
	}
	if c1, _ := companies[1]["is_active"].(bool); c1 {
		t.Errorf("companies[1] is_active = %v, want false", companies[1]["is_active"])
	}
}
