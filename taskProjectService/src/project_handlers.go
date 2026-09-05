package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"taskProjectService/domain"
)

// --- Project Handlers ---

func handleListProjects(w http.ResponseWriter, r *http.Request) {
	// 防御：非 GET 落此函数即静默 200 []（OPT-20260807-021，如 /api/projects 裸路径 POST）
	if r.Method != http.MethodGet {
		writeError(w, r, 405, "method not allowed")
		return
	}
	tenantID := getAuthTenant(r)
	workspaceFilter := r.URL.Query().Get("workspace_id")
	tagFilter := strings.TrimSpace(r.URL.Query().Get("tag"))
	query := `SELECT id,name,COALESCE(description,''),company_id,installed_image_id,container_image_name,COALESCE(tags,'[]'),COALESCE(server_run_template,'{}'),created_at,updated_at
		FROM project_entries WHERE company_id=?`
	args := []interface{}{tenantID}
	if workspaceFilter != "" {
		query = `SELECT p.id,p.name,COALESCE(p.description,''),p.company_id,p.installed_image_id,p.container_image_name,COALESCE(p.tags,'[]'),COALESCE(p.server_run_template,'{}'),p.created_at,p.updated_at
			FROM project_entries p INNER JOIN project_workspaces pw ON pw.project_id=p.id
			WHERE p.company_id=? AND pw.workspace_id=?`
		args = append(args, workspaceFilter)
	}
	query += " ORDER BY name"
	rows, err := db.Query(query, args...)
	if err != nil {
		writeError(w, r, 500, err.Error())
		return
	}
	defer rows.Close()
	projects := []map[string]interface{}{}
	for rows.Next() {
		p, err := scanProjectRow(rows)
		if err != nil {
			continue
		}
		if tagFilter != "" && !projectMatchesTagFilter(tagsRawFromProjectMap(p), tagFilter) {
			continue
		}
		detail, _ := loadProjectDetail(p["id"].(string))
		if detail != nil {
			// OPT-20260902-020: 列表 JSON 对已授权仓 hydrate 真实 token_available（本地 grant 查询，无远端探测）
			hydrateProjectGitReposStatusLocal(detail, getAuthUser(r))
			projects = append(projects, detail)
		} else {
			projects = append(projects, p)
		}
	}
	writeJSON(w, 200, projects)
}

func scanProjectRow(scanner interface {
	Scan(dest ...interface{}) error
}) (map[string]interface{}, error) {
	var id, name, desc, companyID, imageID, imageName, tagsRaw, tmplRaw string
	var ca, ua time.Time
	if err := scanner.Scan(&id, &name, &desc, &companyID, &imageID, &imageName, &tagsRaw, &tmplRaw, &ca, &ua); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"id": id, "name": name, "description": desc, "company": companyID,
		"container_image_id": imageID, "container_image": imageName,
		"tags":                parseJSONField(tagsRaw, []interface{}{}),
		"server_run_template": parseJSONField(tmplRaw, map[string]interface{}{}),
		"created_at":          ca.UTC().Format(time.RFC3339Nano),
		"updated_at":          ua.UTC().Format(time.RFC3339Nano),
	}, nil
}

func handleCreateProject(w http.ResponseWriter, r *http.Request) {
	tenantID := getAuthTenant(r)
	body, err := readJSONBody(r)
	if err != nil {
		writeError(w, r, 400, "invalid JSON")
		return
	}
	id := genID("proj")
	name := strField(body, "name")
	if name == "" {
		writeError(w, r, 400, "name is required")
		return
	}
	var repoEntries []gitRepoEntry
	if repos, ok := body["git_repos"].([]interface{}); ok {
		repoEntries = parseGitRepoItems(repos)
		if errMsg := duplicateCloneAliasError(repoEntries); errMsg != "" {
			writeErrorMap(w, r, 400, map[string]interface{}{"error": errMsg, "git_repos": errMsg})
			return
		}
	}
	tags := "[]"
	if v, ok := body["tags"]; ok {
		b, _ := json.Marshal(v)
		tags = string(b)
	}
	tmpl := "{}"
	if v, ok := body["server_run_template"]; ok {
		b, _ := json.Marshal(v)
		tmpl = string(b)
	}
	imageID := strField(body, "container_image_id")
	if imageID == "" {
		imageID = strField(body, "installed_image_id")
	}
	imageName := strField(body, "container_image")
	autoCloneNested := 1
	if !boolFieldDefault(body, "auto_clone_nested_repos", true) {
		autoCloneNested = 0
	}
	tx, err := db.Begin()
	if err != nil {
		writeError(w, r, 500, err.Error())
		return
	}
	_, err = tx.Exec(`INSERT INTO project_entries(id,name,description,company_id,installed_image_id,container_image_name,tags,server_run_template,auto_clone_nested_repos) VALUES(?,?,?,?,?,?,?,?,?)`,
		id, name, strField(body, "description"), tenantID, imageID, imageName, tags, tmpl, autoCloneNested)
	if err != nil {
		_ = tx.Rollback()
		writeError(w, r, 500, err.Error())
		return
	}
	revID, err := persistProjectRevisionV1(tx, tenantID, id, name, strField(body, "description"), getAuthUser(r), time.Now().UTC())
	if err != nil {
		_ = tx.Rollback()
		writeError(w, r, 500, err.Error())
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, r, 500, err.Error())
		return
	}
	if wsIDs, ok := body["workspaces_ids"].([]interface{}); ok {
		for _, ws := range wsIDs {
			db.Exec("INSERT IGNORE INTO project_workspaces(project_id, workspace_id) VALUES(?,?)", id, fmt.Sprintf("%v", ws))
		}
	}
	for _, entry := range repoEntries {
		db.Exec(
			"INSERT INTO project_repos(id,project_id,repo_url,clone_alias) VALUES(?,?,?,?)",
			genID("prepo"), id, entry.URL, entry.CloneAlias,
		)
	}
	applyCreateProjectGrantTickets(
		id,
		tenantID,
		getAuthUser(r),
		repoEntries,
		grantTicketsFromCreateBody(body),
		r.Header.Get("X-Trace-Id"),
	)
	syncProjectTags(id, tags)
	logInfo("project created: "+id, r.Header.Get("X-Trace-Id"))
	if revID != "" {
		if pubErr := publishProjectRevisionRecordedFn(r.Context(), tenantID, id, revID, 1, getAuthUser(r), "name,description"); pubErr != nil {
			logWarn("PROJECT_REVISION_RECORDED publish failed: "+pubErr.Error(), r.Header.Get("X-Trace-Id"))
		}
	}
	detail, _ := loadProjectDetail(id)
	writeJSON(w, 201, detail)
}

func handleGetProject(w http.ResponseWriter, r *http.Request) {
	pid := resolveRequestID(idAliasKindProject, r.Header.Get("X-Resource-Id"), r.Header.Get("X-Trace-Id"))
	detail, err := loadProjectDetail(pid)
	if err == sql.ErrNoRows {
		writeError(w, r, 404, "project not found")
		return
	}
	if err != nil {
		writeError(w, r, 500, err.Error())
		return
	}
	if rejectIfProjectNotInTenant(w, r, detail, "") {
		return
	}
	// B-086: GitLab OAuth/probe and disk-size must not block page data GET.
	// Live token_status comes from POST /api/projects/validate-git-repos/.
	// OPT-20260902-020: only local grant-based hydrate (no git-oauth / remote), so the
	// db-only contract still holds; authorized repos surface token_available for the
	// create-task OAuth gate to skip the validate POST.
	hydrateProjectGitReposStatusLocal(detail, getAuthUser(r))
	writeJSON(w, 200, detail)
}

func handleUpdateProject(w http.ResponseWriter, r *http.Request) {
	pid := resolveRequestID(idAliasKindProject, r.Header.Get("X-Resource-Id"), r.Header.Get("X-Trace-Id"))
	body, err := readJSONBody(r)
	if err != nil {
		writeError(w, r, 400, "invalid JSON")
		return
	}
	var tenantID, currentImage, currentImageName, tmplRaw, currentName, currentDesc string
	err = db.QueryRow(
		`SELECT company_id, COALESCE(installed_image_id,''), COALESCE(container_image_name,''), COALESCE(server_run_template,'{}'), name, COALESCE(description,'') FROM project_entries WHERE id=?`,
		pid,
	).Scan(&tenantID, &currentImage, &currentImageName, &tmplRaw, &currentName, &currentDesc)
	if err == sql.ErrNoRows {
		writeError(w, r, 404, "project not found")
		return
	}
	if err != nil {
		writeError(w, r, 500, "project lookup failed")
		return
	}
	if rejectIfProjectNotInTenant(w, r, map[string]interface{}{"id": pid, "company": tenantID}, "") {
		return
	}
	healedImageID, err := validateProjectUpdateImageTemplate(tenantID, currentImage, currentImageName, tmplRaw, body)
	if err != nil {
		logWarn("project image/template architecture rejected: "+err.Error(), r.Header.Get("X-Trace-Id"))
		writeError(w, r, 400, err.Error())
		return
	}
	if healedImageID != "" {
		body["container_image_id"] = healedImageID
		if strField(body, "container_image") == "" && strings.TrimSpace(currentImageName) != "" {
			body["container_image"] = currentImageName
		}
		logWarn("project installed image id healed to "+healedImageID, r.Header.Get("X-Trace-Id"))
	}
	newName := currentName
	if v := strField(body, "name"); v != "" {
		newName = v
	}
	newDesc := currentDesc
	if v, ok := body["description"]; ok {
		newDesc = fmt.Sprintf("%v", v)
	}
	tx, txErr := db.Begin()
	if txErr != nil {
		writeError(w, r, 500, txErr.Error())
		return
	}
	if v := strField(body, "name"); v != "" {
		if _, err := tx.Exec("UPDATE project_entries SET name=? WHERE id=?", v, pid); err != nil {
			_ = tx.Rollback()
			writeError(w, r, 500, err.Error())
			return
		}
	}
	if v, ok := body["description"]; ok {
		if _, err := tx.Exec("UPDATE project_entries SET description=? WHERE id=?", fmt.Sprintf("%v", v), pid); err != nil {
			_ = tx.Rollback()
			writeError(w, r, 500, err.Error())
			return
		}
	}
	revID, revNum, revChanged, revRecorded, revErr := persistProjectRevisionIfChanged(
		tx, tenantID, pid, getAuthUser(r),
		domain.VersionedProjectContent{Name: currentName, Description: currentDesc},
		domain.VersionedProjectContent{Name: newName, Description: newDesc},
		time.Now().UTC(),
	)
	if revErr != nil {
		_ = tx.Rollback()
		writeError(w, r, 500, revErr.Error())
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, r, 500, err.Error())
		return
	}
	if v, ok := body["tags"]; ok {
		b, _ := json.Marshal(v)
		db.Exec("UPDATE project_entries SET tags=? WHERE id=?", string(b), pid)
		syncProjectTags(pid, string(b))
	}
	if v, ok := body["server_run_template"]; ok {
		b, _ := json.Marshal(v)
		db.Exec("UPDATE project_entries SET server_run_template=? WHERE id=?", string(b), pid)
	}
	if v, ok := body["container_image_id"]; ok {
		imageID := strings.TrimSpace(fmt.Sprintf("%v", v))
		imageName := strField(body, "container_image")
		if imageID != "" && imageName == "" {
			var tenantID string
			_ = db.QueryRow("SELECT company_id FROM project_entries WHERE id=?", pid).Scan(&tenantID)
			if resolved := resolveInstalledImageName(tenantID, imageID); resolved != "" {
				imageName = resolved
			}
		}
		db.Exec("UPDATE project_entries SET installed_image_id=?, container_image_name=? WHERE id=?", imageID, imageName, pid)
	}
	if wsIDs, ok := body["workspaces_ids"].([]interface{}); ok {
		db.Exec("DELETE FROM project_workspaces WHERE project_id=?", pid)
		for _, ws := range wsIDs {
			db.Exec("INSERT IGNORE INTO project_workspaces(project_id, workspace_id) VALUES(?,?)", pid, fmt.Sprintf("%v", ws))
		}
	}
	if repos, ok := body["git_repos"].([]interface{}); ok {
		entries := parseGitRepoItems(repos)
		if errMsg := duplicateCloneAliasError(entries); errMsg != "" {
			writeErrorMap(w, r, 400, map[string]interface{}{"error": errMsg, "git_repos": errMsg})
			return
		}
		db.Exec("DELETE FROM project_repos WHERE project_id=?", pid)
		for _, entry := range entries {
			db.Exec(
				"INSERT INTO project_repos(id,project_id,repo_url,clone_alias) VALUES(?,?,?,?)",
				genID("prepo"), pid, entry.URL, entry.CloneAlias,
			)
		}
	}
	if _, ok := body["auto_clone_nested_repos"]; ok {
		autoCloneNested := 1
		if !boolFieldDefault(body, "auto_clone_nested_repos", true) {
			autoCloneNested = 0
		}
		db.Exec("UPDATE project_entries SET auto_clone_nested_repos=? WHERE id=?", autoCloneNested, pid)
	}
	db.Exec("UPDATE project_entries SET updated_at=CURRENT_TIMESTAMP WHERE id=?", pid)
	if revRecorded {
		if pubErr := publishProjectRevisionRecordedFn(r.Context(), tenantID, pid, revID, revNum, getAuthUser(r), domain.JoinChangedFields(revChanged)); pubErr != nil {
			logWarn("PROJECT_REVISION_RECORDED publish failed: "+pubErr.Error(), r.Header.Get("X-Trace-Id"))
		}
	}
	handleGetProject(w, r)
}

func handleDeleteProject(w http.ResponseWriter, r *http.Request) {
	pid := resolveRequestID(idAliasKindProject, r.Header.Get("X-Resource-Id"), r.Header.Get("X-Trace-Id"))
	detail, err := loadProjectDetail(pid)
	if err == sql.ErrNoRows {
		writeError(w, r, 404, "project not found")
		return
	}
	if err != nil {
		writeError(w, r, 500, err.Error())
		return
	}
	if rejectIfProjectNotInTenant(w, r, detail, "") {
		return
	}
	db.Exec("DELETE FROM project_repos WHERE project_id=?", pid)
	db.Exec("DELETE FROM project_workspaces WHERE project_id=?", pid)
	db.Exec("DELETE FROM project_entries WHERE id=?", pid)
	deleteProjectTags(pid)
	tenantID, _ := detail["company"].(string)
	if pubErr := publishProjectDeletedFn(r.Context(), tenantID, pid); pubErr != nil {
		logWarn("PROJECT_DELETED publish failed: "+pubErr.Error(), r.Header.Get("X-Trace-Id"))
	}
	logInfo("project deleted: "+pid, r.Header.Get("X-Trace-Id"))
	writeJSON(w, 200, map[string]string{"status": "deleted"})
}

// handleInternalProjectsBatchGet serves POST /api/internal/projects/batch-get/
// Body: {"project_ids": ["id1","id2",...]}
// Returns: {"projects": [{id, name, company_id}, ...]}
// Max 500 project_ids per request.
func handleInternalProjectsBatchGet(w http.ResponseWriter, r *http.Request) {
	body, err := readJSONBody(r)
	if err != nil {
		writeError(w, r, 400, "invalid JSON")
		return
	}
	rawIDs, _ := body["project_ids"].([]interface{})
	if len(rawIDs) == 0 {
		writeJSON(w, 200, map[string]interface{}{"projects": []interface{}{}})
		return
	}

	// Deduplicate and collect
	ids := make([]string, 0, len(rawIDs))
	seen := map[string]bool{}
	for _, raw := range rawIDs {
		pid := strings.TrimSpace(fmt.Sprintf("%v", raw))
		pid = resolveStoredID(idAliasKindProject, pid)
		if pid == "" || pid == "<nil>" || seen[pid] {
			continue
		}
		seen[pid] = true
		ids = append(ids, pid)
		if len(ids) >= 500 {
			break
		}
	}
	if len(ids) == 0 {
		writeJSON(w, 200, map[string]interface{}{"projects": []interface{}{}})
		return
	}

	// Build IN query
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}

	rows, err := db.Query(`SELECT id, name, company_id FROM project_entries WHERE id IN (`+strings.Join(placeholders, ",")+`)`, args...)
	if err != nil {
		writeError(w, r, 500, err.Error())
		return
	}
	defer rows.Close()

	projects := make([]map[string]interface{}, 0, len(ids))
	for rows.Next() {
		var id, name, companyID string
		if err := rows.Scan(&id, &name, &companyID); err != nil {
			continue
		}
		projects = append(projects, map[string]interface{}{
			"id":         id,
			"name":       name,
			"company_id": companyID,
		})
	}
	if projects == nil {
		projects = []map[string]interface{}{}
	}
	writeJSON(w, 200, map[string]interface{}{"projects": projects})
}
