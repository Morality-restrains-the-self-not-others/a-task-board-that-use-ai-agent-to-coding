package main

import (
	"fmt"
	"net/http"
	"strings"

	"tracelog"

	"taskGitOauth/domain"
	"taskGitOauth/infrastructure"
)

// requestTraceID normalizes the inbound W3C trace id (forwarded by taskProjectService,
// OPT-20260810-051) so access-for-user failure logs join the same trace as the caller.
func requestTraceID(r *http.Request) string {
	if r == nil {
		return ""
	}
	return tracelog.NormalizeTraceID(tracelog.CorrelationFromContext(r.Context()).TraceID)
}

func (a *App) handleRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	provider := providerFromPath(r.URL.Path)
	body, err := readJSONNumbered(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "invalid json"})
		return
	}
	providerKey := normalizeProviderKey(fmt.Sprint(body["provider_key"]), provider)
	rt := strings.TrimSpace(fmt.Sprint(body["refresh_token"]))
	if rt == "" || rt == "<nil>" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "refresh_token required"})
		return
	}
	out, err := a.refreshAccessToken(providerKey, rt)
	if err != nil {
		logWarn("internal refresh 失败: %v", err)
		writeJSON(w, http.StatusBadGateway, map[string]any{"detail": truncate(err.Error(), 500)})
		return
	}
	access := strings.TrimSpace(fmt.Sprint(out["access_token"]))
	if access == "" || access == "<nil>" {
		writeJSON(w, http.StatusBadGateway, map[string]any{"detail": "no access_token in github response"})
		return
	}
	resp := map[string]any{"access_token": access}
	if nr := strings.TrimSpace(fmt.Sprint(out["refresh_token"])); nr != "" && nr != "<nil>" {
		resp["refresh_token"] = nr
	}
	writeJSON(w, http.StatusOK, resp)
}

func (a *App) refreshAccessToken(providerKey, refreshPlain string) (map[string]any, error) {
	if a != nil && a.RefreshAccessTokenFn != nil {
		return a.RefreshAccessTokenFn(providerKey, refreshPlain)
	}
	p, sp := domainParseProviderKey(providerKey, "github")
	if p == "gitlab" {
		cfg, err := a.gitlabProviderConfig(providerKey, sp)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(cfg.Website) == "" {
			return nil, fmt.Errorf("未找到 provider_key=%s 对应的配置 (website)", providerKey)
		}
		return infrastructure.RefreshGitLabToken(cfg, refreshPlain)
	}
	return infrastructure.RefreshGitHubToken(a.Cfg, refreshPlain)
}

// gitlabProviderConfig resolves YAML first, then Path A tenant rows in DB
// (gitlab:tenant-{company_id} is never in git-oauth-providers YAML).
func (a *App) gitlabProviderConfig(providerKey, serviceProvider string) (*infrastructure.ProviderConfig, error) {
	sp := strings.TrimSpace(strings.ToLower(serviceProvider))
	for _, row := range a.Cfg.GetProviderConfigs("gitlab") {
		if row.ServiceProvider == sp {
			cp := row
			return &cp, nil
		}
	}
	pc, err := a.resolveProviderByServiceProvider(sp)
	if err != nil {
		return nil, err
	}
	if pc == nil {
		return nil, fmt.Errorf("未找到 provider_key=%s 对应的配置", providerKey)
	}
	return pc, nil
}

func domainParseProviderKey(providerKey, fallback string) (string, string) {
	return splitPK(providerKey, fallback)
}

func splitPK(providerKey, fallback string) (string, string) {
	raw := strings.TrimSpace(strings.ToLower(providerKey))
	fb := strings.TrimSpace(strings.ToLower(fallback))
	if fb == "" {
		fb = "github"
	}
	if i := strings.Index(raw, ":"); i >= 0 {
		p := strings.TrimSpace(raw[:i])
		sp := strings.TrimSpace(raw[i+1:])
		if p == "" {
			p = fb
		}
		if sp == "" {
			sp = "default"
		}
		return p, sp
	}
	if raw == "" {
		return fb, "default"
	}
	return raw, "default"
}

func (a *App) handleTokenUseReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	provider := providerFromPath(r.URL.Path)
	body, err := readJSONNumbered(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "invalid json"})
		return
	}
	providerKey := normalizeProviderKey(fmt.Sprint(body["provider_key"]), provider)
	uid := strings.TrimSpace(fmt.Sprint(body["user_id"]))
	if uid == "" || uid == "<nil>" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "bad user_id"})
		return
	}
	audit, ok := body["audit"].(map[string]any)
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "audit object required"})
		return
	}
	if strings.TrimSpace(fmt.Sprint(audit["action"])) == "" || fmt.Sprint(audit["action"]) == "<nil>" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "audit.action required"})
		return
	}
	if err := a.writeTokenUseAudit(uid, providerKey, audit, "", requestTraceID(r)); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"detail": "audit_write_failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (a *App) handleCredentialDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	provider := providerFromPath(r.URL.Path)
	body, err := readJSONNumbered(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "invalid json"})
		return
	}
	providerKey := normalizeProviderKey(fmt.Sprint(body["provider_key"]), provider)
	uid := strings.TrimSpace(fmt.Sprint(body["user_id"]))
	if uid == "" || uid == "<nil>" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "bad user_id"})
		return
	}
	ghUID := ""
	if v := body["github_user_id"]; v != nil {
		s := strings.TrimSpace(fmt.Sprint(v))
		if s != "" && s != "<nil>" {
			ghUID = s
		}
	}
	n, err := a.DB.DeleteCredentials(providerKey, uid, ghUID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"detail": truncate(err.Error(), 500)})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "deleted_count": n})
}

func (a *App) handleConsumeGrantTicket(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body, err := readJSONNumbered(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "invalid json"})
		return
	}
	id := strings.TrimSpace(fmt.Sprint(body["id"]))
	uid := strings.TrimSpace(fmt.Sprint(body["user_id"]))
	gitsite := strings.TrimSpace(fmt.Sprint(body["gitsite"]))
	if id == "" || id == "<nil>" || uid == "" || uid == "<nil>" || gitsite == "" || gitsite == "<nil>" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "id, user_id, gitsite required"})
		return
	}
	remote, ok, err := a.DB.ConsumeGrantTicket(id, uid, gitsite)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"detail": truncate(err.Error(), 500)})
		return
	}
	if !ok {
		writeJSON(w, http.StatusConflict, map[string]any{"ok": false, "detail": "ticket_unusable"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "remote_user_id": remote})
}

func (a *App) handleCredentialUserIDs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	provider := providerFromPath(r.URL.Path)
	providerKey := normalizeProviderKey(r.URL.Query().Get("provider_key"), provider)
	ids, err := a.DB.ListUserIDs(providerKey)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"detail": truncate(err.Error(), 500)})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user_ids": ids})
}

func (a *App) handleCredentialSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	provider := providerFromPath(r.URL.Path)
	body, err := readJSONNumbered(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "invalid json"})
		return
	}
	providerKey := normalizeProviderKey(fmt.Sprint(body["provider_key"]), provider)
	uid := strings.TrimSpace(fmt.Sprint(body["user_id"]))
	if uid == "" || uid == "<nil>" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "bad user_id"})
		return
	}
	rows, err := a.DB.ListCredentialsForUser(providerKey, uid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"detail": truncate(err.Error(), 500)})
		return
	}
	if len(rows) == 0 {
		writeJSON(w, http.StatusOK, map[string]any{
			"connected": false, "github_user_id": nil, "github_login": nil,
			"scope": nil, "bind_status": nil, "bind_error": nil, "connections": []any{},
		})
		return
	}
	var connections []map[string]any
	for _, row := range rows {
		hasRefresh := strings.TrimSpace(row.RefreshTokenCipher) != ""
		item := map[string]any{
			"connected":      hasRefresh && row.BindStatus == "active",
			"github_user_id": row.RemoteUserID,
			"github_login":   nilIfEmpty(row.RemoteLogin),
			"scope":          nilIfEmpty(row.Scope),
			"bind_status":    nilIfEmpty(row.BindStatus),
			"bind_error":     nilIfEmpty(row.BindError),
			"updated_at":     nil,
		}
		if !row.UpdatedAt.IsZero() {
			item["updated_at"] = row.UpdatedAt.Format("2006-01-02T15:04:05.999999")
		}
		connections = append(connections, item)
	}
	primary := connections[0]
	for _, c := range connections {
		if c["connected"] == true {
			primary = c
			break
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"connected":      primary["connected"],
		"github_user_id": primary["github_user_id"],
		"github_login":   primary["github_login"],
		"scope":          primary["scope"],
		"bind_status":    primary["bind_status"],
		"bind_error":     primary["bind_error"],
		"connections":    connections,
	})
}

func isDirectGitHubAccessToken(plain string) bool {
	p := strings.TrimSpace(plain)
	return strings.HasPrefix(p, "ghu_") ||
		strings.HasPrefix(p, "gho_") ||
		strings.HasPrefix(p, "ghp_") ||
		strings.HasPrefix(p, "github_pat_")
}

func nilIfEmpty(s string) any {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return s
}

func (a *App) handleTaskAuditReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	provider := providerFromPath(r.URL.Path)
	body, err := readJSON(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "invalid json"})
		return
	}
	providerKey := normalizeProviderKey(fmt.Sprint(body["provider_key"]), provider)
	taskID := truncate(strings.TrimSpace(fmt.Sprint(body["task_id"])), 64)
	if taskID == "" || taskID == "<nil>" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "task_id required"})
		return
	}
	workspaceID := truncate(strings.TrimSpace(fmt.Sprint(body["workspace_id"])), 64)
	if workspaceID == "<nil>" {
		workspaceID = ""
	}
	action := truncate(strings.TrimSpace(fmt.Sprint(body["action"])), 64)
	if action == "" || action == "<nil>" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "action required"})
		return
	}
	detail, _ := body["detail"].(map[string]any)
	if err := a.DB.InsertTaskAudit(providerKey, taskID, workspaceID,
		optionalInt64(body["company_id"]), optionalInt64(body["user_id"]),
		action, sanitizeAuditDetail(detail)); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"detail": "audit_write_failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// handleUserAppConnection serves the FE convention path /api/git-oauth/user-app-connection/
// (replaces Django /api/user/{userId}/accounts/{provider}/app/connection/, OPT-049 migration).
// userID comes from the gateway-injected X-User-Id header (X-Auth-User-Id alias); provider is
// resolved from query `provider`, else the service_provider config, else the repo_url host,
// else "github" (legacy default). GET returns connection status; DELETE disconnects.
func (a *App) handleUserAppConnection(w http.ResponseWriter, r *http.Request) {
	uid := strings.TrimSpace(a.effectiveUserID(r))
	if uid == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"detail": "未认证"})
		return
	}
	sp := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("service_provider")))
	if sp == "" || sp == "default" {
		sp = "default"
	}
	providerKey := a.resolveUserAppConnectionProvider(r, sp) + ":" + sp
	repoURL := strings.TrimSpace(r.URL.Query().Get("repo_url"))

	switch r.Method {
	case http.MethodGet:
		// 检查键扩展为配置存储键：任务详情页只传 repo_url（sp 缺省 "default"），
		// 而凭据按配置 service_provider 存储（github:github-official-daydaymoney /
		// gitlab:daydaymoney-gitlab），不扩展会查不到 → 已授权仓库误显 "OAuth 绑定"。
		// repo_url 能匹配到某一实例 website 时只查该实例，避免绑了 gitlab:tencent-sh-1
		// 却把另一 GitLab 仓也当成 connected。
		a.handleUserAppConnectionGet(w, r, a.userAppConnectionLookupKeys(providerKey, repoURL), uid)
	case http.MethodDelete:
		a.handleUserAppConnectionDelete(w, r, providerKey, uid)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"detail": "method not allowed"})
	}
}

func (a *App) handleUserAppConnectionGet(w http.ResponseWriter, r *http.Request, providerKeys []string, userID string) {
	rows, err := a.DB.ListCredentialsForUserKeys(providerKeys, userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"detail": truncate(err.Error(), 500)})
		return
	}
	probe := queryBoolFlag(r.URL.Query().Get("probe_access_token"))
	if len(rows) == 0 {
		body := map[string]any{
			"connected": false, "github_user_id": nil, "github_login": nil,
			"scope": nil, "bind_status": nil, "bind_error": nil, "connections": []any{},
		}
		if probe {
			body["access_token_valid"] = false
			logInfo("event=git_oauth_access_token_probe valid=false reason=not_connected trace_id=%s", requestTraceID(r))
		}
		writeJSON(w, http.StatusOK, body)
		return
	}
	var connections []map[string]any
	for _, row := range rows {
		hasRefresh := strings.TrimSpace(row.RefreshTokenCipher) != ""
		item := map[string]any{
			"connected":      hasRefresh && row.BindStatus == "active",
			"github_user_id": row.RemoteUserID,
			"github_login":   nilIfEmpty(row.RemoteLogin),
			"scope":          nilIfEmpty(row.Scope),
			"bind_status":    nilIfEmpty(row.BindStatus),
			"bind_error":     nilIfEmpty(row.BindError),
			"updated_at":     nil,
		}
		if !row.UpdatedAt.IsZero() {
			item["updated_at"] = row.UpdatedAt.Format("2006-01-02T15:04:05.999999")
		}
		connections = append(connections, item)
	}
	primary := connections[0]
	for _, c := range connections {
		if c["connected"] == true {
			primary = c
			break
		}
	}
	body := map[string]any{
		"connected":      primary["connected"],
		"github_user_id": primary["github_user_id"],
		"github_login":   primary["github_login"],
		"scope":          primary["scope"],
		"bind_status":    primary["bind_status"],
		"bind_error":     primary["bind_error"],
		"connections":    connections,
	}
	if probe {
		valid := false
		networkStatus := ""
		if row := pickActiveCredential(rows); row != nil {
			var skipToken bool
			networkStatus, skipToken = a.gitlabOAuthProbeNetwork(r.URL.Query().Get("repo_url"))
			if skipToken {
				if networkStatus == domain.ReachabilitySkippedIntranet {
					valid = true
					logInfo("event=git_oauth_access_token_probe valid=true reason=skipped_intranet provider=%s trace_id=%s",
						credentialProviderKey(row), requestTraceID(r))
				} else {
					logWarn("event=git_oauth_access_token_probe valid=false reason=gitlab_unreachable provider=%s trace_id=%s",
						credentialProviderKey(row), requestTraceID(r))
				}
			} else {
				valid, _ = a.probeUserAppAccessToken(requestTraceID(r), userID, row)
			}
		} else {
			logInfo("event=git_oauth_access_token_probe valid=false reason=not_connected trace_id=%s", requestTraceID(r))
		}
		body["access_token_valid"] = valid
		if networkStatus != "" {
			body["network_status"] = networkStatus
		}
		delete(body, "access_token")
	}
	writeJSON(w, http.StatusOK, body)
}

func (a *App) handleUserAppConnectionDelete(w http.ResponseWriter, r *http.Request, providerKey string, userID string) {
	deleted, err := a.DB.DeleteCredentials(providerKey, userID, "")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"detail": truncate(err.Error(), 500)})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "deleted_count": deleted})
}
