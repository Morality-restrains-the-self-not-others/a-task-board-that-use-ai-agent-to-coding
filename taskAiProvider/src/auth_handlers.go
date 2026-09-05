package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"taskAiProvider/domain"
	"taskAiProvider/infrastructure"

	"tracelog"
)

var ssoOnlyBody = map[string]any{
	"detail": "镜像市场已关闭本地注册与密码登录，请通过主站单点登录进入。 超级管理员：主站系统管理「容器镜像列表」页内「镜像市场管理（SSO）」； 厂商：在厂商门户申请认证通过后，使用「打开主站 SSO」（需先绑定邮箱账号，且主站邮箱与厂商档案一致；若运营关闭厂商申请审核则可直接进入）。 新厂商档案由运营维护或审核关闭时自动建档。",
	"code":   "sso_only",
}

func (a *App) handleSSOOnly(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusForbidden, ssoOnlyBody)
}

func (a *App) handleSSOExchange(w http.ResponseWriter, r *http.Request) {
	var bridge string
	isGET := r.Method == http.MethodGet

	if r.Method == http.MethodPost {
		var body map[string]any
		if err := readJSON(r, &body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "无效 JSON"})
			return
		}
		bridge, _ = body["bridge"].(string)
	} else if isGET {
		// Browser redirect from SSO bridge (taskAuth sso_bridge.go)
		bridge = strings.TrimSpace(r.URL.Query().Get("bridge"))
	} else {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"detail": "method not allowed"})
		return
	}

	bridge = strings.TrimSpace(bridge)
	if bridge == "" {
		if isGET {
			http.Redirect(w, r, "/?error="+url.QueryEscape("缺少 bridge"), http.StatusFound)
			return
		}
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "缺少 bridge"})
		return
	}
	claims, err := infrastructure.ParseHS256JWT(bridge, a.Cfg.SSOJwtSecret, a.Cfg.SSOJwtIssuer, a.Cfg.SSOAudience, "")
	if err != nil {
		// OPT-20260806-061: 失败路径必须留痕 — 此前仅重定向 /?error=无效的 bridge，
		// 排障时日志全空只能静态分析（bad signature / bad iss / bad aud / typ 缺失）。
		logWarn(r.Context(), "event=SSOBridgeVerifyFailed err=%v iss=%s aud=%s", err, a.Cfg.SSOJwtIssuer, a.Cfg.SSOAudience)
		if strings.Contains(err.Error(), "expired") {
			if isGET {
				http.Redirect(w, r, "/?error="+url.QueryEscape("bridge 已过期"), http.StatusFound)
				return
			}
			writeJSON(w, http.StatusUnauthorized, map[string]any{"detail": "bridge 已过期"})
			return
		}
		if isGET {
			http.Redirect(w, r, "/?error="+url.QueryEscape("无效的 bridge"), http.StatusFound)
			return
		}
		writeJSON(w, http.StatusUnauthorized, map[string]any{"detail": "无效的 bridge"})
		return
	}
	if infrastructure.ClaimString(claims, "typ") == "" {
		// typ 缺失单独留痕（ParseHS256JWT 不校验 typ，需在业务层捕获）
		logWarn(r.Context(), "event=SSOBridgeVerifyFailed reason=missing_typ sub=%s", infrastructure.ClaimString(claims, "sub"))
		if isGET {
			http.Redirect(w, r, "/?error="+url.QueryEscape("无效的 bridge"), http.StatusFound)
			return
		}
		writeJSON(w, http.StatusUnauthorized, map[string]any{"detail": "无效的 bridge"})
		return
	}
	access, role, err := infrastructure.ExchangeBridge(a.Cfg, a.DB, claims)
	if err != nil {
		logWarn(r.Context(), "event=SSOExchangeFailed err=%v", err)
		if isGET {
			http.Redirect(w, r, "/?error="+url.QueryEscape(err.Error()), http.StatusFound)
			return
		}
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": err.Error()})
		return
	}
	logInfo("event=SSOBridgeExchanged role=%s", role)

	if isGET {
		// Browser redirect: deliver token via URL fragment (same pattern as OIDC callback)
		frontend := "/#token=" + access
		if role == "staff" {
			frontend = "/admin#token=" + access
		}
		http.Redirect(w, r, frontend, http.StatusFound)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"access": access, "role": role})
}

func (a *App) requireVendor(w http.ResponseWriter, r *http.Request) (*domain.Vendor, bool) {
	tok := bearerToken(r)
	if tok == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"detail": "未认证"})
		return nil, false
	}
	claims, err := infrastructure.ParseHS256JWT(tok, a.Cfg.SecretKey, infrastructure.JWTIssuer, "", "vendor")
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"detail": "无效或过期的令牌"})
		return nil, false
	}
	sub, _ := strconv.ParseInt(infrastructure.ClaimString(claims, "sub"), 10, 64)
	v, err := a.DB.GetVendorByID(sub)
	if err != nil || v == nil || !v.IsActive {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"detail": "厂商不存在或已禁用"})
		return nil, false
	}
	return v, true
}

func (a *App) requireStaff(w http.ResponseWriter, r *http.Request) (*domain.Staff, bool) {
	tok := bearerToken(r)
	if tok == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"detail": "未认证"})
		return nil, false
	}
	claims, err := infrastructure.ParseHS256JWT(tok, a.Cfg.SecretKey, infrastructure.JWTIssuer, "", "staff")
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"detail": "无效或过期的令牌"})
		return nil, false
	}
	sub, _ := strconv.ParseInt(infrastructure.ClaimString(claims, "sub"), 10, 64)
	s, err := a.DB.GetStaffByID(sub)
	if err != nil || s == nil || !s.IsActive {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"detail": "管理员不存在或已禁用"})
		return nil, false
	}
	return s, true
}

// handleVendorStatus 厂商门户/镜像市场资格四态（OPT-20260806-065）。
// 资格来源：ai_provider_vendor —— 按 saas_user_id 绑定（网关 X-User-Id，或
// provider 同源 Cookie → taskAuth forward-auth）或主邮箱匹配（X-User-Email）。
// 未登录返回 401；DB 故障降级为 none 态并记录日志。
func (a *App) handleVendorStatus(w http.ResponseWriter, r *http.Request) {
	applicant := a.resolveSaasApplicant(r)
	if applicant == nil || applicant.UID <= 0 {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"detail": "未认证"})
		return
	}
	var vendor *domain.Vendor
	v, err := a.DB.GetVendorBySaasUserID(applicant.UID)
	if err != nil && err != sql.ErrNoRows {
		logWarn(r.Context(), "vendor-status saas_user_id lookup failed: %v", err)
	}
	vendor = v
	if vendor == nil {
		if email := applicant.Email; email != "" {
			v, err := a.DB.GetVendorByEmail(email)
			if err != nil && err != sql.ErrNoRows {
				logWarn(r.Context(), "vendor-status email lookup failed: %v", err)
			}
			vendor = v
		}
	}
	status := vendorStatusOf(vendor)
	reviewEnabled := true
	if enabled, err := a.DB.GetMarketplaceSettings(); err == nil {
		reviewEnabled = enabled
	} else {
		logWarn(r.Context(), "vendor-status marketplace settings load failed: %v", err)
	}
	out := map[string]any{
		"is_vendor":                         status == "qualified",
		"status":                            status,
		"has_email":                         applicantHasBindableEmail(applicant.Email),
		"vendor_application_review_enabled": reviewEnabled,
		"vendor":                            nil,
	}
	if vendor != nil {
		out["vendor"] = vendorJSON(vendor)
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *App) handleVendorMe(w http.ResponseWriter, r *http.Request) {
	v, ok := a.requireVendor(w, r)
	if !ok {
		return
	}
	m := map[string]any{
		"id": infrastructure.IDStr(v.ID), "email": v.Email, "company_name": v.CompanyName,
		"contact_name": v.ContactName, "is_active": v.IsActive,
		// OPT-20260806-064: 微信用户合成邮箱（@sso.invalid 永不可投递）标记，前端展示占位。
		"is_synthetic_email": isSyntheticEmail(v.Email),
	}
	if v.SaasUserID != nil {
		m["saas_user_id"] = infrastructure.IDStr(*v.SaasUserID)
	} else {
		m["saas_user_id"] = nil
	}
	writeJSON(w, http.StatusOK, m)
}

func (a *App) handleStaffMe(w http.ResponseWriter, r *http.Request) {
	s, ok := a.requireStaff(w, r)
	if !ok {
		return
	}
	m := map[string]any{
		"id": infrastructure.IDStr(s.ID), "username": s.Username, "display_name": s.DisplayName, "is_active": s.IsActive,
	}
	if s.SaasSuperadminID != nil {
		m["saas_superadmin_id"] = infrastructure.IDStr(*s.SaasSuperadminID)
	} else {
		m["saas_superadmin_id"] = nil
	}
	writeJSON(w, http.StatusOK, m)
}

type oidcStore struct {
	mu   sync.Mutex
	data map[string]oidcSession
}

type oidcSession struct {
	CodeVerifier string
	Role         string
	Expires      time.Time
}

func newOIDCStore() *oidcStore {
	return &oidcStore{data: map[string]oidcSession{}}
}

func (s *oidcStore) put(state string, sess oidcSession) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[state] = sess
}

func (s *oidcStore) take(state string) (oidcSession, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.data[state]
	if ok {
		delete(s.data, state)
	}
	if ok && time.Now().After(sess.Expires) {
		return oidcSession{}, false
	}
	return sess, ok
}

func randomB64(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func (a *App) oidcRedirectURI(r *http.Request) string {
	if a.Cfg.OIDCPublicOrigin != "" {
		return strings.TrimRight(a.Cfg.OIDCPublicOrigin, "/") + "/api/auth/oidc/callback/"
	}
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	host := r.Host
	if fh := r.Header.Get("X-Forwarded-Host"); fh != "" {
		host = fh
	}
	return scheme + "://" + host + "/api/auth/oidc/callback/"
}

func (a *App) handleOIDCAuthorize(w http.ResponseWriter, r *http.Request) {
	role := strings.TrimSpace(r.URL.Query().Get("role"))
	if role != "vendor" && role != "admin" {
		writeJSON(w, 400, map[string]any{"detail": fmt.Sprintf("无效的 role: %s，仅支持 vendor 或 admin", role)})
		return
	}
	state := randomB64(24)
	verifier := randomB64(32)
	nonce := randomB64(16)
	a.OIDC.put(state, oidcSession{CodeVerifier: verifier, Role: role, Expires: time.Now().Add(10 * time.Minute)})

	q := url.Values{}
	q.Set("response_type", "code")
	q.Set("client_id", a.Cfg.OIDCClientID)
	q.Set("redirect_uri", a.oidcRedirectURI(r))
	q.Set("scope", strings.Join(a.Cfg.OIDCScopes, " "))
	q.Set("state", state)
	q.Set("nonce", nonce)
	q.Set("code_challenge", pkceChallenge(verifier))
	q.Set("code_challenge_method", "S256")
	authURL := a.Cfg.OIDCRpIssuer + "/api/oidc/authorize?" + q.Encode()
	logInfo("OIDC authorize role=%s", role)
	http.Redirect(w, r, authURL, http.StatusFound)
}

func (a *App) handleOIDCCallback(w http.ResponseWriter, r *http.Request) {
	if errParam := r.URL.Query().Get("error"); errParam != "" {
		http.Redirect(w, r, "/?error="+url.QueryEscape(errParam), http.StatusFound)
		return
	}
	code := strings.TrimSpace(r.URL.Query().Get("code"))
	state := strings.TrimSpace(r.URL.Query().Get("state"))
	if code == "" {
		writeJSON(w, 400, map[string]any{"detail": "缺少 authorization code"})
		return
	}
	if state == "" {
		writeJSON(w, 400, map[string]any{"detail": "缺少 state 参数"})
		return
	}
	sess, ok := a.OIDC.take(state)
	if !ok {
		writeJSON(w, 400, map[string]any{"detail": "OIDC 会话已过期，请重新发起登录"})
		return
	}

	tokenURL := a.Cfg.OIDCRpIssuer + "/api/oidc/token"
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", a.oidcRedirectURI(r))
	form.Set("client_id", a.Cfg.OIDCClientID)
	form.Set("client_secret", a.Cfg.OIDCClientSecret)
	form.Set("code_verifier", sess.CodeVerifier)

	req, _ := http.NewRequest(http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := tracelog.DirectClient(30 * time.Second)
	resp, err := client.Do(req)
	if err != nil {
		logWarn(r.Context(), "OIDC token exchange failed: %v", err)
		logInfo("event=OidcLoginFailed role=%s", sess.Role)
		writeJSON(w, 400, map[string]any{"detail": "身份认证服务暂时不可用，请稍后重试"})
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		logWarn(r.Context(), "OIDC token status=%d body=%s", resp.StatusCode, string(body))
		logInfo("event=OidcLoginFailed role=%s", sess.Role)
		writeJSON(w, 400, map[string]any{"detail": "身份认证服务暂时不可用，请稍后重试"})
		return
	}
	var tokResp map[string]any
	_ = json.Unmarshal(body, &tokResp)
	idToken, _ := tokResp["id_token"].(string)
	if idToken == "" {
		logInfo("event=OidcLoginFailed role=%s", sess.Role)
		http.Redirect(w, r, "/?error="+url.QueryEscape("缺少 id_token"), http.StatusFound)
		return
	}
	claims, err := infrastructure.ValidateIDToken(idToken, a.JWKS, a.Cfg.OIDCRpIssuer, a.Cfg.OIDCClientID)
	if err != nil {
		logWarn(r.Context(), "OIDC id_token verify failed: %v", err)
		logInfo("event=OidcLoginFailed role=%s", sess.Role)
		http.Redirect(w, r, "/?error="+url.QueryEscape(err.Error()), http.StatusFound)
		return
	}
	email := strings.ToLower(strings.TrimSpace(fmt.Sprint(claims["email"])))
	sub := strings.TrimSpace(fmt.Sprint(claims["sub"]))

	var access string
	var frontend string
	if sess.Role == "vendor" {
		v, err := a.DB.GetVendorByEmail(email)
		if err != nil || !v.IsActive {
			logInfo("event=OidcLoginFailed role=vendor")
			http.Redirect(w, r, "/?error="+url.QueryEscape("未找到厂商账号"), http.StatusFound)
			return
		}
		if sub != "" {
			if sid, e := strconv.ParseInt(sub, 10, 64); e == nil && v.SaasUserID == nil {
				_ = a.DB.BindVendorSaasUser(v.ID, sid)
			}
		}
		access, err = infrastructure.IssueToken(a.Cfg.SecretKey, infrastructure.IDStr(v.ID), "vendor", a.Cfg.JWTTTLSeconds)
		if err != nil {
			writeJSON(w, 500, map[string]any{"detail": "签发失败"})
			return
		}
		frontend = "/#token=" + access
		logInfo("event=VendorLoggedInViaOidc vendor_id=%d", v.ID)
	} else {
		// staff: match by saas_superadmin_id or username/email hint
		var staff *domain.Staff
		if sid, e := strconv.ParseInt(sub, 10, 64); e == nil {
			staff, _ = a.DB.GetStaffBySaasSuperadminID(sid)
		}
		if staff == nil {
			uname := strings.TrimSpace(fmt.Sprint(claims["preferred_username"]))
			if uname == "" {
				uname = email
			}
			if uname == "" {
				uname = sub
			}
			display := strings.TrimSpace(fmt.Sprint(claims["name"]))
			if display == "" {
				display = uname
			}
			saasID, _ := strconv.ParseInt(sub, 10, 64)
			picked, _ := infrastructure.PickStaffUsername(a.DB, saasID, uname)
			staff, err = a.DB.UpsertStaffFromBridge(saasID, picked, display)
			if err != nil {
				http.Redirect(w, r, "/?error="+url.QueryEscape(err.Error()), http.StatusFound)
				return
			}
		}
		access, err = infrastructure.IssueToken(a.Cfg.SecretKey, infrastructure.IDStr(staff.ID), "staff", a.Cfg.JWTTTLSeconds)
		if err != nil {
			writeJSON(w, 500, map[string]any{"detail": "签发失败"})
			return
		}
		frontend = "/admin#token=" + access
		logInfo("event=StaffLoggedInViaOidc staff_id=%d", staff.ID)
	}
	http.Redirect(w, r, frontend, http.StatusFound)
}
