package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var gitHTTPClient = &http.Client{
	// 25s：nested-git 匿名探测 + taskGitOauth Refresh 多轮 dial 重试（边缘 IP HTTP 挂起）
	Timeout: 25 * time.Second,
	Transport: &http.Transport{
		// Bypass HTTP(S)_PROXY — git host APIs must not be hijacked by local SOCKS/dev proxies.
		Proxy: nil,
	},
}

type gitoauthTokenResponse struct {
	AccessToken string `json:"access_token"`
	Detail      string `json:"detail"`
}

func gitoauthBridgeHeaders() map[string]string {
	headers := map[string]string{
		"Content-Type": "application/json",
		"Accept":       "application/json",
	}
	if secret := strings.TrimSpace(cfg.GitoauthBridgeSecret); secret != "" {
		headers["X-GitOauth-Bridge-Secret"] = secret
	}
	return headers
}

// traceHeadersFromRequest extracts W3C traceparent + project trace headers from the
// incoming request so the outbound OAuth call joins the same trace (OPT-20260810-051).
// taskGitOauth 的 tracelog.Middleware 对仅带 X-Trace-Id 而无 X-Parent-Span-Id/traceparent
// 的请求会 400 拒绝（strict.go ErrTraceIdOnlyRejected），故必须整组透传。
func traceHeadersFromRequest(r *http.Request) map[string]string {
	if r == nil {
		return nil
	}
	headers := make(map[string]string)
	for _, h := range []string{"X-Trace-Id", "traceparent", "X-Parent-Span-Id", "tracestate"} {
		if v := strings.TrimSpace(r.Header.Get(h)); v != "" {
			headers[h] = v
		}
	}
	if len(headers) == 0 {
		return nil
	}
	return headers
}

// outboundTrace picks the first non-empty trace-header map from a variadic chain.
// 沿调用链逐层透传（处理函数 → resolve*/list* → fetchGitAccessToken），无请求上下文时为 nil。
func outboundTrace(traceHeaders ...map[string]string) map[string]string {
	for _, h := range traceHeaders {
		if len(h) > 0 {
			return h
		}
	}
	return nil
}

func fetchGitAccessToken(userID int64, repoURL, gitoauthBase string, traceHeaders ...map[string]string) (string, string) {
	base := strings.TrimRight(strings.TrimSpace(gitoauthBase), "/")
	if base == "" {
		base = strings.TrimRight(strings.TrimSpace(cfg.GitoauthBaseURL), "/")
	}
	if base == "" {
		return "", "gitOauth not configured"
	}
	if userID <= 0 {
		return "", "missing user id"
	}
	host, _ := extractHostNetloc(repoURL)
	if host == "" {
		return "", "missing repo host"
	}
	// OPT-20260822-036 / OPT-20260827-036: 统一走 gitsite 内部路径，gitOauth 按
	// repo host 解析 provider（YAML website / 租户 Path A 行），不再按 provider_key 分叉。
	endpoint := base + "/api/internal/gitsite/" + url.PathEscape(host) + "/oauth/access-for-user/"
	body, _ := json.Marshal(map[string]interface{}{
		"user_id": userID,
	})
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err.Error()
	}
	for k, v := range gitoauthBridgeHeaders() {
		req.Header.Set(k, v)
	}
	for k, v := range outboundTrace(traceHeaders...) {
		req.Header.Set(k, v)
	}

	resp, err := gitHTTPClient.Do(req)
	if err != nil {
		return "", err.Error()
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var result gitoauthTokenResponse
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &result)
	}

	if resp.StatusCode == http.StatusNotFound {
		return "", ""
	}
	if resp.StatusCode >= 400 {
		return "", sanitizeGitOauthResolveError(result.Detail, resp.StatusCode)
	}
	token := strings.TrimSpace(result.AccessToken)
	if token == "" {
		return "", "empty access token"
	}
	return token, ""
}

func sanitizeGitOauthResolveError(detail string, status int) string {
	text := strings.TrimSpace(detail)
	lower := strings.ToLower(text)
	switch {
	case text == "":
		return fmt.Sprintf("http %d", status)
	case strings.Contains(lower, "not_found"), strings.Contains(lower, "not bound"):
		return ""
	case strings.Contains(lower, "timeout"), strings.Contains(lower, "deadline"):
		// 必须先于 oauth/token：GitLab 超时文案含 /oauth/token，旧顺序会误判成令牌失效。
		return "gitOauth internal timeout"
	case strings.Contains(lower, "oauth/token"), strings.Contains(lower, "invalid_grant"),
		strings.Contains(lower, "bad request"), strings.Contains(lower, "gitlab refresh"):
		return "GitLab OAuth 令牌已失效或 refresh 失败，请在项目页点击「OAuth 授权」重新绑定"
	default:
		if len(text) > 180 {
			return text[:180]
		}
		return text
	}
}

// gitAuthFailureClass distinguishes "never bound" from refresh/network/service failures
// so UI does not tell users to bind when the binding exists but OAuth refresh timed out.
type gitAuthFailureClass int

const (
	gitAuthFailureUnbound gitAuthFailureClass = iota
	gitAuthFailureTimeout
	gitAuthFailureService
	gitAuthFailureReauth
	gitAuthFailureOther
)

func classifyGitAuthFailure(tokenErr string) gitAuthFailureClass {
	te := strings.TrimSpace(tokenErr)
	if te == "" {
		return gitAuthFailureUnbound
	}
	lower := strings.ToLower(te)
	switch {
	case strings.Contains(lower, "timeout"), strings.Contains(lower, "deadline"):
		return gitAuthFailureTimeout
	case strings.Contains(lower, "http 502"), strings.Contains(lower, "bad gateway"),
		strings.Contains(lower, "http 503"), strings.Contains(lower, "http 504"):
		return gitAuthFailureService
	case strings.Contains(lower, "invalid_grant"), strings.Contains(lower, "refresh http 400"),
		strings.Contains(lower, "refresh http 401"), strings.Contains(lower, "令牌已失效"),
		strings.Contains(lower, "授权已失效"):
		return gitAuthFailureReauth
	default:
		return gitAuthFailureOther
	}
}

// nestedAuthFailureMessage maps token resolve errors for nested-git discovery UX.
func nestedAuthFailureMessage(tokenErr string) string {
	switch classifyGitAuthFailure(tokenErr) {
	case gitAuthFailureTimeout:
		return "无法获取子 Git 仓库列表：Git 授权刷新超时（无法连接 OAuth 端点）。" +
			"公开仓库可匿名探测；私有仓库请检查网络后重试，或重新完成 Git 网站绑定。"
	case gitAuthFailureService:
		return "无法获取子 Git 仓库列表：Git 授权服务暂时不可用，请稍后重试或重新完成 Git 网站绑定。"
	case gitAuthFailureReauth:
		return "无法获取子 Git 仓库列表：Git OAuth 授权已失效，请重新绑定对应 Git 网站后再试。"
	case gitAuthFailureOther:
		return "无法获取子 Git 仓库列表：" + strings.TrimSpace(tokenErr)
	default:
		return nestedNeedsAuthMessage
	}
}

// githubBranchesAuthFailureMessage maps token resolve errors for branch listing UX.
func githubBranchesAuthFailureMessage(tokenErr string) string {
	switch classifyGitAuthFailure(tokenErr) {
	case gitAuthFailureTimeout:
		return "无法获取 GitHub 分支：Git 授权刷新超时（无法连接 OAuth 端点）。" +
			"公开仓库可匿名拉取；私有仓库请检查网络后重试，或重新完成 GitHub 绑定。"
	case gitAuthFailureService:
		return "无法获取 GitHub 分支：Git 授权服务暂时不可用，请稍后重试或重新完成 GitHub 绑定。"
	case gitAuthFailureReauth:
		return "无法获取 GitHub 分支：Git OAuth 授权已失效，请重新绑定 GitHub 后再试。"
	case gitAuthFailureOther:
		return "无法获取 GitHub 分支：" + strings.TrimSpace(tokenErr)
	default:
		return githubNeedsAuthMessage
	}
}
