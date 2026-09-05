package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
)

// handleInternalLayerGitPushPrepare implements POST /api/internal/layer-git-push/prepare
// Response shape matches Django prepare_layer_git_push_view:
//
//	{ok:true, push_body, use_oauth_access_push} or {ok:false, status, detail}
//
// prefer_container_remote=true 仍会 best-effort 换票：多仓前端可能清空 identity_id，
// 容器内 HTTPS origin 并无持久凭据；若能换到 OAuth token 则走 oauth-access-push。
// 无 token 时默认返回 409（避免裸 git/push 触发
// "could not read Username for 'https://github.com': terminal prompts disabled"）。
// 仅当 allow_bare_git_push=true（本地 SSH / 容器已配凭据）才回退裸 git/push。
func handleInternalLayerGitPushPrepare(w http.ResponseWriter, r *http.Request) {
	if !requireInternalSecret(r) {
		writeJSON(w, http.StatusForbidden, map[string]any{"ok": false, "detail": "forbidden"})
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "detail": "method not allowed"})
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "detail": "invalid json"})
		return
	}

	tenantID := strField(body, "tenant_id")
	workspaceID := strField(body, "workspace_id")
	taskID := strField(body, "task_id")
	layerID := strField(body, "layer_id")
	userID := strField(body, "user_id")
	identityID := strField(body, "identity_id")
	repoURL := strField(body, "repo_url")
	preferRemote := false
	if v, ok := body["prefer_container_remote"]; ok {
		switch t := v.(type) {
		case bool:
			preferRemote = t
		case string:
			preferRemote = strings.EqualFold(strings.TrimSpace(t), "true") || t == "1"
		}
	}
	allowBareGitPush := false
	if v, ok := body["allow_bare_git_push"]; ok {
		switch t := v.(type) {
		case bool:
			allowBareGitPush = t
		case string:
			allowBareGitPush = strings.EqualFold(strings.TrimSpace(t), "true") || t == "1"
		}
	}

	if tenantID == "" || taskID == "" || layerID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"ok": false, "status": 400, "detail": "tenant_id, task_id, layer_id required",
		})
		return
	}
	if userID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"ok": false, "status": 401, "detail": "user_id required / user not found",
		})
		return
	}
	// Container Gateway internal-secret bypass used to stamp user_id=internal_gateway.
	// Lookup by that sentinel always 404s as "Git 身份不存在" even when the browser user
	// has a valid identity. Fail closed with a distinct 401 so the miss is not a fake 404.
	if userID == "internal" || userID == "internal_gateway" {
		log.Printf("[taskCloudService] layer-git-push sentinel user_id=%s identity_id=%s tenant_id=%s",
			userID, identityID, tenantID)
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"ok":     false,
			"status": 401,
			"detail": "内部推送缺少真实用户身份，无法查找 Git 身份",
		})
		return
	}

	pushBody := map[string]any{}
	if tb := body["target_branch"]; tb != nil {
		if s, ok := tb.(string); ok && strings.TrimSpace(s) != "" {
			pushBody["target_branch"] = strings.TrimSpace(s)
		} else if !ok {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok": false, "status": 400, "detail": "target_branch 必须为字符串",
			})
			return
		}
	}

	// Priority 1: reuse client-supplied github_auth_by_repo.
	reused := normalizeGithubAuthByRepo(body["github_auth_by_repo"])
	if len(reused) > 0 {
		pushBody["github_auth_by_repo"] = reused
		writeLayerGitPushPrepareOK(w, pushBody, tenantID, workspaceID, taskID, true)
		return
	}

	if identityID != "" && !preferRemote {
		found, err := lookupUserCompanyGitIdentity(r.Context(), identityID, userID, tenantID)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]any{
				"ok": false, "status": 502, "detail": fmt.Sprintf("saas identity lookup failed: %v", err),
			})
			return
		}
		if !found {
			log.Printf("[taskCloudService] layer-git-push identity not found identity_id=%s user_id=%s tenant_id=%s",
				identityID, userID, tenantID)
			writeJSON(w, http.StatusNotFound, map[string]any{
				"ok":     false,
				"status": 404,
				"detail": "Git 身份不存在或不属于当前租户",
			})
			return
		}
	} else if identityID != "" && preferRemote {
		log.Printf("[taskCloudService] layer-git-push prefer_container_remote=true, skip identity gate identity_id=%s (still attempt OAuth)", identityID)
	}

	githubAuth, oauthAuth, errDetail := resolveLayerGitPushOauthMaps(r.Context(), tenantID, workspaceID, taskID, userID, repoURL)
	if len(githubAuth) > 0 {
		pushBody["github_auth_by_repo"] = githubAuth
	}
	if len(oauthAuth) > 0 {
		pushBody["oauth_auth_by_repo"] = oauthAuth
	}
	if len(githubAuth) > 0 || len(oauthAuth) > 0 {
		log.Printf(
			"[taskCloudService] layer-git-push prepare oauth ok layer_id=%s github_repos=%d gitlab_repos=%d",
			layerID, len(githubAuth), len(oauthAuth),
		)
		writeLayerGitPushPrepareOK(w, pushBody, tenantID, workspaceID, taskID, true)
		return
	}

	if preferRemote && allowBareGitPush {
		log.Printf("[taskCloudService] layer-git-push prefer_container_remote + allow_bare_git_push: fallback bare git/push layer_id=%s", layerID)
		writeLayerGitPushPrepareOK(w, pushBody, tenantID, workspaceID, taskID, false)
		return
	}

	detail := errDetail
	if detail == "" {
		detail = "未能换取 Git OAuth 凭据，无法推送到 HTTPS 远端。请重新完成 GitHub/GitLab 授权后重试。"
	}
	// Bridge secret mismatch surfaces as gitOauth {"detail":"unauthorized"}; map to actionable guidance.
	if strings.Contains(strings.ToLower(detail), "unauthorized") {
		detail = "未能换取 Git OAuth 凭据（gitOauth bridge secret 校验失败或授权已失效）。请确认 taskCloudService 与 taskGitOauth 的 bridgeSecret 一致，或重新完成 GitHub/GitLab 授权后重试。"
	}
	status := http.StatusConflict
	if errDetail != "" && !strings.Contains(errDetail, "未找到") && !strings.Contains(errDetail, "not found") &&
		!strings.Contains(errDetail, "凭据") {
		status = http.StatusBadGateway
	}
	log.Printf(
		"[taskCloudService] layer-git-push prepare oauth failed layer_id=%s status=%d detail=%s",
		layerID, status, truncateForLog(detail, 300),
	)
	writeJSON(w, status, map[string]any{
		"ok":     false,
		"status": status,
		"detail": detail,
	})
}

// writeLayerGitPushPrepareOK attaches PR metadata into push_body then writes ok response.
func writeLayerGitPushPrepareOK(
	w http.ResponseWriter,
	pushBody map[string]any,
	tenantID, workspaceID, taskID string,
	useOauthAccessPush bool,
) {
	attachLayerGitPushPRMetadata(pushBody, tenantID, workspaceID, taskID)
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":                    true,
		"push_body":             pushBody,
		"use_oauth_access_push": useOauthAccessPush,
	})
}

// handleInternalLayerGitPushComplete implements POST /api/internal/layer-git-push/complete
// Best-effort: log and return ok; never calls Django.
func handleInternalLayerGitPushComplete(w http.ResponseWriter, r *http.Request) {
	if !requireInternalSecret(r) {
		writeJSON(w, http.StatusForbidden, map[string]any{"ok": false, "detail": "forbidden"})
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "detail": "method not allowed"})
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "detail": "invalid json"})
		return
	}
	tenantID := strField(body, "tenant_id")
	taskID := strField(body, "task_id")
	if tenantID == "" || taskID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"ok": false, "detail": "tenant_id and task_id required",
		})
		return
	}
	log.Printf("[taskCloudService] layer-git-push complete best-effort tenant=%s task=%s layer=%s user=%s",
		tenantID, taskID, strField(body, "layer_id"), strField(body, "user_id"))
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func normalizeGithubAuthByRepo(raw any) map[string]string {
	m, ok := raw.(map[string]any)
	if !ok || len(m) == 0 {
		// json.Unmarshal may produce map[string]interface{}; also accept already-typed via round-trip
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		slug := strings.TrimSpace(k)
		token := strings.TrimSpace(fmt.Sprintf("%v", v))
		if slug == "" || token == "" || token == "<nil>" {
			continue
		}
		out[slug] = token
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
