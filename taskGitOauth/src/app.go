package main

import (
	"bytes"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"gatewayauth"
	"taskGitOauth/infrastructure"
	"tracelog"
)

type App struct {
	Cfg    *infrastructure.Config
	DB     *infrastructure.DB
	Fernet *infrastructure.Fernet
	Sess   *infrastructure.SessionStore
	Cache  *infrastructure.AccessTokenCache

	// ExchangeGitHubCodeFn — 可注入换票函数（测试用；nil 时用真实实现）。
	ExchangeGitHubCodeFn func(cfg *infrastructure.Config, code, redirectURI string) (map[string]any, error)
	// GitAPIDoFn 可注入 GitHub/GitLab HTTP（测试 mock）；nil 时直连。
	GitAPIDoFn func(req *http.Request) (*http.Response, error)
	// RefreshAccessTokenFn 可注入换票（测试 mock）；nil 时走真实 GitHub/GitLab。
	RefreshAccessTokenFn func(providerKey, refreshPlain string) (map[string]any, error)
	// MergeAccessTokenFn 可注入当前用户对该 host 的 access token（测试）。
	MergeAccessTokenFn func(userID string, htmlURL string) (string, error)
	// ResolveDisplayNameFn 可注入 user_id → 展示名解析（测试用）；nil 时走
	// taskTenantService members/display-name 内部端点，失败回退 user_id 本身。
	ResolveDisplayNameFn func(userID string, tenantID string) string
}

func NewApp(cfg *infrastructure.Config, db *infrastructure.DB) *App {
	return &App{
		Cfg:    cfg,
		DB:     db,
		Fernet: infrastructure.NewFernet(cfg.SecretKey),
		Sess:   infrastructure.NewSessionStore(cfg.SecretKey, infrastructure.CookieDomainFromPublicBase(cfg.PublicBaseURL)),
		Cache:  infrastructure.NewAccessTokenCache(),
	}
}

// gatewayUserMiddleware promotes verified APISIX forward-auth identity
// (X-User-Id + gateway secret) into X-Auth-User-Id, the canonical header read
// by effectiveUserID. Aligns taskGitOauth with the other business services
// (OPT-20260812-044). Does not reject unauthenticated requests — handlers decide.
func (a *App) gatewayUserMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gatewayauth.ApplyGatewayUser(r, a.Cfg.GatewayInternalSecret)
		next.ServeHTTP(w, r)
	})
}

func (a *App) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/health/", a.handleHealth)
	mux.HandleFunc("/api/health", a.handleHealth)
	mux.HandleFunc("/api/swagger/", a.handleSwagger)
	mux.HandleFunc("/api/schema/", a.handleSchema)

	// Convention: /api/git-oauth/{funcName}/key/value/...
	mux.HandleFunc("/api/git-oauth/github-start/", a.handleGithubStart)
	mux.HandleFunc("/api/git-oauth/github-start-from-gateway/", a.handleGithubStartFromGateway)
	mux.HandleFunc("/api/git-oauth/github-app-start/", a.handleGithubStartFromGateway)
	mux.HandleFunc("/api/git-oauth/github-callback/", a.handleGithubCallback)

	mux.HandleFunc("/api/git-oauth/gitlab-start/", a.handleGitlabStart)
	mux.HandleFunc("/api/git-oauth/gitlab-start-from-gateway/", a.handleGitlabStartFromGateway)
	mux.HandleFunc("/api/git-oauth/gitlab-app-start/", a.handleGitlabStartFromGateway)
	mux.HandleFunc("/api/git-oauth/gitlab-callback/", a.handleGitlabCallback)

	mux.HandleFunc("/api/git-oauth/providers/", a.handleGitOauthProviders)
	// Browser OAuth callback contract path — SSOT redirect_uri (conf/base.yaml
	// subdomains.gitoauth) and provider app whitelists point at
	// /api/accounts/<service_provider>/oauth/callback/; must NOT be flattened
	// under /api/git-oauth/ (regression 8c47b8a caused callback 404).
	mux.HandleFunc("/api/accounts/", a.handleAccountsDynamic)
	// v2 浏览器回调契约路径 — SSOT redirect_uri（conf/base.yaml subdomains.base +
	// provider YAML）指向 /redirect/gitsite/<gitsite>/oauth/callback/；<gitsite> 为
	// target.website 主机名（如 github.com / gitlab.daydaymoney.com），由
	// ResolveProviderByGitsite 解析到 provider 后分发（2026-08-07 迁移，旧
	// /api/accounts/ 契约路径保留兼容）。
	mux.HandleFunc("/redirect/gitsite/", a.handleGitsiteCallback)
	mux.HandleFunc("/api/git-oauth/user-app-connection/", a.handleUserAppConnection)
	mux.HandleFunc("/api/git-oauth/tenant-connection/", a.handleTenantRoutes)
	mux.HandleFunc("/api/git-oauth/merge-request-status/", a.handleMergeRequestStatus)
	mux.HandleFunc("/api/git-oauth/merge-request-merge/", a.handleMergeRequestMerge)

	internal := func(h http.HandlerFunc) http.HandlerFunc {
		return a.requireBridgeSecret(h)
	}
	mux.HandleFunc("/api/internal/git-oauth/github-refresh/", internal(a.handleRefresh))
	mux.HandleFunc("/api/internal/git-oauth/github-access-for-user/", internal(a.handleAccessForUser))
	mux.HandleFunc("/api/internal/git-oauth/github-token-use-report/", internal(a.handleTokenUseReport))
	mux.HandleFunc("/api/internal/git-oauth/github-credential-delete/", internal(a.handleCredentialDelete))
	mux.HandleFunc("/api/internal/git-oauth/github-credential-user-ids/", internal(a.handleCredentialUserIDs))
	mux.HandleFunc("/api/internal/git-oauth/github-credential-summary/", internal(a.handleCredentialSummary))
	mux.HandleFunc("/api/internal/git-oauth/github-audit-report/", internal(a.handleTaskAuditReport))
	// 新契约别名：taskProjectService / taskCredentialService 调用
	// /api/internal/{provider}/oauth/access-for-user/（旧 /api/internal/git-oauth/ 路径保留兼容，
	// 路径契约失配曾致 access-for-user 404 → token_error，OPT-20260807-071 复盘）。
	mux.HandleFunc("/api/internal/github/oauth/access-for-user/", internal(a.handleAccessForUser))
	mux.HandleFunc("/api/internal/gitlab/oauth/access-for-user/", internal(a.handleAccessForUser))
	// OPT-20260822-036: 内部换票路径按 Git site 寻址（v97 target）。
	// /api/internal/gitsite/{site}/oauth/access-for-user/；{site} URL 编码（host[:port]），
	// 经 ResolveProviderByGitsite 解析到 provider；旧 github|gitlab|git-oauth 别名保留。
	mux.HandleFunc("/api/internal/gitsite/", internal(a.handleGitsiteAccessForUser))

	mux.HandleFunc("/api/internal/git-oauth/gitlab-refresh/", internal(a.handleRefresh))
	mux.HandleFunc("/api/internal/git-oauth/gitlab-access-for-user/", internal(a.handleAccessForUser))
	mux.HandleFunc("/api/internal/git-oauth/gitlab-token-use-report/", internal(a.handleTokenUseReport))
	mux.HandleFunc("/api/internal/git-oauth/gitlab-credential-delete/", internal(a.handleCredentialDelete))
	mux.HandleFunc("/api/internal/git-oauth/gitlab-credential-user-ids/", internal(a.handleCredentialUserIDs))
	mux.HandleFunc("/api/internal/git-oauth/gitlab-credential-summary/", internal(a.handleCredentialSummary))
	mux.HandleFunc("/api/internal/git-oauth/gitlab-audit-report/", internal(a.handleTaskAuditReport))
	mux.HandleFunc("/api/internal/git-oauth/gitlab-tenant-connection/", internal(a.handleInternalTenantGitlabOAuthConnection))
	mux.HandleFunc("/api/internal/git-oauth/grant-ticket/consume/", internal(a.handleConsumeGrantTicket))
}

func (a *App) handleTenantRoutes(w http.ResponseWriter, r *http.Request) {
	// Canonical: /api/git-oauth/tenant-connection/tenant_id/{tid}/
	// Legacy:    /api/git-oauth/tenant-connection/{tid}/gitlab-oauth-connection/
	if isTenantGitlabReachabilityPath(r.URL.Path) {
		a.handleTenantGitlabReachability(w, r)
		return
	}
	if isTenantGitlabConnectionPath(r.URL.Path) {
		a.handleTenantGitlabOAuthConnection(w, r)
		return
	}
	http.NotFound(w, r)
}

func (a *App) requireBridgeSecret(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !a.Cfg.RequireBridgeSecret {
			next(w, r)
			return
		}
		want := strings.TrimSpace(a.Cfg.BridgeJWTSecret)
		got := strings.TrimSpace(r.Header.Get("X-GitOauth-Bridge-Secret"))
		if want == "" || got == "" || len(want) != len(got) ||
			subtle.ConstantTimeCompare([]byte(got), []byte(want)) != 1 {
			logWarn("internal API bridge secret rejected path=%s", r.URL.Path)
			writeJSON(w, http.StatusUnauthorized, map[string]any{"detail": "unauthorized"})
			return
		}
		next(w, r)
	}
}

// handleGitOauthProviders returns the merged provider catalog for the Git OAuth
// site settings page (replaces Django GitOauthProvidersCatalogView, OPT-049).
// When company_id / X-Tenant-Id is present, the tenant Path A GitLab connection
// is merged so project-page OAuth buttons can bind that instance.
func (a *App) handleGitOauthProviders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"detail": "method not allowed"})
		return
	}
	all := a.Cfg.GetAllProviderConfigs()
	providers := make([]map[string]any, 0, len(all)+1)
	for _, pc := range all {
		providers = append(providers, providerCatalogEntry(pc))
	}
	providers = a.mergeTenantProviderCatalog(providers, catalogCompanyID(r))
	writeJSON(w, http.StatusOK, map[string]any{"providers": providers})
}

func (a *App) handleAccountsDynamic(w http.ResponseWriter, r *http.Request) {
	// /api/accounts/<service_provider>/oauth/callback/
	path := strings.TrimPrefix(r.URL.Path, "/api/accounts/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 3 && parts[1] == "oauth" && parts[2] == "callback" {
		sp := parts[0]
		if sp == "github" {
			a.handleGithubCallback(w, r)
			return
		}
		if sp == "gitlab" {
			a.handleGitlabCallback(w, r)
			return
		}
		a.handleServiceProviderCallback(w, r, sp)
		return
	}
	http.NotFound(w, r)
}

// handleGitsiteCallback — v2 浏览器回调契约路径（SSOT redirect_uri）：
// ${scheme}://${subdomains.base}/redirect/gitsite/<gitsite>/oauth/callback/。
// <gitsite> = provider target.website 主机名（如 github.com / gitlab.daydaymoney.com），
// 按 ResolveProviderByGitsite 解析出 provider 后分发到既有回调处理（state 内携带
// redirect_uri / service_provider，跨子域签名自足，见 OAuthBrowserState）。
func (a *App) handleGitsiteCallback(w http.ResponseWriter, r *http.Request) {
	// /redirect/gitsite/<gitsite>/oauth/callback/
	path := strings.TrimPrefix(r.URL.Path, "/redirect/gitsite/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 3 && parts[1] == "oauth" && parts[2] == "callback" {
		gitsite := strings.TrimSpace(strings.ToLower(parts[0]))
		pc := a.Cfg.ResolveProviderByGitsite(gitsite)
		if pc == nil {
			logWarn("Git OAuth callback unknown gitsite=%s path=%s", gitsite, r.URL.Path)
			writeJSON(w, http.StatusNotFound, map[string]any{
				"detail":  "未找到 gitsite 对应的 OAuth 配置",
				"gitsite": gitsite,
			})
			return
		}
		switch pc.Provider {
		case "github":
			a.handleGithubCallbackWithSP(w, r, pc.ServiceProvider)
		case "gitlab":
			a.handleGitlabCallbackWithSP(w, r, pc.ServiceProvider)
		default:
			writeJSON(w, http.StatusNotFound, map[string]any{
				"detail":   fmt.Sprintf("provider '%s' 暂不支持 OAuth 回调", pc.Provider),
				"provider": pc.Provider,
				"gitsite":  gitsite,
			})
		}
		return
	}
	http.NotFound(w, r)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func readJSON(r *http.Request) (map[string]any, error) {
	defer r.Body.Close()
	b, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	if len(strings.TrimSpace(string(b))) == 0 {
		return map[string]any{}, nil
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = map[string]any{}
	}
	return out, nil
}

// readJSONNumbered：同 readJSON，但数字以 json.Number 保留原文（不落 float64）。
// 雪花 ID（如 873438061961179136，> 2^53）经 float64 会丢精度，导致按 user_id
// 精确匹配查询 not_found（access-for-user 曾因此 404→not_found 链，OPT-20260807-071）。
func readJSONNumbered(r *http.Request) (map[string]any, error) {
	defer r.Body.Close()
	b, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	if len(strings.TrimSpace(string(b))) == 0 {
		return map[string]any{}, nil
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	var out map[string]any
	if err := dec.Decode(&out); err != nil {
		return nil, err
	}
	if out == nil {
		out = map[string]any{}
	}
	return out, nil
}

func (a *App) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	headerBase := strings.TrimSpace(r.Header.Get("X-GitOauth-Allowed-Host"))
	latency, err := a.DB.PingOK()
	dbCheck := map[string]any{"ok": true, "latency_ms": latency}
	ok := true
	if err != nil {
		ok = false
		dbCheck = map[string]any{"ok": false, "error": err.Error()}
	}
	payload := map[string]any{
		"service":                      "gitOauth",
		"ok":                           ok,
		"checks":                       map[string]any{"database": dbCheck},
		"gitlab_provider_config_count": len(a.Cfg.GetProviderConfigs("gitlab")),
		"public_base_url":              headerBase,
		"from_nginx_header":            headerBase != "",
	}
	if headerBase == "" {
		payload["public_base_url"] = a.Cfg.PublicBaseURL
	}
	status := http.StatusOK
	if !ok {
		status = http.StatusServiceUnavailable
	}
	writeJSON(w, status, payload)
}

func (a *App) handleSchema(w http.ResponseWriter, r *http.Request) {
	a.handleSwagger(w, r)
}

// --- helpers ---

var returnKeyRE = regexp.MustCompile(`^[0-9a-f]{32}$`)

func sanitizeAppNext(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" || len(s) > 2048 {
		return ""
	}
	if !strings.HasPrefix(s, "/") || strings.HasPrefix(s, "//") || strings.Contains(s, "://") {
		return ""
	}
	if strings.ContainsAny(s, "\n\r\x00") {
		return ""
	}
	return s
}

func sanitizeReturnKey(raw string) string {
	s := strings.TrimSpace(raw)
	if returnKeyRE.MatchString(s) {
		return s
	}
	return ""
}

func (a *App) frontendRedirect(feb, pathSuffix string) string {
	base := strings.TrimRight(strings.TrimSpace(feb), "/")
	if base == "" {
		base = a.Cfg.FrontendBase
	}
	if base == "" {
		return "/"
	}
	return base + pathSuffix
}

func (a *App) febFromRequest(r *http.Request, fallback string) string {
	origin := strings.TrimRight(strings.TrimSpace(r.Header.Get("Origin")), "/")
	allowed := map[string]bool{}
	for _, o := range a.Cfg.PublicEntryOrigins {
		allowed[o] = true
	}
	if origin != "" && allowed[origin] {
		return origin
	}
	referer := strings.TrimSpace(r.Header.Get("Referer"))
	if referer != "" {
		if u, err := url.Parse(referer); err == nil && u.Scheme != "" && u.Host != "" {
			cand := strings.TrimRight(u.Scheme+"://"+u.Host, "/")
			if allowed[cand] {
				return cand
			}
		}
	}
	return strings.TrimRight(strings.TrimSpace(fallback), "/")
}

func pathWithParamOK(path, key string) string {
	re := regexp.MustCompile(`(?:^|[?&])` + regexp.QuoteMeta(key) + `=ok(?:&|$)`)
	if re.MatchString(path) {
		return path
	}
	joiner := "?"
	if strings.Contains(path, "?") {
		joiner = "&"
	}
	return path + joiner + key + "=ok"
}

// withRequestTraceID appends trace_id from the inbound request context so the SPA
// can mount data-traceId on OAuth failure UI (meta rule 24). Omits when empty.
func withRequestTraceID(r *http.Request, pathSuffix string) string {
	if r == nil || strings.TrimSpace(pathSuffix) == "" {
		return pathSuffix
	}
	tid := tracelog.NormalizeTraceID(tracelog.CorrelationFromContext(r.Context()).TraceID)
	if tid == "" {
		return pathSuffix
	}
	u, err := url.Parse(pathSuffix)
	if err != nil || u == nil {
		joiner := "?"
		if strings.Contains(pathSuffix, "?") {
			joiner = "&"
		}
		return pathSuffix + joiner + "trace_id=" + url.QueryEscape(tid)
	}
	q := u.Query()
	if q.Get("trace_id") == "" && q.Get("traceId") == "" {
		q.Set("trace_id", tid)
		u.RawQuery = q.Encode()
	}
	return u.String()
}

func (a *App) frontendOAuthErrorRedirect(w http.ResponseWriter, r *http.Request, feb, pathSuffix string) {
	redirect(w, r, a.frontendRedirect(feb, withRequestTraceID(r, pathSuffix)))
}

func (a *App) decodeStartJWT(token, typ, audience string) (map[string]any, error) {
	secret := strings.TrimSpace(a.Cfg.BridgeJWTSecret)
	if secret == "" {
		return nil, fmt.Errorf("未配置 GITOAUTH_BRIDGE_JWT_SECRET")
	}
	claims, err := infrastructure.ParseHS256JWT(token, secret, a.Cfg.BridgeJWTIssuer, audience, typ)
	if err != nil {
		return nil, err
	}
	return claims, nil
}

func sessionGetString(sess map[string]any, key string) string {
	if sess == nil {
		return ""
	}
	v, ok := sess[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return strconv.FormatInt(int64(t), 10)
	default:
		return fmt.Sprint(t)
	}
}

func sessionGetInt(sess map[string]any, key string) int64 {
	v, ok := asInt64(sess[key])
	if !ok {
		return 0
	}
	return v
}

func redirect(w http.ResponseWriter, r *http.Request, loc string) {
	http.Redirect(w, r, loc, http.StatusFound)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func logWarn(msg string, args ...any) {
	log.Printf("[taskGitOauth] "+msg, args...)
}

func logInfo(msg string, args ...any) {
	log.Printf("[taskGitOauth] "+msg, args...)
}
