package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// handleInternalContainerSnapshot serves GET /api/internal/tasks/{id}/container-snapshot.
// Service-to-service read model for task-credential-service (no JWT; optional internal secret).
func handleInternalContainerSnapshot(w http.ResponseWriter, r *http.Request) {
	if !requireInternalSecret(r) {
		writeError(w, r, http.StatusForbidden, "forbidden")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	taskID := strings.TrimSpace(r.Header.Get("X-Task-Id"))
	if taskID == "" {
		writeError(w, r, http.StatusBadRequest, "task id required")
		return
	}
	commentID := strings.TrimSpace(r.URL.Query().Get("comment_id"))
	snap, err := buildContainerSnapshot(r.Context(), taskID, commentID)
	if err == sql.ErrNoRows {
		writeError(w, r, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, snap)
}

func requireInternalSecret(r *http.Request) bool {
	if cfg.InternalSecret == "" {
		return true
	}
	return r.Header.Get("X-Internal-Secret") == cfg.InternalSecret
}

// handleInternalCommentCreatedBy serves GET /api/internal/comments/{id}/created-by.
// Service-to-service comment author lookup for task-credential-service HTTP fallback
// (OPT-20260820-021). Owner-service resolves the author without a taskID, so the
// created_by_id is exposed as an independent internal API instead of only riding
// along inside container-snapshot.
func handleInternalCommentCreatedBy(w http.ResponseWriter, r *http.Request, commentID string) {
	if !requireInternalSecret(r) {
		writeError(w, r, http.StatusForbidden, "forbidden")
		return
	}
	commentID = strings.TrimSpace(commentID)
	if commentID == "" {
		writeError(w, r, http.StatusBadRequest, "comment id required")
		return
	}
	var raw string
	err := db.QueryRow(`SELECT COALESCE(created_by_id, '') FROM task_comments WHERE id = ? LIMIT 1`, commentID).Scan(&raw)
	if err == sql.ErrNoRows {
		writeError(w, r, http.StatusNotFound, "comment not found")
		return
	}
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"comment_id": commentID,
		"user_id":    strings.TrimSpace(raw),
	})
}

func buildContainerSnapshot(ctx context.Context, taskID, commentID string) (map[string]interface{}, error) {
	t, err := loadTask(taskID)
	if err != nil {
		return nil, err
	}
	bs := loadBranchStrategy(t.ID)
	targetBranch := ""
	if bs != nil {
		work := strings.TrimSpace(strFromAny(bs["work_branch_name"]))
		target := strings.TrimSpace(strFromAny(bs["target_branch_name"]))
		if work != "" {
			targetBranch = work
		} else if target != "" {
			targetBranch = target
		}
	}
	projects, err := loadProjectsForContainerSnapshot(ctx, t.ID, t.TenantID)
	if err != nil {
		return nil, err
	}
	identities, err := loadSnapshotRepoIdentities(t.ID, commentID)
	if err != nil {
		return nil, err
	}
	out := map[string]interface{}{
		"id":                 t.ID,
		"title":              t.Title,
		"description":        t.Description,
		"workspace_id":       t.WorkspaceID,
		"tenant_id":          t.TenantID,
		"auto_run":           t.AutoRun,
		"owner_id":           t.OwnerID,
		"installed_image_id": t.InstalledImageID,
		"target_branch":      targetBranch,
		"projects":           projects,
		"repo_identities":    identities,
	}
	if bs != nil {
		out["branch_strategy"] = bs
	}
	return out, nil
}

func loadRepoIdentitiesList(taskID string) []map[string]string {
	rows, err := db.Query(`SELECT repo_url, COALESCE(git_identity_id, '') FROM task_repo_identities WHERE task_id=?`, taskID)
	if err != nil {
		return []map[string]string{}
	}
	defer rows.Close()
	out := []map[string]string{}
	for rows.Next() {
		var repoURL, gid string
		if err := rows.Scan(&repoURL, &gid); err != nil {
			continue
		}
		repoURL = strings.TrimSpace(repoURL)
		if repoURL == "" {
			continue
		}
		out = append(out, map[string]string{
			"repo_url":        repoURL,
			"git_identity_id": strings.TrimSpace(gid),
		})
	}
	return out
}

func loadSnapshotRepoIdentities(taskID, commentID string) ([]map[string]string, error) {
	commentID = strings.TrimSpace(commentID)
	if commentID == "" {
		return enrichRepoIdentityMapsWithGitUsers(loadRepoIdentitiesList(taskID)), nil
	}
	raw, found, err := loadCommentRepoIdentitiesJSON(taskID, commentID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	if !found {
		return nil, sql.ErrNoRows
	}
	selections, _ := ParseRepoIdentities(repoIdentitiesFromJSONColumn(raw))
	if len(selections) > 0 && commentIdentitiesNeedOAuthSeed(selections) {
		userID := loadCommentCreatedByID(commentID)
		before := make([]RepoIdentitySelection, len(selections))
		copy(before, selections)
		repaired := repairCommentOAuthL2FromForkAndProject(taskID, commentID, userID, selections)
		if commentOAuthL2Changed(before, repaired) {
			nextJSON := RepoIdentitiesJSON(repaired)
			if _, updErr := db.Exec(
				`UPDATE task_comments SET repo_identities_json=? WHERE id=? AND task_id=?`,
				nextJSON, commentID, taskID,
			); updErr != nil {
				log.Printf("[taskTaskService] event=comment_oauth_l2_repair_failed task_id=%s comment_id=%s err=%v",
					taskID, commentID, updErr)
			} else {
				log.Printf("[taskTaskService] event=comment_oauth_l2_repaired task_id=%s comment_id=%s user_id=%s",
					taskID, commentID, userID)
				raw = nextJSON
			}
		}
	}
	fromComment := enrichRepoIdentityMapsWithGitUsers(repoIdentityMapsFromJSON(raw))
	if len(fromComment) > 0 {
		return fromComment, nil
	}
	return enrichRepoIdentityMapsWithGitUsers(loadRepoIdentitiesList(taskID)), nil
}

// loadProjectsForContainerSnapshot groups task_projects by project_id with repo URLs and names.
func loadProjectsForContainerSnapshot(ctx context.Context, taskID, tenantID string) ([]map[string]interface{}, error) {
	rows, err := db.Query(`
		SELECT project_id,
			COALESCE(repo_address, ''),
			COALESCE(base_branch, ''),
			COALESCE(target_branch, '')
		FROM task_projects
		WHERE task_id = ?
		ORDER BY created_at
	`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type projAcc struct {
		name                 string
		urls                 []string
		aliases              map[string]string
		branches             []map[string]string
		autoCloneNestedRepos bool
	}
	grouped := map[string]*projAcc{}
	order := []string{}
	nameCache := map[string]string{}
	aliasCache := map[string]map[string]string{}
	autoCloneCache := map[string]bool{}

	for rows.Next() {
		var projectID, repoURL, baseBranch, targetBranch string
		if err := rows.Scan(&projectID, &repoURL, &baseBranch, &targetBranch); err != nil {
			return nil, err
		}
		repoURL = strings.TrimSpace(repoURL)
		if repoURL == "" {
			urls, _ := getProjectRepoURLs(ctx, tenantID, projectID)
			if len(urls) == 0 {
				continue
			}
			repoURL = urls[0]
		}
		acc, ok := grouped[projectID]
		if !ok {
			name := nameCache[projectID]
			if name == "" {
				name = lookupProjectName(tenantID, projectID)
				nameCache[projectID] = name
			}
			aliases := aliasCache[projectID]
			if aliases == nil {
				aliases = map[string]string{}
				autoClone := true
				if meta, err := loadProjectRepoMeta(ctx, tenantID, projectID); err == nil {
					for _, e := range meta.Entries {
						if e.CloneAlias != "" {
							aliases[strings.TrimSpace(e.URL)] = e.CloneAlias
						}
					}
					autoClone = meta.AutoCloneNestedRepos
				}
				aliasCache[projectID] = aliases
				autoCloneCache[projectID] = autoClone
			}
			autoClone := autoCloneCache[projectID]
			acc = &projAcc{name: name, aliases: aliases, autoCloneNestedRepos: autoClone}
			grouped[projectID] = acc
			order = append(order, projectID)
		}
		acc.urls = append(acc.urls, repoURL)
		acc.branches = append(acc.branches, map[string]string{
			"git_repo":      repoURL,
			"base_branch":   strings.TrimSpace(baseBranch),
			"target_branch": strings.TrimSpace(targetBranch),
		})
	}

	out := make([]map[string]interface{}, 0, len(order))
	for _, pid := range order {
		acc := grouped[pid]
		entries := make([]map[string]interface{}, 0, len(acc.urls))
		for _, u := range acc.urls {
			entry := map[string]interface{}{"url": u, "clone_alias": ""}
			if acc.aliases != nil {
				if a := strings.TrimSpace(acc.aliases[u]); a != "" {
					entry["clone_alias"] = a
				}
			}
			entries = append(entries, entry)
		}
		out = append(out, map[string]interface{}{
			"project_id":              pid,
			"project_name":            acc.name,
			"git_repos":               acc.urls,
			"git_repo_entries":        entries,
			"repo_branches":           acc.branches,
			"auto_clone_nested_repos": acc.autoCloneNestedRepos,
		})
	}
	return out, nil
}

func lookupProjectName(tenantID, projectID string) string {
	status, data, err := projectGetObject(
		context.Background(),
		fmt.Sprintf("/api/projects/tenant_id/%s/%s", tenantID, projectID),
		tenantID,
	)
	if err != nil || status != 200 || data == nil {
		return ""
	}
	return strings.TrimSpace(strFromAny(data["name"]))
}

func strFromAny(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return strings.TrimSpace(fmt.Sprintf("%v", v))
}

// handleInternalTasksBatchGet serves POST /api/internal/tasks/batch-get/
// Body: {"task_ids": ["id1","id2",...]}
// Returns: {"tasks": [{id, title, tenant_id, workspace_id}, ...]}
// Max 500 task_ids per request.
func handleInternalTasksBatchGet(w http.ResponseWriter, r *http.Request) {
	if !requireInternalSecret(r) {
		writeError(w, r, http.StatusForbidden, "forbidden")
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeError(w, r, 400, "invalid JSON")
		return
	}
	rawIDs, _ := body["task_ids"].([]interface{})
	if len(rawIDs) == 0 {
		writeJSON(w, 200, map[string]interface{}{"tasks": []interface{}{}})
		return
	}

	// Deduplicate and collect
	ids := make([]string, 0, len(rawIDs))
	seen := map[string]bool{}
	for _, raw := range rawIDs {
		tid := strings.TrimSpace(fmt.Sprintf("%v", raw))
		if tid == "" || tid == "<nil>" || seen[tid] {
			continue
		}
		seen[tid] = true
		ids = append(ids, tid)
		if len(ids) >= 500 {
			break
		}
	}
	if len(ids) == 0 {
		writeJSON(w, 200, map[string]interface{}{"tasks": []interface{}{}})
		return
	}

	// Build IN query
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}

	rows, err := db.Query(`SELECT id, title, tenant_id, workspace_id FROM task_tasks WHERE id IN (`+strings.Join(placeholders, ",")+`)`, args...)
	if err != nil {
		writeError(w, r, 500, err.Error())
		return
	}
	defer rows.Close()

	type taskRow struct {
		ID, Title, TenantID, WorkspaceID string
	}
	tasks := make([]map[string]interface{}, 0, len(ids))
	taskIDsFound := make([]string, 0, len(ids))
	for rows.Next() {
		var tr taskRow
		if err := rows.Scan(&tr.ID, &tr.Title, &tr.TenantID, &tr.WorkspaceID); err != nil {
			continue
		}
		tasks = append(tasks, map[string]interface{}{
			"id":           tr.ID,
			"title":        tr.Title,
			"tenant_id":    tr.TenantID,
			"workspace_id": tr.WorkspaceID,
			"project_ids":  []interface{}{},
		})
		taskIDsFound = append(taskIDsFound, tr.ID)
	}

	// Batch-resolve project_ids via task_projects table
	if len(taskIDsFound) > 0 {
		taskProjPlaceholders := make([]string, len(taskIDsFound))
		taskProjArgs := make([]interface{}, len(taskIDsFound))
		for i, tid := range taskIDsFound {
			taskProjPlaceholders[i] = "?"
			taskProjArgs[i] = tid
		}
		projRows, err := db.Query(`SELECT task_id, project_id FROM task_projects WHERE task_id IN (`+strings.Join(taskProjPlaceholders, ",")+`) ORDER BY created_at`, taskProjArgs...)
		if err == nil {
			defer projRows.Close()
			projMap := make(map[string][]interface{}, len(taskIDsFound))
			for projRows.Next() {
				var tid, pid string
				if err := projRows.Scan(&tid, &pid); err != nil {
					continue
				}
				projMap[tid] = append(projMap[tid], pid)
			}
			for i, t := range tasks {
				if pids, ok := projMap[t["id"].(string)]; ok {
					tasks[i]["project_ids"] = pids
				}
			}
		}
	}

	if tasks == nil {
		tasks = []map[string]interface{}{}
	}
	writeJSON(w, 200, map[string]interface{}{"tasks": tasks})
}

// handleInternalExpirePosts serves POST /api/internal/tasks/expire-posts/
// 每日到期扫描（taskEvents 定时触发）：找出 post_expires_at 已到期的帖子，
// 逐一发布 TASK_POST_EXPIRED（通知用户续存）。幂等：只扫描不修改数据。
func handleInternalExpirePosts(w http.ResponseWriter, r *http.Request) {
	if !requireInternalSecret(r) {
		writeError(w, r, http.StatusForbidden, "forbidden")
		return
	}
	rows, err := db.Query(`SELECT id, tenant_id, workspace_id FROM task_tasks WHERE post_expires_at IS NOT NULL AND post_expires_at < ?`, time.Now().UTC())
	if err != nil {
		writeError(w, r, 500, err.Error())
		return
	}
	type expiredRow struct {
		ID, TenantID, WorkspaceID string
	}
	expired := []expiredRow{}
	for rows.Next() {
		var er expiredRow
		if err := rows.Scan(&er.ID, &er.TenantID, &er.WorkspaceID); err != nil {
			continue
		}
		expired = append(expired, er)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		writeError(w, r, 500, err.Error())
		return
	}

	published := 0
	for _, er := range expired {
		ctx := r.Context()
		if err := publishTaskPostExpiredFn(ctx, er.TenantID, er.WorkspaceID, er.ID); err != nil {
			log.Printf("[taskTaskService] publish TASK_POST_EXPIRED failed task_id=%s: %v", er.ID, err)
			continue
		}
		published++
	}
	writeJSON(w, 200, map[string]interface{}{
		"expired_count": len(expired),
		"published":     published,
	})
}
