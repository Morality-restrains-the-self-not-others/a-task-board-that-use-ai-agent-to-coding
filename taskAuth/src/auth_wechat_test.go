package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// wechat identity matching (v64): unionid-first, (app_key, openid) fallback,
// bind transfer, conflict detection, bind/unbind.

func mustCreateWechatTestUser(t *testing.T, identifier string) string {
	t.Helper()
	// 空昵称：测试夹具不预写 profile，避免挡住「补齐昵称」用例。
	userID, err := createWechatUser(identifier, "", "", timeNowUTC())
	if err != nil {
		t.Fatalf("createWechatUser: %v", err)
	}
	return userID
}

func TestFindOrCreateWeChatUser_UnionIDFirstHit(t *testing.T) {
	setupAuthTestDB(t)
	unionID := "u-test-union-001"
	u1 := mustCreateWechatTestUser(t, unionID)
	upsertWechatIdentity(u1, "web", "wx123", "o-openid-web-001", unionID, "u1", "", timeNowUTC())

	// 同一用户经 inapp 应用登录（openid 不同，unionid 相同）→ 命中同一用户
	got, created, err := findOrCreateWeChatUser("inapp", "wx456", unionID, "o-openid-inapp-001", "u1b", "")
	if err != nil {
		t.Fatalf("findOrCreate: %v", err)
	}
	if created {
		t.Fatalf("unionid hit should not create a new user")
	}
	if got != u1 {
		t.Fatalf("unionid hit: got user %s want %s", got, u1)
	}
	// 别名已绑定到 inapp
	owner, ok := findWechatUserByAppOpenID("inapp", "o-openid-inapp-001")
	if !ok || owner != u1 {
		t.Fatalf("inapp alias not bound to %s: got %s ok=%v", u1, owner, ok)
	}
}

func TestFindOrCreateWeChatUser_SameAppReloginNoUnionID(t *testing.T) {
	setupAuthTestDB(t)
	u1 := mustCreateWechatTestUser(t, "o-openid-web-002")
	upsertWechatIdentity(u1, "web", "wx123", "o-openid-web-002", "", "u1", "", timeNowUTC())

	// unionid 缺失（首次授权场景），同应用重登 → 不分裂
	got, created, err := findOrCreateWeChatUser("web", "wx123", "", "o-openid-web-002", "u1b", "")
	if err != nil {
		t.Fatalf("findOrCreate: %v", err)
	}
	if created {
		t.Fatalf("same-app relogin should not create a new user")
	}
	if got != u1 {
		t.Fatalf("same-app relogin: got user %s want %s", got, u1)
	}
}

func TestFindOrCreateWeChatUser_NewAppCreatesNewUser(t *testing.T) {
	setupAuthTestDB(t)
	u1 := mustCreateWechatTestUser(t, "o-openid-web-003")
	upsertWechatIdentity(u1, "web", "wx123", "o-openid-web-003", "", "u1", "", timeNowUTC())

	// 另一应用 + 新 openid + 无 unionid → 新建用户（预期行为：unionid 缺失时按 app 隔离）
	got, created, err := findOrCreateWeChatUser("inapp", "wx456", "", "o-openid-inapp-003", "u2", "")
	if err != nil {
		t.Fatalf("findOrCreate: %v", err)
	}
	if !created {
		t.Fatalf("expected created=true for brand-new user")
	}
	if got == u1 {
		t.Fatalf("expected new user, got existing %s", got)
	}
	owner, _ := findWechatUserByAppOpenID("inapp", "o-openid-inapp-003")
	if owner != got {
		t.Fatalf("inapp alias owner mismatch: %s vs %s", owner, got)
	}
}

func TestFindOrCreateWeChatUser_BindTransfer(t *testing.T) {
	setupAuthTestDB(t)
	// 分裂场景: U2 以 openid（无 unionid）建号；之后 unionid 可得
	u2 := mustCreateWechatTestUser(t, "o-openid-web-004")
	upsertWechatIdentity(u2, "web", "wx123", "o-openid-web-004", "", "u2", "", timeNowUTC())

	got, created, err := findOrCreateWeChatUser("web", "wx123", "u-test-union-004", "o-openid-web-004", "u2b", "")
	if err != nil {
		t.Fatalf("findOrCreate: %v", err)
	}
	// 绑定转移 → 新 unionid 真源账号持有该 openid 别名
	if !created {
		t.Fatalf("bind transfer should create a new unionid user")
	}
	if got == u2 {
		t.Fatalf("bind transfer expected new unionid user, got legacy %s", got)
	}
	owner, ok := findWechatUserByAppOpenID("web", "o-openid-web-004")
	if !ok || owner != got {
		t.Fatalf("alias not transferred to %s: got %s ok=%v", got, owner, ok)
	}
	if u := wechatUnionIDOf(got); u != "u-test-union-004" {
		t.Fatalf("unionid not recorded on new user: %q", u)
	}
}

func TestFindOrCreateWeChatUser_ConflictRejected(t *testing.T) {
	setupAuthTestDB(t)
	// U1 已有 unionid U-A；U2 的 openid 在同一 app（异常: openid 双射不一致）
	u1 := mustCreateWechatTestUser(t, "u-test-union-005a")
	upsertWechatIdentity(u1, "web", "wx123", "o-openid-web-005", "u-test-union-005a", "u1", "", timeNowUTC())
	u2 := mustCreateWechatTestUser(t, "o-openid-web-005b")
	upsertWechatIdentity(u2, "web", "wx123", "o-openid-web-005b", "", "u2", "", timeNowUTC())

	// U1 登录携带 unionid 但 openid 属于 U2 的 app 域（人为构造冲突）
	// 直接构造: (web, o-openid-web-005) 归属 U1 且 U1 已有 unionid，
	// 新登录 unionid=U2 的 unionid... 简化: 让 openid 归属的用户已有不同 unionid
	upsertWechatIdentity(u1, "web", "wx123", "o-openid-web-005b", "u-test-union-005a", "u1", "", timeNowUTC())
	_, _, err := findOrCreateWeChatUser("web", "wx123", "u-test-union-005b", "o-openid-web-005b", "u2", "")
	if !errors.Is(err, errWechatIdentityConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestBindWeChatIdentity_ConflictAndIdempotent(t *testing.T) {
	setupAuthTestDB(t)
	owner := mustCreateWechatTestUser(t, "u-test-union-006a")
	upsertWechatIdentity(owner, "web", "wx123", "o-openid-web-006", "u-test-union-006a", "owner", "", timeNowUTC())
	me := mustCreateWechatTestUser(t, "u-test-union-006b")

	// 绑定已被占用的微信 → 冲突
	if _, err := bindWeChatIdentity(me, "web", "wx123", "u-test-union-006a", "o-openid-web-006", "x", ""); !errors.Is(err, errWechatIdentityConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
	// 绑定全新微信 → 成功
	got, err := bindWeChatIdentity(me, "web", "wx123", "u-test-union-006c", "o-openid-web-006c", "me", "")
	if err != nil || got != me {
		t.Fatalf("bind: got %s err %v", got, err)
	}
	// 重复绑定（幂等）
	got2, err := bindWeChatIdentity(me, "web", "wx123", "u-test-union-006c", "o-openid-web-006c", "me", "")
	if err != nil || got2 != me {
		t.Fatalf("rebind: got %s err %v", got2, err)
	}
}

func TestUnbindWeChatIdentity_RemainingMethodCheck(t *testing.T) {
	setupAuthTestDB(t)
	// 仅微信一种登录方式的账号 → 解绑被拒
	onlyWechat := mustCreateWechatTestUser(t, "u-test-union-007")
	upsertWechatIdentity(onlyWechat, "web", "wx123", "o-openid-web-007", "u-test-union-007", "", "", timeNowUTC())
	if err := unbindWeChatIdentity(onlyWechat, "web"); err == nil || !strings.Contains(err.Error(), "无可用登录方式") {
		t.Fatalf("expected remaining-method rejection, got %v", err)
	}

	// 有 email + wechat → 解绑成功且 wechat 别名被移除
	both := mustCreateWechatTestUser(t, "u-test-union-008")
	upsertWechatIdentity(both, "web", "wx123", "o-openid-web-008", "u-test-union-008", "", "", timeNowUTC())
	if _, err := db.Exec(`
		INSERT INTO auth_login_method (id, content_type_id, object_id, method_type, identifier, password_hash, is_verified, created_at, updated_at)
		VALUES (?, ?, ?, 'email', 't@example.com', '', 1, NOW(), NOW())`,
		generateSnowflakeID(), cfg.UserContentTypeID, both); err != nil {
		t.Fatalf("seed email lm: %v", err)
	}
	if err := unbindWeChatIdentity(both, "web"); err != nil {
		t.Fatalf("unbind with email: %v", err)
	}
	if _, ok := findWechatUserByAppOpenID("web", "o-openid-web-008"); ok {
		t.Fatalf("alias should be removed after unbind")
	}
}

func TestWeChatStateAppKeyEncoding(t *testing.T) {
	state, err := generateWeChatState("inapp", "user-42", "/projects/")
	if err != nil {
		t.Fatalf("generate state: %v", err)
	}
	if got := parseAppKeyFromState(state); got != "inapp" {
		t.Fatalf("app key decode: got %q want inapp", got)
	}
	entry, ok := consumeWeChatState(state)
	if !ok || entry.AppKey != "inapp" || entry.BindUserID != "user-42" || entry.Next != "/projects/" {
		t.Fatalf("state consume: %+v ok=%v", entry, ok)
	}
	if _, ok := consumeWeChatState(state); ok {
		t.Fatalf("state should be single-use")
	}
}

// TestWeChatStateEmbedsTraceID 验证 traceId 埋入 state 末段（扫码链路可关联）：
//  1. state 形态 = base64url(app_key).random.traceId（三段）
//  2. parseTraceIDFromState 能从 state 中还原 traceId（含无效/过期回调场景）
//  3. 既有 app_key 前缀解析不受影响（向后兼容）
func TestWeChatStateEmbedsTraceID(t *testing.T) {
	state, err := generateWeChatState("web", "", "")
	if err != nil {
		t.Fatalf("generate state: %v", err)
	}
	// 三段式：app_key 段 + random 段 + traceId 段
	if got := strings.Count(state, "."); got != 2 {
		t.Fatalf("state should be 3 segments (app_key.random.traceId), got %d dots in %q", got, state)
	}
	traceID := parseTraceIDFromState(state)
	if traceID == "" {
		t.Fatalf("traceId segment missing in state %q", state)
	}
	if !strings.HasSuffix(state, "."+traceID) {
		t.Fatalf("traceId should be the trailing segment: state=%q traceId=%q", state, traceID)
	}
	// 消费时 entry 携带同一 traceId
	entry, ok := consumeWeChatState(state)
	if !ok || entry.TraceID != traceID {
		t.Fatalf("consumed entry should carry traceId: entry=%+v ok=%v want=%q", entry, ok, traceID)
	}
	// app_key 解析不受 traceId 段影响（向后兼容）
	if got := parseAppKeyFromState(state); got != "web" {
		t.Fatalf("app key decode with traceId: got %q want web", got)
	}
}

// TestParseTraceIDFromState 覆盖 traceId 提取边界：空 state、无 traceId 的旧形态 state。
func TestParseTraceIDFromState(t *testing.T) {
	if got := parseTraceIDFromState(""); got != "" {
		t.Fatalf("empty state: got %q want empty", got)
	}
	// 旧形态（无 traceId 段）：返回空，不误伤
	legacy := base64.RawURLEncoding.EncodeToString([]byte("web")) + ".legacyrandom"
	if got := parseTraceIDFromState(legacy); got != "" {
		t.Fatalf("legacy state: got %q want empty", got)
	}
	// 含空白注入的 state：拒绝
	if got := parseTraceIDFromState("web.rand.bad trace"); got != "" {
		t.Fatalf("state with whitespace: got %q want empty", got)
	}
}

// TestSweepExpiredWeChatStates 验证 GC 清扫语义：仅删除超过 TTL 的条目，
// 未过期条目保留；清扫与消费共用锁，不破坏并发安全。
func TestSweepExpiredWeChatStates(t *testing.T) {
	// 直接构造新旧两条目（绕过 generate 的时间戳，精确控制年龄）
	fresh := "fresh.state.00000000-0000-4000-8000-000000000001"
	stale := "stale.state.00000000-0000-4000-8000-000000000002"
	now := time.Now().UTC()

	wechatStateMu.Lock()
	wechatStateStore[fresh] = wechatStateEntry{State: fresh, CreatedAt: now.Add(-1 * time.Minute)}
	wechatStateStore[stale] = wechatStateEntry{State: stale, CreatedAt: now.Add(-wechatStateTTL - 1*time.Minute)}
	wechatStateMu.Unlock()

	wechatStateMu.Lock()
	got := sweepExpiredWeChatStates(now)
	wechatStateMu.Unlock()
	if got != 1 {
		t.Fatalf("sweep: got %d expired, want 1", got)
	}

	wechatStateMu.RLock()
	_, freshOK := wechatStateStore[fresh]
	_, staleOK := wechatStateStore[stale]
	wechatStateMu.RUnlock()
	if !freshOK {
		t.Fatalf("fresh entry should survive sweep")
	}
	if staleOK {
		t.Fatalf("stale entry should be swept")
	}

	// 已消费条目仍正常：GC 不干扰消费路径
	if entry, ok := consumeWeChatState(fresh); !ok || entry.State != fresh {
		t.Fatalf("consume after sweep: ok=%v entry=%+v", ok, entry)
	}
}

func TestSanitizeNextPath(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		{"/projects/", "/projects/"},
		{"/tenant/123/work-panel/", "/tenant/123/work-panel/"},
		{"  /a/b  ", "/a/b"},
		{"", ""},
		{"//evil.com", ""},
		{"https://evil.com/x", ""},
		{"javascript:alert(1)", ""},
		{"http://evil.com", ""},
		{"/path\ninjected", ""},
		{"/path\x00injected", ""},
		{"/" + strings.Repeat("a", 3000), ""},
	}
	for _, c := range cases {
		if got := sanitizeNextPath(c.raw); got != c.want {
			t.Fatalf("sanitizeNextPath(%q) = %q, want %q", c.raw, got, c.want)
		}
	}
}

// --- 回调重定向落点回归（「微信扫码登录后跳转到首页」修复）---

// mockWechatHTTP 替换 wechatHTTPGet，按 URL 特征返回微信 API 的 mock 响应。
func mockWechatHTTP(t *testing.T, tokenJSON, userInfoJSON string) {
	t.Helper()
	old := wechatHTTPGet
	wechatHTTPGet = func(urlStr string) ([]byte, error) {
		// userinfo URL 同时包含 access_token 查询参数，必须先匹配 userinfo，
		// 否则昵称等用户信息响应会被 token mock 抢占（昵称断言类测试必挂）。
		switch {
		case strings.Contains(urlStr, "/sns/userinfo"):
			return []byte(userInfoJSON), nil
		case strings.Contains(urlStr, "/sns/oauth2/access_token"):
			return []byte(tokenJSON), nil
		}
		return nil, fmt.Errorf("unexpected wechat url: %s", urlStr)
	}
	t.Cleanup(func() { wechatHTTPGet = old })
}

// setTestWechatApp 配置 wechatApps（web 扫码应用），测试结束恢复。
func setTestWechatApp(t *testing.T) {
	t.Helper()
	old := wechatApps
	wechatApps = map[string]*WeChatAppConfig{
		"web": {Key: "web", AppID: "wx-test", AppSecret: "sec-test", RedirectURI: "https://example.com/api/auth/wechat/callback/", Type: "qr"},
	}
	t.Cleanup(func() { wechatApps = old })
}

// wechatCallbackTestCfg 保留 UserContentTypeID（setupAuthTestDB 加载），仅覆盖前端地址。
func wechatCallbackTestCfg(t *testing.T) {
	t.Helper()
	prev := cfg
	cfg = Config{FrontendBase: "https://www.daydaymoney.com", UserContentTypeID: prev.UserContentTypeID}
	t.Cleanup(func() { cfg = prev })
}

// runWeChatCallback 走完整回调：mock 微信 API + 携带 state 调用 handler，返回 Location。
func runWeChatCallback(t *testing.T, state string) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/auth/wechat/web/callback/?code=test-code&state="+state, nil)
	rec := httptest.NewRecorder()
	handleWeChatCallback(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("callback status = %d, want 302 (body: %s)", rec.Code, rec.Body.String())
	}
	return rec.Header().Get("Location")
}

// TestWeChatCallbackRedirectComputedDestination — 无用户 next 时，回调按角色计算登录落点，
// 禁止兜底到公开首页 "/"（回归：此前仅回 wechat_token，前端 next||'/' 落到首页）。
func TestWeChatCallbackRedirectComputedDestination(t *testing.T) {
	setupAuthTestDB(t)
	wechatCallbackTestCfg(t)
	setWechatPolicyEnabled(t, true)
	setTestWechatApp(t)
	mockWechatHTTP(t,
		`{"access_token":"at-cb-1","expires_in":7200,"openid":"o-cb-1","unionid":"u-cb-1"}`,
		`{"openid":"o-cb-1","unionid":"u-cb-1","nickname":"wx-user"}`,
	)
	// 已存在用户（unionid 命中）→ 登录其账号
	userID := mustCreateWechatTestUser(t, "u-cb-1")
	upsertWechatIdentity(userID, "web", "wx-test", "o-cb-1", "u-cb-1", "wx-user", "", timeNowUTC())

	state, err := generateWeChatState("web", "", "")
	if err != nil {
		t.Fatalf("generate state: %v", err)
	}
	loc := runWeChatCallback(t, state)
	// 新用户（无公司、非平台角色）→ /onboarding/；不得出现 next 缺失或落到 "/"
	if !strings.Contains(loc, "wechat_token=") {
		t.Fatalf("callback redirect missing wechat_token: %s", loc)
	}
	if !strings.Contains(loc, "next=%2Fonboarding%2F") {
		t.Fatalf("callback redirect missing computed next=/onboarding/: %s", loc)
	}
	if strings.HasSuffix(strings.Split(loc, "?")[1], "next=/") {
		t.Fatalf("callback redirect must never land on public homepage: %s", loc)
	}
}

// TestWeChatCallbackRedirectPreservesUserNext — 用户发起登录时携带 next，
// 回调后原样回跳该业务路径（而非计算落点/首页）。
func TestWeChatCallbackRedirectPreservesUserNext(t *testing.T) {
	setupAuthTestDB(t)
	wechatCallbackTestCfg(t)
	setWechatPolicyEnabled(t, true)
	setTestWechatApp(t)
	mockWechatHTTP(t,
		`{"access_token":"at-cb-2","expires_in":7200,"openid":"o-cb-2","unionid":"u-cb-2"}`,
		`{"openid":"o-cb-2","unionid":"u-cb-2","nickname":"wx-user"}`,
	)
	userID := mustCreateWechatTestUser(t, "u-cb-2")
	upsertWechatIdentity(userID, "web", "wx-test", "o-cb-2", "u-cb-2", "wx-user", "", timeNowUTC())

	state, err := generateWeChatState("web", "", "/tenant/88/work-panel/")
	if err != nil {
		t.Fatalf("generate state: %v", err)
	}
	loc := runWeChatCallback(t, state)
	if !strings.Contains(loc, "next=%2Ftenant%2F88%2Fwork-panel%2F") {
		t.Fatalf("callback redirect lost user next: %s", loc)
	}
}

// mockTenantMembersServer 启动 mock taskTenantService：GET /api/internal/tenant/members
// 按 user_id 返回成员 JSON（无匹配 → 空数组），模拟 onboarding 已建公司的数据层。
func mockTenantMembersServer(t *testing.T, membersByUser map[string]string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/internal/tenant/members", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		uid := r.URL.Query().Get("user_id")
		if body, ok := membersByUser[uid]; ok {
			fmt.Fprint(w, body)
			return
		}
		fmt.Fprint(w, `[]`)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// TestWeChatCallbackRedirectOnboardingNextWithCompany — 用户已设置公司名后，
// 二次扫码登录携带 next=/onboarding/（陈旧链接/书签/Onboarding 页 401 循环产物）
// 时，回调弃用该 next 改按角色计算落点：有公司 → /tenant/{id}/work-panel/。
// 回归：数据层有公司但落点仍为 onboarding 的「第二次扫码登录依旧跳到公司名称设置页」。
func TestWeChatCallbackRedirectOnboardingNextWithCompany(t *testing.T) {
	setupAuthTestDB(t)
	setWechatPolicyEnabled(t, true)
	setTestWechatApp(t)
	mockWechatHTTP(t,
		`{"access_token":"at-cb-5","expires_in":7200,"openid":"o-cb-5","unionid":"u-cb-5"}`,
		`{"openid":"o-cb-5","unionid":"u-cb-5","nickname":"wx-user"}`,
	)
	userID := mustCreateWechatTestUser(t, "u-cb-5")
	upsertWechatIdentity(userID, "web", "wx-test", "o-cb-5", "u-cb-5", "wx-user", "", timeNowUTC())

	// 已有公司（onboarding 首次设置完成后的数据层）
	tenantSrv := mockTenantMembersServer(t, map[string]string{
		userID: `[{"id":"m5","user_id":"` + userID + `","company_id":"c-5","is_admin":true,"is_active":true,"workspace_id":"","member_name":"软刀","company_name":"软刀"}]`,
	})
	prevCfg := cfg
	cfg = Config{FrontendBase: "https://www.daydaymoney.com", UserContentTypeID: prevCfg.UserContentTypeID, TenantServiceURL: tenantSrv.URL}
	t.Cleanup(func() { cfg = prevCfg })

	state, err := generateWeChatState("web", "", "/onboarding/")
	if err != nil {
		t.Fatalf("generate state: %v", err)
	}
	loc := runWeChatCallback(t, state)
	if !strings.Contains(loc, "next=%2Ftenant%2Fc-5%2Fwork-panel%2F") {
		t.Fatalf("company user with next=/onboarding/ must land on work-panel, got: %s", loc)
	}
	if strings.Contains(loc, "next=%2Fonboarding%2F") {
		t.Fatalf("company user must never land on onboarding: %s", loc)
	}
}

// TestWeChatCallbackRedirectOnboardingNextNoCompany — 无公司新用户携带
// next=/onboarding/ 时，回调弃用该 next 后按角色计算仍为 /onboarding/
// （首次扫码登录建公司引导不受影响）。
func TestWeChatCallbackRedirectOnboardingNextNoCompany(t *testing.T) {
	setupAuthTestDB(t)
	setWechatPolicyEnabled(t, true)
	setTestWechatApp(t)
	mockWechatHTTP(t,
		`{"access_token":"at-cb-6","expires_in":7200,"openid":"o-cb-6","unionid":"u-cb-6"}`,
		`{"openid":"o-cb-6","unionid":"u-cb-6","nickname":"wx-user"}`,
	)
	userID := mustCreateWechatTestUser(t, "u-cb-6")
	upsertWechatIdentity(userID, "web", "wx-test", "o-cb-6", "u-cb-6", "wx-user", "", timeNowUTC())

	// 无公司：mock 服务返回空成员
	tenantSrv := mockTenantMembersServer(t, nil)
	prevCfg := cfg
	cfg = Config{FrontendBase: "https://www.daydaymoney.com", UserContentTypeID: prevCfg.UserContentTypeID, TenantServiceURL: tenantSrv.URL}
	t.Cleanup(func() { cfg = prevCfg })

	state, err := generateWeChatState("web", "", "/onboarding/")
	if err != nil {
		t.Fatalf("generate state: %v", err)
	}
	loc := runWeChatCallback(t, state)
	if !strings.Contains(loc, "next=%2Fonboarding%2F") {
		t.Fatalf("company-less user must still land on onboarding, got: %s", loc)
	}
}

// TestWeChatCallbackRedirectRejectsMaliciousNext — 恶意/非法 next 被净化丢弃，
// 回退到计算落点（防开放重定向；前端 PostLoginReturnUrl 规则对齐）。
func TestWeChatCallbackRedirectRejectsMaliciousNext(t *testing.T) {
	setupAuthTestDB(t)
	wechatCallbackTestCfg(t)
	setWechatPolicyEnabled(t, true)
	setTestWechatApp(t)
	mockWechatHTTP(t,
		`{"access_token":"at-cb-3","expires_in":7200,"openid":"o-cb-3","unionid":"u-cb-3"}`,
		`{"openid":"o-cb-3","unionid":"u-cb-3","nickname":"wx-user"}`,
	)
	userID := mustCreateWechatTestUser(t, "u-cb-3")
	upsertWechatIdentity(userID, "web", "wx-test", "o-cb-3", "u-cb-3", "wx-user", "", timeNowUTC())

	state, err := generateWeChatState("web", "", "https://evil.example.com/phish")
	if err != nil {
		t.Fatalf("generate state: %v", err)
	}
	loc := runWeChatCallback(t, state)
	if strings.Contains(loc, "evil.example.com") {
		t.Fatalf("callback redirect leaked malicious next: %s", loc)
	}
	if !strings.Contains(loc, "next=%2Fonboarding%2F") {
		t.Fatalf("callback redirect should fall back to computed next after sanitization: %s", loc)
	}
}

// TestWeChatCallbackBindFlowUnchanged — 绑定流程仍只回 wechat_bound，不带 next。
func TestWeChatCallbackBindFlowUnchanged(t *testing.T) {
	setupAuthTestDB(t)
	wechatCallbackTestCfg(t)
	setWechatPolicyEnabled(t, true)
	setTestWechatApp(t)
	mockWechatHTTP(t,
		`{"access_token":"at-cb-4","expires_in":7200,"openid":"o-cb-4","unionid":"u-cb-4"}`,
		`{"openid":"o-cb-4","unionid":"u-cb-4","nickname":"wx-user"}`,
	)
	bindUserID := mustCreateWechatTestUser(t, "u-bind-target")

	state, err := generateWeChatState("web", bindUserID, "/projects/")
	if err != nil {
		t.Fatalf("generate state: %v", err)
	}
	loc := runWeChatCallback(t, state)
	if !strings.Contains(loc, "wechat_bound=1") {
		t.Fatalf("bind flow should redirect with wechat_bound=1: %s", loc)
	}
	if strings.Contains(loc, "wechat_token=") || strings.Contains(loc, "next=") {
		t.Fatalf("bind flow must not carry token/next: %s", loc)
	}
}

// referralRecorder 捕获任务推荐内部接口 bind-from-code 的调用（OPT-20260820-036 回归）。
type referralRecorder struct {
	reqs chan map[string]string
}

func (r *referralRecorder) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/api/internal/referral/bind-from-code/" {
			http.NotFound(w, req)
			return
		}
		var body map[string]string
		_ = json.NewDecoder(req.Body).Decode(&body)
		select {
		case r.reqs <- body:
		default:
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
}

// waitForReferralCall 轮询等待异步 bind-from-code 调用（≤2s），未收到则测试失败。
func waitForReferralCall(t *testing.T, rec *referralRecorder, wantCode string) map[string]string {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case body := <-rec.reqs:
			return body
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
	t.Fatalf("referral bind-from-code not called within timeout (want code %s)", wantCode)
	return nil
}

// TestWeChatStateCarriesAccessCode — state 携带 accessCode（variadic 空则字段为空）。
func TestWeChatStateCarriesAccessCode(t *testing.T) {
	state, err := generateWeChatState("web", "", "", "DR2AKvP9J9")
	if err != nil {
		t.Fatalf("generate state: %v", err)
	}
	entry, ok := consumeWeChatState(state)
	if !ok {
		t.Fatalf("state consume failed")
	}
	if entry.AccessCode != "DR2AKvP9J9" {
		t.Fatalf("accessCode not carried through state: got %q", entry.AccessCode)
	}

	// 无 accessCode → 空字段（既有调用不受影响）
	state2, err := generateWeChatState("web", "", "")
	if err != nil {
		t.Fatalf("generate state: %v", err)
	}
	entry2, ok := consumeWeChatState(state2)
	if !ok {
		t.Fatalf("state consume failed")
	}
	if entry2.AccessCode != "" {
		t.Fatalf("accessCode should be empty when not provided: got %q", entry2.AccessCode)
	}
}

// TestWeChatCallbackBindsReferralForNewUser — 首次扫码自动注册 + state 带 accessCode
// → 回调后 POST bind-from-code（OPT-20260820-036）。
func TestWeChatCallbackBindsReferralForNewUser(t *testing.T) {
	setupAuthTestDB(t)
	wechatCallbackTestCfg(t)
	setWechatPolicyEnabled(t, true)
	setTestWechatApp(t)
	mockWechatHTTP(t,
		`{"access_token":"at-rf-1","expires_in":7200,"openid":"o-rf-1","unionid":"u-rf-1"}`,
		`{"openid":"o-rf-1","unionid":"u-rf-1","nickname":"wx-user"}`,
	)
	rec := &referralRecorder{reqs: make(chan map[string]string, 4)}
	srv := httptest.NewServer(rec.handler())
	defer srv.Close()
	prev := cfg.ReferralServiceURL
	cfg.ReferralServiceURL = srv.URL
	t.Cleanup(func() { cfg.ReferralServiceURL = prev })

	state, err := generateWeChatState("web", "", "", "DR2AKvP9J9")
	if err != nil {
		t.Fatalf("generate state: %v", err)
	}
	loc := runWeChatCallback(t, state)
	if !strings.Contains(loc, "wechat_token=") {
		t.Fatalf("callback should login new user: %s", loc)
	}
	body := waitForReferralCall(t, rec, "DR2AKvP9J9")
	if body["access_code"] != "DR2AKvP9J9" || body["referred_user_id"] == "" {
		t.Fatalf("referral bind body unexpected: %v", body)
	}
}

// TestWeChatCallbackSkipsReferralForExistingUser — 存量用户重登不绑定推荐。
func TestWeChatCallbackSkipsReferralForExistingUser(t *testing.T) {
	setupAuthTestDB(t)
	wechatCallbackTestCfg(t)
	setWechatPolicyEnabled(t, true)
	setTestWechatApp(t)
	mockWechatHTTP(t,
		`{"access_token":"at-rf-2","expires_in":7200,"openid":"o-rf-2","unionid":"u-rf-2"}`,
		`{"openid":"o-rf-2","unionid":"u-rf-2","nickname":"wx-user"}`,
	)
	userID := mustCreateWechatTestUser(t, "u-rf-2")
	upsertWechatIdentity(userID, "web", "wx-test", "o-rf-2", "u-rf-2", "wx-user", "", timeNowUTC())

	rec := &referralRecorder{reqs: make(chan map[string]string, 4)}
	srv := httptest.NewServer(rec.handler())
	defer srv.Close()
	prev := cfg.ReferralServiceURL
	cfg.ReferralServiceURL = srv.URL
	t.Cleanup(func() { cfg.ReferralServiceURL = prev })

	state, err := generateWeChatState("web", "", "", "DR2AKvP9J9")
	if err != nil {
		t.Fatalf("generate state: %v", err)
	}
	loc := runWeChatCallback(t, state)
	if !strings.Contains(loc, "wechat_token=") {
		t.Fatalf("callback should login existing user: %s", loc)
	}
	select {
	case body := <-rec.reqs:
		t.Fatalf("existing user must not bind referral, got %v", body)
	case <-time.After(300 * time.Millisecond):
	}
}

// 注：TestPhoneOTPLoginAutoRegisterBindsReferral 已随 2026-08-24 手机号+验证码
// 登录移除而删除（OTP 自动开通不再存在）。推荐绑定行为由
// TestPhoneRegisterNewUserBindsReferral（phone_register 显式注册）继续覆盖。
