package interfaces

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"taskCredentialService/domain"
)

func (h *Handlers) handleTokenInitLegacyGone(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusNotFound, map[string]string{
		"detail": "use /v1/token/init/tenant/{tenantId}/workspace/{workspaceId}/task/{taskId}/comment/{commentId}",
	})
}

// parseTokenInitPath extracts scope from:
// /v1/token/init/tenant/{tid}/workspace/{wid}/task/{tk}/comment/{cid}
func parseTokenInitPath(path string) (tenantID, workspaceID, taskID, commentID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 11 {
		return "", "", "", "", false
	}
	if parts[0] != "v1" || parts[1] != "token" || parts[2] != "init" {
		return "", "", "", "", false
	}
	if parts[3] != "tenant" || parts[5] != "workspace" || parts[7] != "task" || parts[9] != "comment" {
		return "", "", "", "", false
	}
	tenantID = strings.TrimSpace(parts[4])
	workspaceID = strings.TrimSpace(parts[6])
	taskID = strings.TrimSpace(parts[8])
	commentID = strings.TrimSpace(parts[10])
	if tenantID == "" || workspaceID == "" || taskID == "" || commentID == "" || commentID == "-" {
		return "", "", "", "", false
	}
	return tenantID, workspaceID, taskID, commentID, true
}

func commentIDFromInitRequest(r *http.Request, pathCommentID string) (string, error) {
	if c := strings.TrimSpace(pathCommentID); c != "" {
		return c, nil
	}
	if c := strings.TrimSpace(r.URL.Query().Get("comment_id")); c != "" {
		return c, nil
	}
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		return "", err
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return "", nil
	}
	var body struct {
		CommentID string `json:"comment_id"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return "", err
	}
	return strings.TrimSpace(body.CommentID), nil
}

func (h *Handlers) handleTokenInit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, nil)
		return
	}
	tenantID, workspaceID, taskID, pathCommentID, ok := parseTokenInitPath(r.URL.Path)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"detail": "expected /v1/token/init/tenant/{tenantId}/workspace/{workspaceId}/task/{taskId}/comment/{commentId}",
		})
		return
	}
	commentID, err := commentIDFromInitRequest(r, pathCommentID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "invalid json"})
		return
	}
	if commentID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"detail":     "comment_id is required",
			"error_code": "COMMENT_ID_REQUIRED",
		})
		return
	}
	scope := domain.TaskScope{
		TenantID:    tenantID,
		WorkspaceID: workspaceID,
		TaskID:      taskID,
		CommentID:   commentID,
	}
	token, err := h.svc.Token.IssueToken(scope, nil)
	if err != nil {
		if de, ok := err.(*domain.DomainError); ok && de.Code == "COMMENT_ID_REQUIRED" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"detail": de.Message, "error_code": de.Code})
			return
		}
		log.Printf("[task-credential-service] token init failed tenant=%s task=%s comment=%s: %v", tenantID, taskID, commentID, err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": err.Error()})
		return
	}
	log.Printf("[task-credential-service] token init ok tenant=%s workspace=%s task=%s comment=%s", tenantID, workspaceID, taskID, commentID)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":        "ok",
		"access_token":  token.ContainerAccessToken,
		"refresh_token": token.ContainerRefreshToken,
		"expires_at":    token.ContainerAccessTokenExpiresAt,
		"comment_id":    token.CommentID,
	})
}

func (h *Handlers) handleValidateToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"detail": "method not allowed"})
		return
	}
	var req struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "invalid json"})
		return
	}
	accessToken := strings.TrimSpace(req.AccessToken)
	if accessToken == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "access_token required"})
		return
	}

	token, err := h.svc.Token.FindByAccessToken(accessToken)
	if err != nil || token == nil {
		log.Printf("[task-credential-service] validate-token: token not found")
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"detail":     "无效的 access_token",
			"error_code": "TOKEN_ACCESS_INVALID",
		})
		return
	}
	if h.svc.Token.IsExpired(token) {
		log.Printf("[task-credential-service] validate-token: token expired task=%s", token.TaskID)
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"detail":     "access_token 已过期",
			"error_code": "TOKEN_ACCESS_EXPIRED",
		})
		return
	}

	log.Printf("[task-credential-service] validate-token: OK task=%s company=%s comment=%s", token.TaskID, token.CompanyID, token.CommentID)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"valid":        true,
		"company_id":   token.CompanyID,
		"workspace_id": token.WorkspaceID,
		"task_id":      token.TaskID,
		"comment_id":   token.CommentID,
		"expires_at":   token.ContainerAccessTokenExpiresAt,
	})
}

func (h *Handlers) handleExchangeRefresh(w http.ResponseWriter, r *http.Request, tenantID, workspaceID, taskID string) {
	var req struct {
		AccessToken         string `json:"access_token"`
		BusinessAPIEndpoint string `json:"business_api_endpoint"`
		CommentID           string `json:"comment_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "invalid json"})
		return
	}
	accessToken := strings.TrimSpace(req.AccessToken)
	if accessToken == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"detail":     "access_token 必填",
			"error_code": "TOKEN_ACCESS_MISSING",
		})
		return
	}
	businessAPIEndpoint := strings.TrimSpace(req.BusinessAPIEndpoint)
	if businessAPIEndpoint == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"detail":     "business_api_endpoint 必填",
			"error_code": "BUSINESS_API_ENDPOINT_MISSING",
		})
		return
	}

	scope := domain.TaskScope{TenantID: tenantID, WorkspaceID: workspaceID, TaskID: taskID, CommentID: strings.TrimSpace(req.CommentID)}
	refreshToken, err := h.svc.Token.ExchangeRefresh(accessToken, businessAPIEndpoint, scope)
	if err != nil {
		de, ok := err.(*domain.DomainError)
		if !ok {
			log.Printf("[task-credential-service] exchange-refresh: internal error task=%s: %v", taskID, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": err.Error()})
			return
		}
		switch de.Code {
		case "TOKEN_NOT_FOUND", "TOKEN_EXPIRED":
			writeJSON(w, http.StatusUnauthorized, map[string]string{
				"detail":     "无效的 access_token",
				"error_code": "TOKEN_ACCESS_INVALID",
			})
		case "TOKEN_EXCHANGE_ALREADY_DONE":
			writeJSON(w, http.StatusForbidden, map[string]string{
				"detail":     de.Message + "，请使用 server-container-token/refresh-access/ 接口",
				"error_code": de.Code,
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

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"refresh_token": refreshToken,
		"task_id":       taskID,
		"comment_id":    scope.CommentID,
	})
}

func (h *Handlers) handleRefreshAccess(w http.ResponseWriter, r *http.Request, tenantID, workspaceID, taskID string) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
		CommentID    string `json:"comment_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "invalid json"})
		return
	}
	refreshToken := strings.TrimSpace(req.RefreshToken)
	if refreshToken == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"detail":     "refresh_token 必填",
			"error_code": "TOKEN_REFRESH_MISSING",
		})
		return
	}

	scope := domain.TaskScope{TenantID: tenantID, WorkspaceID: workspaceID, TaskID: taskID, CommentID: strings.TrimSpace(req.CommentID)}
	newAccessToken, expiresAt, err := h.svc.Token.RefreshAccess(refreshToken, scope)
	if err != nil {
		de, ok := err.(*domain.DomainError)
		if !ok {
			log.Printf("[task-credential-service] refresh-access: internal error task=%s: %v", taskID, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": err.Error()})
			return
		}
		switch de.Code {
		case "TOKEN_NOT_FOUND":
			writeJSON(w, http.StatusUnauthorized, map[string]string{
				"detail":     "无效的 refresh_token",
				"error_code": "TOKEN_REFRESH_INVALID",
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

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"access_token": newAccessToken,
		"expires_at":   expiresAt,
		"task_id":      taskID,
		"comment_id":   scope.CommentID,
	})
}

func (h *Handlers) handleGetTokenByScope(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"detail": "method not allowed"})
		return
	}
	q := r.URL.Query()
	tenantID := strings.TrimSpace(q.Get("tenant_id"))
	workspaceID := strings.TrimSpace(q.Get("workspace_id"))
	taskID := strings.TrimSpace(q.Get("task_id"))
	commentID := strings.TrimSpace(q.Get("comment_id"))

	scope := domain.TaskScope{
		TenantID:    tenantID,
		WorkspaceID: workspaceID,
		TaskID:      taskID,
		CommentID:   commentID,
	}
	if err := scope.Validate(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"detail":     "tenant_id, workspace_id, task_id are required",
			"error_code": "INVALID_SCOPE",
		})
		return
	}

	token, err := h.svc.Token.EnsureAccessByScope(scope)
	if err != nil {
		if de, ok := err.(*domain.DomainError); ok {
			switch de.Code {
			case "TOKEN_NOT_FOUND":
				writeJSON(w, http.StatusNotFound, map[string]string{
					"detail":     "no token found for this scope",
					"error_code": "TOKEN_NOT_FOUND",
				})
				return
			case "TOKEN_EXPIRED":
				writeJSON(w, http.StatusNotFound, map[string]string{
					"detail":     "token exists but access_token has expired",
					"error_code": "TOKEN_EXPIRED",
				})
				return
			case "COMMENT_ID_REQUIRED":
				writeJSON(w, http.StatusBadRequest, map[string]string{
					"detail":     "comment_id is required when multiple tokens exist for this task",
					"error_code": "COMMENT_ID_REQUIRED",
				})
				return
			case "SCOPE_MISMATCH":
				writeJSON(w, http.StatusForbidden, map[string]string{
					"detail":     de.Message,
					"error_code": de.Code,
				})
				return
			}
		}
		log.Printf("[task-credential-service] get-token-by-scope: error task=%s: %v", taskID, err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"access_token":  token.ContainerAccessToken,
		"refresh_token": token.ContainerRefreshToken,
		"expires_at":    token.ContainerAccessTokenExpiresAt,
		"task_id":       token.TaskID,
		"company_id":    token.CompanyID,
		"workspace_id":  token.WorkspaceID,
		"comment_id":    token.CommentID,
	})
}
