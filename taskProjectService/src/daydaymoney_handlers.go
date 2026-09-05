package main

import (
	"encoding/json"
	"net/http"
	"strings"

	"daydaymoneymeta"
)

func handleAidevRoute(w http.ResponseWriter, r *http.Request, tenantID string, segs []string) {
	if len(segs) == 0 {
		writeError(w, r, 404, "not found")
		return
	}
	switch segs[0] {
	case "resolve":
		if r.Method != http.MethodGet {
			writeError(w, r, 405, "method not allowed")
			return
		}
		handleAidevResolve(w, r, tenantID)
	case "parse-yaml":
		if r.Method != http.MethodPost {
			writeError(w, r, 405, "method not allowed")
			return
		}
		handleAidevParseYAML(w, r, tenantID)
	default:
		writeError(w, r, 404, "not found")
	}
}

func handleAidevParseYAML(w http.ResponseWriter, r *http.Request, tenantID string) {
	_ = tenantID
	body, err := readJSONBody(r)
	if err != nil {
		writeError(w, r, 400, "invalid JSON")
		return
	}
	raw := strField(body, "yaml")
	meta, err := daydaymoneymeta.ParseYAML(raw)
	if err != nil {
		writeError(w, r, 400, err.Error())
		return
	}
	writeJSON(w, 200, map[string]interface{}{
		"status":       "success",
		"service_id":   meta.ServiceID,
		"display_name": meta.DisplayName,
		"description":  meta.Description,
		"tags":         meta.Tags,
		"version":      meta.Version,
	})
}

func handleAidevResolve(w http.ResponseWriter, r *http.Request, tenantID string) {
	serviceID := strings.TrimSpace(r.URL.Query().Get("service_id"))
	tag := strings.TrimSpace(r.URL.Query().Get("tag"))
	if serviceID == "" && tag == "" {
		writeError(w, r, 400, "service_id or tag is required")
		return
	}
	needles := []string{}
	if serviceID != "" {
		needles = append(needles, normalizeTag(daydaymoneymeta.ServiceTag(serviceID)))
	}
	if tag != "" {
		needles = append(needles, normalizeTag(tag))
	}
	requireAll := serviceID != "" && tag != ""

	// Use project_tags index when available; fall back to full scan.
	matches := []map[string]interface{}{}
	projectIDs := resolveProjectIDsByTags(tenantID, needles, requireAll)
	if projectIDs != nil {
		// Index path: only fetch matching projects
		for _, pid := range projectIDs {
			var name, tagsRaw string
			if err := db.QueryRow(`SELECT name, COALESCE(tags,'[]') FROM project_entries WHERE id=? AND company_id=?`, pid, tenantID).Scan(&name, &tagsRaw); err != nil {
				continue
			}
			tags := projectTagsAsStrings(tagsRaw)
			matched := matchingTags(tags, needles, requireAll)
			if len(matched) == 0 {
				continue
			}
			appendMatchRows(&matches, tenantID, pid, name, matched)
		}
	} else {
		// Fallback: full scan (index table may be empty for legacy data)
		rows, err := db.Query(`SELECT id,name,COALESCE(tags,'[]') FROM project_entries WHERE company_id=? ORDER BY name`, tenantID)
		if err != nil {
			writeError(w, r, 500, err.Error())
			return
		}
		defer rows.Close()
		for rows.Next() {
			var id, name, tagsRaw string
			if err := rows.Scan(&id, &name, &tagsRaw); err != nil {
				continue
			}
			tags := projectTagsAsStrings(tagsRaw)
			matched := matchingTags(tags, needles, requireAll)
			if len(matched) == 0 {
				continue
			}
			appendMatchRows(&matches, tenantID, id, name, matched)
		}
	}

	writeJSON(w, 200, map[string]interface{}{
		"status": "success",
		"query": map[string]string{
			"service_id": serviceID,
			"tag":        tag,
		},
		"matches": matches,
	})
}

// resolveProjectIDsByTags returns matching project IDs via the project_tags index.
// Returns nil when the index should not be used (e.g., empty or legacy data).
func resolveProjectIDsByTags(tenantID string, needles []string, requireAll bool) []string {
	if len(needles) == 0 {
		return nil
	}
	// Build IN clause
	placeholders := make([]string, len(needles))
	args := make([]interface{}, 0, len(needles)+1)
	args = append(args, tenantID)
	for i, n := range needles {
		placeholders[i] = "?"
		args = append(args, n)
	}
	query := `SELECT DISTINCT pt.project_id FROM project_tags pt
		INNER JOIN project_entries p ON p.id = pt.project_id
		WHERE p.company_id = ? AND pt.tag IN (` + strings.Join(placeholders, ",") + `)`
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	counts := map[string]int{}
	for rows.Next() {
		var pid string
		if err := rows.Scan(&pid); err != nil {
			continue
		}
		counts[pid]++
	}
	if len(counts) == 0 {
		// No results from index — could be legacy data; return nil to trigger fallback.
		// But if the index has rows and nil matched, return empty slice to avoid full scan.
		var tagCount int
		_ = db.QueryRow(`SELECT COUNT(*) FROM project_tags`).Scan(&tagCount)
		if tagCount == 0 {
			return nil // fallback to full scan
		}
		return []string{} // index exists but no match
	}
	if requireAll {
		out := make([]string, 0, len(counts))
		for pid, cnt := range counts {
			if cnt == len(needles) {
				out = append(out, pid)
			}
		}
		if len(out) == 0 {
			return []string{}
		}
		return out
	}
	out := make([]string, 0, len(counts))
	for pid := range counts {
		out = append(out, pid)
	}
	return out
}

func appendMatchRows(matches *[]map[string]interface{}, tenantID, projectID, projectName string, matched []string) {
	wsRows, err := db.Query(`SELECT pw.workspace_id, COALESCE(w.name,'')
		FROM project_workspaces pw LEFT JOIN project_workspace_entries w ON w.id=pw.workspace_id
		WHERE pw.project_id=? ORDER BY w.name`, projectID)
	if err != nil {
		return
	}
	defer wsRows.Close()
	wsCount := 0
	for wsRows.Next() {
		var wsID, wsName string
		if err := wsRows.Scan(&wsID, &wsName); err != nil {
			continue
		}
		wsCount++
		*matches = append(*matches, map[string]interface{}{
			"company_id":     tenantID,
			"project_id":     projectID,
			"project_name":   projectName,
			"workspace_id":   wsID,
			"workspace_name": wsName,
			"matched_tags":   matched,
		})
	}
	if wsCount == 0 {
		*matches = append(*matches, map[string]interface{}{
			"company_id":     tenantID,
			"project_id":     projectID,
			"project_name":   projectName,
			"workspace_id":   "",
			"workspace_name": "",
			"matched_tags":   matched,
		})
	}
}

func projectTagsAsStrings(tagsRaw string) []string {
	raw := parseJSONField(tagsRaw, []interface{}{})
	arr, _ := raw.([]interface{})
	out := make([]string, 0, len(arr))
	for _, v := range arr {
		s, ok := v.(string)
		if !ok {
			continue
		}
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

// normalizeTag lowercases and trims a tag for index lookups (case-insensitive matching).
func normalizeTag(tag string) string {
	return strings.ToLower(strings.TrimSpace(tag))
}

// syncProjectTags replaces the project_tags index rows for a given project.
func syncProjectTags(projectID string, tagsRaw string) {
	_, _ = db.Exec(`DELETE FROM project_tags WHERE project_id=?`, projectID)
	tags := projectTagsAsStrings(tagsRaw)
	for _, t := range tags {
		nt := normalizeTag(t)
		if nt == "" {
			continue
		}
		_, _ = db.Exec(`INSERT IGNORE INTO project_tags(tag, project_id) VALUES(?,?)`, nt, projectID)
	}
}

// OPT-20260719-019: mergeAidevServiceTagsFromYAML parses daydaymoney.yaml content and
// ensures svc:{service_id} is present in the project's tags. Call this from
// create/sync handlers once git file reading is available.
func mergeAidevServiceTagsFromYAML(projectID string, yamlContent string) bool {
	yamlContent = strings.TrimSpace(yamlContent)
	if yamlContent == "" {
		return false
	}
	meta, err := daydaymoneymeta.ParseYAML(yamlContent)
	if err != nil || meta.ServiceID == "" {
		return false
	}
	svcTag := normalizeTag(daydaymoneymeta.ServiceTag(meta.ServiceID))
	if svcTag == "" {
		return false
	}
	// Check if tag already exists
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM project_tags WHERE project_id=? AND tag=?`, projectID, svcTag).Scan(&count)
	if count > 0 {
		return false // already present
	}
	_, _ = db.Exec(`INSERT IGNORE INTO project_tags(tag, project_id) VALUES(?,?)`, svcTag, projectID)
	return true
}

// deleteProjectTags removes all project_tags rows for a project.
func deleteProjectTags(projectID string) {
	_, _ = db.Exec(`DELETE FROM project_tags WHERE project_id=?`, projectID)
}

// matchingTags: if requireAll, every needle must match; else any needle.
func matchingTags(tags, needles []string, requireAll bool) []string {
	hit := make([]string, 0, len(needles))
	for _, n := range needles {
		if daydaymoneymeta.TagsContain(tags, n) {
			hit = append(hit, n)
		}
	}
	if requireAll && len(hit) != len(needles) {
		return nil
	}
	if !requireAll && len(hit) == 0 {
		return nil
	}
	return hit
}

func projectMatchesTagFilter(tagsRaw string, tagFilter string) bool {
	tagFilter = strings.TrimSpace(tagFilter)
	if tagFilter == "" {
		return true
	}
	return daydaymoneymeta.TagsContain(projectTagsAsStrings(tagsRaw), tagFilter)
}

// used by list to keep JSON tags stable when filtering
func tagsRawFromProjectMap(p map[string]interface{}) string {
	v, ok := p["tags"]
	if !ok {
		return "[]"
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "[]"
	}
	return string(b)
}
