// Package interfaces contains HTTP handlers. These are thin adapters that
// parse requests, invoke application services, and write responses.
package interfaces

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"taskCredentialService/domain"
	"taskCredentialService/infrastructure"
)

// Handlers holds HTTP handler methods with application services injected.
type Handlers struct {
	svc *infrastructure.AppServices
	// fetchActiveContextPack overrides AIComment lookup in tests; nil uses svc.AIComment.
	fetchActiveContextPack func(taskID string) map[string]interface{}
}

// NewHandlers creates handler set with assembled services.
func NewHandlers(svc *infrastructure.AppServices) *Handlers {
	return &Handlers{svc: svc}
}

func (h *Handlers) activeContextPack(taskID string) map[string]interface{} {
	if h.fetchActiveContextPack != nil {
		return h.fetchActiveContextPack(taskID)
	}
	if h.svc == nil {
		return nil
	}
	return infrastructure.SoftFetchActiveContextPack(h.svc.AIComment, taskID)
}

// RegisterRoutes sets up the HTTP route table.
func (h *Handlers) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/health", h.handleHealth)

	// Container callback endpoints — same URL paths as Django.
	mux.HandleFunc("/api/tenant/", h.handleContainerAPI)

	// Internal token init: POST /v1/token/init/tenant/{tid}/workspace/{wid}/task/{tk}/comment/{cid}
	mux.HandleFunc("/v1/token/init", h.handleTokenInitLegacyGone)
	mux.HandleFunc("/v1/token/init/", h.handleTokenInit)

	// Internal token validation endpoint for Django downstream views.
	mux.HandleFunc("/v1/token/validate", h.handleValidateToken)

	// Read-only endpoint: get current access token by scope (no mutation).
	mux.HandleFunc("/v1/token/by-scope", h.handleGetTokenByScope)

	// Internal audit append for relay lifecycle event consumer.
	mux.HandleFunc("/v1/audit/append", h.handleAuditAppend)
}

func (h *Handlers) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleContainerAPI routes container callback requests by path suffix.
func (h *Handlers) handleContainerAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"detail": "method not allowed"})
		return
	}
	// Parse tenant/workspace/task/comment/{cid} from URL path:
	// /api/tenant/{tid}/workspace/{wid}/task/{tk}/comment/{cid}/cloud/server-container-token/{action}/
	tenantID, workspaceID, taskID, pathCommentID, action, ok := parseContainerAPIPath(r.URL.Path)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"detail": "not found"})
		return
	}

	switch action {
	case "repo-clone-credentials":
		h.handleRepoCloneCredentials(w, r, tenantID, workspaceID, taskID, pathCommentID)
	case "task-detail":
		h.handleTaskDetail(w, r, tenantID, workspaceID, taskID, pathCommentID)
	case "layer-github-oauth-access-tokens":
		h.handleLayerOauthTokens(w, r, tenantID, workspaceID, taskID, pathCommentID)
	case "exchange-refresh":
		h.handleExchangeRefresh(w, r, tenantID, workspaceID, taskID)
	case "refresh-access":
		h.handleRefreshAccess(w, r, tenantID, workspaceID, taskID)
	default:
		writeJSON(w, http.StatusNotFound, map[string]string{"detail": "unknown action: " + action})
	}
}

func (h *Handlers) handleRepoCloneCredentials(w http.ResponseWriter, r *http.Request, tenantID, workspaceID, taskID, pathCommentID string) {
	var req struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "invalid json"})
		return
	}
	scope := domain.TaskScope{TenantID: tenantID, WorkspaceID: workspaceID, TaskID: taskID}
	token, err := h.svc.Credential.ValidateToken(req.AccessToken, scope)
	if err != nil {
		if de, ok := err.(*domain.DomainError); ok {
			writeJSON(w, statusFromDomainError(de.Code), map[string]string{"detail": de.Message, "error_code": de.Code})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": err.Error()})
		return
	}
	if rejectCommentScopeMismatch(w, pathCommentID, tokenCommentID(token)) {
		return
	}
	result, err := h.svc.Credential.BuildRepoCloneCredentials(req.AccessToken, scope)
	if err != nil {
		if de, ok := err.(*domain.DomainError); ok {
			writeJSON(w, statusFromDomainError(de.Code), map[string]string{"detail": de.Message, "error_code": de.Code})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": err.Error()})
		return
	}
	if len(result.TokenRefreshFailures) > 0 {
		writeJSON(w, http.StatusBadGateway, map[string]interface{}{
			"detail":                   "token refresh failed",
			"error_code":               "REPO_CLONE_TOKEN_REFRESH_FAILED",
			"token_refresh_failures":   result.TokenRefreshFailures,
			"missing_repo_credentials": result.MissingIdentityRepos,
		})
		return
	}
	if len(result.MissingIdentityRepos) > 0 {
		writeJSON(w, http.StatusConflict, map[string]interface{}{
			"detail":                   "repo clone credentials incomplete",
			"error_code":               "REPO_CLONE_CREDENTIALS_INCOMPLETE",
			"missing_repo_credentials": result.MissingIdentityRepos,
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"company_id":             tenantID,
		"workspace_id":           workspaceID,
		"task_id":                taskID,
		"repo_clone_credentials": result.Credentials,
		"repo_count":             len(result.Credentials),
	})
}

func (h *Handlers) handleTaskDetail(w http.ResponseWriter, r *http.Request, tenantID, workspaceID, taskID, pathCommentID string) {
	var req struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "invalid json"})
		return
	}
	scope := domain.TaskScope{TenantID: tenantID, WorkspaceID: workspaceID, TaskID: taskID}
	token, err := h.svc.Credential.ValidateToken(req.AccessToken, scope)
	if err != nil {
		if de, ok := err.(*domain.DomainError); ok {
			writeJSON(w, statusFromDomainError(de.Code), map[string]string{"detail": de.Message})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": err.Error()})
		return
	}
	if rejectCommentScopeMismatch(w, pathCommentID, tokenCommentID(token)) {
		return
	}
	commentID := tokenCommentID(token)
	detail, err := h.svc.TaskDetail.FetchTaskDetail(taskID, commentID)
	if err != nil {
		log.Printf("[task-credential-service] task-detail failed task=%s: %v", taskID, err)
		writeJSON(w, http.StatusBadGateway, map[string]string{
			"detail":     "无法从任务服务加载任务详情（快照缺失），请检查 taskTaskService 与业务库",
			"error_code": "TASK_SNAPSHOT_MISSING",
		})
		return
	}
	// 路径中的租户/工作空间/任务优先（与 Django container task-detail 一致）
	detail.CompanyID = tenantID
	detail.WorkspaceID = workspaceID
	detail.TaskID = taskID
	h.svc.TaskDetail.AttachIdlePolicy(detail, commentID)
	if detail.Task != nil {
		params := map[string]interface{}{}
		if detail.Task.BranchStrategy != nil {
			params["branch_strategy"] = detail.Task.BranchStrategy
		}
		var repoBranchPlans []domain.RepoBranchEntry
		for _, pr := range detail.ProjectRepos {
			repoBranchPlans = append(repoBranchPlans, pr.RepoBranches...)
		}
		if repoBranchPlans == nil {
			repoBranchPlans = []domain.RepoBranchEntry{}
		}
		repoGitIdentities := detail.RepoGitIdentities
		if repoGitIdentities == nil {
			repoGitIdentities = []domain.RepoGitIdentityDTO{}
		}
		resp := map[string]interface{}{
			"company_id":           detail.CompanyID,
			"workspace_id":         detail.WorkspaceID,
			"task_id":              detail.TaskID,
			"idle_recycle_minutes": detail.IdleRecycleMinutes,
			"instruction_idle":     detail.InstructionIdle,
			"task": map[string]interface{}{
				"id":                 detail.Task.ID,
				"title":              detail.Task.Title,
				"description":        detail.Task.Description,
				"parameters":         params,
				"target_branch":      detail.Task.TargetBranch,
				"auto_run":           detail.Task.AutoRun,
				"installed_image_id": strings.TrimSpace(detail.Task.InstalledImageID),
			},
			"project_repos":       detail.ProjectRepos,
			"repo_branch_plans":   repoBranchPlans,
			"repo_git_identities": repoGitIdentities,
		}
		if len(detail.MachineReleaseSTS) > 0 {
			resp["machine_release_sts"] = detail.MachineReleaseSTS
		}
		if pack := h.activeContextPack(taskID); pack != nil {
			if v, ok := pack["at_mention_run"]; ok {
				resp["at_mention_run"] = v
			}
			if v, ok := pack["comment_thread"]; ok {
				resp["comment_thread"] = v
			}
		} else if deferred := synthesizeDeferredAtMentionRun(commentID, detail.Task); deferred != nil {
			resp["at_mention_run"] = deferred
		}
		writeJSON(w, http.StatusOK, resp)
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (h *Handlers) handleLayerOauthTokens(w http.ResponseWriter, r *http.Request, tenantID, workspaceID, taskID, pathCommentID string) {
	var req struct {
		AccessToken   string   `json:"access_token"`
		RepoMatchKeys []string `json:"repo_match_keys"`
		RepoSlugs     []string `json:"repo_slugs"`
		TargetBranch  string   `json:"target_branch"` // 可选；响应附带 pr_base_branch / pr_title（merge_target）
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "invalid json"})
		return
	}
	accessToken := strings.TrimSpace(req.AccessToken)
	if accessToken == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "access_token 必填"})
		return
	}

	scope := domain.TaskScope{TenantID: tenantID, WorkspaceID: workspaceID, TaskID: taskID}
	token, tokErr := h.svc.Credential.ValidateToken(accessToken, scope)
	if tokErr != nil {
		if de, ok := tokErr.(*domain.DomainError); ok {
			switch de.Code {
			case "TOKEN_NOT_FOUND", "TOKEN_EXPIRED":
				writeJSON(w, http.StatusUnauthorized, map[string]string{
					"detail":     "无效的 access_token",
					"error_code": "TOKEN_ACCESS_INVALID",
				})
			case "SCOPE_MISMATCH":
				writeJSON(w, http.StatusForbidden, map[string]string{
					"detail":     "URL 中的租户/工作空间/任务与令牌不匹配",
					"error_code": "TOKEN_SCOPE_MISMATCH",
				})
			default:
				writeJSON(w, http.StatusBadRequest, map[string]string{"detail": de.Message, "error_code": de.Code})
			}
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": tokErr.Error()})
		return
	}
	if rejectCommentScopeMismatch(w, pathCommentID, tokenCommentID(token)) {
		return
	}
	result, err := h.svc.LayerOauth.ResolveLayerOauthTokens(accessToken, scope, req.RepoMatchKeys, req.RepoSlugs)
	if err != nil {
		if de, ok := err.(*domain.DomainError); ok {
			switch de.Code {
			case "TOKEN_NOT_FOUND", "TOKEN_EXPIRED":
				writeJSON(w, http.StatusUnauthorized, map[string]string{
					"detail":     "无效的 access_token",
					"error_code": "TOKEN_ACCESS_INVALID",
				})
			case "SCOPE_MISMATCH":
				writeJSON(w, http.StatusForbidden, map[string]string{
					"detail":     "URL 中的租户/工作空间/任务与令牌不匹配",
					"error_code": "TOKEN_SCOPE_MISMATCH",
				})
			default:
				writeJSON(w, http.StatusBadRequest, map[string]string{"detail": de.Message, "error_code": de.Code})
			}
			return
		}
		log.Printf("[task-credential-service] layer-oauth failed task=%s: %v", taskID, err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": err.Error()})
		return
	}

	if !result.OK {
		writeJSON(w, http.StatusConflict, map[string]interface{}{
			"ok":                  false,
			"detail":              result.Detail,
			"github_auth_by_repo": result.GithubAuthByRepo,
			"error_code":          result.ErrorCode,
			"failed_stage":        result.FailedStage,
			"retryable":           result.Retryable,
			"detail_safe":         result.DetailSafe,
		})
		return
	}

	payload := map[string]interface{}{
		"ok":                  true,
		"github_auth_by_repo": result.GithubAuthByRepo,
	}
	if len(result.GitAuthByRepoMatchKey) > 0 {
		payload["git_auth_by_repo_match_key"] = result.GitAuthByRepoMatchKey
	}
	if result.PartialError != "" {
		payload["partial_error"] = result.PartialError
	}
	// PR 元数据：供容器 oauth-refresh-push / auto_run 交付创建 PR（与 Django 契约对齐）
	if h.svc != nil && h.svc.TaskDetail != nil {
		if detail, err := h.svc.TaskDetail.FetchTaskDetail(taskID, ""); err == nil && detail != nil && detail.Task != nil {
			if title := strings.TrimSpace(detail.Task.Title); title != "" {
				payload["pr_title"] = title
			}
			if detail.Task.BranchStrategy != nil {
				if base := strings.TrimSpace(detail.Task.BranchStrategy.MergeTargetBranchName); base != "" {
					payload["pr_base_branch"] = base
				}
			}
		}
	}
	if tb := strings.TrimSpace(req.TargetBranch); tb != "" {
		payload["target_branch"] = tb
	}
	writeJSON(w, http.StatusOK, payload)
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

func statusFromDomainError(code string) int {
	switch code {
	case "TOKEN_NOT_FOUND", "TOKEN_EXPIRED":
		return http.StatusUnauthorized
	case "SCOPE_MISMATCH":
		return http.StatusForbidden
	default:
		return http.StatusBadRequest
	}
}

func tokenCommentID(token *domain.ContainerToken) string {
	if token == nil {
		return ""
	}
	return strings.TrimSpace(token.CommentID)
}

// synthesizeDeferredAtMentionRun builds parent+image pack when no pending agent comment exists
// (OPT-20260823-008: container POSTs the agent row after bootstrap).
func synthesizeDeferredAtMentionRun(parentCommentID string, task *domain.TaskSnapshot) map[string]interface{} {
	parent := strings.TrimSpace(parentCommentID)
	if parent == "" || parent == "-" || task == nil {
		return nil
	}
	imageID := strings.TrimSpace(task.InstalledImageID)
	if imageID == "" {
		return nil
	}
	source := "at_mention"
	if task.AutoRun {
		source = "auto_run"
	}
	return map[string]interface{}{
		"parent_comment_id": parent,
		"installed_image":   map[string]string{"id": imageID},
		"source":            source,
	}
}

func commentScopeMismatch(pathCommentID, tokenCmt string) bool {
	p := strings.TrimSpace(pathCommentID)
	t := strings.TrimSpace(tokenCmt)
	if p == "" || p == "-" || t == "" || t == "-" {
		return false
	}
	return p != t
}

func rejectCommentScopeMismatch(w http.ResponseWriter, pathCommentID, tokenCmt string) bool {
	if !commentScopeMismatch(pathCommentID, tokenCmt) {
		return false
	}
	writeJSON(w, http.StatusForbidden, map[string]string{
		"detail":     "URL 中的评论与令牌不匹配",
		"error_code": "TOKEN_SCOPE_MISMATCH",
	})
	return true
}
