package infrastructure

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	githubTokenURL   = "https://github.com/login/oauth/access_token"
	githubUserAPI    = "https://api.github.com/user"
	oauthHTTPTimeout = 60 * time.Second
)

// GitHubExchangeRejectedError — GitHub 明确拒绝换票（HTTP 4xx，如 code 已过期、
// redirect_uri 与 App 白名单不匹配）。区别于网络类失败（exchange_failed）：
// 前端据此展示「重新授权」类提示而非「网络问题」提示。
type GitHubExchangeRejectedError struct {
	StatusCode int
	Body       string
}

func (e *GitHubExchangeRejectedError) Error() string {
	return fmt.Sprintf("github token http %d", e.StatusCode)
}

// GitLabExchangeRejectedError — GitLab 明确拒绝换票（HTTP 4xx，或 200 但 body 携带 error，
// 如 code 已过期 invalid_grant / redirect_uri 不匹配）。区别于网络类失败（exchange_failed）：
// 前端据此展示「重新授权」类提示而非「网络问题」提示。
type GitLabExchangeRejectedError struct {
	StatusCode int
	Body       string
}

func (e *GitLabExchangeRejectedError) Error() string {
	return fmt.Sprintf("gitlab token http %d", e.StatusCode)
}

// Outbound OAuth HTTP must not inherit shell HTTP(S)_PROXY (dead local SOCKS).
// Dial/header timeouts reduce hangs that surface as github=exchange_failed.
// Optional conf outbound_proxy is tried only after direct network failure.
var httpClient = newOAuthHTTPClient()

func ExchangeGitHubCode(cfg *Config, code, redirectURI string) (map[string]any, error) {
	cid := cfg.GithubClientID
	sec := cfg.GithubClientSecret
	uri := strings.TrimSpace(redirectURI)
	if uri == "" {
		uri = cfg.GithubRedirectURI
	}
	if cid == "" || sec == "" {
		return nil, fmt.Errorf("未配置 GITHUB_APP_CLIENT_ID / GITHUB_APP_CLIENT_SECRET")
	}
	if uri == "" {
		return nil, fmt.Errorf("未配置 GITHUB_APP_REDIRECT_URI")
	}
	form := url.Values{}
	form.Set("client_id", cid)
	form.Set("client_secret", sec)
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", uri)
	body := form.Encode()
	clients := oauthHTTPClients(cfg)
	var lastErr error
	for ci, client := range clients {
		attempts := 1
		if ci == 0 && len(clients) == 1 {
			// 无 proxy：多轮直连重试，配合 dial demote 绕过「TCP 通但 HTTP 挂起」边缘。
			attempts = 4
		} else if ci == 0 {
			attempts = 2
		}
		for attempt := 0; attempt < attempts; attempt++ {
			if attempt > 0 || ci > 0 {
				time.Sleep(300 * time.Millisecond)
			}
			if ci > 0 && attempt == 0 {
				log.Printf("[taskGitOauth] GitHub 换票直连失败，回退 outbound_proxy: %v", lastErr)
			}
			req, err := http.NewRequest(http.MethodPost, githubTokenURL, strings.NewReader(body))
			if err != nil {
				return nil, err
			}
			req.Header.Set("Accept", "application/json")
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			resp, err := client.Do(req)
			if err != nil {
				if lastErr == nil {
					lastErr = err // 保留首个错误（直连优先），不被回退代理错误覆盖
				}
				ReportGithubDialHTTPFailure()
				if isRetryableNetErr(err) && (attempt+1 < attempts || ci+1 < len(clients)) {
					continue
				}
				return nil, lastErr
			}
			raw, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if resp.StatusCode >= 400 {
				return nil, &GitHubExchangeRejectedError{StatusCode: resp.StatusCode, Body: string(raw)}
			}
			out, err := parseGitHubTokenBody(raw, resp.Header.Get("Content-Type"))
			if err != nil {
				// HTTP 200 但 GitHub 在 body 中拒绝（bad_verification_code / redirect_uri_mismatch 等）
				return nil, &GitHubExchangeRejectedError{StatusCode: resp.StatusCode, Body: string(raw)}
			}
			return out, nil
		}
	}
	return nil, lastErr
}

func isRetryableNetErr(err error) bool {
	if err == nil {
		return false
	}
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return true
	}
	msg := strings.ToLower(err.Error())
	for _, needle := range []string{
		"timeout", "deadline exceeded", "temporarily unavailable",
		"connection reset", "connection refused", "tls handshake timeout",
		"i/o timeout", "unexpected eof",
	} {
		if strings.Contains(msg, needle) {
			return true
		}
	}
	return false
}

func RefreshGitHubToken(cfg *Config, refreshToken string) (map[string]any, error) {
	cid := cfg.GithubClientID
	sec := cfg.GithubClientSecret
	if cid == "" || sec == "" {
		return nil, fmt.Errorf("未配置 GITHUB_APP_CLIENT_ID / GITHUB_APP_CLIENT_SECRET")
	}
	form := url.Values{}
	form.Set("client_id", cid)
	form.Set("client_secret", sec)
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)
	body := form.Encode()
	clients := oauthHTTPClients(cfg)
	var lastErr error
	for ci, client := range clients {
		// 与 ExchangeGitHubCode 对齐：无 proxy 时直连重试，HTTP 挂起后轮转 dial IP。
		// access-for-user 只走 Refresh；旧实现单次失败会误杀 nested-git / auto_run。
		attempts := 1
		if ci == 0 && len(clients) == 1 {
			attempts = 4 // 与 Exchange 对齐：无 proxy 时靠 demote + 多轮绕过挂起边缘
		} else if ci == 0 {
			attempts = 2
		}
		for attempt := 0; attempt < attempts; attempt++ {
			if attempt > 0 || ci > 0 {
				time.Sleep(300 * time.Millisecond)
			}
			if ci > 0 && attempt == 0 {
				log.Printf("[taskGitOauth] GitHub refresh 直连失败，回退 outbound_proxy: %v", lastErr)
			}
			req, err := http.NewRequest(http.MethodPost, githubTokenURL, strings.NewReader(body))
			if err != nil {
				return nil, err
			}
			req.Header.Set("Accept", "application/json")
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			resp, err := client.Do(req)
			if err != nil {
				if lastErr == nil {
					lastErr = err
				}
				ReportGithubDialHTTPFailure()
				if isRetryableNetErr(err) && (attempt+1 < attempts || ci+1 < len(clients)) {
					continue
				}
				return nil, lastErr
			}
			raw, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if resp.StatusCode >= 400 {
				return nil, fmt.Errorf("github refresh http %d", resp.StatusCode)
			}
			return parseGitHubTokenBody(raw, resp.Header.Get("Content-Type"))
		}
	}
	return nil, lastErr
}

func FetchGitHubProfile(cfg *Config, accessToken string) (map[string]any, error) {
	clients := oauthHTTPClients(cfg)
	var lastErr error
	for ci, client := range clients {
		// 与 RefreshGitHubToken 对齐（OPT-20260817-037）：无 proxy 时靠多轮直连重试绕过
		// 挂起边缘 IP；资料拉取失败会阻断 OAuth 回调落库/展示，与 refresh 同类脆弱点。
		attempts := 1
		if ci == 0 && len(clients) == 1 {
			attempts = 4
		} else if ci == 0 {
			attempts = 2
		}
		for attempt := 0; attempt < attempts; attempt++ {
			if attempt > 0 || ci > 0 {
				time.Sleep(300 * time.Millisecond)
			}
			req, err := http.NewRequest(http.MethodGet, githubUserAPI, nil)
			if err != nil {
				return nil, err
			}
			req.Header.Set("Accept", "application/vnd.github+json")
			req.Header.Set("Authorization", "Bearer "+accessToken)
			req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
			resp, err := client.Do(req)
			if err != nil {
				if lastErr == nil {
					lastErr = err
				}
				ReportGithubDialHTTPFailure()
				if isRetryableNetErr(err) && (attempt+1 < attempts || ci+1 < len(clients)) {
					continue
				}
				return nil, lastErr
			}
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if resp.StatusCode >= 400 {
				return nil, fmt.Errorf("github /user http %d", resp.StatusCode)
			}
			var out map[string]any
			if err := json.Unmarshal(body, &out); err != nil {
				return map[string]any{}, nil
			}
			return out, nil
		}
	}
	return nil, lastErr
}

func parseOAuthTokenBody(body []byte, contentType string) map[string]any {
	txt := strings.TrimSpace(string(body))
	merged := map[string]any{}
	ctype := strings.ToLower(contentType)
	tryJSON := func(s string) {
		var j map[string]any
		if err := json.Unmarshal([]byte(s), &j); err == nil && j != nil {
			for k, v := range j {
				merged[k] = v
			}
		}
	}
	if strings.Contains(ctype, "json") || strings.HasPrefix(txt, "{") {
		tryJSON(txt)
	}
	if strings.Contains(txt, "=") {
		vals, _ := url.ParseQuery(txt)
		for k, vs := range vals {
			if len(vs) > 0 {
				merged[k] = vs[0]
			}
		}
	}
	return merged
}

// parseGitHubTokenBody parses GitHub token responses and surfaces OAuth errors
// even when HTTP status is 200 (GitHub commonly returns error in JSON body).
func parseGitHubTokenBody(body []byte, contentType string) (map[string]any, error) {
	out := parseOAuthTokenBody(body, contentType)
	if errVal, ok := out["error"]; ok {
		errStr := strings.TrimSpace(fmt.Sprint(errVal))
		if errStr != "" && errStr != "<nil>" {
			desc := strings.TrimSpace(fmt.Sprint(out["error_description"]))
			if desc != "" && desc != "<nil>" {
				return nil, fmt.Errorf("github token error: %s (%s)", errStr, desc)
			}
			return nil, fmt.Errorf("github token error: %s", errStr)
		}
	}
	return out, nil
}
