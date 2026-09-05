package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

// gitIdentityRow is the internal row representation for DB queries.
type gitIdentityRow struct {
	ID                string `json:"id"`
	UserID            string `json:"user_id"`
	CompanyID         string `json:"company_id"`
	Label             string `json:"label"`
	GitUserName       string `json:"git_user_name"`
	GitUserEmail      string `json:"git_user_email"`
	GitRemoteUsername string `json:"git_remote_username"`
	IsDefault         bool   `json:"is_default"`
	CreatedAt         string `json:"created_at"`
	UpdatedAt         string `json:"updated_at"`
	CompanyName       string `json:"company_name"`
}

// handleProfileGitIdentities handles git identities CRUD.
// Supports two path formats:
//   - /api/git-identities/user/{userId}/  (primary, avoids APISIX /api/user/* radixtree conflicts)
//   - /api/user/{userId}/profile/git-identities/  (legacy Django path, kept for compatibility)
//
// OPT-052: Replaces retired Django saas-backend endpoint (saas DB decommissioned 2026-07-30).
// task_git_identities table now lives in task_task.db owned by taskTaskService.
func handleProfileGitIdentities(w http.ResponseWriter, r *http.Request) {
	var urlUserID string

	// Try new path: /api/git-identities/user/{userId}[/...]
	if strings.HasPrefix(r.URL.Path, "/api/git-identities/") {
		parts := cleanPath(r, "/api/git-identities/")
		if len(parts) < 2 || parts[0] != "user" {
			writeError(w, r, http.StatusNotFound, "not found")
			return
		}
		urlUserID = strings.TrimSpace(parts[1])
	} else {
		// Legacy path: /api/user/{userId}/profile/git-identities/
		parts := cleanPath(r, "/api/user/")
		if len(parts) < 3 || parts[1] != "profile" || parts[2] != "git-identities" {
			writeError(w, r, http.StatusNotFound, "not found")
			return
		}
		urlUserID = strings.TrimSpace(parts[0])
	}

	// Auth: must match authenticated user (or internal call)
	authUserID := getAuthUser(r)
	if !isInternalCall(r) && authUserID != "" && authUserID != urlUserID {
		writeError(w, r, http.StatusForbidden, "无权访问其他用户的 Git 身份")
		return
	}

	switch r.Method {
	case http.MethodGet:
		handleListGitIdentities(w, r, urlUserID)
	case http.MethodPost:
		handleCreateGitIdentity(w, r, urlUserID)
	case http.MethodPatch:
		handlePatchGitIdentity(w, r, urlUserID)
	case http.MethodOptions:
		w.WriteHeader(http.StatusNoContent)
	default:
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func handleListGitIdentities(w http.ResponseWriter, r *http.Request, userID string) {
	rows, err := db.Query(`
		SELECT id, user_id, company_id, COALESCE(label,''), COALESCE(git_user_name,''),
		       COALESCE(git_user_email,''), COALESCE(git_remote_username,''),
		       COALESCE(is_default,0), created_at, updated_at
		FROM task_git_identities
		WHERE user_id = ?
		ORDER BY company_id, is_default DESC, created_at DESC
	`, userID)
	if err != nil {
		log.Printf("[taskTaskService] git-identities list user=%s: %v", userID, err)
		writeErrorDetail(w, r, http.StatusInternalServerError, "获取身份列表失败")
		return
	}
	defer rows.Close()

	var identities []gitIdentityRow
	for rows.Next() {
		var row gitIdentityRow
		var createdAt, updatedAt any
		if err := rows.Scan(&row.ID, &row.UserID, &row.CompanyID, &row.Label,
			&row.GitUserName, &row.GitUserEmail, &row.GitRemoteUsername,
			&row.IsDefault, &createdAt, &updatedAt); err != nil {
			log.Printf("[taskTaskService] git-identities scan: %v", err)
			continue
		}
		row.CreatedAt = fmt.Sprintf("%v", createdAt)
		row.UpdatedAt = fmt.Sprintf("%v", updatedAt)
		identities = append(identities, row)
	}

	// Batch-resolve company names via tenant service
	companyIDs := collectCompanyIDs(identities)
	companyNames := batchResolveCompanyNames(companyIDs)
	for i := range identities {
		if name, ok := companyNames[identities[i].CompanyID]; ok {
			identities[i].CompanyName = name
		}
	}

	if identities == nil {
		identities = []gitIdentityRow{}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"identities": identities,
	})
}

func handleCreateGitIdentity(w http.ResponseWriter, r *http.Request, userID string) {
	body, err := readJSONBody(r)
	if err != nil {
		writeErrorDetail(w, r, http.StatusBadRequest, "invalid json")
		return
	}

	companyID := strField(body, "company_id")
	gitUserName := strField(body, "git_user_name")
	gitUserEmail := strField(body, "git_user_email")

	if companyID == "" || gitUserName == "" || gitUserEmail == "" {
		writeErrorDetail(w, r, http.StatusBadRequest, "请选择公司，并填写 Git 用户名和 Git 邮箱")
		return
	}

	label := strField(body, "label")
	gitRemoteUsername := strField(body, "git_remote_username")
	isDefault := false
	if v, ok := body["is_default"]; ok {
		switch t := v.(type) {
		case bool:
			isDefault = t
		case float64:
			isDefault = t != 0
		}
	}

	id := genID("gi")

	// If setting as default, unset other defaults for same company+user
	if isDefault {
		_, _ = db.Exec(`UPDATE task_git_identities SET is_default=0, updated_at=NOW() WHERE user_id=? AND company_id=?`, userID, companyID)
	}

	_, err = db.Exec(`
		INSERT INTO task_git_identities (id, user_id, company_id, label, git_user_name, git_user_email, git_remote_username, is_default, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, NOW(), NOW())
	`, id, userID, companyID, label, gitUserName, gitUserEmail, gitRemoteUsername, isDefault)
	if err != nil {
		log.Printf("[taskTaskService] git-identities create user=%s company=%s: %v", userID, companyID, err)
		writeErrorDetail(w, r, http.StatusInternalServerError, "创建身份失败")
		return
	}

	log.Printf("[taskTaskService] git-identity created id=%s user=%s company=%s default=%v", id, userID, companyID, isDefault)

	// Read back the created row to return full identity
	var created gitIdentityRow
	var ca, ua any
	err = db.QueryRow(`
		SELECT id, user_id, company_id, COALESCE(label,''), COALESCE(git_user_name,''),
		       COALESCE(git_user_email,''), COALESCE(git_remote_username,''),
		       COALESCE(is_default,0), created_at, updated_at
		FROM task_git_identities WHERE id = ?
	`, id).Scan(&created.ID, &created.UserID, &created.CompanyID, &created.Label,
		&created.GitUserName, &created.GitUserEmail, &created.GitRemoteUsername,
		&created.IsDefault, &ca, &ua)
	if err == nil {
		created.CreatedAt = fmt.Sprintf("%v", ca)
		created.UpdatedAt = fmt.Sprintf("%v", ua)
	}

	writeJSON(w, http.StatusCreated, created)
}

func handlePatchGitIdentity(w http.ResponseWriter, r *http.Request, userID string) {
	body, err := readJSONBody(r)
	if err != nil {
		writeErrorDetail(w, r, http.StatusBadRequest, "invalid json")
		return
	}

	identityID := strField(body, "identity_id")
	if identityID == "" {
		writeErrorDetail(w, r, http.StatusBadRequest, "identity_id is required")
		return
	}

	// Verify ownership: identity belongs to this user
	var companyID string
	err = db.QueryRow(`SELECT company_id FROM task_git_identities WHERE id=? AND user_id=? LIMIT 1`, identityID, userID).Scan(&companyID)
	if err != nil {
		writeErrorDetail(w, r, http.StatusNotFound, "身份不存在")
		return
	}

	// Handle is_default
	if v, ok := body["is_default"]; ok {
		isDefault := false
		switch t := v.(type) {
		case bool:
			isDefault = t
		case float64:
			isDefault = t != 0
		}
		if isDefault && companyID != "" {
			// Unset other defaults for same company+user
			_, _ = db.Exec(`UPDATE task_git_identities SET is_default=0, updated_at=NOW() WHERE user_id=? AND company_id=?`, userID, companyID)
		}
		_, err = db.Exec(`UPDATE task_git_identities SET is_default=?, updated_at=NOW() WHERE id=? AND user_id=?`, isDefault, identityID, userID)
		if err != nil {
			log.Printf("[taskTaskService] git-identities patch is_default id=%s: %v", identityID, err)
			writeErrorDetail(w, r, http.StatusInternalServerError, "设置默认身份失败")
			return
		}
		log.Printf("[taskTaskService] git-identity set-default id=%s user=%s company=%s", identityID, userID, companyID)
	}

	// Return updated identity
	var resp gitIdentityRow
	var ca, ua any
	err = db.QueryRow(`
		SELECT id, user_id, company_id, COALESCE(label,''), COALESCE(git_user_name,''),
		       COALESCE(git_user_email,''), COALESCE(git_remote_username,''),
		       COALESCE(is_default,0), created_at, updated_at
		FROM task_git_identities WHERE id = ?
	`, identityID).Scan(&resp.ID, &resp.UserID, &resp.CompanyID, &resp.Label,
		&resp.GitUserName, &resp.GitUserEmail, &resp.GitRemoteUsername,
		&resp.IsDefault, &ca, &ua)
	if err == nil {
		resp.CreatedAt = fmt.Sprintf("%v", ca)
		resp.UpdatedAt = fmt.Sprintf("%v", ua)
	}

	writeJSON(w, http.StatusOK, resp)
}

// collectCompanyIDs extracts unique company IDs from identity rows.
func collectCompanyIDs(identities []gitIdentityRow) []string {
	seen := map[string]bool{}
	var ids []string
	for _, idn := range identities {
		cid := strings.TrimSpace(idn.CompanyID)
		if cid != "" && !seen[cid] {
			seen[cid] = true
			ids = append(ids, cid)
		}
	}
	return ids
}

// batchResolveCompanyNames fetches company names from the tenant service.
func batchResolveCompanyNames(companyIDs []string) map[string]string {
	out := map[string]string{}
	base := strings.TrimRight(cfg.TaskTenantServiceURL, "/")
	if base == "" {
		return out
	}

	for _, cid := range companyIDs {
		url := fmt.Sprintf("%s/api/internal/tenant/companies/creator?company_id=%s", base, cid)
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			continue
		}
		if cfg.InternalSecret != "" {
			req.Header.Set("X-Internal-Secret", cfg.InternalSecret)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			continue
		}
		if resp.StatusCode == http.StatusOK {
			var body map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&body); err == nil {
				if name, ok := body["company_name"].(string); ok && name != "" {
					out[cid] = name
				}
			}
		}
		resp.Body.Close()
	}

	return out
}
