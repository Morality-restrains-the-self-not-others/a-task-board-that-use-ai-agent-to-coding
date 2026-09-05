package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"tracelog"
)

// localHTTPClient is used for calls to onlineServiceJS (127.0.0.1) to avoid
// system proxy interference (matching Python's ProxyHandler({}) behavior).
var localHTTPClient = &http.Client{
	Transport: &http.Transport{
		Proxy: nil,
		DialContext: (&net.Dialer{
			Timeout: 5 * time.Second,
		}).DialContext,
	},
	Timeout: 60 * time.Second,
}

// defaultHTTPClient is used for calls to the Django backend (token exchange, push).
var defaultHTTPClient = newBackendHTTPClient(30 * time.Second)

// extractOrigin parses a URL string and returns scheme://netloc.
func extractOrigin(raw string) string {
	text := strings.TrimSpace(raw)
	if text == "" {
		return ""
	}
	parsed, err := url.Parse(text)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ""
	}
	return normalizeLoopbackOrigin(fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host))
}

// normalizeLoopbackOrigin maps localhost to 127.0.0.1 so macOS Go resolves IPv4
// when Django/listeners bind IPv4 only (localhost may resolve to ::1 first).
func normalizeLoopbackOrigin(origin string) string {
	text := strings.TrimRight(strings.TrimSpace(origin), "/")
	if text == "" {
		return ""
	}
	parsed, err := url.Parse(text)
	if err != nil || parsed.Host == "" {
		return text
	}
	host, port, splitErr := net.SplitHostPort(parsed.Host)
	if splitErr != nil {
		host = parsed.Host
		port = ""
	}
	if !strings.EqualFold(strings.TrimSpace(host), "localhost") {
		return text
	}
	if port != "" {
		parsed.Host = net.JoinHostPort("127.0.0.1", port)
	} else {
		parsed.Host = "127.0.0.1"
	}
	return strings.TrimRight(parsed.String(), "/")
}

func businessAPIEndpointForExchange(env map[string]string) string {
	direct := strings.TrimSpace(env["BUSINESS_API_ENDPOINT"])
	if direct == "" {
		direct = strings.TrimSpace(env["BusinessApiEndPoint"])
	}
	if direct != "" {
		return strings.TrimRight(direct, "/")
	}
	origin := extractOrigin(env["BUSINESS_API_ENDPOINT_ORIGIN"])
	if origin != "" {
		return strings.TrimRight(origin, "/") + "/api"
	}
	return ""
}

func expandEnvForRuntime(envIn map[string]string, tenantID, workspaceID, taskID string) map[string]string {
	env := make(map[string]string)
	for k, v := range envIn {
		k = strings.TrimSpace(k)
		if k != "" {
			env[k] = v
		}
	}

	taskOrigin := extractOrigin(env["TASK_API_ENDPOINT_ORIGIN"])
	bizOrigin := extractOrigin(env["BUSINESS_API_ENDPOINT_ORIGIN"])

	tid := strings.TrimSpace(tenantID)
	if tid == "" {
		tid = strings.TrimSpace(env["tenantId"])
	}
	wid := strings.TrimSpace(workspaceID)
	if wid == "" {
		wid = strings.TrimSpace(env["workspaceId"])
	}
	task := strings.TrimSpace(taskID)
	if task == "" {
		task = strings.TrimSpace(env["taskId"])
	}

	if taskOrigin != "" && tid != "" && wid != "" && task != "" {
		comment := strings.TrimSpace(env["COMMENT_ID"])
		if comment != "" && comment != "-" {
			prefix := fmt.Sprintf("%s/api/tenant/%s/workspace/%s/task/%s/comment/%s/cloud",
				strings.TrimRight(taskOrigin, "/"), tid, wid, task, comment)
			if _, ok := env["TASK_API_ENDPOINT"]; !ok {
				env["TASK_API_ENDPOINT"] = prefix
			}
			if _, ok := env["TaskApiEndPoint"]; !ok {
				env["TaskApiEndPoint"] = prefix
			}
		}
	}

	if bizOrigin != "" {
		bizAPI := strings.TrimRight(bizOrigin, "/") + "/api"
		if _, ok := env["BUSINESS_API_ENDPOINT"]; !ok {
			env["BUSINESS_API_ENDPOINT"] = bizAPI
		}
		if _, ok := env["BusinessApiEndPoint"]; !ok {
			env["BusinessApiEndPoint"] = bizAPI
		}
	}

	if tid != "" {
		if _, ok := env["tenantId"]; !ok {
			env["tenantId"] = tid
		}
	}
	if wid != "" {
		if _, ok := env["workspaceId"]; !ok {
			env["workspaceId"] = wid
		}
	}
	if task != "" {
		if _, ok := env["taskId"]; !ok {
			env["taskId"] = task
		}
	}

	return env
}

// tokenExchangeResult is the outcome of exchange-refresh + refresh-access.
type tokenExchangeResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time // zero if expires_at missing/unparseable
}

// accessTokenExpiresLayout matches taskCredentialService RefreshAccess UTC format.
const accessTokenExpiresLayout = "2006-01-02 15:04:05"

func parseAccessTokenExpiresAt(raw string) time.Time {
	text := strings.TrimSpace(raw)
	if text == "" {
		return time.Time{}
	}
	t, err := time.ParseInLocation(accessTokenExpiresLayout, text, time.UTC)
	if err != nil {
		return time.Time{}
	}
	return t
}

// accessTokenNeedsRefresh reports whether proactive refresh should run.
// Unknown expiry (zero) → false (do not guess).
func accessTokenNeedsRefresh(expiresAt time.Time, now time.Time, skew time.Duration) bool {
	if expiresAt.IsZero() {
		return false
	}
	if skew < 0 {
		skew = 0
	}
	return !now.Before(expiresAt.Add(-skew))
}

func cloudTokenAPIPrefix(taskAPIOrigin, tenantID, workspaceID, taskID, commentID string) (string, error) {
	origin := extractOrigin(taskAPIOrigin)
	if origin == "" {
		origin = strings.TrimRight(strings.TrimSpace(taskAPIOrigin), "/")
	}
	tid := strings.TrimSpace(tenantID)
	wid := strings.TrimSpace(workspaceID)
	task := strings.TrimSpace(taskID)
	comment := strings.TrimSpace(commentID)
	if origin == "" || tid == "" || wid == "" || task == "" {
		return "", fmt.Errorf("token exchange requires task_api_origin, tenant_id, workspace_id, task_id")
	}
	if comment == "" || comment == "-" {
		return "", fmt.Errorf("token exchange requires comment_id")
	}
	prefix := fmt.Sprintf("%s/api/tenant/%s/workspace/%s/task/%s/comment/%s/cloud",
		strings.TrimRight(origin, "/"), tid, wid, task, comment)
	return prefix, nil
}

func refreshAccessToken(refreshToken, taskAPIOrigin, tenantID, workspaceID, taskID, commentID string) (tokenExchangeResult, error) {
	var out tokenExchangeResult
	prefix, err := cloudTokenAPIPrefix(taskAPIOrigin, tenantID, workspaceID, taskID, commentID)
	if err != nil {
		return out, err
	}
	rt := strings.TrimSpace(refreshToken)
	if rt == "" {
		return out, fmt.Errorf("refresh-access requires refresh_token")
	}

	accessURL := prefix + "/server-container-token/refresh-access/"
	var lastErr error
	for attempt := 1; attempt <= tokenExchangeRetries; attempt++ {
		body := map[string]string{"refresh_token": rt}
		resp, err := doJSONPost(activeCorrelationCtx(), defaultHTTPClient, accessURL, body, tokenExchangeTimeout)
		if err != nil {
			lastErr = err
			appendLog(fmt.Sprintf("[relayToTrae] token-refresh: refresh-access attempt %d failed: %v", attempt, err))
			if attempt >= tokenExchangeRetries {
				return out, fmt.Errorf("refresh-access failed after %d attempts: %w", tokenExchangeRetries, lastErr)
			}
			continue
		}
		at, ok := resp["access_token"].(string)
		if !ok || strings.TrimSpace(at) == "" {
			lastErr = fmt.Errorf("refresh-access missing access_token")
			appendLog(fmt.Sprintf("[relayToTrae] token-refresh: %v", lastErr))
			if attempt >= tokenExchangeRetries {
				return out, lastErr
			}
			continue
		}
		out.AccessToken = strings.TrimSpace(at)
		out.RefreshToken = rt
		if expRaw, ok := resp["expires_at"].(string); ok {
			out.ExpiresAt = parseAccessTokenExpiresAt(expRaw)
		}
		appendLog(fmt.Sprintf(
			"[relayToTrae] token-refresh: refresh-access OK new_access_token len=%d expires_at_set=%v",
			len(out.AccessToken), !out.ExpiresAt.IsZero(),
		))
		return out, nil
	}
	return out, lastErr
}

// performTokenExchange exchanges the initial access_token for a new one.
// Step 1: exchange-refresh → get refresh_token
// Step 2: refresh-access → get new access_token (+ expires_at when present)
//
// Production relay start no longer calls this: onlineServiceJS performs the first
// exchange (cloud UserData parity). Kept for unit tests and optional tooling.
func performTokenExchange(accessToken, taskAPIOrigin, businessAPIEndpoint, tenantID, workspaceID, taskID, commentID string) (tokenExchangeResult, error) {
	var out tokenExchangeResult
	prefix, err := cloudTokenAPIPrefix(taskAPIOrigin, tenantID, workspaceID, taskID, commentID)
	if err != nil {
		return out, err
	}
	biz := strings.TrimSpace(businessAPIEndpoint)

	appendLog("[relayToTrae] token-exchange: begin")

	// Step 1: exchange-refresh
	refreshURL := prefix + "/server-container-token/exchange-refresh/"
	appendLog(fmt.Sprintf("[relayToTrae] token-exchange: POST %s", refreshURL))

	var refreshToken string
	var lastErr error
	for attempt := 1; attempt <= tokenExchangeRetries; attempt++ {
		body := map[string]string{
			"access_token":          accessToken,
			"business_api_endpoint": biz,
		}
		resp, err := doJSONPost(activeCorrelationCtx(), defaultHTTPClient, refreshURL, body, tokenExchangeTimeout)
		if err != nil {
			lastErr = err
			appendLog(fmt.Sprintf("[relayToTrae] token-exchange: exchange-refresh attempt %d failed: %v", attempt, err))
			if attempt >= tokenExchangeRetries {
				return out, fmt.Errorf("exchange-refresh failed after %d attempts: %w", tokenExchangeRetries, lastErr)
			}
			continue
		}
		rt, ok := resp["refresh_token"].(string)
		if !ok || strings.TrimSpace(rt) == "" {
			lastErr = fmt.Errorf("exchange-refresh missing refresh_token")
			appendLog(fmt.Sprintf("[relayToTrae] token-exchange: %v", lastErr))
			if attempt >= tokenExchangeRetries {
				return out, lastErr
			}
			continue
		}
		refreshToken = strings.TrimSpace(rt)
		appendLog(fmt.Sprintf("[relayToTrae] token-exchange: exchange-refresh OK refresh_token len=%d", len(refreshToken)))
		break
	}

	// 与 refresh-access 错开，减轻启动瞬间对同一 SQLite 文件的连续写。
	time.Sleep(150 * time.Millisecond)

	refreshed, err := refreshAccessToken(refreshToken, taskAPIOrigin, tenantID, workspaceID, taskID, commentID)
	if err != nil {
		return out, err
	}
	out = refreshed
	appendLog("[relayToTrae] token-exchange: done")
	return out, nil
}

// ensureFreshAccessTokenLocked refreshes state.AccessToken when remaining TTL is within skew.
// Caller must NOT hold stateMu (this function acquires it). On refresh failure, leaves tokens unchanged.
func ensureFreshAccessToken(now time.Time) error {
	stateMu.Lock()
	expiresAt := state.AccessTokenExpiresAt
	refreshToken := strings.TrimSpace(state.RefreshToken)
	taskOrigin := ""
	tenantID := ""
	workspaceID := ""
	commentID := ""
	taskID := trimTaskID(state.ActiveTaskID)
	if taskID != "" {
		if reg, ok := registeredTasks[taskID]; ok && reg != nil {
			taskOrigin = reg.TaskAPIOrigin
			tenantID = reg.TenantID
			workspaceID = reg.WorkspaceID
			commentID = reg.CommentID
		}
	}
	// Fallback: any registered task (single-active runtime).
	if taskOrigin == "" {
		for _, reg := range registeredTasks {
			if reg == nil {
				continue
			}
			taskOrigin = reg.TaskAPIOrigin
			tenantID = reg.TenantID
			workspaceID = reg.WorkspaceID
			commentID = reg.CommentID
			if taskID == "" {
				taskID = reg.TaskID
			}
			break
		}
	}
	needs := accessTokenNeedsRefresh(expiresAt, now, tokenProactiveRefreshSkew)
	stateMu.Unlock()

	if !needs {
		return nil
	}
	if refreshToken == "" || taskOrigin == "" || tenantID == "" || workspaceID == "" || taskID == "" {
		// Soft skip: keep current access_token; 401 unregister remains the hard fallback.
		appendLog("[relayToTrae] token-refresh: skipped (missing refresh_token or task scope)")
		return nil
	}

	appendLog(fmt.Sprintf(
		"[relayToTrae] token-refresh: proactive refresh (expires_at=%s skew=%s)",
		expiresAt.UTC().Format(accessTokenExpiresLayout), tokenProactiveRefreshSkew,
	))
	result, err := refreshAccessToken(refreshToken, taskOrigin, tenantID, workspaceID, taskID, commentID)
	if err != nil {
		return err
	}

	stateMu.Lock()
	state.AccessToken = result.AccessToken
	if result.RefreshToken != "" {
		state.RefreshToken = result.RefreshToken
	}
	if !result.ExpiresAt.IsZero() {
		state.AccessTokenExpiresAt = result.ExpiresAt
	}
	// Keep registered push tokens in sync immediately.
	for _, reg := range registeredTasks {
		if reg != nil {
			reg.AccessToken = result.AccessToken
		}
	}
	stateMu.Unlock()
	return nil
}

func doJSONPost(ctx context.Context, client *http.Client, urlStr string, body interface{}, timeout time.Duration) (map[string]interface{}, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", urlStr, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	tracelog.ApplyOutboundHeaders(req, ctx)

	// Apply timeout at the client level for this request.
	clientWithTimeout := newBackendHTTPClient(timeout)
	if client != nil && client.Transport != nil {
		clientWithTimeout.Transport = client.Transport
	}

	resp, err := clientWithTimeout.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, &httpError{Code: resp.StatusCode, Body: string(respBody)}
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		if len(bytes.TrimSpace(respBody)) == 0 {
			return map[string]interface{}{}, nil
		}
		return nil, err
	}
	return result, nil
}

type httpError struct {
	Code int
	Body string
}

func (e *httpError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.Code, e.Body)
}
