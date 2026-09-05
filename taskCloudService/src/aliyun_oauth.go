package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ── Aliyun OAuth 2.0 (RAM) integration ──────────────────────────────────────
//
// OAuth flow:
//   1. GET /api/cloud/oauth/aliyun/login → returns auth_url, frontend redirects user
//   2. Aliyun redirects to /callback/cloudplatform/oauth2.0/aliyun?code=...&state=...
//   3. Go exchanges code for token, stores in cloud_oauth_tokens table, redirects to frontend

const aliyunOAuthAuthorizeURL = "https://signin.aliyun.com/oauth2/v1/auth"
const aliyunOAuthTokenURL = "https://oauth.aliyuncs.com/v1/token"

func genOAuthState(tenantID string) (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return tenantID + "_" + hex.EncodeToString(b), nil
}

func parseTenantFromState(state string) string {
	idx := strings.Index(state, "_")
	if idx < 0 {
		return ""
	}
	return state[:idx]
}

func buildAliyunOAuthLoginURL(state string) string {
	u, _ := url.Parse(aliyunOAuthAuthorizeURL)
	q := u.Query()
	q.Set("client_id", cfg.AliyunOAuthClientID)
	q.Set("redirect_uri", cfg.AliyunOAuthRedirectURI)
	q.Set("response_type", "code")
	q.Set("scope", "all") // RAM full access scope
	q.Set("state", state)
	q.Set("access_type", "offline")
	u.RawQuery = q.Encode()
	return u.String()
}

type aliyunTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	Scope        string `json:"scope"`
	Error        string `json:"error"`
	ErrorDesc    string `json:"error_description"`
}

func exchangeAliyunOAuthCode(code string) (*aliyunTokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("client_id", cfg.AliyunOAuthClientID)
	data.Set("client_secret", cfg.AliyunOAuthClientSecret)
	data.Set("redirect_uri", cfg.AliyunOAuthRedirectURI)

	req, err := http.NewRequest("POST", aliyunOAuthTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token exchange request failed: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read token response: %w", err)
	}

	var tr aliyunTokenResponse
	if err := json.Unmarshal(raw, &tr); err != nil {
		return nil, fmt.Errorf("parse token response: %w (body=%s)", err, string(raw))
	}
	if tr.Error != "" {
		return nil, fmt.Errorf("aliyun oauth error: %s (%s)", tr.Error, tr.ErrorDesc)
	}
	return &tr, nil
}

func storeAliyunOAuthToken(tenantID, accessToken, refreshToken, scope string, expiresIn int) (string, error) {
	id := genID("oat")
	var expiresAt interface{}
	if expiresIn > 0 {
		expiresAt = time.Now().Add(time.Duration(expiresIn) * time.Second)
	}
	_, err := db.Exec(
		`INSERT INTO cloud_oauth_tokens(id,company_id,platform_type,authorization_type,access_token,refresh_token,scope,expires_at)
		VALUES(?,?,?,?,?,?,?,?)`,
		id, tenantID, "aliyun", "oauth", accessToken, refreshToken, scope, expiresAt,
	)
	if err != nil {
		return "", fmt.Errorf("store oauth token: %w", err)
	}
	return id, nil
}

// ── Handlers ────────────────────────────────────────────────────────────────

func handleAliyunOAuthLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, r, 405, "method not allowed")
		return
	}
	if cfg.AliyunOAuthClientID == "" {
		writeJSON(w, 500, map[string]string{"error": "阿里云 OAuth 未配置 (ALIYUN_OAUTH_CLIENT_ID)", "trace_id": traceIDFromRequest(r)})
		return
	}

	tenantID := resolveGlobalCloudTenantID(r)
	if tenantID == "" {
		// Try query param for initial OAuth setup
		tenantID = strings.TrimSpace(r.URL.Query().Get("tenant_id"))
	}
	if tenantID == "" {
		cloudQueryError(w, 400, "无法解析租户")
		return
	}
	if !ensureTenantMember(w, r, tenantID) {
		return
	}

	state, err := genOAuthState(tenantID)
	if err != nil {
		cloudQueryError(w, 500, "生成 OAuth state 失败")
		return
	}
	authURL := buildAliyunOAuthLoginURL(state)
	writeJSON(w, 200, map[string]string{"auth_url": authURL})
}

func handleAliyunOAuthCallback(w http.ResponseWriter, r *http.Request) {
	code := strings.TrimSpace(r.URL.Query().Get("code"))
	if code == "" {
		writeErrorJSON(w, r, 400, "missing code")
		return
	}
	state := strings.TrimSpace(r.URL.Query().Get("state"))
	tenantID := parseTenantFromState(state)

	if cfg.AliyunOAuthClientID == "" || cfg.AliyunOAuthClientSecret == "" {
		logInfo("aliyun oauth callback: config missing (ALIYUN_OAUTH_CLIENT_ID/SECRET)", r.Header.Get("X-Trace-Id"))
		writeJSON(w, 500, map[string]string{"error": "阿里云 OAuth 未配置", "trace_id": traceIDFromRequest(r)})
		return
	}

	token, err := exchangeAliyunOAuthCode(code)
	if err != nil {
		logInfo("aliyun oauth token exchange failed: "+err.Error(), r.Header.Get("X-Trace-Id"))
		writeJSON(w, 500, map[string]string{"error": "Token 交换失败: " + err.Error(), "trace_id": traceIDFromRequest(r)})
		return
	}

	if tenantID == "" {
		logInfo("aliyun oauth: cannot resolve tenant from state="+state, r.Header.Get("X-Trace-Id"))
		// Try to redirect with token stored temporarily; frontend will resolve tenant
		writeJSON(w, 200, map[string]interface{}{
			"status":        "ok",
			"access_token":  maskCredential(token.AccessToken),
			"expires_in":    token.ExpiresIn,
			"message":       "token obtained, please provide tenant_id to complete authorization",
			"requires_tenant": true,
		})
		return
	}

	tokenID, err := storeAliyunOAuthToken(tenantID, token.AccessToken, token.RefreshToken, token.Scope, token.ExpiresIn)
	if err != nil {
		logInfo("aliyun oauth store token failed: "+err.Error(), r.Header.Get("X-Trace-Id"))
		writeJSON(w, 500, map[string]string{"error": "存储 Token 失败: " + err.Error(), "trace_id": traceIDFromRequest(r)})
		return
	}

	logInfo("aliyun oauth token stored: "+tokenID+" tenant="+tenantID, r.Header.Get("X-Trace-Id"))
	writeJSON(w, 200, map[string]interface{}{
		"status":    "ok",
		"token_id":  tokenID,
		"tenant_id": tenantID,
		"message":   "阿里云 OAuth 授权成功",
	})
}
