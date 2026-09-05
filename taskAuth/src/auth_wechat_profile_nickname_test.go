package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ensureWechatProfileNickname 行为契约（微信扫码登录昵称修复）:
// 1. 个人昵称为空（或无 profile 行）→ 用微信昵称补齐
// 2. 个人昵称已有值 → 不覆盖（微信昵称仅作初始值）
// 3. 微信昵称为空 → 不动作，也不创建 profile 行

func profilePersonalNickname(t *testing.T, userID string) string {
	t.Helper()
	payload, err := buildUserProfileJSON(userID)
	if err != nil {
		t.Fatalf("buildUserProfileJSON: %v", err)
	}
	nickname, _ := payload["personal_nickname"].(string)
	return nickname
}

func TestEnsureWechatProfileNickname_SetsWhenProfileMissing(t *testing.T) {
	setupAuthTestDB(t)
	userID := mustCreateWechatTestUser(t, "o-nick-profile-missing-001")

	if err := ensureWechatProfileNickname(userID, "微信昵称A"); err != nil {
		t.Fatalf("ensureWechatProfileNickname: %v", err)
	}
	if got := profilePersonalNickname(t, userID); got != "微信昵称A" {
		t.Fatalf("personal_nickname = %q, want %q", got, "微信昵称A")
	}
}

func TestEnsureWechatProfileNickname_SetsWhenProfileRowEmpty(t *testing.T) {
	setupAuthTestDB(t)
	userID := mustCreateWechatTestUser(t, "o-nick-profile-empty-001")
	if err := upsertUserProfile(userID, ""); err != nil {
		t.Fatalf("upsertUserProfile(empty): %v", err)
	}

	if err := ensureWechatProfileNickname(userID, "微信昵称B"); err != nil {
		t.Fatalf("ensureWechatProfileNickname: %v", err)
	}
	if got := profilePersonalNickname(t, userID); got != "微信昵称B" {
		t.Fatalf("personal_nickname = %q, want %q", got, "微信昵称B")
	}
}

// 用户已自定义个人昵称 → 微信昵称不得覆盖（微信昵称仅作初始值）
func TestEnsureWechatProfileNickname_DoesNotOverwriteExisting(t *testing.T) {
	setupAuthTestDB(t)
	userID := mustCreateWechatTestUser(t, "o-nick-keep-001")
	if err := upsertUserProfile(userID, "用户自定义昵称"); err != nil {
		t.Fatalf("upsertUserProfile: %v", err)
	}

	if err := ensureWechatProfileNickname(userID, "微信昵称C"); err != nil {
		t.Fatalf("ensureWechatProfileNickname: %v", err)
	}
	if got := profilePersonalNickname(t, userID); got != "用户自定义昵称" {
		t.Fatalf("personal_nickname = %q, want %q (must not be overwritten)", got, "用户自定义昵称")
	}
}

func TestEnsureWechatProfileNickname_EmptyNicknameNoop(t *testing.T) {
	setupAuthTestDB(t)
	userID := mustCreateWechatTestUser(t, "o-nick-empty-001")

	if err := ensureWechatProfileNickname(userID, "  "); err != nil {
		t.Fatalf("ensureWechatProfileNickname: %v", err)
	}
	if got := profilePersonalNickname(t, userID); got != "" {
		t.Fatalf("personal_nickname = %q, want empty (no-op)", got)
	}
}

// --- 回调级端到端（修复核心行为）：扫码登录后个人昵称 = 微信昵称 ---

// TestWeChatCallbackLoginSetsPersonalNickname — 新用户首次扫码登录（回调建号）后，
// 个人昵称同步落库为微信昵称（此前依赖 USER_CREATED 事件链，存在部署/时序缺口）。
func TestWeChatCallbackLoginSetsPersonalNickname(t *testing.T) {
	setupAuthTestDB(t)
	wechatCallbackTestCfg(t)
	setWechatPolicyEnabled(t, true)
	setTestWechatApp(t)
	mockWechatHTTP(t,
		`{"access_token":"at-nick-1","expires_in":7200,"openid":"o-nick-1","unionid":"u-nick-1"}`,
		`{"openid":"o-nick-1","unionid":"u-nick-1","nickname":"微信昵称甲"}`,
	)

	state, err := generateWeChatState("web", "", "")
	if err != nil {
		t.Fatalf("generate state: %v", err)
	}
	loc := runWeChatCallback(t, state)
	if !strings.Contains(loc, "wechat_token=") {
		t.Fatalf("callback redirect missing wechat_token: %s", loc)
	}

	// 建号 → 个人昵称必须已同步为微信昵称（不依赖事件链）
	userID, ok := findWechatUserByUnionID("u-nick-1")
	if !ok {
		t.Fatalf("wechat user not created via callback")
	}
	if got := profilePersonalNickname(t, userID); got != "微信昵称甲" {
		t.Fatalf("personal_nickname = %q, want %q", got, "微信昵称甲")
	}
}

// TestWeChatCallbackLoginKeepsCustomNickname — 存量用户已自定义个人昵称，
// 再次扫码登录不得被微信昵称覆盖。
func TestWeChatCallbackLoginKeepsCustomNickname(t *testing.T) {
	setupAuthTestDB(t)
	wechatCallbackTestCfg(t)
	setWechatPolicyEnabled(t, true)
	setTestWechatApp(t)
	mockWechatHTTP(t,
		`{"access_token":"at-nick-2","expires_in":7200,"openid":"o-nick-2","unionid":"u-nick-2"}`,
		`{"openid":"o-nick-2","unionid":"u-nick-2","nickname":"微信昵称乙"}`,
	)
	userID := mustCreateWechatTestUser(t, "u-nick-2")
	upsertWechatIdentity(userID, "web", "wx-test", "o-nick-2", "u-nick-2", "微信昵称乙", "", timeNowUTC())
	if err := upsertUserProfile(userID, "用户自定义昵称"); err != nil {
		t.Fatalf("upsertUserProfile: %v", err)
	}

	state, err := generateWeChatState("web", "", "")
	if err != nil {
		t.Fatalf("generate state: %v", err)
	}
	runWeChatCallback(t, state)

	if got := profilePersonalNickname(t, userID); got != "用户自定义昵称" {
		t.Fatalf("personal_nickname = %q, want %q (custom nickname must be kept)", got, "用户自定义昵称")
	}
}

// TestWeChatCallbackBindSetsPersonalNickname — 绑定流程同样补齐个人昵称。
func TestWeChatCallbackBindSetsPersonalNickname(t *testing.T) {
	setupAuthTestDB(t)
	wechatCallbackTestCfg(t)
	setWechatPolicyEnabled(t, true)
	setTestWechatApp(t)
	mockWechatHTTP(t,
		`{"access_token":"at-nick-3","expires_in":7200,"openid":"o-nick-3","unionid":"u-nick-3"}`,
		`{"openid":"o-nick-3","unionid":"u-nick-3","nickname":"微信昵称丙"}`,
	)
	userID := mustCreateWechatTestUser(t, "o-nick-3-bind-owner")

	state, err := generateWeChatState("web", userID, "")
	if err != nil {
		t.Fatalf("generate state: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/auth/wechat/web/callback/?code=test-code&state="+state, nil)
	rec := httptest.NewRecorder()
	handleWeChatCallback(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("bind callback status = %d (body: %s)", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Header().Get("Location"), "wechat_bound=1") {
		t.Fatalf("bind callback should redirect with wechat_bound=1: %s", rec.Header().Get("Location"))
	}
	if got := profilePersonalNickname(t, userID); got != "微信昵称丙" {
		t.Fatalf("personal_nickname = %q, want %q after bind", got, "微信昵称丙")
	}
}

// OPT-20260812-038: 个人昵称写入后同步创建者 member_name（转发 sync-nickname）。
func TestEnsureWechatProfileNickname_SyncsCreatorMemberName(t *testing.T) {
	setupAuthTestDB(t)
	userID := mustCreateWechatTestUser(t, "o-nick-sync-001")

	var synced []map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/internal/tenant/members/sync-nickname" && r.Method == http.MethodPatch {
			body, _ := io.ReadAll(r.Body)
			var payload map[string]string
			_ = json.Unmarshal(body, &payload)
			synced = append(synced, payload)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": true, "updated": 1})
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)
	prevTenant := cfg.TenantServiceURL
	cfg.TenantServiceURL = srv.URL
	t.Cleanup(func() { cfg.TenantServiceURL = prevTenant })

	if err := ensureWechatProfileNickname(userID, "微信昵称同步"); err != nil {
		t.Fatalf("ensureWechatProfileNickname: %v", err)
	}
	if len(synced) != 1 {
		t.Fatalf("expected 1 sync call, got %d", len(synced))
	}
	if synced[0]["user_id"] != userID || synced[0]["nickname"] != "微信昵称同步" {
		t.Fatalf("unexpected sync payload: %+v", synced[0])
	}
}

// 已有个人昵称 → 不写入也不触发同步（微信昵称仅作初始值）。
func TestEnsureWechatProfileNickname_NoSyncWhenNicknameKept(t *testing.T) {
	setupAuthTestDB(t)
	userID := mustCreateWechatTestUser(t, "o-nick-nosync-001")
	if err := upsertUserProfile(userID, "已有昵称"); err != nil {
		t.Fatalf("upsertUserProfile: %v", err)
	}
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
	}))
	t.Cleanup(srv.Close)
	prevTenant := cfg.TenantServiceURL
	cfg.TenantServiceURL = srv.URL
	t.Cleanup(func() { cfg.TenantServiceURL = prevTenant })

	if err := ensureWechatProfileNickname(userID, "微信昵称X"); err != nil {
		t.Fatalf("ensureWechatProfileNickname: %v", err)
	}
	if calls != 0 {
		t.Fatalf("expected 0 sync calls when nickname kept, got %d", calls)
	}
}

// OPT-20260822-034: 微信建号先写 profile 再发 USER_CREATED，
// 建公司侧可通过 POST /api/internal/users/batch/details/ 读到昵称。
func TestCreateWechatUserProfileAvailableBeforeUserCreated(t *testing.T) {
	setupAuthTestDB(t)
	origSecret := cfg.InternalSecret
	cfg.InternalSecret = "test-secret"
	t.Cleanup(func() { cfg.InternalSecret = origSecret })

	var usernameAtPublish string
	var eventUsername string
	orig := publishWechatUserCreated
	t.Cleanup(func() { publishWechatUserCreated = orig })
	publishWechatUserCreated = func(userID, phone, email, username string) {
		eventUsername = username
		usernameAtPublish = batchDetailsUsername(t, userID)
		orig(userID, phone, email, username)
	}

	userID, err := createWechatUser("u-opt-034-union", "软刀", "", timeNowUTC())
	if err != nil {
		t.Fatalf("createWechatUser: %v", err)
	}
	if eventUsername != "" {
		t.Fatalf("USER_CREATED username must stay empty (not phone/raw nickname), got %q", eventUsername)
	}
	if usernameAtPublish != "软刀" {
		t.Fatalf("batch/details username at USER_CREATED = %q, want 软刀", usernameAtPublish)
	}
	if got := profilePersonalNickname(t, userID); got != "软刀" {
		t.Fatalf("profile personal_nickname = %q, want 软刀", got)
	}
}

func TestCreateWechatUserEmptyNicknameStillPublishes(t *testing.T) {
	setupAuthTestDB(t)
	published := false
	orig := publishWechatUserCreated
	t.Cleanup(func() { publishWechatUserCreated = orig })
	publishWechatUserCreated = func(userID, phone, email, username string) {
		published = true
		orig(userID, phone, email, username)
	}

	userID, err := createWechatUser("u-opt-034-empty", "", "", timeNowUTC())
	if err != nil {
		t.Fatalf("createWechatUser: %v", err)
	}
	if !published {
		t.Fatal("USER_CREATED must still fire when wechat nickname is empty")
	}
	if got := profilePersonalNickname(t, userID); got != "" {
		t.Fatalf("empty wechat nickname must not seed profile, got %q", got)
	}
}

func batchDetailsUsername(t *testing.T, userID string) string {
	t.Helper()
	body := strings.NewReader(`{"user_ids":["` + userID + `"]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/internal/users/batch/details/", body)
	req.Header.Set("X-TaskAuth-Internal-Secret", cfg.InternalSecret)
	rec := httptest.NewRecorder()
	handleBatchUserDetails(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("batch/details status=%d body=%s", rec.Code, rec.Body.String())
	}
	var payload struct {
		Results map[string]struct {
			Username string `json:"username"`
		} `json:"results"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode batch/details: %v", err)
	}
	return payload.Results[userID].Username
}
