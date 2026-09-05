package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"authz"
)

// handleInternalLookupGitIdentity — POST /api/internal/git-identities/lookup/
// Body: {identity_id, user_id, company_id}
// Resolves a Git identity by id with ownership checks (user_id must match;
// company_id must match when non-empty). Used by taskCloudService to replace
// cross-database SQL into task_task.task_git_identities (OPT-20260820-016).
func handleInternalLookupGitIdentity(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	// Empty InternalSecret does not open this endpoint. Callers must send
	// X-Auth-User-Id: internal and/or X-Internal-Secret (Cloud lookup
	// omitted the user header → 403 "internal only" on git push prepare).
	if !isInternalCall(r) && !internalSecretOK(r) {
		writeError(w, r, http.StatusForbidden, "internal only")
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeErrorDetail(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	identityID := strings.TrimSpace(strField(body, "identity_id"))
	userID := strings.TrimSpace(strField(body, "user_id"))
	companyID := strings.TrimSpace(strField(body, "company_id"))
	if identityID == "" || userID == "" {
		writeErrorDetail(w, r, http.StatusBadRequest, "identity_id and user_id required")
		return
	}
	row, err := loadGitIdentityByID(identityID)
	if err != nil {
		log.Printf("[taskTaskService] git identity lookup: id=%s err=%v", identityID, err)
		writeErrorDetail(w, r, http.StatusInternalServerError, "lookup failed")
		return
	}
	if row == nil || row.UserID != userID {
		writeJSON(w, http.StatusOK, map[string]interface{}{"found": false})
		return
	}
	if companyID != "" && row.CompanyID != companyID {
		writeJSON(w, http.StatusOK, map[string]interface{}{"found": false})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"found":          true,
		"git_user_name":  row.GitUserName,
		"git_user_email": row.GitUserEmail,
	})
}

// handleInternalEnsureDefaultGitIdentity — POST /api/internal/git-identities/ensure-default/
// Body: {user_id, company_id, member_id, member_name}
// Creates system-auto identity if missing (idempotent by email).
func handleInternalEnsureDefaultGitIdentity(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !isInternalCall(r) && !internalSecretOK(r) {
		writeError(w, r, http.StatusForbidden, "internal only")
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeErrorDetail(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	userID := strField(body, "user_id")
	companyID := strField(body, "company_id")
	memberID := strField(body, "member_id")
	memberName := strField(body, "member_name")
	if userID == "" || companyID == "" || memberID == "" {
		writeErrorDetail(w, r, http.StatusBadRequest, "user_id, company_id, member_id required")
		return
	}

	email := BuildSystemGitEmail(memberID, companyID)
	name := ResolveSystemGitUserName(memberName, userID)

	existing, err := findGitIdentityByEmail(userID, companyID, email)
	if err != nil {
		log.Printf("[taskTaskService] ensure-default lookup: %v", err)
		writeErrorDetail(w, r, http.StatusInternalServerError, "查询身份失败")
		return
	}
	if existing != nil {
		log.Printf("[taskTaskService] ensure-default idempotent id=%s user=%s company=%s", existing.ID, userID, companyID)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"created":  false,
			"identity": existing,
		})
		return
	}

	id := genID("gi")
	// Prefer default when user has no default yet for this company.
	var hasDefault int
	_ = db.QueryRow(`SELECT COUNT(1) FROM task_git_identities WHERE user_id=? AND company_id=? AND is_default=1`,
		userID, companyID).Scan(&hasDefault)
	isDefault := hasDefault == 0
	if isDefault {
		_, _ = db.Exec(`UPDATE task_git_identities SET is_default=0, updated_at=NOW() WHERE user_id=? AND company_id=?`, userID, companyID)
	}

	_, err = db.Exec(`
		INSERT INTO task_git_identities (id, user_id, company_id, label, git_user_name, git_user_email, git_remote_username, is_default, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, '', ?, NOW(), NOW())
	`, id, userID, companyID, systemAutoGitIdentityLabel, name, email, isDefault)
	if err != nil {
		log.Printf("[taskTaskService] ensure-default insert: %v", err)
		writeErrorDetail(w, r, http.StatusInternalServerError, "创建身份失败")
		return
	}

	created, err := loadGitIdentityByID(id)
	if err != nil || created == nil {
		writeJSON(w, http.StatusCreated, map[string]interface{}{
			"created": true,
			"identity": gitIdentityRow{
				ID: id, UserID: userID, CompanyID: companyID,
				Label: systemAutoGitIdentityLabel, GitUserName: name, GitUserEmail: email, IsDefault: isDefault,
			},
		})
		return
	}
	log.Printf("[taskTaskService] ensure-default created id=%s user=%s company=%s email=%s", id, userID, companyID, email)
	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"created":  true,
		"identity": created,
	})
}

func internalSecretOK(r *http.Request) bool {
	secret := strings.TrimSpace(cfg.InternalSecret)
	if secret == "" {
		return false
	}
	return r.Header.Get("X-Internal-Secret") == secret
}

func findGitIdentityByEmail(userID, companyID, email string) (*gitIdentityRow, error) {
	row := &gitIdentityRow{}
	var ca, ua any
	err := db.QueryRow(`
		SELECT id, user_id, company_id, COALESCE(label,''), COALESCE(git_user_name,''),
		       COALESCE(git_user_email,''), COALESCE(git_remote_username,''),
		       COALESCE(is_default,0), created_at, updated_at
		FROM task_git_identities
		WHERE user_id=? AND company_id=? AND git_user_email=?
		LIMIT 1
	`, userID, companyID, email).Scan(
		&row.ID, &row.UserID, &row.CompanyID, &row.Label,
		&row.GitUserName, &row.GitUserEmail, &row.GitRemoteUsername,
		&row.IsDefault, &ca, &ua,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	row.CreatedAt = fmt.Sprintf("%v", ca)
	row.UpdatedAt = fmt.Sprintf("%v", ua)
	return row, nil
}

func loadGitIdentityByID(id string) (*gitIdentityRow, error) {
	row := &gitIdentityRow{}
	var ca, ua any
	err := db.QueryRow(`
		SELECT id, user_id, company_id, COALESCE(label,''), COALESCE(git_user_name,''),
		       COALESCE(git_user_email,''), COALESCE(git_remote_username,''),
		       COALESCE(is_default,0), created_at, updated_at
		FROM task_git_identities WHERE id=? LIMIT 1
	`, id).Scan(
		&row.ID, &row.UserID, &row.CompanyID, &row.Label,
		&row.GitUserName, &row.GitUserEmail, &row.GitRemoteUsername,
		&row.IsDefault, &ca, &ua,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	row.CreatedAt = fmt.Sprintf("%v", ca)
	row.UpdatedAt = fmt.Sprintf("%v", ua)
	return row, nil
}

// canManageMemberGitIdentities: self OR member:manage OR group-members:manage（限本组）。
func canManageMemberGitIdentities(r *http.Request, tenantID, targetUserID string) bool {
	authUser := getAuthUser(r)
	if authUser != "" && authUser == targetUserID {
		return true
	}
	if authz.HasPerm(r, tenantID, authz.PermMemberManage) {
		return true
	}
	if authz.HasPerm(r, tenantID, authz.PermGroupMembersManage) {
		// 组管理员（group-members:manage）仅可代管其所管小组内成员（OPT-20260812-029），
		// 避免越权管理其他组员身份。
		return userInAdminGroup(r, tenantID, authUser, targetUserID)
	}
	return false
}

// userInAdminGroup queries taskTenantService internal API to verify targetUserID
// belongs to at least one group that adminUserID administers within tenantID.
// Fail-closed on infra errors: 越权管理比短暂不可用更危险。
func userInAdminGroup(r *http.Request, tenantID, adminUserID, targetUserID string) bool {
	if adminUserID == "" {
		return false
	}
	if cfg.TaskTenantServiceURL == "" {
		return false
	}
	base := strings.TrimRight(cfg.TaskTenantServiceURL, "/")
	u := fmt.Sprintf("%s/api/internal/tenant/groups/user-in-admin-groups?admin_user_id=%s&user_id=%s&company_id=%s",
		base, url.QueryEscape(adminUserID), url.QueryEscape(targetUserID), url.QueryEscape(tenantID))
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return false
	}
	if cfg.InternalSecret != "" {
		req.Header.Set("X-Internal-Secret", cfg.InternalSecret)
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[taskTaskService] user-in-admin-groups: tenant service unreachable admin=%s target=%s: %v", adminUserID, targetUserID, err)
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false
	}
	raw, _ := io.ReadAll(resp.Body)
	var result struct {
		InAdminGroup bool `json:"in_admin_group"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return false
	}
	return result.InAdminGroup
}

// handleTenantMemberGitIdentities — /api/git-identities/tenant/{tid}/member/{memberId}/
func handleTenantMemberGitIdentities(w http.ResponseWriter, r *http.Request, tenantID, memberID string) {
	member, err := tenantGetMemberByID(tenantID, memberID)
	if err != nil {
		log.Printf("[taskTaskService] tenant member by-id: %v", err)
		writeErrorDetail(w, r, http.StatusBadGateway, "成员查询失败")
		return
	}
	if member == nil {
		writeErrorDetail(w, r, http.StatusNotFound, "成员不存在")
		return
	}
	userID := strField(member, "user_id")
	if userID == "" {
		writeErrorDetail(w, r, http.StatusNotFound, "成员不存在")
		return
	}
	if !canManageMemberGitIdentities(r, tenantID, userID) {
		writeError(w, r, http.StatusForbidden, "无权管理该成员的 Git 身份")
		return
	}

	switch r.Method {
	case http.MethodGet:
		list, err := listGitIdentitiesForUserCompany(userID, tenantID)
		if err != nil {
			writeErrorDetail(w, r, http.StatusInternalServerError, "获取身份列表失败")
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"member_id":  memberID,
			"user_id":    userID,
			"company_id": tenantID,
			"identities": list,
		})
	case http.MethodPost:
		body, err := readJSONBody(r)
		if err != nil {
			writeErrorDetail(w, r, http.StatusBadRequest, "invalid json")
			return
		}
		gitUserName := strField(body, "git_user_name")
		gitUserEmail := strField(body, "git_user_email")
		if gitUserName == "" || gitUserEmail == "" {
			writeErrorDetail(w, r, http.StatusBadRequest, "请填写 Git 用户名和 Git 邮箱")
			return
		}
		label := strField(body, "label")
		isDefault := boolFromAny(body["is_default"])
		id := genID("gi")
		if isDefault {
			_, _ = db.Exec(`UPDATE task_git_identities SET is_default=0, updated_at=NOW() WHERE user_id=? AND company_id=?`, userID, tenantID)
		}
		_, err = db.Exec(`
			INSERT INTO task_git_identities (id, user_id, company_id, label, git_user_name, git_user_email, git_remote_username, is_default, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, NOW(), NOW())
		`, id, userID, tenantID, label, gitUserName, gitUserEmail, strField(body, "git_remote_username"), isDefault)
		if err != nil {
			writeErrorDetail(w, r, http.StatusInternalServerError, "创建身份失败")
			return
		}
		created, _ := loadGitIdentityByID(id)
		if created == nil {
			writeJSON(w, http.StatusCreated, gitIdentityRow{ID: id, UserID: userID, CompanyID: tenantID, Label: label, GitUserName: gitUserName, GitUserEmail: gitUserEmail, IsDefault: isDefault})
			return
		}
		writeJSON(w, http.StatusCreated, created)
	case http.MethodOptions:
		w.WriteHeader(http.StatusNoContent)
	default:
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleTenantIdentityMutation — /api/git-identities/tenant/{tid}/identity/{identityId}/
func handleTenantIdentityMutation(w http.ResponseWriter, r *http.Request, tenantID, identityID string) {
	row, err := loadGitIdentityByID(identityID)
	if err != nil || row == nil {
		writeErrorDetail(w, r, http.StatusNotFound, "身份不存在")
		return
	}
	if row.CompanyID != tenantID {
		writeErrorDetail(w, r, http.StatusNotFound, "身份不存在")
		return
	}
	if !canManageMemberGitIdentities(r, tenantID, row.UserID) {
		writeError(w, r, http.StatusForbidden, "无权管理该 Git 身份")
		return
	}

	switch r.Method {
	case http.MethodPatch:
		body, err := readJSONBody(r)
		if err != nil {
			writeErrorDetail(w, r, http.StatusBadRequest, "invalid json")
			return
		}
		if v, ok := body["is_default"]; ok {
			isDefault := boolFromAny(v)
			if isDefault {
				_, _ = db.Exec(`UPDATE task_git_identities SET is_default=0, updated_at=NOW() WHERE user_id=? AND company_id=?`, row.UserID, tenantID)
			}
			_, err = db.Exec(`UPDATE task_git_identities SET is_default=?, updated_at=NOW() WHERE id=?`, isDefault, identityID)
			if err != nil {
				writeErrorDetail(w, r, http.StatusInternalServerError, "更新失败")
				return
			}
		}
		if name := strField(body, "git_user_name"); name != "" {
			_, _ = db.Exec(`UPDATE task_git_identities SET git_user_name=?, updated_at=NOW() WHERE id=?`, name, identityID)
		}
		if email := strField(body, "git_user_email"); email != "" {
			_, _ = db.Exec(`UPDATE task_git_identities SET git_user_email=?, updated_at=NOW() WHERE id=?`, email, identityID)
		}
		if _, ok := body["label"]; ok {
			_, _ = db.Exec(`UPDATE task_git_identities SET label=?, updated_at=NOW() WHERE id=?`, strField(body, "label"), identityID)
		}
		updated, _ := loadGitIdentityByID(identityID)
		writeJSON(w, http.StatusOK, updated)
	case http.MethodDelete:
		_, err := db.Exec(`DELETE FROM task_git_identities WHERE id=? AND company_id=?`, identityID, tenantID)
		if err != nil {
			writeErrorDetail(w, r, http.StatusInternalServerError, "删除失败")
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"deleted": true, "id": identityID})
	case http.MethodOptions:
		w.WriteHeader(http.StatusNoContent)
	default:
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func boolFromAny(v interface{}) bool {
	switch t := v.(type) {
	case bool:
		return t
	case float64:
		return t != 0
	case string:
		return t == "1" || strings.EqualFold(t, "true")
	default:
		return false
	}
}

func listGitIdentitiesForUserCompany(userID, companyID string) ([]gitIdentityRow, error) {
	rows, err := db.Query(`
		SELECT id, user_id, company_id, COALESCE(label,''), COALESCE(git_user_name,''),
		       COALESCE(git_user_email,''), COALESCE(git_remote_username,''),
		       COALESCE(is_default,0), created_at, updated_at
		FROM task_git_identities
		WHERE user_id=? AND company_id=?
		ORDER BY is_default DESC, created_at DESC
	`, userID, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]gitIdentityRow, 0)
	for rows.Next() {
		var row gitIdentityRow
		var ca, ua any
		if err := rows.Scan(&row.ID, &row.UserID, &row.CompanyID, &row.Label,
			&row.GitUserName, &row.GitUserEmail, &row.GitRemoteUsername,
			&row.IsDefault, &ca, &ua); err != nil {
			continue
		}
		row.CreatedAt = fmt.Sprintf("%v", ca)
		row.UpdatedAt = fmt.Sprintf("%v", ua)
		out = append(out, row)
	}
	return out, nil
}

func tenantGetMemberByID(tenantID, memberID string) (map[string]interface{}, error) {
	base := strings.TrimRight(cfg.TaskTenantServiceURL, "/")
	if base == "" {
		return nil, fmt.Errorf("task tenant service URL not configured")
	}
	url := fmt.Sprintf("%s/api/internal/tenant/members/by-id?member_id=%s&company_id=%s",
		base, memberID, tenantID)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if cfg.InternalSecret != "" {
		req.Header.Set("X-Internal-Secret", cfg.InternalSecret)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, string(raw))
	}
	raw, _ := io.ReadAll(resp.Body)
	var out map[string]interface{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// dispatchGitIdentitiesRoutes routes under /api/git-identities/
func dispatchGitIdentitiesRoutes(w http.ResponseWriter, r *http.Request) {
	parts := cleanPath(r, "/api/git-identities/")
	if len(parts) >= 4 && parts[0] == "tenant" && parts[2] == "member" {
		handleTenantMemberGitIdentities(w, r, parts[1], parts[3])
		return
	}
	if len(parts) >= 4 && parts[0] == "tenant" && parts[2] == "identity" {
		handleTenantIdentityMutation(w, r, parts[1], parts[3])
		return
	}
	handleProfileGitIdentities(w, r)
}
