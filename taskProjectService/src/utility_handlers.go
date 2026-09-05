package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"
)

var projectActionSegments = map[string]bool{
	"switch":                     true,
	"batch-delete":               true,
	"batch-get":                  true,
	"gitlab-remote-repos":        true,
	"batch-from-gitlab-repos":    true,
	"combined-from-gitlab-repos": true,
	"workspace-access":           true,
	"validate-git-repo":          true,
	"validate-git-repos":         true,
	"translate-branch-title":     true,
	"manage-deliverable-system":  true,
	"manage-progress-column":     true,
}

func isProjectActionSegment(seg string) bool {
	return projectActionSegments[seg]
}

func handleSwitchWorkspace(w http.ResponseWriter, r *http.Request, tenantID string) {
	if r.Method != http.MethodPost {
		writeError(w, r, 405, "method not allowed")
		return
	}
	userID, ok := requireAuthUser(w, r)
	if !ok {
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeError(w, r, 400, "invalid JSON")
		return
	}
	wsID := strField(body, "workspace_id")
	if wsID == "" {
		writeError(w, r, 400, "工作空间ID不能为空")
		return
	}
	switchWorkspaceCore(w, r, tenantID, userID, wsID)
}

// switchWorkspaceCore 执行切换工作空间的核心逻辑。request body 的解析由各入口
// handler 负责并在本函数前完成——切勿在入口之间二次 readJSONBody，否则 body 已
// 被消费导致 workspace_id 恒为空（400 根因，见 OPT-20260808-xxx）。
func switchWorkspaceCore(w http.ResponseWriter, r *http.Request, tenantID, userID, wsID string) {
	var name, companyID string
	err := db.QueryRow("SELECT name, company_id FROM project_workspace_entries WHERE id=? AND company_id=?", wsID, tenantID).Scan(&name, &companyID)
	if err == sql.ErrNoRows {
		writeError(w, r, 404, "工作空间不存在或无权访问")
		return
	}
	if err != nil {
		writeError(w, r, 500, err.Error())
		return
	}
	// 校验工作空间级访问权限：有 access 行则须命中 user_id；无 access 行则对租户开放
	if !checkWorkspaceAccess(wsID, userID) {
		writeError(w, r, 403, "无权访问该工作空间")
		return
	}
	if err := persistWorkspaceSelection(userID, tenantID, wsID); err != nil {
		logInfo("persist workspace warning: "+err.Error(), r.Header.Get("X-Trace-Id"))
	}
	writeJSON(w, 200, map[string]interface{}{
		"success": true, "workspace_id": wsID, "workspace_name": name,
	})
}

func handleBatchDeleteProjects(w http.ResponseWriter, r *http.Request, tenantID string) {
	if r.Method != http.MethodPost {
		writeError(w, r, 405, "method not allowed")
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeError(w, r, 400, "invalid JSON")
		return
	}
	rawIDs, ok := body["project_ids"].([]interface{})
	if !ok || len(rawIDs) == 0 {
		writeError(w, r, 400, "project_ids 不能为空")
		return
	}
	deleted := []map[string]string{}
	errors := []map[string]string{}
	for _, raw := range rawIDs {
		pid := strings.TrimSpace(fmt.Sprintf("%v", raw))
		if pid == "" {
			continue
		}
		var name string
		err := db.QueryRow("SELECT name FROM project_entries WHERE id=? AND company_id=?", pid, tenantID).Scan(&name)
		if err == sql.ErrNoRows {
			errors = append(errors, map[string]string{"id": pid, "error": "项目不存在或无权访问"})
			continue
		}
		if err != nil {
			errors = append(errors, map[string]string{"id": pid, "error": err.Error()})
			continue
		}
		db.Exec("DELETE FROM project_repos WHERE project_id=?", pid)
		db.Exec("DELETE FROM project_workspaces WHERE project_id=?", pid)
		db.Exec("DELETE FROM project_entries WHERE id=?", pid)
		if pubErr := publishProjectDeletedFn(r.Context(), tenantID, pid); pubErr != nil {
			logWarn("PROJECT_DELETED publish failed: "+pubErr.Error(), r.Header.Get("X-Trace-Id"))
		}
		deleted = append(deleted, map[string]string{"id": pid, "name": name})
	}
	logInfo(fmt.Sprintf("batch delete tenant=%s deleted=%d errors=%d", tenantID, len(deleted), len(errors)), r.Header.Get("X-Trace-Id"))
	writeJSON(w, 200, map[string]interface{}{"deleted": deleted, "errors": errors})
}

const maxBatchGetProjects = 500

// handleBatchGetProjects — POST /api/tenant/{tid}/projects/batch-get/
// body {"ids":[...], "lite": true?}；lite 时跳过 loadProjectDetail（仅基础字段，含 name）。
func handleBatchGetProjects(w http.ResponseWriter, r *http.Request, tenantID string) {
	if r.Method != http.MethodPost {
		writeError(w, r, 405, "method not allowed")
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeError(w, r, 400, "invalid JSON")
		return
	}
	rawIDs, _ := body["ids"].([]interface{})
	if rawIDs == nil {
		rawIDs, _ = body["project_ids"].([]interface{})
	}
	ids := make([]string, 0, len(rawIDs))
	seen := map[string]struct{}{}
	for _, raw := range rawIDs {
		pid := strings.TrimSpace(fmt.Sprintf("%v", raw))
		if pid == "" || pid == "<nil>" {
			continue
		}
		if _, ok := seen[pid]; ok {
			continue
		}
		seen[pid] = struct{}{}
		ids = append(ids, pid)
		if len(ids) >= maxBatchGetProjects {
			break
		}
	}
	if len(ids) == 0 {
		writeJSON(w, 200, map[string]interface{}{"projects": []interface{}{}})
		return
	}
	lite := false
	switch v := body["lite"].(type) {
	case bool:
		lite = v
	case string:
		lite = strings.EqualFold(strings.TrimSpace(v), "true") || strings.TrimSpace(v) == "1"
	}
	placeholders := make([]string, len(ids))
	args := make([]interface{}, 0, len(ids)+1)
	args = append(args, tenantID)
	for i, id := range ids {
		placeholders[i] = "?"
		args = append(args, id)
	}
	q := `SELECT id,name,description,company_id,installed_image_id,container_image_name,tags,server_run_template,created_at,updated_at
		FROM project_entries WHERE company_id=? AND id IN (` + strings.Join(placeholders, ",") + `)`
	rows, err := db.Query(q, args...)
	if err != nil {
		writeError(w, r, 500, err.Error())
		return
	}
	defer rows.Close()
	projects := make([]map[string]interface{}, 0, len(ids))
	for rows.Next() {
		p, err := scanProjectRow(rows)
		if err != nil {
			continue
		}
		if lite {
			projects = append(projects, p)
			continue
		}
		detail, _ := loadProjectDetail(p["id"].(string))
		if detail != nil {
			projects = append(projects, detail)
		} else {
			projects = append(projects, p)
		}
	}
	writeJSON(w, 200, map[string]interface{}{"projects": projects})
}

const maxBatchGetWorkspaces = 500

// handleBatchGetWorkspaces — POST /api/tenant/{tid}/workspaces/batch-get/ body {"ids":[...]}
func handleBatchGetWorkspaces(w http.ResponseWriter, r *http.Request, tenantID string) {
	if r.Method != http.MethodPost {
		writeError(w, r, 405, "method not allowed")
		return
	}
	userID, ok := requireAuthUser(w, r)
	if !ok {
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeError(w, r, 400, "invalid JSON")
		return
	}
	rawIDs, _ := body["ids"].([]interface{})
	if rawIDs == nil {
		rawIDs, _ = body["workspace_ids"].([]interface{})
	}
	ids := make([]string, 0, len(rawIDs))
	seen := map[string]struct{}{}
	for _, raw := range rawIDs {
		wid := strings.TrimSpace(fmt.Sprintf("%v", raw))
		if wid == "" || wid == "<nil>" {
			continue
		}
		if _, ok := seen[wid]; ok {
			continue
		}
		seen[wid] = struct{}{}
		ids = append(ids, wid)
		if len(ids) >= maxBatchGetWorkspaces {
			break
		}
	}
	if len(ids) == 0 {
		writeJSON(w, 200, map[string]interface{}{"workspaces": []interface{}{}})
		return
	}
	placeholders := make([]string, len(ids))
	args := make([]interface{}, 0, len(ids)+1)
	args = append(args, tenantID)
	for i, id := range ids {
		placeholders[i] = "?"
		args = append(args, id)
	}
	q := `SELECT id,name,description,company_id,deliverable_system_id,deliverable_system_from,is_default,task_archive_tier,allow_personal_feature_params,container_image_at_mode_enabled,created_at,updated_at
		FROM project_workspace_entries w WHERE w.company_id=? AND w.id IN (` + strings.Join(placeholders, ",") + `)`
	if !isInternalCall(r) {
		q += ` AND (
			EXISTS (SELECT 1 FROM project_workspace_accesses a WHERE a.workspace_id=w.id AND a.user_id=?)
			OR NOT EXISTS (SELECT 1 FROM project_workspace_accesses a WHERE a.workspace_id=w.id)
		)`
		args = append(args, userID)
	}
	q += ` ORDER BY w.name`
	rows, err := db.Query(q, args...)
	if err != nil {
		writeError(w, r, 500, err.Error())
		return
	}
	defer rows.Close()
	workspaces := make([]map[string]interface{}, 0, len(ids))
	for rows.Next() {
		var id, name, desc, cid, dsid, dsfrom, tier string
		var isDef, allowPP, containerImageAtMode bool
		var ca, ua time.Time
		if err := rows.Scan(&id, &name, &desc, &cid, &dsid, &dsfrom, &isDef, &tier, &allowPP, &containerImageAtMode, &ca, &ua); err != nil {
			continue
		}
		workspaces = append(workspaces, map[string]interface{}{
			"id": id, "name": name, "description": desc,
			"company_id": cid, "company_name": "",
			"deliverable_system_id": dsid, "deliverable_system_from": dsfrom,
			"is_default": isDef, "is_current": false,
			"task_archive_tier": tier, "allow_personal_feature_params": allowPP,
			"container_image_at_mode_enabled": containerImageAtMode,
			"created_at":                      ca.UTC().Format(time.RFC3339Nano),
			"updated_at":                      ua.UTC().Format(time.RFC3339Nano),
		})
	}
	writeJSON(w, 200, map[string]interface{}{"workspaces": workspaces})
}

func handleValidateGitRepo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, 405, "method not allowed")
		return
	}
	userID := getAuthUser(r)
	urlParam := r.URL.Query().Get("url")
	if urlParam == "" {
		urlParam = r.URL.Query().Get("git_repo")
	}
	if strings.TrimSpace(urlParam) == "" {
		writeJSON(w, 400, map[string]interface{}{
			"is_accessible": false,
			"message":       "请输入有效的 Git 仓库地址",
			"token_status":  tokenStatusNotApplicable,
		})
		return
	}
	result := validateGitRepoForUser(userID, urlParam, true, requestTenantID(r), strings.TrimSpace(r.URL.Query().Get("project_id")), traceHeadersFromRequest(r))
	writeJSON(w, 200, result.toMap())
}

const maxValidateGitReposBatch = 100

// handleValidateGitRepos batches validation for many repo URLs in one round-trip.
// Body: { "urls": ["…"], "probe_access": true|false }
// - probe_access=false (default): token_status via gitOauth only (autorun skip path)
// - probe_access=true: also probe remote; 401/403/404 downgrades token_available → token_error
func handleValidateGitRepos(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, 405, "method not allowed")
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeError(w, r, 400, "invalid JSON")
		return
	}
	rawURLs, _ := body["urls"].([]interface{})
	if len(rawURLs) == 0 {
		rawURLs, _ = body["repo_urls"].([]interface{})
	}
	urls := make([]string, 0, len(rawURLs))
	seen := map[string]bool{}
	for _, item := range rawURLs {
		u := strings.TrimSpace(fmt.Sprintf("%v", item))
		if u == "" || seen[u] {
			continue
		}
		seen[u] = true
		urls = append(urls, u)
	}
	if len(urls) == 0 {
		writeError(w, r, 400, "urls 不能为空")
		return
	}
	if len(urls) > maxValidateGitReposBatch {
		writeJSON(w, 400, map[string]interface{}{
			"error": fmt.Sprintf("urls 最多 %d 个", maxValidateGitReposBatch),
		})
		return
	}
	userID := getAuthUser(r)
	if strings.TrimSpace(userID) == "" {
		writeError(w, r, 401, "unauthenticated")
		return
	}
	probeAccess := false
	switch v := body["probe_access"].(type) {
	case bool:
		probeAccess = v
	case string:
		probeAccess = strings.EqualFold(strings.TrimSpace(v), "true") || v == "1"
	}
	projectID := ""
	if v, ok := body["project_id"]; ok && v != nil {
		projectID = strings.TrimSpace(fmt.Sprintf("%v", v))
		if projectID == "<nil>" {
			projectID = ""
		}
	}
	validated := validateGitReposForUser(userID, urls, probeAccess, requestTenantID(r), projectID, traceHeadersFromRequest(r))
	results := make([]map[string]interface{}, 0, len(validated))
	for _, item := range validated {
		results = append(results, item.toMap())
	}
	logInfo(
		fmt.Sprintf("validate-git-repos user=%s count=%d probe=%v", userID, len(results), probeAccess),
		r.Header.Get("X-Trace-Id"),
	)
	writeJSON(w, 200, map[string]interface{}{"results": results})
}

func handleRepoAccessCheck(w http.ResponseWriter, r *http.Request, tenantID, projectID string) {
	if r.Method != http.MethodGet {
		writeError(w, r, 405, "method not allowed")
		return
	}
	detail, err := loadProjectDetail(projectID)
	if err == sql.ErrNoRows {
		writeError(w, r, 404, "project not found")
		return
	}
	if err != nil {
		writeError(w, r, 500, err.Error())
		return
	}
	if rejectIfProjectNotInTenant(w, r, detail, tenantID) {
		return
	}
	repos := collectProjectGitRepoURLs(detail)
	userID := getAuthUser(r)
	result := checkProjectPrimaryRepoAccess(userID, repos, tenantID, projectID, traceHeadersFromRequest(r))
	logInfo(
		fmt.Sprintf("repo-access-check project=%s user=%s status=%s", projectID, userID, result.AccessStatus),
		r.Header.Get("X-Trace-Id"),
	)
	writeJSON(w, 200, result.toMap())
}
