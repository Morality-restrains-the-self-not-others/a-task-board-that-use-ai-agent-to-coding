package main

import (
	"database/sql"
	"encoding/json"
	"strings"
	"time"
)

func parseJSONField(raw string, fallback interface{}) interface{} {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}
	switch fallback.(type) {
	case map[string]interface{}:
		out := map[string]interface{}{}
		if err := json.Unmarshal([]byte(raw), &out); err != nil {
			return fallback
		}
		return out
	case []interface{}:
		out := []interface{}{}
		if err := json.Unmarshal([]byte(raw), &out); err != nil {
			return fallback
		}
		return out
	default:
		return fallback
	}
}

func loadProjectDetail(projectID string) (map[string]interface{}, error) {
	row := db.QueryRow(`
		SELECT id,name,COALESCE(description,''),company_id,installed_image_id,container_image_name,COALESCE(tags,'[]'),COALESCE(server_run_template,'{}'),
			COALESCE(auto_clone_nested_repos,1),created_at,updated_at
		FROM project_entries WHERE id=?`, projectID)
	var id, name, desc, companyID, imageID, imageName, tagsRaw, tmplRaw string
	var autoCloneNested int
	var ca, ua time.Time
	if err := row.Scan(&id, &name, &desc, &companyID, &imageID, &imageName, &tagsRaw, &tmplRaw, &autoCloneNested, &ca, &ua); err != nil {
		return nil, err
	}

	workspaceIDs := []string{}
	wsRows, _ := db.Query("SELECT workspace_id FROM project_workspaces WHERE project_id=? ORDER BY workspace_id", projectID)
	if wsRows != nil {
		defer wsRows.Close()
		for wsRows.Next() {
			var wid string
			wsRows.Scan(&wid)
			workspaceIDs = append(workspaceIDs, wid)
		}
	}

	gitRepos := []string{}
	gitRepoEntries := []map[string]interface{}{}
	repoRows, _ := db.Query(
		"SELECT repo_url, COALESCE(clone_alias, '') FROM project_repos WHERE project_id=? ORDER BY created_at",
		projectID,
	)
	if repoRows != nil {
		defer repoRows.Close()
		for repoRows.Next() {
			var url, alias string
			repoRows.Scan(&url, &alias)
			gitRepos = append(gitRepos, url)
			gitRepoEntries = append(gitRepoEntries, map[string]interface{}{
				"url":         url,
				"clone_alias": alias,
			})
		}
	}

	gitReposStatus := []map[string]interface{}{}
	for _, url := range gitRepos {
		gitReposStatus = append(gitReposStatus, map[string]interface{}{
			"repo_url":               url,
			"token_status":           "not_applicable",
			"oauth_provider":         "",
			"oauth_service_provider": "",
		})
	}

	tags, _ := parseJSONField(tagsRaw, []interface{}{}).([]interface{})
	tmpl, _ := parseJSONField(tmplRaw, map[string]interface{}{}).(map[string]interface{})

	if strings.TrimSpace(imageName) == "" && strings.TrimSpace(imageID) != "" {
		if resolved := resolveInstalledImageName(companyID, imageID); resolved != "" {
			imageName = resolved
			_, _ = db.Exec(
				`UPDATE project_entries SET container_image_name=? WHERE id=? AND (container_image_name IS NULL OR container_image_name='')`,
				imageName, id,
			)
		}
	}

	return map[string]interface{}{
		"id":                      id,
		"name":                    name,
		"description":             desc,
		"tags":                    tags,
		"container_image":         imageName,
		"container_image_id":      imageID,
		"workspaces":              workspaceIDs,
		"server_run_template":     tmpl,
		"company":                 companyID,
		"auto_clone_nested_repos": autoCloneNested != 0,
		"created_at":              ca.UTC().Format(time.RFC3339Nano),
		"updated_at":              ua.UTC().Format(time.RFC3339Nano),
		"git_repos":               gitRepos,
		"git_repo_entries":        gitRepoEntries,
		"git_repos_status":        gitReposStatus,
	}, nil
}

func loadWorkspaceRow(row *sql.Row) (map[string]interface{}, error) {
	var id, name, desc, cid, dsid, dsfrom, tier string
	var isDef, allowPP, containerImageAtMode bool
	var ca, ua time.Time
	if err := row.Scan(&id, &name, &desc, &cid, &dsid, &dsfrom, &isDef, &tier, &allowPP, &containerImageAtMode, &ca, &ua); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"id":          id,
		"name":        name,
		"description": desc,
		"company_id":  cid,
		// Company display name is owned by accounts; do not echo company_id here.
		"company_name":                    "",
		"deliverable_system_id":           dsid,
		"deliverable_system_from":         dsfrom,
		"is_default":                      isDef,
		"is_current":                      false,
		"task_archive_tier":               tier,
		"allow_personal_feature_params":   allowPP,
		"container_image_at_mode_enabled": containerImageAtMode,
		"created_at":                      ca.UTC().Format(time.RFC3339Nano),
		"updated_at":                      ua.UTC().Format(time.RFC3339Nano),
	}, nil
}
