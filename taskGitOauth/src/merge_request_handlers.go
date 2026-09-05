package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"taskGitOauth/domain"
	"taskGitOauth/infrastructure"
)

// errGitOAuthNotConnected 是内部哨兵；HTTP 响应必须用 detailGitOAuthNotConnected，禁止把英文哨兵泄漏给前端弹层。
var errGitOAuthNotConnected = errors.New("git oauth not connected")

const detailGitOAuthNotConnected = "尚未绑定 Git 网站 OAuth，请先在账号中心完成授权后再合并"

// detailGitOAuthReauthRequired 用于 refresh_token 被 Git 站点拒绝（400/401）或无法解密：凭据行仍在，但不能当「未绑定」。
const detailGitOAuthReauthRequired = "Git OAuth 授权已失效，请重新绑定后再合并"

const mergeRequestStatusMaxURLs = 20

func extractFuncTenantID(path, funcName string) string {
	prefix := "/api/git-oauth/" + funcName + "/tenant_id/"
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	rest := strings.Trim(strings.TrimPrefix(path, prefix), "/")
	if rest == "" {
		return ""
	}
	return strings.TrimSpace(strings.SplitN(rest, "/", 2)[0])
}

func (a *App) mergeRequestHostAllowed(host string) bool {
	h := strings.ToLower(strings.TrimSpace(host))
	if h == "github.com" || h == "www.github.com" {
		return true
	}
	if h == "" {
		return false
	}
	return a.providerConfigForRepoURL("https://"+h+"/") != nil
}

func (a *App) handleMergeRequestStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	tid := extractFuncTenantID(r.URL.Path, "merge-request-status")
	if tid == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "missing tenant id"})
		return
	}
	if !a.ensureTenantMember(w, r, tid) {
		return
	}
	userID := a.effectiveUserID(r)
	body, err := readJSON(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "invalid json"})
		return
	}
	rawURLs, _ := body["html_urls"].([]any)
	if len(rawURLs) > mergeRequestStatusMaxURLs {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "html_urls exceeds 20"})
		return
	}
	results := make([]map[string]any, 0, len(rawURLs))
	nameCache := map[string]string{}
	for _, raw := range rawURLs {
		htmlURL := strings.TrimSpace(fmt.Sprint(raw))
		item := map[string]any{"html_url": htmlURL}
		state := "unknown"
		st, err := a.lookupMergeRequestStatus(r, userID, htmlURL)
		if err != nil {
			item["error"] = err.Error()
		} else {
			state = st.State
			if st.Title != "" {
				item["title"] = st.Title
			}
		}
		item["state"] = state
		if state == "merged" {
			// 审计回显：已由谁合并（展示名）+ 何时点击（RFC3339）。
			if mergedBy, mergedAt := a.mergeAuditContext(tid, htmlURL, nameCache); mergedBy != "" || mergedAt != "" {
				if mergedBy != "" {
					item["merged_by"] = mergedBy
				}
				if mergedAt != "" {
					item["merged_at"] = mergedAt
				}
			}
		}
		results = append(results, item)
	}
	logInfo("event=merge_request_status_ok tenant_id=%s user_id=%s count=%d trace_id=%s",
		tid, userID, len(results), requestTraceID(r))
	writeJSON(w, http.StatusOK, map[string]any{"results": results})
}

func (a *App) handleMergeRequestMerge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	tid := extractFuncTenantID(r.URL.Path, "merge-request-merge")
	if tid == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "missing tenant id"})
		return
	}
	if !a.ensureTenantMember(w, r, tid) {
		return
	}
	userID := a.effectiveUserID(r)
	body, err := readJSON(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "invalid json"})
		return
	}
	htmlURL := strings.TrimSpace(fmt.Sprint(body["html_url"]))
	taskID := truncate(strings.TrimSpace(fmt.Sprint(body["task_id"])), 64)
	commentID := truncate(strings.TrimSpace(fmt.Sprint(body["comment_id"])), 64)
	if htmlURL == "" || htmlURL == "<nil>" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "html_url required"})
		return
	}
	ref, err := domain.ParseMergeRequestURL(htmlURL)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": err.Error()})
		return
	}
	if !a.mergeRequestHostAllowed(ref.Host) {
		logWarn("event=merge_request_ssrf_denied host=%s user_id=%s trace_id=%s", ref.Host, userID, requestTraceID(r))
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "html_url host is not an allowed git provider"})
		return
	}
	a.applyGitLabAPIOrigin(&ref)
	traceID := requestTraceID(r)
	stageAt := time.Now()
	token, err := a.accessTokenForMergeURL(userID, htmlURL)
	logInfo("event=merge_request_stage stage=token host=%s elapsed_ms=%d err=%v trace_id=%s",
		ref.Host, time.Since(stageAt).Milliseconds(), err, traceID)
	if isGitOAuthNotConnectedErr(err, token) {
		logWarn("event=merge_request_no_token user_id=%s host=%s trace_id=%s", userID, ref.Host, traceID)
		writeJSON(w, http.StatusConflict, map[string]any{"detail": detailGitOAuthNotConnected})
		return
	}
	if err != nil {
		logWarn("event=merge_request_token_failed user_id=%s host=%s err=%v trace_id=%s", userID, ref.Host, err, traceID)
		writeJSON(w, http.StatusBadGateway, map[string]any{"detail": mergeTokenUserError(err)})
		return
	}
	stageAt = time.Now()
	st, err := a.gitFetchMergeRequest(token, ref)
	logInfo("event=merge_request_stage stage=status host=%s elapsed_ms=%d err=%v trace_id=%s",
		ref.Host, time.Since(stageAt).Milliseconds(), err, traceID)
	if err != nil {
		logWarn("event=merge_request_status_failed host=%s err=%v trace_id=%s", ref.Host, err, traceID)
		writeGitAPIError(w, ref.Host, err, "status")
		return
	}
	noop := st.State == "merged"
	if !noop {
		stageAt = time.Now()
		err := a.gitMergeMergeRequest(token, ref)
		logInfo("event=merge_request_stage stage=merge host=%s elapsed_ms=%d err=%v trace_id=%s",
			ref.Host, time.Since(stageAt).Milliseconds(), err, traceID)
		if err != nil {
			stageAt = time.Now()
			st2, lookupErr := a.gitFetchMergeRequest(token, ref)
			logInfo("event=merge_request_stage stage=reconcile host=%s elapsed_ms=%d lookup_err=%v state=%s trace_id=%s",
				ref.Host, time.Since(stageAt).Milliseconds(), lookupErr, st2.State, traceID)
			if lookupErr == nil && st2.State == "merged" {
				noop = true
			} else {
				a.writeMergeAudit(ref, taskID, tid, userID, commentID, token, "failed", err.Error())
				logWarn("event=merge_request_merge_failed user_id=%s html_url_host=%s err=%v trace_id=%s",
					userID, ref.Host, err, traceID)
				writeGitAPIError(w, ref.Host, err, "merge")
				return
			}
		}
	}
	result := "merged"
	if noop {
		result = "noop_already_merged"
	}
	a.writeMergeAudit(ref, taskID, tid, userID, commentID, token, result, "")
	logInfo("event=merge_request_merged user_id=%s result=%s host=%s trace_id=%s",
		userID, result, ref.Host, traceID)
	publishDomainEvent(a.Cfg, "GIT_MERGE_REQUEST_MERGED", map[string]any{
		"tenant_id":         tid,
		"task_id":           taskID,
		"comment_id":        commentID,
		"html_url":          htmlURL,
		"provider":          ref.Provider,
		"merged_by_user_id": userID,
		"noop":              noop,
	}, taskID)
	// 审计回显：合并成功后立即返回「已由谁合并 + 点击时间」，前端徽章无需等轮询。
	mergedAt := time.Now().UTC().Format(time.RFC3339)
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":        true,
		"merged":    true,
		"noop":      noop,
		"state":     "merged",
		"html_url":  htmlURL,
		"merged_by": a.userDisplayName(tid, userID, nil),
		"merged_at": mergedAt,
	})
}

type mergeRequestState struct {
	State string
	Title string
}

func (a *App) lookupMergeRequestStatus(r *http.Request, userID, htmlURL string) (mergeRequestState, error) {
	ref, err := domain.ParseMergeRequestURL(htmlURL)
	if err != nil {
		return mergeRequestState{}, err
	}
	if !a.mergeRequestHostAllowed(ref.Host) {
		return mergeRequestState{}, fmt.Errorf("html_url host is not an allowed git provider")
	}
	a.applyGitLabAPIOrigin(&ref)
	token, err := a.accessTokenForMergeURL(userID, htmlURL)
	if isGitOAuthNotConnectedErr(err, token) {
		return mergeRequestState{}, fmt.Errorf("%s", detailGitOAuthNotConnected)
	}
	if err != nil {
		if isGitOAuthRefreshAuthFailure(err) {
			return mergeRequestState{}, fmt.Errorf("%s", detailGitOAuthReauthRequired)
		}
		return mergeRequestState{}, err
	}
	st, err := a.gitFetchMergeRequest(token, ref)
	if err != nil && isGitAPITimeout(err) {
		return mergeRequestState{}, fmt.Errorf("%s", gitAPITimeoutUserError(ref.Host, "status"))
	}
	return st, err
}

func isGitOAuthNotConnectedErr(err error, token string) bool {
	if strings.TrimSpace(token) != "" && err == nil {
		return false
	}
	if err == nil {
		return true
	}
	return errors.Is(err, errGitOAuthNotConnected)
}

func isGitOAuthRefreshAuthFailure(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "decrypt_failed") {
		return true
	}
	return strings.Contains(msg, "refresh http 400") || strings.Contains(msg, "refresh http 401")
}

func mergeTokenUserError(err error) string {
	if err == nil {
		return "无法获取 Git 访问令牌"
	}
	if isGitOAuthRefreshAuthFailure(err) {
		return detailGitOAuthReauthRequired
	}
	return "无法获取 Git 访问令牌：" + truncate(err.Error(), 200)
}

func (a *App) accessTokenForMergeURL(userID, htmlURL string) (string, error) {
	if a.MergeAccessTokenFn != nil {
		return a.MergeAccessTokenFn(userID, htmlURL)
	}
	row, err := a.findActiveCredentialForMergeURL(userID, htmlURL)
	if err != nil {
		return "", err
	}
	if row == nil || strings.TrimSpace(row.RefreshTokenCipher) == "" {
		return "", errGitOAuthNotConnected
	}
	return a.issueAccessTokenFromCredential(userID, row)
}

func (a *App) findActiveCredentialForMergeURL(userID, htmlURL string) (*infrastructure.CredentialRow, error) {
	keys := a.userAppConnectionLookupKeys("", htmlURL)
	if len(keys) == 0 {
		ref, err := domain.ParseMergeRequestURL(htmlURL)
		if err != nil {
			return nil, err
		}
		if ref.Provider == domain.MergeProviderGitHub {
			keys = []string{"github"}
		}
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("unknown git provider")
	}
	rows, err := a.DB.ListCredentialsForUserKeys(keys, userID)
	if err != nil {
		return nil, err
	}
	for i := range rows {
		row := rows[i]
		if strings.TrimSpace(row.RefreshTokenCipher) == "" {
			continue
		}
		if row.BindStatus != "" && row.BindStatus != "active" {
			continue
		}
		return &row, nil
	}
	return nil, errGitOAuthNotConnected
}

// mergeAuditContext 从审计表取该 PR 最近一次成功合并的「谁 + 何时」：
// mergedBy 为展示名（解析失败回退 user_id），mergedAt 为点击时间 RFC3339。
// nameCache 为请求内 user_id → 展示名缓存（可为 nil），避免同请求重复回源。
func (a *App) mergeAuditContext(tenantID, htmlURL string, nameCache map[string]string) (mergedBy, mergedAt string) {
	if a.DB == nil {
		return "", ""
	}
	row, err := a.DB.SelectLatestMergeAuditByURL(htmlURL)
	if err != nil || row == nil {
		return "", ""
	}
	if !row.CreatedAt.IsZero() {
		mergedAt = row.CreatedAt.UTC().Format(time.RFC3339)
	}
	uid := strings.TrimSpace(row.UserID)
	if uid == "" {
		return "", mergedAt
	}
	mergedBy = a.userDisplayName(tenantID, uid, nameCache)
	return mergedBy, mergedAt
}

// userDisplayName 解析 user_id 的对外展示名（「已由 xxx 合并」的 xxx）。
// 优先级：注入函数（测试）→ taskTenantService members/display-name → user_id 本身。
func (a *App) userDisplayName(tenantID, userID string, cache map[string]string) string {
	if cache != nil {
		if v, ok := cache[userID]; ok {
			return v
		}
	}
	name := a.resolveDisplayNameFromTenantService(tenantID, userID)
	if cache != nil {
		cache[userID] = name
	}
	return name
}

func (a *App) resolveDisplayNameFromTenantService(tenantID, userID string) string {
	if a.ResolveDisplayNameFn != nil {
		return a.ResolveDisplayNameFn(userID, tenantID)
	}
	base := strings.TrimRight(strings.TrimSpace(a.Cfg.TaskTenantServiceURL), "/")
	if base == "" {
		return userID
	}
	u := fmt.Sprintf("%s/api/internal/tenant/members/display-name?user_id=%s&company_id=%s",
		base, url.QueryEscape(userID), url.QueryEscape(tenantID))
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return userID
	}
	req.Header.Set("Accept", "application/json")
	if sec := strings.TrimSpace(a.Cfg.DjangoInternalSecret); sec != "" {
		req.Header.Set("X-Internal-Secret", sec)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return userID
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return userID
	}
	var out struct {
		DisplayName string `json:"display_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return userID
	}
	if n := strings.TrimSpace(out.DisplayName); n != "" {
		return n
	}
	return userID
}

func (a *App) writeMergeAudit(ref domain.MergeRequestRef, taskID, tenantID, userID, commentID, accessToken, result, errMsg string) {
	if a.DB == nil {
		return
	}
	detail := map[string]any{
		"html_url":   ref.HTMLURL,
		"comment_id": commentID,
		"user_id":    userID,
		"result":     result,
		"provider":   ref.Provider,
		"host":       ref.Host,
	}
	if errMsg != "" {
		detail["error"] = truncate(errMsg, 300)
	}
	if result != "failed" {
		// 精确点击时间（RFC3339），供状态回显「已由 xxx 合并 + 悬停时间」；
		// 历史行无此字段时读路径回退 created_at 列。
		detail["merged_at"] = time.Now().UTC().Format(time.RFC3339)
	}
	companyID := optionalInt64(tenantID)
	userPtr := optionalInt64(userID)
	_ = a.DB.InsertTaskAudit(ref.Provider, taskID, "", companyID, userPtr, "merge_request_merge", detail)
	fp := accessTokenFingerprint(accessToken)
	_ = a.DB.InsertAccessAudit(ref.Host, userID, companyID, nil, "merge_request_merge", fp, detail)
}
