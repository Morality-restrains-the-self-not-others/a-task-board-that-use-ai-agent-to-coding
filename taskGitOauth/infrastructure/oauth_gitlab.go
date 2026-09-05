package infrastructure

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

func ExchangeGitLabCode(pc *ProviderConfig, code, redirectURI string) (map[string]any, error) {
	if pc == nil {
		return nil, fmt.Errorf("未配置 gitlab provider")
	}
	cid := strings.TrimSpace(pc.ClientID)
	sec := strings.TrimSpace(pc.ClientSecret)
	uri := strings.TrimSpace(redirectURI)
	if uri == "" {
		uri = pc.RedirectURI
	}
	if cid == "" || sec == "" {
		return nil, fmt.Errorf("未配置 GITLAB client_id / client_secret")
	}
	if uri == "" {
		return nil, fmt.Errorf("未配置 GITLAB redirect_uri")
	}
	tokenURL, _, err := gitlabEndpoints(pc)
	if err != nil {
		return nil, err
	}
	form := url.Values{}
	form.Set("client_id", cid)
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", uri)
	req, err := http.NewRequest(http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(cid, sec)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := gitlabHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, &GitLabExchangeRejectedError{StatusCode: resp.StatusCode, Body: string(body)}
	}
	var out map[string]any
	_ = json.Unmarshal(body, &out)
	if out == nil {
		out = map[string]any{}
	}
	// GitLab 200 但 body 携带 error（invalid_grant / redirect_uri_mismatch 等）→ 拒绝语义
	if errVal, ok := out["error"]; ok {
		if errStr := strings.TrimSpace(fmt.Sprint(errVal)); errStr != "" && errStr != "<nil>" {
			return nil, &GitLabExchangeRejectedError{StatusCode: resp.StatusCode, Body: string(body)}
		}
	}
	return out, nil
}

func RefreshGitLabToken(pc *ProviderConfig, refreshToken string) (map[string]any, error) {
	if pc == nil {
		return nil, fmt.Errorf("未配置 gitlab provider")
	}
	cid := strings.TrimSpace(pc.ClientID)
	sec := strings.TrimSpace(pc.ClientSecret)
	if cid == "" || sec == "" {
		return nil, fmt.Errorf("未配置 GITLAB client_id / client_secret")
	}
	tokenURL, _, err := gitlabEndpoints(pc)
	if err != nil {
		return nil, err
	}
	form := url.Values{}
	uri := strings.TrimSpace(pc.RedirectURI)
	if uri == "" {
		return nil, fmt.Errorf("未配置 GITLAB redirect_uri")
	}
	form.Set("client_id", cid)
	form.Set("client_secret", sec)
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)
	// GitLab Doorkeeper：redirect_uri 必须与授权时完全一致，否则 /oauth/token 返回 400 invalid_grant。
	form.Set("redirect_uri", uri)
	req, err := http.NewRequest(http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(cid, sec)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := gitlabHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, gitlabRefreshHTTPError(resp.StatusCode, body)
	}
	var out map[string]any
	_ = json.Unmarshal(body, &out)
	if out == nil {
		out = map[string]any{}
	}
	if errVal, ok := out["error"]; ok {
		if errStr := strings.TrimSpace(fmt.Sprint(errVal)); errStr != "" && errStr != "<nil>" {
			return nil, gitlabRefreshHTTPError(resp.StatusCode, body)
		}
	}
	return out, nil
}

func gitlabOAuthErrorCode(body []byte) string {
	var parsed map[string]any
	if err := json.Unmarshal(body, &parsed); err != nil || parsed == nil {
		return ""
	}
	code := strings.ToLower(strings.TrimSpace(fmt.Sprint(parsed["error"])))
	switch code {
	case "invalid_grant", "invalid_request", "invalid_client", "unauthorized_client", "unsupported_grant_type":
		return code
	default:
		return ""
	}
}

func gitlabRefreshHTTPError(status int, body []byte) error {
	if status < 400 {
		status = http.StatusBadRequest
	}
	if code := gitlabOAuthErrorCode(body); code != "" {
		return fmt.Errorf("gitlab refresh http %d: %s", status, code)
	}
	return fmt.Errorf("gitlab refresh http %d", status)
}

func FetchGitLabProfile(pc *ProviderConfig, accessToken string) (map[string]any, error) {
	_, userURL, err := gitlabEndpoints(pc)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodGet, userURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := gitlabHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("gitlab /user http %d", resp.StatusCode)
	}
	var out map[string]any
	_ = json.Unmarshal(body, &out)
	if out == nil {
		out = map[string]any{}
	}
	return out, nil
}

func gitlabEndpoints(pc *ProviderConfig) (tokenURL, userURL string, err error) {
	website := strings.TrimRight(strings.TrimSpace(pc.Website), "/")
	if website == "" {
		return "", "", fmt.Errorf("未配置 gitlab provider 的 website")
	}
	u, err := url.Parse(website)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", "", fmt.Errorf("gitlab.website 必须为有效 origin")
	}
	origin := u.Scheme + "://" + u.Host
	return origin + "/oauth/token", origin + "/api/v4/user", nil
}
