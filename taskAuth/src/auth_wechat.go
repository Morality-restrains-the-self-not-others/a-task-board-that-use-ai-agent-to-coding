package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// --- WeChat config (v64 multi-app) ---

// WeChatAppConfig holds one WeChat Open Platform app credential.
type WeChatAppConfig struct {
	Key            string // 应用逻辑名: 'web' / 'inapp' / 'miniapp' / 'mp'
	AppID          string
	AppSecret      string
	RedirectURI    string
	Type           string // 'qr' | 'web_oauth' | 'mp'（服务号消息回调，无 redirectUri）
	Token          string // 服务号服务器配置 Token（验签）；仅 type=mp
	EncodingAESKey string // 服务号 EncodingAESKey（43 字符）；空则仅明文 XML
}

var wechatApps map[string]*WeChatAppConfig

func wechatEnabled() bool {
	return len(wechatApps) > 0
}

// wechatLoginPolicyEnabled 返回管理员是否开启「微信扫码登录」策略
// （auth_system_feature_policy.enable_wechat_login，产品默认为开启，2026-08-09 调整）。
// 修复：此前登录链路仅检查应用凭据是否配置（wechatEnabled），管理员关闭
// 开关后用户仍可扫码登录 — 现在登录链路必须同时满足「已配置」与「策略开启」。
// DB 读取失败按关闭处理（fail-closed），与「微信应用未配置时入口不可用」同为
// 保守姿态，与产品默认值无关。
//
// OPT-20260806-028: 每次扫码登录「发起 + 回调」各直查一次策略行，登录高峰期
// 属无谓 DB 往返 — 加 30s TTL 内存缓存（管理员开关修改后最多延迟 30s 生效，
// 可接受）。DB 读取失败时清缓存并按 fail-closed 拒绝。
var (
	wechatPolicyCacheMu    sync.Mutex
	wechatPolicyCacheValid bool
	wechatPolicyCacheVal   bool
	wechatPolicyCacheAt    time.Time
)

const wechatPolicyCacheTTL = 30 * time.Second

func wechatLoginPolicyEnabled() bool {
	wechatPolicyCacheMu.Lock()
	defer wechatPolicyCacheMu.Unlock()
	if wechatPolicyCacheValid && time.Since(wechatPolicyCacheAt) < wechatPolicyCacheTTL {
		return wechatPolicyCacheVal
	}
	row, err := loadFeaturePolicy()
	if err != nil {
		// fail-closed + 清缓存：DB 恢复后重新加载
		wechatPolicyCacheValid = false
		log.Printf("[taskAuth] event=wechat_policy_check err=%v enabled=false fail_closed=true", err)
		return false
	}
	wechatPolicyCacheVal = row.EnableWechatLogin
	wechatPolicyCacheAt = time.Now()
	wechatPolicyCacheValid = true
	return wechatPolicyCacheVal
}

// invalidateWechatPolicyCache 管理员保存策略后立即失效缓存（保存即生效）。
func invalidateWechatPolicyCache() {
	wechatPolicyCacheMu.Lock()
	wechatPolicyCacheValid = false
	wechatPolicyCacheMu.Unlock()
}

// wechatAppByKey returns the app for key if fully configured, nil otherwise.
func wechatAppByKey(key string) *WeChatAppConfig {
	a, ok := wechatApps[key]
	if !ok || a.AppID == "" || a.AppSecret == "" || a.RedirectURI == "" {
		return nil
	}
	return a
}

// wechatDefaultApp prefers 'web', falls back to any configured app.
func wechatDefaultApp() *WeChatAppConfig {
	if a := wechatAppByKey("web"); a != nil {
		return a
	}
	for _, a := range wechatApps {
		if a.AppID != "" && a.AppSecret != "" && a.RedirectURI != "" {
			return a
		}
	}
	return nil
}

// --- OAuth URLs ---

const (
	wechatQRConnectURL        = "https://open.weixin.qq.com/connect/qrconnect"
	wechatOAuth2AuthorizeURL  = "https://open.weixin.qq.com/connect/oauth2/authorize"
	wechatOAuthAccessTokenURL = "https://api.weixin.qq.com/sns/oauth2/access_token"
	wechatOAuthUserInfoURL    = "https://api.weixin.qq.com/sns/userinfo"
)

// --- State token (single-process in-memory; state 编码 app_key 防回调混淆) ---

type wechatStateEntry struct {
	State      string
	AppKey     string
	BindUserID string // 非空 = 绑定流程（state 携带绑定上下文; 微信回调无法带自定义头，只能靠 state）
	Next       string // 登录流程原始回跳地址（同源相对路径，防开放重定向）— 回调后拼回前端
	TraceID    string // 扫码链路 traceId：state 第三段，全链路（发起→回调）可关联（Grafana 检索）
	AccessCode string // 分享链接携带的推荐码（OPT-20260820-036）：发起扫码时从 URL/sessionStorage 读入，
	// 回调登录态/新注册用户创建后回填 billing_referral_edge。非登录态绑定上下文。
	CreatedAt time.Time
}

// wechatStateStore 并发访问受锁保护：HTTP 处理器（generate/consume）与
// 后台 GC 协程（startWechatStateGC）均读写此 map。
var (
	wechatStateMu    sync.RWMutex
	wechatStateStore = make(map[string]wechatStateEntry)
)

// newWechatTraceID 生成 UUIDv4 形态的扫码链路 traceId（与前端 X-Trace-Id 同形态，便于混排检索）。
func newWechatTraceID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	buf[6] = (buf[6] & 0x0f) | 0x40 // version 4
	buf[8] = (buf[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%x-%x-%x-%x-%x", buf[0:4], buf[4:6], buf[6:8], buf[8:10], buf[10:16]), nil
}

// generateWeChatState 生成 state：base64url(app_key) + "." + random + "." + traceId。
// traceId 埋入 state 末段：微信回调无法携带自定义 header，扫码全链路状态变化
// （发起 → 回调 → 过期/失效）只能靠 state 关联；Grafana 检索该 traceId 即可还原。
// parseAppKeyFromState 只读首段，向后兼容既有 state 形态。
// accessCode（可选，variadic 保持既有调用不变）：分享链接带 ?accessCode= 时透传到
// 回调，新注册用户创建后回填 billing_referral_edge（OPT-20260820-036）。
func generateWeChatState(appKey, bindUserID, next string, accessCodes ...string) (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	traceID, err := newWechatTraceID()
	if err != nil {
		return "", err
	}
	// state = base64url(app_key) + "." + random + "." + traceId
	state := base64.RawURLEncoding.EncodeToString([]byte(appKey)) + "." +
		base64.RawURLEncoding.EncodeToString(buf) + "." + traceID
	accessCode := ""
	if len(accessCodes) > 0 {
		accessCode = strings.TrimSpace(accessCodes[0])
	}
	wechatStateMu.Lock()
	wechatStateStore[state] = wechatStateEntry{
		State:      state,
		AppKey:     appKey,
		BindUserID: bindUserID,
		Next:       next,
		TraceID:    traceID,
		AccessCode: accessCode,
		CreatedAt:  time.Now().UTC(),
	}
	wechatStateMu.Unlock()
	log.Printf("[taskAuth] event=wechat_state_issued trace_id=%s app=%s bind=%s next=%q",
		traceID, appKey, bindUserID, next)
	return state, nil
}

func consumeWeChatState(state string) (wechatStateEntry, bool) {
	wechatStateMu.Lock()
	entry, ok := wechatStateStore[state]
	if !ok {
		wechatStateMu.Unlock()
		return wechatStateEntry{}, false
	}
	delete(wechatStateStore, state)
	wechatStateMu.Unlock()
	if time.Since(entry.CreatedAt) > wechatStateTTL {
		return wechatStateEntry{}, false
	}
	return entry, true
}

// wechatStateTTL 与 consumeWeChatState 的过期判定一致：超过该时长的
// state 视为无效（回调消费时也会拒绝）。GC 按此阈值清扫。
const wechatStateTTL = 10 * time.Minute

// sweepExpiredWeChatStates 清除所有超过 TTL 的 state 条目，返回清扫数量。
// 供 GC 循环与测试共用；调用方需自行加锁。
func sweepExpiredWeChatStates(now time.Time) int {
	expired := 0
	for s, entry := range wechatStateStore {
		if now.Sub(entry.CreatedAt) > wechatStateTTL {
			delete(wechatStateStore, s)
			expired++
			log.Printf("[taskAuth] event=wechat_state_gc_expire trace_id=%s app=%s age_sec=%.0f",
				entry.TraceID, entry.AppKey, now.Sub(entry.CreatedAt).Seconds())
		}
	}
	return expired
}

// startWechatStateGC 周期清扫过期 state 条目：state 仅在回调消费时删除，
// 用户放弃扫码/二维码过期等场景会留下永不消费的条目，长期运行内存持续增长。
// 每分钟清扫一次 >TTL 条目；扫到过期条目输出 traceId 便于扫码链路排障。
func startWechatStateGC() {
	go func() {
		for {
			time.Sleep(1 * time.Minute)
			now := time.Now().UTC()
			wechatStateMu.Lock()
			expired := sweepExpiredWeChatStates(now)
			remaining := len(wechatStateStore)
			wechatStateMu.Unlock()
			if expired > 0 {
				log.Printf("[taskAuth] event=wechat_state_gc swept=%d remaining=%d", expired, remaining)
			}
		}
	}()
}

// parseAppKeyFromState decodes the app_key prefix from a state string.
func parseAppKeyFromState(state string) string {
	if i := strings.Index(state, "."); i > 0 {
		if b, err := base64.RawURLEncoding.DecodeString(state[:i]); err == nil {
			return string(b)
		}
	}
	return ""
}

// parseTraceIDFromState extracts the trailing traceId segment from a state string.
// 仅识别新三段式（app_key.random.traceId）；旧两段式 state（无 traceId 段）返回空，
// 避免把 random 段误当 traceId 写入日志。回调状态无效/过期时仍能输出 traceId，
// 保证扫码链路日志可关联。
func parseTraceIDFromState(state string) string {
	if state == "" || strings.Count(state, ".") < 2 {
		return ""
	}
	if i := strings.LastIndex(state, "."); i > 0 {
		t := state[i+1:]
		if t != "" && !strings.ContainsAny(t, " \t\r\n") {
			return t
		}
	}
	return ""
}

// --- Handlers ---

// sanitizeNextPath 仅接受同源相对路径（/ 开头、不含 scheme、非 //、无换行），
// 用于 OAuth 登录回跳 next 参数，防开放重定向（与前端 PostLoginReturnUrl.normalize 对齐）。
func sanitizeNextPath(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || len(raw) > 2048 {
		return ""
	}
	if !strings.HasPrefix(raw, "/") {
		return ""
	}
	if strings.HasPrefix(raw, "//") {
		return ""
	}
	if strings.Contains(raw, "://") {
		return ""
	}
	if strings.ContainsAny(raw, "\n\r\x00") {
		return ""
	}
	return raw
}

// handleWeChatLogin initiates WeChat login.
// GET /api/auth/wechat/login/?app=web|inapp&next=/path （默认 web，兼容旧无参调用）
func handleWeChatLogin(w http.ResponseWriter, r *http.Request) {
	if !wechatEnabled() {
		writeError(w, r, http.StatusServiceUnavailable, "微信登录未配置")
		return
	}
	if !wechatLoginPolicyEnabled() {
		log.Printf("[taskAuth] event=wechat_login_rejected reason=policy_disabled ip=%s", resolveClientIP(r))
		writeError(w, r, http.StatusForbidden, "微信登录未开启")
		return
	}
	app := resolveWechatAppFromRequest(w, r)
	if app == nil {
		return
	}
	next := sanitizeNextPath(r.URL.Query().Get("next"))
	// OPT-20260820-036: 分享链接带 ?accessCode= 时透传到回调；发起扫码时前端从
	// sessionStorage 读入（readStoredReferralAccessCode），此处只做服务端兜底透传。
	accessCode := strings.TrimSpace(r.URL.Query().Get("accessCode"))
	state, err := generateWeChatState(app.Key, "", next, accessCode)
	if err != nil {
		log.Printf("[taskAuth] wechat state generation failed: %v", err)
		writeError(w, r, http.StatusInternalServerError, "内部错误")
		return
	}
	http.Redirect(w, r, wechatAuthorizeURL(app, state), http.StatusFound)
}

// handleWeChatCallback handles WeChat OAuth callback (login and bind).
// GET /api/auth/wechat/{web,inapp}/callback/?code=CODE&state=STATE
// （旧路径 /api/auth/wechat/callback/ 保持注册，state 决定 app）
func handleWeChatCallback(w http.ResponseWriter, r *http.Request) {
	if !wechatEnabled() {
		writeError(w, r, http.StatusServiceUnavailable, "微信登录未配置")
		return
	}

	code := strings.TrimSpace(r.URL.Query().Get("code"))
	stateParam := strings.TrimSpace(r.URL.Query().Get("state"))
	errParam := strings.TrimSpace(r.URL.Query().Get("errcode"))

	if errParam != "" || code == "" {
		msg := "微信授权失败"
		if errMsg := r.URL.Query().Get("errmsg"); errMsg != "" {
			msg = "微信授权失败: " + errMsg
		}
		redirectToLoginWithError(w, r, msg)
		return
	}

	state, ok := consumeWeChatState(stateParam)
	if !ok {
		// 状态无效/过期：仍输出 traceId（state 末段），保证扫码链路可关联检索
		log.Printf("[taskAuth] event=wechat_state_consumed trace_id=%s ok=false reason=missing_or_expired",
			parseTraceIDFromState(stateParam))
		redirectToLoginWithError(w, r, "微信登录状态无效或已过期，请重新登录")
		return
	}
	log.Printf("[taskAuth] event=wechat_state_consumed trace_id=%s app=%s ok=true", state.TraceID, state.AppKey)
	app := wechatAppByKey(state.AppKey)
	if app == nil {
		log.Printf("[taskAuth] event=wechat_state_consumed trace_id=%s ok=false reason=app_config_missing", state.TraceID)
		redirectToLoginWithError(w, r, "微信应用配置缺失")
		return
	}

	// 登录流程受管理员开关控制（纵深防御：用户在扫码途中开关被关闭，回调也拒绝发 token）。
	// 绑定流程（登录态用户账号管理）不受此开关约束。
	if state.BindUserID == "" && !wechatLoginPolicyEnabled() {
		log.Printf("[taskAuth] event=wechat_state_consumed trace_id=%s ok=false reason=policy_disabled", state.TraceID)
		redirectToLoginWithError(w, r, "微信登录未开启")
		return
	}

	// Step 1: Exchange code for access_token（使用 state 编码的 app 凭据）
	accessTokenResp, err := exchangeWeChatCodeForToken(app, code)
	if err != nil {
		log.Printf("[taskAuth] event=wechat_token_exchange trace_id=%s ok=false err=%v", state.TraceID, err)
		redirectToLoginWithError(w, r, "微信登录失败，请稍后重试")
		return
	}
	if accessTokenResp.UnionID == "" && accessTokenResp.OpenID == "" {
		log.Printf("[taskAuth] event=wechat_token_exchange trace_id=%s ok=false err=empty_openid_unionid", state.TraceID)
		redirectToLoginWithError(w, r, "获取微信用户信息失败")
		return
	}

	// Step 2: Get user info
	userInfo, err := fetchWeChatUserInfo(accessTokenResp.AccessToken, accessTokenResp.OpenID)
	if err != nil {
		log.Printf("[taskAuth] event=wechat_userinfo trace_id=%s ok=false err=%v", state.TraceID, err)
		redirectToLoginWithError(w, r, "获取微信用户信息失败")
		return
	}

	unionID := userInfo.UnionID
	if unionID == "" {
		unionID = accessTokenResp.UnionID
	}
	openID := userInfo.OpenID

	// 绑定流程（登录态发起，state 携带 bind 上下文）优先于「登录或创建」
	if state.BindUserID != "" {
		userID, err := bindWeChatIdentity(state.BindUserID, state.AppKey, app.AppID, unionID, openID, userInfo.Nickname, userInfo.HeadImgURL)
		if err != nil {
			if errors.Is(err, errWechatIdentityConflict) {
				redirectToLoginWithError(w, r, "该微信已绑定其他账号，请直接用该微信登录")
				return
			}
			log.Printf("[taskAuth] event=wechat_bind trace_id=%s ok=false user=%s err=%v", state.TraceID, state.BindUserID, err)
			redirectToLoginWithError(w, r, "微信绑定失败，请稍后重试")
			return
		}
		log.Printf("[taskAuth] event=wechat_bind trace_id=%s ok=true user=%s app=%s", state.TraceID, userID, state.AppKey)
		// 绑定后补齐个人昵称（仅空时写入，不覆盖用户自定义昵称）
		if err := ensureWechatProfileNickname(state.BindUserID, userInfo.Nickname); err != nil {
			log.Printf("[taskAuth] event=wechat_bind_profile_nickname trace_id=%s user=%s err=%v (non-fatal)",
				state.TraceID, state.BindUserID, err)
		}
		redirectToFrontend(w, r, wechatBindSuccessPath(state.Next))
		return
	}

	// Step 3: Find or create user by unionid（unionid 优先，openid 按 app 维度兜底）
	userID, created, err := findOrCreateWeChatUser(state.AppKey, app.AppID, unionID, openID, userInfo.Nickname, userInfo.HeadImgURL)
	if err != nil {
		if errors.Is(err, errWechatIdentityConflict) {
			log.Printf("[taskAuth] event=wechat_user_match trace_id=%s ok=false reason=identity_conflict user_id=%s", state.TraceID, userID)
			redirectToLoginWithError(w, r, "微信身份异常，请联系客服处理")
			return
		}
		log.Printf("[taskAuth] event=wechat_user_match trace_id=%s ok=false err=%v", state.TraceID, err)
		redirectToLoginWithError(w, r, "登录失败，请稍后重试")
		return
	}

	// OPT-20260820-036: 首次扫码自动注册且分享链接带 accessCode 时回填推荐关系
	// （存量用户重登不绑定，避免把已有账号误记到分享者名下）。
	if created && state.AccessCode != "" {
		bindReferralAfterRegisterAsync(userID, state.AccessCode)
	}

	// 登录后补齐个人昵称（同步直写，仅空时写入）：修复「微信昵称未设置到个人昵称」。
	// 不依赖 USER_CREATED 事件链（syncprofile intent 存在部署/时序缺口）；
	// USER_CREATED 事件本身不再携带昵称（避免昵称被误用作公司名称）。
	if err := ensureWechatProfileNickname(userID, userInfo.Nickname); err != nil {
		log.Printf("[taskAuth] event=wechat_login_profile_nickname trace_id=%s user=%s err=%v (non-fatal)",
			state.TraceID, userID, err)
	}

	// Step 4: Generate token
	token, err := getOrCreateToken(userID, cfg.UserContentTypeID, resolveClientIP(r))
	if err != nil {
		log.Printf("[taskAuth] event=wechat_token_issue trace_id=%s ok=false user_id=%s err=%v", state.TraceID, userID, err)
		redirectToLoginWithError(w, r, "登录失败，请稍后重试")
		return
	}
	log.Printf("[taskAuth] event=wechat_login_success trace_id=%s user_id=%s app=%s", state.TraceID, userID, state.AppKey)

	touchLastLogin(userID)
	recordSuccessfulLoginFromRequest(r, userID, unionID, "wechat", "wechat", state.AppKey)

	// Step 6: Redirect to frontend with token + 登录落点。
	// next 优先取用户发起登录时携带的原始回跳（handleWeChatLogin 存入 state），
	// 否则按角色计算（超管→/system-admin/，有公司→/tenant/{id}/work-panel/，无公司→/onboarding/；
	// 仅在无用户 next 时调用，避免回调路径额外依赖 tenant service）。
	// 例外：next 指向 /onboarding 引导页时弃用该 next，改按角色计算落点 —
	// 已设置公司名的用户可能经陈旧链接/书签/Onboarding 页 401 循环携带
	// next=/onboarding/ 发起扫码，无条件回显会让「第二次扫码登录依旧跳到公司
	// 名称设置页」；角色计算保证有公司→work-panel、无公司才→onboarding。
	// 修复：此前回调仅回 wechat_token，前端兜底跳到公开首页 "/"，导致
	// 「微信扫码登录后跳转到首页」— 见 auth_wechat_test.go 回归用例。
	dest := sanitizeNextPath(state.Next)
	if dest == "/onboarding" || strings.HasPrefix(dest, "/onboarding/") {
		dest = ""
	}
	if dest == "" {
		dest = resolveLoginRedirect(buildLoginUserJSON(userID, token))
	}
	redirectToFrontend(w, r, "/auth/login/?wechat_token="+url.QueryEscape(token)+"&next="+url.QueryEscape(dest))
}

// handleWeChatBind initiates WeChat bind for the authenticated user.
// GET /api/auth/wechat/bind/?app=web|inapp （Authorization: Token xxx）
func handleWeChatBind(w http.ResponseWriter, r *http.Request) {
	if !wechatEnabled() {
		writeError(w, r, http.StatusServiceUnavailable, "微信登录未配置")
		return
	}
	userID, err := resolveTokenUserIDFromRequest(r)
	if err != nil {
		writeErrorDetail(w, r, http.StatusUnauthorized, "未登录")
		return
	}
	app := resolveWechatAppFromRequest(w, r)
	if app == nil {
		return
	}
	state, err := generateWeChatState(app.Key, userID, wechatBindNextFromRequest(r))
	if err != nil {
		log.Printf("[taskAuth] wechat bind state generation failed: %v", err)
		writeError(w, r, http.StatusInternalServerError, "内部错误")
		return
	}
	http.Redirect(w, r, wechatAuthorizeURL(app, state), http.StatusFound)
}

// handleWeChatUnbind unbinds a WeChat app identity from the authenticated user.
// DELETE /api/auth/wechat/unbind/  body: {"app_key": "web"}
func handleWeChatUnbind(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID, err := resolveTokenUserIDFromRequest(r)
	if err != nil {
		writeErrorDetail(w, r, http.StatusUnauthorized, "未登录")
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeErrorDetail(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	appKey := strings.TrimSpace(strField(body, "app_key"))
	if appKey == "" {
		appKey = "web"
	}
	if err := unbindWeChatIdentity(userID, appKey); err != nil {
		writeErrorDetail(w, r, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"unbound": true, "app_key": appKey})
}

// resolveWechatAppFromRequest picks the app by ?app= query (default 'web').
// On unknown/incomplete app it writes an error response and returns nil.
func resolveWechatAppFromRequest(w http.ResponseWriter, r *http.Request) *WeChatAppConfig {
	appKey := strings.TrimSpace(r.URL.Query().Get("app"))
	if appKey == "" {
		appKey = "web"
	}
	app := wechatAppByKey(appKey)
	if app == nil {
		writeError(w, r, http.StatusBadRequest, "未知微信应用: "+appKey)
		return nil
	}
	return app
}

// wechatAuthorizeURL builds the OAuth authorize URL for the app type.
func wechatAuthorizeURL(app *WeChatAppConfig, state string) string {
	if app.Type == "web_oauth" {
		return fmt.Sprintf("%s?appid=%s&redirect_uri=%s&response_type=code&scope=snsapi_userinfo&state=%s#wechat_redirect",
			wechatOAuth2AuthorizeURL,
			url.QueryEscape(app.AppID),
			url.QueryEscape(app.RedirectURI),
			url.QueryEscape(state),
		)
	}
	// 默认: 开放平台网站应用扫码
	return fmt.Sprintf("%s?appid=%s&redirect_uri=%s&response_type=code&scope=snsapi_login&state=%s#wechat_redirect",
		wechatQRConnectURL,
		url.QueryEscape(app.AppID),
		url.QueryEscape(app.RedirectURI),
		url.QueryEscape(state),
	)
}

// --- WeChat API types ---

type wechatAccessTokenResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	OpenID       string `json:"openid"`
	Scope        string `json:"scope"`
	UnionID      string `json:"unionid"`
	ErrCode      int    `json:"errcode"`
	ErrMsg       string `json:"errmsg"`
}

type wechatUserInfoResponse struct {
	OpenID     string `json:"openid"`
	Nickname   string `json:"nickname"`
	Sex        int    `json:"sex"`
	Province   string `json:"province"`
	City       string `json:"city"`
	Country    string `json:"country"`
	HeadImgURL string `json:"headimgurl"`
	UnionID    string `json:"unionid"`
	ErrCode    int    `json:"errcode"`
	ErrMsg     string `json:"errmsg"`
}

// --- API calls ---

func exchangeWeChatCodeForToken(app *WeChatAppConfig, code string) (*wechatAccessTokenResponse, error) {
	apiURL := fmt.Sprintf("%s?appid=%s&secret=%s&code=%s&grant_type=authorization_code",
		wechatOAuthAccessTokenURL,
		url.QueryEscape(app.AppID),
		url.QueryEscape(app.AppSecret),
		url.QueryEscape(code),
	)
	resp, err := wechatHTTPGet(apiURL)
	if err != nil {
		return nil, fmt.Errorf("http get: %w", err)
	}
	var result wechatAccessTokenResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("json parse: %w (body: %s)", err, string(resp))
	}
	if result.ErrCode != 0 {
		return nil, fmt.Errorf("wechat api error: %d %s", result.ErrCode, result.ErrMsg)
	}
	return &result, nil
}

func fetchWeChatUserInfo(accessToken, openID string) (*wechatUserInfoResponse, error) {
	apiURL := fmt.Sprintf("%s?access_token=%s&openid=%s",
		wechatOAuthUserInfoURL,
		url.QueryEscape(accessToken),
		url.QueryEscape(openID),
	)
	resp, err := wechatHTTPGet(apiURL)
	if err != nil {
		return nil, fmt.Errorf("http get: %w", err)
	}
	var result wechatUserInfoResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("json parse: %w (body: %s)", err, string(resp))
	}
	if result.ErrCode != 0 {
		return nil, fmt.Errorf("wechat api error: %d %s", result.ErrCode, result.ErrMsg)
	}
	return &result, nil
}

// --- Identity matching (v64) ---

// errWechatIdentityConflict: (app_key, openid) 已归属另一用户且该用户已有不同 unionid —
// 微信侧双射异常，拒绝登录/绑定，发冲突事件人工介入。
var errWechatIdentityConflict = errors.New("wechat identity conflict")

// findOrCreateWeChatUser 登录匹配顺序（unionid 为身份真源）:
//
//  1. unionid 非空:
//     a. 查 wechat_identity by unionid → 命中: 登录该 user + upsert (app_key, openid) 别名
//     b. 未命中 → 查 (app_key, openid):
//     - 命中且其 user 无其他 unionid → 绑定转移: 新建 unionid 用户, openid 别名改绑 + WechatIdentityLinked
//     - 命中但其 user 有不同 unionid → 冲突: 拒绝 + WechatIdentityConflict
//     c. 均未命中 → 新建 user + (app_key, openid, unionid)
//  2. unionid 为空（首次授权场景）:
//     a. 查 (app_key, openid) → 命中: 登录该 user（同应用重登不分裂）
//     b. 未命中 → 兼容层: auth_login_method 裸 openid 行 → 登录 + 收敛 wechat_identity
//     c. 未命中 → 新建 user + (app_key, openid, unionid=”)
func findOrCreateWeChatUser(appKey, appID, unionID, openID, nickname, headImgURL string) (string, bool, error) {
	now := timeNowUTC()

	if unionID != "" {
		// 1a. unionid 真源命中
		if userID, ok := findWechatUserByUnionID(unionID); ok {
			upsertWechatIdentity(userID, appKey, appID, openID, unionID, nickname, headImgURL, now)
			touchWechatLoginMethod(userID, unionID, now)
			return userID, false, nil
		}
		// 1b. (app_key, openid) 归属检查
		if ownerID, ok := findWechatUserByAppOpenID(appKey, openID); ok {
			if ownerUnionID := wechatUnionIDOf(ownerID); ownerUnionID != "" {
				// 双射异常 — 拒绝
				go publishWechatIdentityConflict(context.Background(), ownerID, appKey, openID, unionID, ownerUnionID)
				return "", false, errWechatIdentityConflict
			}
			// 绑定转移: 该 openid 用户即当前 unionid 用户
			userID, err := createWechatUser(unionID, nickname, headImgURL, now)
			if err != nil {
				return "", false, err
			}
			moveWechatOpenIDAlias(appKey, openID, ownerID, userID, now)
			rebindLegacyOpenIDLoginMethod(ownerID, userID, openID, now)
			upsertWechatIdentity(userID, appKey, appID, openID, unionID, nickname, headImgURL, now)
			go publishWechatIdentityLinked(context.Background(), userID, ownerID, appKey, openID, unionID)
			return userID, true, nil
		}
		// 1c. 新建
		userID, err := createWechatUser(unionID, nickname, headImgURL, now)
		if err != nil {
			return "", false, err
		}
		upsertWechatIdentity(userID, appKey, appID, openID, unionID, nickname, headImgURL, now)
		return userID, true, nil
	}

	// 2a. 同应用重登
	if ownerID, ok := findWechatUserByAppOpenID(appKey, openID); ok {
		upsertWechatIdentity(ownerID, appKey, appID, openID, "", nickname, headImgURL, now)
		return ownerID, false, nil
	}
	// 2b. 兼容层: 存量 auth_login_method 裸 openid 行（无 unionid 建号场景）
	if ownerID, ok := findLegacyWechatOpenIDUser(openID); ok {
		upsertWechatIdentity(ownerID, appKey, appID, openID, "", nickname, headImgURL, now)
		return ownerID, false, nil
	}
	// 2c. 新建（openid 兜底）
	userID, err := createWechatUser(openID, nickname, headImgURL, now)
	if err != nil {
		return "", false, err
	}
	upsertWechatIdentity(userID, appKey, appID, openID, "", nickname, headImgURL, now)
	return userID, true, nil
}

// bindWeChatIdentity 绑定流程（登录态）:
//   - unionid 已归属其他 user → 冲突（409 语义）
//   - 已归属当前 user 或 (app_key, openid) 已归属当前 user → 幂等成功
//   - 未归属 → 绑定: wechat_identity + auth_login_method unionid 行 + WechatBound
func bindWeChatIdentity(bindUserID, appKey, appID, unionID, openID, nickname, headImgURL string) (string, error) {
	now := timeNowUTC()
	if unionID != "" {
		if ownerID, ok := findWechatUserByUnionID(unionID); ok {
			if ownerID != bindUserID {
				go publishWechatIdentityConflict(context.Background(), ownerID, appKey, openID, unionID, "")
				return "", errWechatIdentityConflict
			}
			upsertWechatIdentity(bindUserID, appKey, appID, openID, unionID, nickname, headImgURL, now)
			return bindUserID, nil
		}
	}
	if ownerID, ok := findWechatUserByAppOpenID(appKey, openID); ok {
		if ownerID != bindUserID {
			if u := wechatUnionIDOf(ownerID); u != "" && unionID != "" && u != unionID {
				go publishWechatIdentityConflict(context.Background(), ownerID, appKey, openID, unionID, u)
				return "", errWechatIdentityConflict
			}
			// openid 已被旧账号占用且无 unionid → 绑定转移
			moveWechatOpenIDAlias(appKey, openID, ownerID, bindUserID, now)
			rebindLegacyOpenIDLoginMethod(ownerID, bindUserID, openID, now)
		}
		upsertWechatIdentity(bindUserID, appKey, appID, openID, unionID, nickname, headImgURL, now)
		return bindUserID, nil
	}
	upsertWechatIdentity(bindUserID, appKey, appID, openID, unionID, nickname, headImgURL, now)
	if unionID != "" {
		ensureWechatLoginMethod(bindUserID, unionID, now)
	}
	go publishWechatBound(context.Background(), bindUserID, appKey, openID, unionID)
	return bindUserID, nil
}

// unbindWeChatIdentity 解绑: 移除指定 app 的 wechat_identity 别名。
// 若用户已无任何微信别名 → void auth_login_method wechat 行（unionid 行与裸 openid 行）。
// 校验账号至少剩余一种登录方式，避免裸号。
func unbindWeChatIdentity(userID, appKey string) error {
	now := timeNowUTC()

	// 删除该 app 的别名行
	if _, err := db.Exec(`DELETE FROM wechat_identity WHERE user_id = ? AND app_key = ?`, userID, appKey); err != nil {
		return fmt.Errorf("remove wechat identity: %w", err)
	}

	// 若无任何剩余微信别名 → void 登录方式行（unionid 行跨应用共享，仅在全解绑时处理）
	var remainingAliases int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM wechat_identity WHERE user_id = ?`, userID).Scan(&remainingAliases); err != nil {
		return fmt.Errorf("count wechat aliases: %w", err)
	}
	if remainingAliases == 0 {
		if _, err := db.Exec(`
			UPDATE auth_login_method
			SET binding_voided_at = ?
			WHERE object_id = ? AND method_type = 'wechat' AND binding_voided_at IS NULL`, now, userID); err != nil {
			return fmt.Errorf("void login method: %w", err)
		}
	}

	// 剩余登录方式校验
	var remaining int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM auth_login_method
		WHERE object_id = ? AND binding_voided_at IS NULL`, userID).Scan(&remaining); err != nil {
		return fmt.Errorf("count login methods: %w", err)
	}
	if remaining == 0 {
		return fmt.Errorf("解绑后账号将无可用登录方式，已保留微信登录")
	}

	go publishWechatUnbound(context.Background(), userID, appKey)
	return nil
}

// --- wechat_identity / login_method helpers ---

func findWechatUserByUnionID(unionID string) (string, bool) {
	if unionID == "" {
		return "", false
	}
	var userID string
	err := db.QueryRow(`
		SELECT user_id FROM wechat_identity
		WHERE unionid = ? AND unionid != ''
		ORDER BY created_at LIMIT 1`, unionID).Scan(&userID)
	if err != nil {
		return "", false
	}
	return userID, true
}

func findWechatUserByAppOpenID(appKey, openID string) (string, bool) {
	if appKey == "" || openID == "" {
		return "", false
	}
	var userID string
	err := db.QueryRow(`
		SELECT user_id FROM wechat_identity
		WHERE app_key = ? AND openid = ?
		LIMIT 1`, appKey, openID).Scan(&userID)
	if err != nil {
		return "", false
	}
	return userID, true
}

// wechatUnionIDOf returns the unionid recorded for a user (any app), "" if none.
func wechatUnionIDOf(userID string) string {
	var unionID string
	err := db.QueryRow(`
		SELECT unionid FROM wechat_identity
		WHERE user_id = ? AND unionid != ''
		ORDER BY created_at LIMIT 1`, userID).Scan(&unionID)
	if err != nil {
		return ""
	}
	return unionID
}

// findLegacyWechatOpenIDUser 兼容层: 存量 auth_login_method 裸 openid 行（无 unionid 建号场景）。
func findLegacyWechatOpenIDUser(openID string) (string, bool) {
	if openID == "" {
		return "", false
	}
	var userID string
	err := db.QueryRow(`
		SELECT object_id FROM auth_login_method
		WHERE method_type = 'wechat' AND identifier = ? AND binding_voided_at IS NULL
		LIMIT 1`, openID).Scan(&userID)
	if err != nil {
		return "", false
	}
	return userID, true
}

// createWechatUser 新建用户 + unionid（或 openid）登录方式行。
func createWechatUser(identifier, nickname, headImgURL string, now string) (string, error) {
	userID := fmt.Sprintf("%d", generateSnowflakeID())

	tx, err := db.Begin()
	if err != nil {
		return "", fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`
		INSERT INTO auth_user (id, password, last_login, is_superuser, is_staff, is_active, date_joined)
		VALUES (?, '', NULL, 0, 0, 1, ?)`, userID, now); err != nil {
		return "", fmt.Errorf("insert user: %w", err)
	}

	if _, err := tx.Exec(`
		INSERT INTO auth_login_method (
			id, content_type_id, object_id, method_type, identifier,
			password_hash, is_verified, created_at, updated_at
		) VALUES (?, ?, ?, 'wechat', ?, '', 1, ?, ?)`,
		generateSnowflakeID(), cfg.UserContentTypeID, userID, identifier, now, now); err != nil {
		return "", fmt.Errorf("insert login method: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("commit: %w", err)
	}

	// OPT-20260822-034: 先写入 auth_user_profile，再发 USER_CREATED。
	// 建公司消费者经 POST /api/internal/users/batch/details/ 取 username；
	// 事件 username 仍留空，避免把裸昵称/手机号直接当公司名。
	if err := ensureWechatProfileNickname(userID, nickname); err != nil {
		log.Printf("[taskAuth] event=wechat_create_profile_nickname user_id=%s ok=false err=%v", userID, err)
	} else if strings.TrimSpace(nickname) != "" {
		log.Printf("[taskAuth] event=wechat_create_profile_nickname user_id=%s ok=true", userID)
	}
	publishWechatUserCreated(userID, "", "", "")
	return userID, nil
}

// publishWechatUserCreated 接缝：单测可替换，断言发事件时 profile 已可读。
var publishWechatUserCreated = publishUserCreatedAsync

// ensureWechatProfileNickname 用微信昵称补齐个人昵称（auth_user_profile.username）。
// 仅在个人昵称为空时写入：用户后续自行设置的昵称不会被微信昵称覆盖
// （微信昵称仅作初始值）。同步直写而非依赖 USER_CREATED 事件链：
// syncprofile intent 存在部署/时序缺口，且事件已不再携带昵称。
// 覆盖新用户（建号即写）与存量用户（下次扫码登录自愈）。
func ensureWechatProfileNickname(userID, nickname string) error {
	if strings.TrimSpace(nickname) == "" {
		return nil
	}
	var current string
	err := db.QueryRow(`SELECT COALESCE(username, '') FROM auth_user_profile WHERE user_id = ?`, userID).Scan(&current)
	if err == sql.ErrNoRows {
		if err := upsertUserProfile(userID, nickname); err != nil {
			return err
		}
		syncWechatCreatorMemberName(userID, nickname)
		return nil
	}
	if err != nil {
		return fmt.Errorf("read profile: %w", err)
	}
	if strings.TrimSpace(current) != "" {
		return nil // 已有昵称，不覆盖
	}
	if err := upsertUserProfile(userID, nickname); err != nil {
		return err
	}
	syncWechatCreatorMemberName(userID, nickname)
	return nil
}

// syncWechatCreatorMemberName 在个人昵称写入后，把该用户各公司误种子/空 member_name
// 行回填为个人昵称（OPT-20260812-038）。读路径自愈依赖访问 company_members，写入时
// 同步可让角色列表/Git 身份等展示面直接拿到正确成员名，且覆盖「公司创建晚于昵称写
// 入」的竞态（下次扫码登录再兜底）。非致命：失败仅记录日志，不阻断登录。
func syncWechatCreatorMemberName(userID, nickname string) {
	if strings.TrimSpace(userID) == "" || strings.TrimSpace(nickname) == "" {
		return
	}
	resp, err := tenantForwardJSON(http.MethodPatch, "/api/internal/tenant/members/sync-nickname", map[string]string{
		"user_id":  userID,
		"nickname": nickname,
	})
	if err != nil {
		log.Printf("[taskAuth] sync creator member_name user=%s: %v (non-fatal)", userID, err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("[taskAuth] sync creator member_name user=%s status=%d (non-fatal)", userID, resp.StatusCode)
	}
}

// upsertWechatIdentity 幂等写入 wechat_identity（(app_key, openid) 唯一键驱动）。
func upsertWechatIdentity(userID, appKey, appID, openID, unionID, nickname, headImgURL, now string) {
	if appKey == "" || openID == "" {
		return
	}
	_, err := db.Exec(`
		INSERT INTO wechat_identity (
			id, user_id, app_key, app_id, openid, unionid, nickname, avatar_url, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			user_id = VALUES(user_id),
			unionid = VALUES(unionid),
			nickname = VALUES(nickname),
			avatar_url = VALUES(avatar_url),
			updated_at = VALUES(updated_at)`,
		generateSnowflakeID(), userID, appKey, appID, openID, unionID, nickname, headImgURL, now, now)
	if err != nil {
		log.Printf("[taskAuth] upsert wechat_identity user=%s app=%s: %v", userID, appKey, err)
		return
	}
	// 关注在先、扫码登录在后：用 unionId 认领 pending 并绑定 mp openId。
	if strings.TrimSpace(unionID) != "" && appKey != "mp" {
		claimMpSubscribePending(userID, unionID)
	}
}

// moveWechatOpenIDAlias 将 (app_key, openid) 别名改绑到新用户（绑定转移）。
func moveWechatOpenIDAlias(appKey, openID, fromUserID, toUserID, now string) {
	_, err := db.Exec(`
		UPDATE wechat_identity
		SET user_id = ?, updated_at = ?
		WHERE app_key = ? AND openid = ? AND user_id = ?`,
		toUserID, now, appKey, openID, fromUserID)
	if err != nil {
		log.Printf("[taskAuth] move wechat alias app=%s openid=%s from=%s to=%s: %v", appKey, openID, fromUserID, toUserID, err)
	}
}

// rebindLegacyOpenIDLoginMethod void 旧用户的裸 openid 行，并在新用户下重建（兼容层收敛）。
func rebindLegacyOpenIDLoginMethod(fromUserID, toUserID, openID, now string) {
	if _, err := db.Exec(`
		UPDATE auth_login_method SET binding_voided_at = ?
		WHERE object_id = ? AND method_type = 'wechat' AND identifier = ? AND binding_voided_at IS NULL`,
		now, fromUserID, openID); err != nil {
		log.Printf("[taskAuth] void legacy openid lm from=%s: %v", fromUserID, err)
		return
	}
	_, err := db.Exec(`
		INSERT INTO auth_login_method (
			id, content_type_id, object_id, method_type, identifier,
			password_hash, is_verified, created_at, updated_at
		) VALUES (?, ?, ?, 'wechat', ?, '', 1, ?, ?)`,
		generateSnowflakeID(), cfg.UserContentTypeID, toUserID, openID, now, now)
	if err != nil {
		log.Printf("[taskAuth] rebind legacy openid lm to=%s: %v", toUserID, err)
	}
}

// touchWechatLoginMethod 更新既有 unionid 登录方式的时间戳（保持活跃标记）。
func touchWechatLoginMethod(userID, unionID, now string) {
	_, _ = db.Exec(`
		UPDATE auth_login_method
		SET updated_at = ? WHERE object_id = ? AND method_type = 'wechat' AND identifier = ?`,
		now, userID, unionID)
}

// ensureWechatLoginMethod 确保 unionid 登录方式行存在（绑定流程）。
func ensureWechatLoginMethod(userID, unionID, now string) {
	var exists int
	_ = db.QueryRow(`
		SELECT COUNT(*) FROM auth_login_method
		WHERE object_id = ? AND method_type = 'wechat' AND identifier = ? AND binding_voided_at IS NULL`,
		userID, unionID).Scan(&exists)
	if exists > 0 {
		return
	}
	_, err := db.Exec(`
		INSERT INTO auth_login_method (
			id, content_type_id, object_id, method_type, identifier,
			password_hash, is_verified, created_at, updated_at
		) VALUES (?, ?, ?, 'wechat', ?, '', 1, ?, ?)`,
		generateSnowflakeID(), cfg.UserContentTypeID, userID, unionID, now, now)
	if err != nil {
		log.Printf("[taskAuth] ensure wechat lm user=%s: %v", userID, err)
	}
}

// --- Helpers ---

func redirectToFrontend(w http.ResponseWriter, r *http.Request, pathAndQuery string) {
	base := cfg.FrontendBase
	if base == "" {
		base = cfg.GatewayPublicBase
	}
	if base != "" {
		pathAndQuery = base + pathAndQuery
	}
	http.Redirect(w, r, pathAndQuery, http.StatusFound)
}

func redirectToLoginWithError(w http.ResponseWriter, r *http.Request, msg string) {
	redirectToFrontend(w, r, "/auth/login/?wechat_error="+url.QueryEscape(msg))
}
