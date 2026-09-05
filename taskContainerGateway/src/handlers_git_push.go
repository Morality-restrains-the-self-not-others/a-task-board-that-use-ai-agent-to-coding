package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"tracelog"
)

func gitPushReadTimeoutSec() float64 {
	raw := strings.TrimSpace(os.Getenv("CONTAINER_LAYER_GIT_PUSH_READ_TIMEOUT"))
	if raw == "" {
		return 45
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil || v <= 0 {
		return 45
	}
	return v
}

func forwardToOnlineServiceTimeout(
	ctx context.Context,
	method string,
	upstreamURL string,
	accessToken string,
	body []byte,
	readSec float64,
) (int, []byte, error) {
	var reader io.Reader
	if len(body) > 0 {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, upstreamURL, reader)
	if err != nil {
		return 0, nil, err
	}
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Access-Token", accessToken)
	tracelog.ApplyOutboundHeaders(req, ctx)
	transport := &http.Transport{Proxy: nil}
	connectSec := cfg.ForwardConnectSec
	if connectSec <= 0 {
		connectSec = 15
	}
	if readSec <= 0 {
		readSec = cfg.ForwardReadSec
	}
	timeout := time.Duration(connectSec)*time.Second + time.Duration(readSec)*time.Second
	client := &http.Client{Transport: transport, Timeout: timeout}
	start := time.Now()
	resp, err := client.Do(req)
	duration := time.Since(start).Milliseconds()
	logURL := sanitizeUpstreamURLForLog(upstreamURL)
	if err != nil {
		tracelog.LogForwardStage(ctx, "upstream_forward", map[string]any{
			"upstream_url":    logURL,
			"upstream_status": 502,
			"duration_ms":     duration,
			"detail":          fmt.Sprintf("upstream error: %v", err),
		})
		payload, _ := json.Marshal(map[string]string{
			"detail": fmt.Sprintf("upstream error: %v", err),
		})
		return http.StatusBadGateway, payload, nil
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		tracelog.LogForwardStage(ctx, "upstream_forward", map[string]any{
			"upstream_url":    logURL,
			"upstream_status": 502,
			"duration_ms":     duration,
			"detail":          "failed to read upstream response",
		})
		return http.StatusBadGateway, []byte(`{"detail":"failed to read upstream response"}`), nil
	}
	if len(respBody) == 0 {
		respBody = []byte("{}")
	}
	tracelog.LogForwardStage(ctx, "upstream_forward", map[string]any{
		"upstream_url":    logURL,
		"upstream_status": resp.StatusCode,
		"duration_ms":     duration,
	})
	return resp.StatusCode, respBody, nil
}

func handleContainerLayerGitPush(
	w http.ResponseWriter,
	r *http.Request,
	sc scope,
	target containerTarget,
	session validateSessionResult,
	rawBody []byte,
) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"detail": "method not allowed, use POST"})
		return
	}
	ctx := r.Context()
	body := parseJSONBody(rawBody)
	layerID := strField(body, "layer_id")
	if layerID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "layer_id 必填"})
		return
	}

	preparePayload := map[string]any{
		"tenant_id":               sc.TenantID,
		"workspace_id":            sc.WorkspaceID,
		"task_id":                 sc.TaskID,
		"layer_id":                layerID,
		"user_id":                 session.UserID,
		"identity_id":             strField(body, "identity_id"),
		"prefer_container_remote": body["prefer_container_remote"],
		"repo_url":                strField(body, "repo_url"),
	}
	if v, ok := body["target_branch"]; ok {
		preparePayload["target_branch"] = v
	}
	if v, ok := body["github_auth_by_repo"]; ok {
		preparePayload["github_auth_by_repo"] = v
	}

	prepStatus, prepBody := cloudPostJSON(
		ctx,
		"/api/internal/layer-git-push/prepare",
		preparePayload,
		"cloud_prepare_git_push",
	)
	if prepStatus != http.StatusOK {
		writeRawJSON(w, prepStatus, prepBody)
		return
	}
	var prepared struct {
		OK                 bool           `json:"ok"`
		PushBody           map[string]any `json:"push_body"`
		UseOauthAccessPush bool           `json:"use_oauth_access_push"`
		Detail             string         `json:"detail"`
	}
	if err := json.Unmarshal(prepBody, &prepared); err != nil || !prepared.OK {
		if prepared.Detail != "" {
			writeJSON(w, http.StatusBadGateway, map[string]string{"detail": prepared.Detail})
			return
		}
		writeJSON(w, http.StatusBadGateway, map[string]string{"detail": "invalid prepare-layer-git-push response"})
		return
	}
	if prepared.PushBody == nil {
		prepared.PushBody = map[string]any{}
	}

	pathSuffix := "git/push"
	if prepared.UseOauthAccessPush {
		pathSuffix = "git/oauth-access-push"
	}
	upstreamURL := scopedAPIRoot(target.BaseURL, sc) + "/layers/" + url.PathEscape(layerID) + "/" + pathSuffix
	forwardBody, err := json.Marshal(prepared.PushBody)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "failed to encode push body"})
		return
	}

	upStatus, upBody, _ := forwardToOnlineServiceTimeout(
		ctx,
		http.MethodPost,
		upstreamURL,
		target.AccessToken,
		forwardBody,
		gitPushReadTimeoutSec(),
	)
	if upStatus >= 400 {
		writeRawJSON(w, upStatus, upBody)
		return
	}

	var upstream map[string]any
	if err := json.Unmarshal(upBody, &upstream); err != nil || upstream == nil {
		upstream = map[string]any{"ok": true}
	}
	// 前端 TaskDetail 读 github_pull_request.html_url；容器仅写 github_oauth_multirepo.repos[].pr。
	if _, hasPR := upstream["github_pull_request"]; !hasPR {
		if prSummary := githubPullRequestSummaryFromOauthRepos(upstream); prSummary != nil {
			upstream["github_pull_request"] = prSummary
		}
	}
	attachPrHtmlURLToGitRemote(upstream)
	// OPT-20260817-042: SaaS follow-up 创建的 PR 容器侧可能未落盘，整页刷新后层图会丢 PR 锚点。
	// 网关在拿到 pr_html_url 后兜底调用容器 remember 端点持久化（幂等，失败仅记日志）。
	if html := prHtmlURLFromGitRemote(upstream); html != "" {
		go rememberContainerLayerPrHtmlURL(ctx, target.BaseURL, sc, layerID, html, target.AccessToken)
	}

	pushedTB := strField(body, "target_branch")
	completePayload := map[string]any{
		"tenant_id":            sc.TenantID,
		"workspace_id":         sc.WorkspaceID,
		"task_id":              sc.TaskID,
		"layer_id":             layerID,
		"user_id":              session.UserID,
		"container_upstream":   upstream,
		"pushed_target_branch": pushedTB,
		"target_branch":        body["target_branch"],
	}
	traceID := tracelog.TraceIDFromContext(ctx)
	go func() {
		bg, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if traceID != "" {
			bg = tracelog.ContextWithTraceID(bg, traceID)
		}
		_, _ = cloudPostJSON(bg, "/api/internal/layer-git-push/complete", completePayload, "cloud_complete_git_push")
	}()

	outBody, err := json.Marshal(upstream)
	if err != nil {
		writeRawJSON(w, upStatus, upBody)
		return
	}
	writeRawJSON(w, upStatus, outBody)
}

// githubPullRequestSummaryFromOauthRepos maps container oauth-access-push repos into
// the github_pull_request shape expected by TaskDetail (html_url / skipped / repos).
func githubPullRequestSummaryFromOauthRepos(upstream map[string]any) map[string]any {
	multi, _ := upstream["github_oauth_multirepo"].(map[string]any)
	if multi == nil {
		return nil
	}
	rawRepos, ok := multi["repos"].([]any)
	if !ok || len(rawRepos) == 0 {
		return nil
	}
	summaries := make([]map[string]any, 0, len(rawRepos))
	firstHTML := ""
	var firstPRErr string
	sawPushOK := false
	for _, rowAny := range rawRepos {
		row, ok := rowAny.(map[string]any)
		if !ok {
			continue
		}
		pushOK, _ := row["push_ok"].(bool)
		if !pushOK {
			continue
		}
		sawPushOK = true
		slug := strings.TrimSpace(fmt.Sprint(row["github_slug"]))
		if slug == "<nil>" {
			slug = ""
		}
		pr, _ := row["pr"].(map[string]any)
		if pr != nil {
			html := strings.TrimSpace(fmt.Sprint(pr["html_url"]))
			if html == "" || html == "<nil>" {
				html = ""
			}
			if html != "" && firstHTML == "" {
				firstHTML = html
			}
			summaries = append(summaries, map[string]any{
				"github_slug": slug,
				"html_url":    html,
				"number":      pr["number"],
				"state":       pr["state"],
			})
			continue
		}
		if errRaw := row["pr_error"]; errRaw != nil {
			errText := strings.TrimSpace(fmt.Sprint(errRaw))
			if errText != "" && errText != "<nil>" {
				if firstPRErr == "" {
					firstPRErr = errText
				}
				summaries = append(summaries, map[string]any{
					"github_slug": slug,
					"pr_error":    errText,
				})
			}
		}
	}
	if !sawPushOK {
		return nil
	}
	if firstHTML != "" {
		return map[string]any{
			"multirepo": true,
			"html_url":  firstHTML,
			"repos":     summaries,
		}
	}
	if firstPRErr != "" {
		return map[string]any{
			"multirepo": true,
			"skipped":   "pr_api_error",
			"detail":    firstPRErr,
			"repos":     summaries,
		}
	}
	// 推送成功但容器未尝试建 PR（常见：prepare 未注入 pr_base_branch）
	return map[string]any{
		"multirepo": true,
		"skipped":   "pr_base_missing",
		"detail":    "推送成功但未创建 PR：缺少 pr_base_branch（merge 目标分支）",
		"repos":     summaries,
	}
}

// attachPrHtmlURLToGitRemote copies github_pull_request.html_url onto git_remote.pr_html_url
// so TaskDetail can attach a clickable PR link on the layer node after push.
func attachPrHtmlURLToGitRemote(upstream map[string]any) {
	if upstream == nil {
		return
	}
	pr, _ := upstream["github_pull_request"].(map[string]any)
	if pr == nil {
		return
	}
	html := strings.TrimSpace(fmt.Sprint(pr["html_url"]))
	if html == "" || html == "<nil>" {
		return
	}
	if gr, ok := upstream["git_remote"].(map[string]any); ok && gr != nil {
		gr["pr_html_url"] = html
		return
	}
	upstream["git_remote"] = map[string]any{"pr_html_url": html}
}

// prHtmlURLFromGitRemote extracts git_remote.pr_html_url from the upstream push response.
func prHtmlURLFromGitRemote(upstream map[string]any) string {
	if upstream == nil {
		return ""
	}
	gr, ok := upstream["git_remote"].(map[string]any)
	if !ok || gr == nil {
		return ""
	}
	html := strings.TrimSpace(fmt.Sprint(gr["pr_html_url"]))
	if html == "" || html == "<nil>" {
		return ""
	}
	return html
}

// rememberContainerLayerPrHtmlURL best-effort 通知容器把 PR/合并请求 URL 持久化到层文件系统
// （OPT-20260817-042）。容器 oauth-access-push 成功路径已自行 remember；本调用兜底覆盖
// SaaS follow-up 等未落盘路径，幂等（相同 URL 覆盖写）。失败仅记 trace 阶段日志，不影响推送结果。
func rememberContainerLayerPrHtmlURL(ctx context.Context, baseURL string, sc scope, layerID, htmlURL, accessToken string) {
	htmlURL = strings.TrimSpace(htmlURL)
	if htmlURL == "" || layerID == "" {
		return
	}
	rememberURL := scopedAPIRoot(baseURL, sc) + "/layers/" + url.PathEscape(layerID) + "/git/remember-pr-html-url"
	body, _ := json.Marshal(map[string]string{"html_url": htmlURL})
	bg, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	if tr := tracelog.TraceIDFromContext(ctx); tr != "" {
		bg = tracelog.ContextWithTraceID(bg, tr)
	}
	status, respBody, _ := forwardToOnlineServiceTimeout(bg, http.MethodPost, rememberURL, accessToken, body, 4)
	detail := strings.TrimSpace(string(respBody))
	if len(detail) > 200 {
		detail = detail[:200]
	}
	tracelog.LogForwardStage(ctx, "remember_pr_html_url", map[string]any{
		"layer_id":        layerID,
		"upstream_status": status,
		"detail":          detail,
	})
}

func handleContainerLayerGitPushAuthContext(
	w http.ResponseWriter,
	r *http.Request,
	sc scope,
	session validateSessionResult,
) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"detail": "method not allowed, use GET"})
		return
	}
	ctx := r.Context()
	q := url.Values{}
	q.Set("tenant_id", sc.TenantID)
	q.Set("workspace_id", sc.WorkspaceID)
	q.Set("task_id", sc.TaskID)
	q.Set("user_id", session.UserID)
	status, body := cloudGetJSON(
		ctx,
		"/api/internal/layer-git-push/auth-context?"+q.Encode(),
		"cloud_git_push_auth_context",
	)
	writeRawJSON(w, status, body)
}
