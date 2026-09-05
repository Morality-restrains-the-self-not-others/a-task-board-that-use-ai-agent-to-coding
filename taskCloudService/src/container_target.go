package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"tracelog"
)

// detectStreamBackend 仅做能力探测；过长超时会吃掉 gateway 30s 预算（曾出现 7s+23s→502）。
var probeHTTP = &http.Client{Timeout: 2 * time.Second}
var localProbeHTTP = &http.Client{Timeout: 400 * time.Millisecond}
var credentialHTTP = &http.Client{Timeout: 5 * time.Second}

func normalizeURLOrigin(raw string) (origin, token string) {
	text := strings.TrimSpace(raw)
	if text == "" {
		return "", ""
	}
	if !strings.HasPrefix(text, "http://") && !strings.HasPrefix(text, "https://") {
		text = "http://" + text
	}
	u, err := url.Parse(text)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", ""
	}
	origin = strings.TrimRight(fmt.Sprintf("%s://%s", u.Scheme, u.Host), "/")
	if q := u.Query().Get("access_token"); q != "" {
		token = strings.TrimSpace(q)
	}
	if token == "" {
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		// /ui/tenant/{t}/workspace/{w}/task/{task}/{token}
		if len(parts) >= 8 &&
			parts[0] == "ui" &&
			parts[1] == "tenant" &&
			parts[3] == "workspace" &&
			parts[5] == "task" {
			token, _ = url.PathUnescape(parts[7])
			token = strings.TrimSpace(token)
		} else if len(parts) >= 2 && parts[0] == "ui" {
			token, _ = url.PathUnescape(parts[1])
			token = strings.TrimSpace(token)
		}
	}
	return origin, token
}

// scopeTokenFetch holds by-scope outcome for observability (409 body / logs).
type scopeTokenFetch struct {
	Token      string
	HTTPStatus int
	ErrorCode  string
	Detail     string
}

func fetchTokenByScope(tenantID, workspaceID, taskID, commentID string) string {
	return fetchTokenByScopeResult(tenantID, workspaceID, taskID, commentID).Token
}

func fetchTokenByScopeResult(tenantID, workspaceID, taskID, commentID string) scopeTokenFetch {
	base := strings.TrimRight(cfg.CredentialServiceURL, "/")
	if base == "" {
		return scopeTokenFetch{ErrorCode: "CREDENTIAL_SERVICE_UNCONFIGURED", Detail: "CredentialServiceURL empty"}
	}
	reqURL := base + "/v1/token/by-scope?tenant_id=" + url.QueryEscape(tenantID) +
		"&workspace_id=" + url.QueryEscape(workspaceID) +
		"&task_id=" + url.QueryEscape(taskID)
	if cid := strings.TrimSpace(commentID); cid != "" {
		reqURL += "&comment_id=" + url.QueryEscape(cid)
	}
	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return scopeTokenFetch{ErrorCode: "CREDENTIAL_FETCH_FAILED", Detail: err.Error()}
	}
	req.Header.Set("Accept", "application/json")
	resp, err := credentialHTTP.Do(req)
	if err != nil {
		return scopeTokenFetch{ErrorCode: "CREDENTIAL_FETCH_FAILED", Detail: err.Error()}
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var data map[string]interface{}
	_ = json.Unmarshal(raw, &data)
	detail := strings.TrimSpace(fmt.Sprintf("%v", data["detail"]))
	if detail == "<nil>" {
		detail = ""
	}
	errCode := strings.TrimSpace(fmt.Sprintf("%v", data["error_code"]))
	if errCode == "<nil>" {
		errCode = ""
	}
	if resp.StatusCode != http.StatusOK {
		if errCode == "" {
			errCode = "CREDENTIAL_BY_SCOPE_HTTP_ERROR"
		}
		if detail == "" {
			detail = fmt.Sprintf("by-scope HTTP %d", resp.StatusCode)
		}
		return scopeTokenFetch{HTTPStatus: resp.StatusCode, ErrorCode: errCode, Detail: detail}
	}
	tok := strings.TrimSpace(fmt.Sprintf("%v", data["access_token"]))
	if tok == "" || tok == "<nil>" {
		return scopeTokenFetch{
			HTTPStatus: resp.StatusCode,
			ErrorCode:  "EMPTY_ACCESS_TOKEN",
			Detail:     "by-scope 200 but access_token empty",
		}
	}
	return scopeTokenFetch{Token: tok, HTTPStatus: resp.StatusCode}
}

// resolveContainerTarget resolves base URL + access token for SaaS→container forwarding.
func resolveContainerTarget(cfg *CloudServerConfig, overridePageURL string) (baseURL, token string) {
	baseURL, token, _, _, _ = resolveContainerTargetDiag(cfg, overridePageURL)
	return baseURL, token
}

// resolveContainerTargetDiag is resolveContainerTarget plus by-scope failure diagnostics.
func resolveContainerTargetDiag(cfg *CloudServerConfig, overridePageURL string) (baseURL, token, tokenErrCode, tokenErrDetail string, credentialStatus int) {
	if cfg == nil {
		return "", "", "MISSING_CLOUD_CONFIG", "cloud server config nil", 0
	}
	overrideOrigin, overrideToken := normalizeURLOrigin(overridePageURL)
	serverOrigin, serverToken := normalizeURLOrigin(cfg.ServerURL)
	bizOrigin, bizToken := normalizeURLOrigin(cfg.BusinessAPIEndpoint)
	baseURL = overrideOrigin
	if baseURL == "" {
		baseURL = serverOrigin
	}
	if baseURL == "" {
		baseURL = bizOrigin
	}
	token = overrideToken
	if token == "" {
		token = serverToken
	}
	if token == "" {
		token = bizToken
	}
	if token == "" && cfg.CompanyID != "" && cfg.TaskID != "" {
		scope := fetchTokenByScopeResult(cfg.CompanyID, cfg.WorkspaceID, cfg.TaskID, cfg.CommentID)
		token = scope.Token
		credentialStatus = scope.HTTPStatus
		if token == "" {
			tokenErrCode = scope.ErrorCode
			tokenErrDetail = scope.Detail
			if tokenErrCode == "" {
				tokenErrCode = "MISSING_ACCESS_TOKEN"
			}
		}
	} else if token == "" {
		tokenErrCode = "MISSING_ACCESS_TOKEN"
		tokenErrDetail = "no token in server_url/business_api_endpoint and by-scope not attempted"
	}
	// relay/host-network 同机：若登记了不可达公网 IP，而本机 :port 已监听，改走 loopback。
	if overrideOrigin == "" {
		baseURL = preferLoopbackContainerBaseURL(baseURL)
	}
	return baseURL, token, tokenErrCode, tokenErrDetail, credentialStatus
}

// preferLoopbackContainerBaseURL rewrites a non-loopback http(s) origin to 127.0.0.1
// when the local same-port health endpoint responds quickly (relay colocated SaaS).
func preferLoopbackContainerBaseURL(baseURL string) string {
	origin := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if origin == "" {
		return baseURL
	}
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		return baseURL
	}
	host := strings.ToLower(u.Hostname())
	if host == "127.0.0.1" || host == "localhost" || host == "::1" {
		return baseURL
	}
	port := u.Port()
	if port == "" {
		if u.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	localOrigin := fmt.Sprintf("http://127.0.0.1:%s", port)
	req, err := http.NewRequest(http.MethodGet, localOrigin+"/api/health/", nil)
	if err != nil {
		return baseURL
	}
	resp, err := localProbeHTTP.Do(req)
	if err != nil {
		return baseURL
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 500 {
		return localOrigin
	}
	return baseURL
}

// probeCloneDone 探测容器 /api/requirements/task-gate 的 clone_done 标记。
// 克隆未完成（clone_done=false）、server_url/凭证缺失或探测失败均返回 false，
// 供指令闲置回收跳过仍处于克隆期的机器（OPT-20260825-001）。
func probeCloneDone(ctx context.Context, cfg *CloudServerConfig) bool {
	if cfg == nil {
		return false
	}
	baseURL, token := resolveContainerTarget(cfg, "")
	if strings.TrimSpace(baseURL) == "" || strings.TrimSpace(token) == "" {
		return false
	}
	req, err := http.NewRequest(http.MethodGet, strings.TrimRight(baseURL, "/")+"/api/requirements/task-gate", nil)
	if err != nil {
		return false
	}
	req.Header.Set("X-Access-Token", token)
	req.Header.Set("Accept", "application/json")
	tracelog.ApplyOutboundHeaders(req, ctx)
	resp, err := probeHTTP.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false
	}
	raw, _ := io.ReadAll(resp.Body)
	var data map[string]interface{}
	if json.Unmarshal(raw, &data) != nil {
		return false
	}
	done, _ := data["clone_done"].(bool)
	return done
}

func detectStreamBackend(baseURL, token, traceID string) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	tok := strings.TrimSpace(token)
	if base == "" || tok == "" {
		return "legacy"
	}
	req, err := http.NewRequest(http.MethodGet, base+"/api/requirements/task-gate", nil)
	if err != nil {
		return "legacy"
	}
	req.Header.Set("X-Access-Token", tok)
	req.Header.Set("Accept", "application/json")
	tracelog.ApplyOutboundHeaders(req, context.Background())
	resp, err := probeHTTP.Do(req)
	if err != nil {
		return "legacy"
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "legacy"
	}
	raw, _ := io.ReadAll(resp.Body)
	var data map[string]interface{}
	if json.Unmarshal(raw, &data) != nil {
		return "legacy"
	}
	if _, ok := data["clone_done"]; ok {
		return "trae"
	}
	return "legacy"
}
