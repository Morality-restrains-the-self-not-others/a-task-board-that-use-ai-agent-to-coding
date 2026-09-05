package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

func canonicalRepoKey(repoURL string) string {
	raw := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(repoURL), "/"))
	if strings.HasSuffix(strings.ToLower(raw), ".git") {
		raw = raw[:len(raw)-4]
	}
	return strings.ToLower(raw)
}

func collectImportedRepoKeys(tenantID string) (singleKeys, anyKeys map[string]bool) {
	singleKeys = map[string]bool{}
	anyKeys = map[string]bool{}
	rows, err := db.Query(`
		SELECT pr.repo_url, COUNT(*) OVER (PARTITION BY pr.project_id) AS repo_count
		FROM project_repos pr
		INNER JOIN project_entries p ON p.id = pr.project_id
		WHERE p.company_id=?`, tenantID)
	if err != nil {
		return singleKeys, anyKeys
	}
	defer rows.Close()
	for rows.Next() {
		var url string
		var count int
		if rows.Scan(&url, &count) != nil {
			continue
		}
		key := canonicalRepoKey(url)
		anyKeys[key] = true
		if count == 1 {
			singleKeys[key] = true
		}
	}
	return singleKeys, anyKeys
}

func mergeGitlabImportFlags(payload map[string]interface{}, tenantID string) {
	singleKeys, anyKeys := collectImportedRepoKeys(tenantID)
	repos, ok := payload["repos"].([]interface{})
	if !ok {
		return
	}
	for _, item := range repos {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		url, _ := m["http_url_to_repo"].(string)
		key := canonicalRepoKey(url)
		m["imported_in_single_repo_project"] = singleKeys[key]
		m["imported_in_any_project"] = anyKeys[key]
	}
}

func handleGitlabRemoteRepos(w http.ResponseWriter, r *http.Request, tenantID string) {
	if r.Method != http.MethodGet {
		writeError(w, r, 405, "method not allowed")
		return
	}
	userID := getAuthUser(r)
	gitlabHost := strings.TrimSpace(r.URL.Query().Get("gitlab_host"))

	// Resolve provider for the given GitLab host (empty host uses default/fallback).
	entry, ok := findProviderByGitlabHost(gitlabHost)
	if !ok {
		writeErrorMap(w, r, 400, map[string]interface{}{
			"error":       "未找到匹配的 GitLab 站点配置，请确认 gitlab_host 参数",
			"oauth_bound": false,
		})
		return
	}

	// Always include provider metadata so the frontend can drive OAuth start flows.
	basePayload := map[string]interface{}{
		"provider_key":   entry.ProviderKey,
		"gitlab_website": entry.WebsiteOrigin,
	}

	// Fetch OAuth token for the user — 统一走 gitsite 路径（OPT-20260827-036），
	// 按 GitLab 站点 host 寻址，不再按 provider_key 分叉。
	// SPA 首屏可不传 gitlab_host（findProviderByGitlabHost 已 fallback）；换票须用
	// WebsiteOrigin/Host，否则 extractHostNetloc("") → "missing repo host"。
	tokenRepoURL := gitlabHost
	if strings.TrimSpace(tokenRepoURL) == "" {
		tokenRepoURL = entry.WebsiteOrigin
	}
	if strings.TrimSpace(tokenRepoURL) == "" {
		tokenRepoURL = entry.Host
	}
	uid, _ := strconv.ParseInt(strings.TrimSpace(userID), 10, 64)
	token, tokenErr := fetchGitAccessToken(uid, tokenRepoURL, entry.GitoauthBase, traceHeadersFromRequest(r))
	if token == "" {
		msg := "无法获取 GitLab 授权：请先在个人资料完成 GitLab 绑定"
		if tokenErr != "" {
			msg = "无法获取 GitLab 授权：" + tokenErr
		}
		basePayload["oauth_bound"] = false
		basePayload["error"] = msg
		writeErrorMap(w, r, 401, basePayload)
		return
	}

	// List user's GitLab projects
	projects, err := listGitLabUserProjects(entry.WebsiteOrigin, token)
	if err != "" {
		basePayload["oauth_bound"] = true
		basePayload["error"] = err
		writeErrorMap(w, r, 502, basePayload)
		return
	}

	// Fetch GitLab username for display in the frontend modal header.
	if glUser, glUserErr := fetchGitLabUsername(entry.WebsiteOrigin, token); glUserErr == "" && glUser != "" {
		basePayload["gitlab_login"] = glUser
	}
	basePayload["repos"] = projects
	basePayload["oauth_bound"] = true
	mergeGitlabImportFlags(basePayload, tenantID)
	writeJSON(w, 200, basePayload)
}

func handleBatchFromGitlabRepos(w http.ResponseWriter, r *http.Request, tenantID string) {
	if r.Method != http.MethodPost {
		writeError(w, r, 405, "method not allowed")
		return
	}
	delegateGitlabCreate(w, r, tenantID, false)
}

func handleCombinedFromGitlabRepos(w http.ResponseWriter, r *http.Request, tenantID string) {
	if r.Method != http.MethodPost {
		writeError(w, r, 405, "method not allowed")
		return
	}
	delegateGitlabCreate(w, r, tenantID, true)
}

func delegateGitlabCreate(w http.ResponseWriter, r *http.Request, tenantID string, combined bool) {
	body, err := readJSONBody(r)
	if err != nil {
		writeError(w, r, 400, "invalid JSON")
		return
	}
	wsID := strField(body, "workspace_id")
	if wsID == "" {
		writeError(w, r, 400, "缺少 workspace_id")
		return
	}
	var wsCompany string
	if err := db.QueryRow("SELECT company_id FROM project_workspace_entries WHERE id=? AND company_id=?", wsID, tenantID).Scan(&wsCompany); err == sql.ErrNoRows {
		writeError(w, r, 404, "工作空间不存在或无权访问")
		return
	}
	repos, _ := body["repos"].([]interface{})
	if len(repos) == 0 {
		writeError(w, r, 400, "repos 不能为空")
		return
	}
	userID := getAuthUser(r)
	_ = userID // used for auth context; project creation does not need per-repo user binding

	// Create projects directly in Go (Django task2app has been retired).
	if combined {
		id := genID("proj")
		name := strField(body, "name")
		if name == "" {
			writeError(w, r, 400, "name is required")
			return
		}
		desc := strField(body, "description")
		_, err := db.Exec(
			`INSERT INTO project_entries(id,name,description,company_id,server_run_template) VALUES(?,?,?,?,?)`,
			id, name, desc, tenantID, "{}",
		)
		if err != nil {
			writeError(w, r, 500, err.Error())
			return
		}
		db.Exec("INSERT IGNORE INTO project_workspaces(project_id, workspace_id) VALUES(?,?)", id, wsID)
		for _, repo := range repos {
			m, ok := repo.(map[string]interface{})
			if !ok {
				continue
			}
			repoURL := strField(m, "http_url_to_repo")
			if repoURL == "" {
				repoURL = strField(m, "url")
			}
			if repoURL == "" {
				continue
			}
			db.Exec("INSERT INTO project_repos(id,project_id,repo_url) VALUES(?,?,?)", genID("prepo"), id, repoURL)
		}
		detail, _ := loadProjectDetail(id)
		writeJSON(w, 201, detail)
		return
	}

	// Batch: one project per repo
	var created []map[string]interface{}
	for _, repo := range repos {
		m, ok := repo.(map[string]interface{})
		if !ok {
			continue
		}
		repoURL := strField(m, "http_url_to_repo")
		if repoURL == "" {
			repoURL = strField(m, "url")
		}
		if repoURL == "" {
			continue
		}
		name := strField(m, "name")
		if name == "" {
			name = repoNameFromURL(repoURL)
		}
		id := genID("proj")
		_, err := db.Exec(
			`INSERT INTO project_entries(id,name,description,company_id,server_run_template) VALUES(?,?,?,?,?)`,
			id, name, "", tenantID, "{}",
		)
		if err != nil {
			continue
		}
		db.Exec("INSERT IGNORE INTO project_workspaces(project_id, workspace_id) VALUES(?,?)", id, wsID)
		db.Exec("INSERT INTO project_repos(id,project_id,repo_url) VALUES(?,?,?)", genID("prepo"), id, repoURL)
		detail, _ := loadProjectDetail(id)
		if detail != nil {
			created = append(created, detail)
		}
	}
	writeJSON(w, 201, map[string]interface{}{"projects": created, "count": len(created)})
}

// findProviderByGitlabHost matches a GitLab host (e.g. "gitlab.daydaymoney.com") to a provider entry.
// When host is empty, falls back to the first gitlab provider (auto-detection for SPA
// clients that don't know the host ahead of time — they get it back in the response).
func findProviderByGitlabHost(host string) (providerEntry, bool) {
	host = strings.TrimSpace(strings.ToLower(host))
	if providerResolver == nil {
		return providerEntry{}, false
	}
	// Strip scheme if present
	if strings.HasPrefix(host, "http://") || strings.HasPrefix(host, "https://") {
		u, err := url.Parse(host)
		if err == nil && u.Host != "" {
			host = strings.ToLower(u.Host)
		}
	}
	if host != "" {
		for _, e := range providerResolver.entries {
			if strings.EqualFold(e.Host, host) || strings.EqualFold(e.Netloc, host) {
				return e, true
			}
		}
	}
	// Fallback: match any gitlab provider by prefix (also handles empty host)
	for _, e := range providerResolver.entries {
		if strings.HasPrefix(e.ProviderKey, "gitlab:") {
			return e, true
		}
	}
	return providerEntry{}, false
}

// listGitLabUserProjects lists the authenticated user's GitLab projects via API v4.
func listGitLabUserProjects(gitlabOrigin, token string) ([]map[string]interface{}, string) {
	origin := strings.TrimRight(strings.TrimSpace(gitlabOrigin), "/")
	if origin == "" {
		return nil, "GitLab host not configured"
	}
	apiURL := origin + "/api/v4/projects?membership=true&simple=true&per_page=100&order_by=last_activity_at"
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, "构建 GitLab API 请求失败: " + err.Error()
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := gitHTTPClient.Do(req)
	if err != nil {
		return nil, "GitLab API 请求失败: " + err.Error()
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, "GitLab 授权已过期，请重新完成 GitLab 绑定"
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Sprintf("GitLab API 返回 %d: %s", resp.StatusCode, truncateStr(string(raw), 200))
	}

	var projects []map[string]interface{}
	if err := json.Unmarshal(raw, &projects); err != nil {
		return nil, "解析 GitLab API 响应失败: " + err.Error()
	}

	// Normalize field names to match expected format
	result := make([]map[string]interface{}, 0, len(projects))
	for _, p := range projects {
		item := map[string]interface{}{
			"id":                  p["id"],
			"name":                p["name"],
			"path_with_namespace": p["path_with_namespace"],
			"http_url_to_repo":    p["http_url_to_repo"],
			"description":         p["description"],
			"last_activity_at":    p["last_activity_at"],
		}
		result = append(result, item)
	}
	return result, ""
}

// repoNameFromURL extracts a human-readable repo name from a Git URL.
func repoNameFromURL(repoURL string) string {
	s := strings.TrimSpace(repoURL)
	// Remove trailing .git and slashes
	s = strings.TrimSuffix(s, "/")
	s = strings.TrimSuffix(s, ".git")
	// Take last segment
	idx := strings.LastIndex(s, "/")
	if idx >= 0 {
		return s[idx+1:]
	}
	return s
}

func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// fetchGitLabUsername calls the GitLab /api/v4/user endpoint to retrieve
// the authenticated user's username for display in the frontend modal.
func fetchGitLabUsername(gitlabOrigin, token string) (string, string) {
	origin := strings.TrimRight(strings.TrimSpace(gitlabOrigin), "/")
	if origin == "" {
		return "", "GitLab host not configured"
	}
	req, err := http.NewRequest(http.MethodGet, origin+"/api/v4/user", nil)
	if err != nil {
		return "", "构建 GitLab API 请求失败: " + err.Error()
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := gitHTTPClient.Do(req)
	if err != nil {
		return "", "GitLab API 请求失败: " + err.Error()
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Sprintf("GitLab API 返回 %d", resp.StatusCode)
	}

	var user struct {
		Username string `json:"username"`
	}
	if err := json.Unmarshal(raw, &user); err != nil {
		return "", "解析 GitLab 用户信息失败"
	}
	return user.Username, ""
}
