package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

// github-credential-status / github-credential-approve Go 重新实现。
// OPT-052: Django saas-backend retired 2026-07-30，原实现为代理 Django
// /cloud/compute/github-credential-{status,approve}/；本文件以 taskGitOauth
// credential-summary + cloud_task_repo_github_bindings 表本地实现：
//   - github_connections ← taskGitOauth git_oauth_appusercredential（用户 GitHub App 连接）
//   - repo_bindings      ← cloud_task_repo_github_bindings（任务级 repo → github_user_id）
//   - 任务关联 repos     ← taskTaskService task detail projects[].stored_repo_address

// githubCredentialConnection is one entry of the gitOauth credential-summary
// connections array (mirrors git_oauth_appusercredential fields).
type githubCredentialConnection struct {
	Connected    bool
	GithubUserID string
	GithubLogin  string
	Scope        string
}

// githubCredentialSummary is the parsed gitOauth credential-summary response.
type githubCredentialSummary struct {
	Connected      bool
	GithubUserID   string
	GithubLogin    string
	Connections    []githubCredentialConnection
	ConnectionByID map[string]githubCredentialConnection
}

// repoGithubBinding is one row of cloud_task_repo_github_bindings.
type repoGithubBinding struct {
	RepoURL      string
	RepoSlug     string
	GithubUserID string
}

// fetchGithubCredentialSummary calls gitOauth credential-summary-for-user.
// gitOauth 404（无凭据记录）视为空摘要而非错误；5xx/网络错误返回 err（调用方 500）。
func fetchGithubCredentialSummary(userID string) (*githubCredentialSummary, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" || userID == "<nil>" {
		return &githubCredentialSummary{}, nil
	}
	base := strings.TrimRight(strings.TrimSpace(cfg.GitOauthBaseURL), "/")
	if base == "" {
		return &githubCredentialSummary{}, nil
	}
	payload, _ := json.Marshal(map[string]any{
		"user_id":      userID,
		"provider_key": defaultGithubProviderKey(),
	})
	req, err := http.NewRequest(http.MethodPost, base+"/api/internal/git-oauth/github-credential-summary/", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("gitOauth request error: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if secret := strings.TrimSpace(cfg.GitOauthBridgeSecret); secret != "" {
		req.Header.Set("X-GitOauth-Bridge-Secret", secret)
	}
	client := &http.Client{Transport: &http.Transport{Proxy: nil}, Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gitOauth unreachable: %v", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusNotFound {
		return &githubCredentialSummary{}, nil
	}
	if resp.StatusCode >= 400 {
		detail := strings.TrimSpace(string(raw))
		if detail == "" {
			detail = fmt.Sprintf("http %d", resp.StatusCode)
		}
		return nil, fmt.Errorf("gitOauth error: %s", detail)
	}
	var parsed struct {
		Connected    bool   `json:"connected"`
		GithubUserID string `json:"github_user_id"`
		GithubLogin  string `json:"github_login"`
		Connections  []struct {
			Connected    bool   `json:"connected"`
			GithubUserID string `json:"github_user_id"`
			GithubLogin  string `json:"github_login"`
			Scope        string `json:"scope"`
		} `json:"connections"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("invalid json from gitOauth: %v", err)
	}
	summary := &githubCredentialSummary{
		Connected:      parsed.Connected,
		GithubUserID:   strings.TrimSpace(parsed.GithubUserID),
		GithubLogin:    strings.TrimSpace(parsed.GithubLogin),
		ConnectionByID: map[string]githubCredentialConnection{},
	}
	for _, c := range parsed.Connections {
		conn := githubCredentialConnection{
			Connected:    c.Connected,
			GithubUserID: strings.TrimSpace(c.GithubUserID),
			GithubLogin:  strings.TrimSpace(c.GithubLogin),
			Scope:        strings.TrimSpace(c.Scope),
		}
		summary.Connections = append(summary.Connections, conn)
		if conn.GithubUserID != "" {
			summary.ConnectionByID[conn.GithubUserID] = conn
		}
	}
	return summary, nil
}

// githubLoginByUserID resolves a GitHub login for the bound github_user_id
// from the connection list; returns "" when the account is not connected.
func (s *githubCredentialSummary) githubLoginByUserID(userID string) string {
	if s == nil {
		return ""
	}
	if conn, ok := s.ConnectionByID[userID]; ok {
		return conn.GithubLogin
	}
	// fallback: primary login when it matches the bound id
	if s.GithubUserID == userID {
		return s.GithubLogin
	}
	return ""
}

// loadTaskRepoGithubBindings returns all binding rows for a task keyed by repo_slug.
func loadTaskRepoGithubBindings(taskID string) (map[string]repoGithubBinding, error) {
	out := map[string]repoGithubBinding{}
	if taskID == "" {
		return out, nil
	}
	rows, err := db.Query(
		`SELECT repo_url, repo_slug, github_user_id FROM cloud_task_repo_github_bindings WHERE task_id=?`,
		taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var b repoGithubBinding
		if err := rows.Scan(&b.RepoURL, &b.RepoSlug, &b.GithubUserID); err != nil {
			return nil, err
		}
		if slug := strings.ToLower(strings.TrimSpace(b.RepoSlug)); slug != "" {
			out[slug] = b
		}
	}
	return out, rows.Err()
}

// upsertTaskRepoGithubBinding inserts or updates the repo→github_user_id binding.
func upsertTaskRepoGithubBinding(taskID, repoURL, repoSlug, githubUserID string) error {
	taskID = strings.TrimSpace(taskID)
	repoURL = strings.TrimSpace(repoURL)
	repoSlug = strings.TrimSpace(strings.ToLower(repoSlug))
	githubUserID = strings.TrimSpace(githubUserID)
	if taskID == "" || repoURL == "" || repoSlug == "" || githubUserID == "" {
		return fmt.Errorf("missing task/repo/github_user context")
	}
	_, err := db.Exec(`
INSERT INTO cloud_task_repo_github_bindings (id, task_id, repo_url, repo_slug, github_user_id)
VALUES (?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE repo_slug=VALUES(repo_slug), github_user_id=VALUES(github_user_id)`,
		genID("ctrgb"), taskID, repoURL, repoSlug, githubUserID)
	return err
}

// handleGithubCredentialStatus previously proxied to Django saas-backend.
// OPT-052: Django retired 2026-07-30 — reimplemented in Go (gitOauth summary +
// cloud_task_repo_github_bindings). 响应契约见 OpenAPI GithubCredentialStatusResponse。
func handleGithubCredentialStatus(w http.ResponseWriter, r *http.Request, tenantID, workspaceID, taskID string) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	if taskID == "" {
		taskID = strings.TrimSpace(r.URL.Query().Get("task_id"))
	}
	if taskID == "" || tenantID == "" || workspaceID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"status": "error", "message": "缺少任务或租户上下文"})
		return
	}

	summary, err := fetchGithubCredentialSummary(getAuthUser(r))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"status": "error", "message": "GitHub 账号状态查询失败", "detail": err.Error(),
		})
		return
	}

	// 任务关联 GitHub 仓库（slug 排序保证响应稳定）。
	// status 端点对收集失败降级为空列表（展示层可接受，已授权绑定信息仍返回）。
	githubRepos, _ := collectTaskGithubRepos(r.Context(), tenantID, workspaceID, taskID)

	bindings, err := loadTaskRepoGithubBindings(taskID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"status": "error", "message": "GitHub 绑定读取失败", "detail": err.Error(),
		})
		return
	}

	repoBindings := make([]map[string]interface{}, 0, len(githubRepos))
	allRepoBound := len(githubRepos) > 0
	for _, repo := range githubRepos {
		var selectedUserID, selectedLogin interface{}
		if b, ok := bindings[repo.RepoSlug]; ok && b.GithubUserID != "" {
			selectedUserID = b.GithubUserID
			selectedLogin = summary.githubLoginByUserID(b.GithubUserID)
		} else {
			allRepoBound = false
		}
		repoBindings = append(repoBindings, map[string]interface{}{
			"repo_url":                repo.RepoURL,
			"repo_slug":               repo.RepoSlug,
			"selected_github_user_id": selectedUserID,
			"selected_github_login":   selectedLogin,
		})
	}

	connections := make([]map[string]interface{}, 0, len(summary.Connections))
	for _, c := range summary.Connections {
		connections = append(connections, map[string]interface{}{
			"connected":      c.Connected,
			"github_user_id": c.GithubUserID,
			"github_login":   nilIfEmpty(c.GithubLogin),
			"scope":          nilIfEmpty(c.Scope),
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"approved_for_task":    allRepoBound,
		"github_app_connected": summary.Connected,
		"github_login":         summary.GithubLogin,
		"github_connections":   connections,
		"repo_bindings":        repoBindings,
		"all_repo_bound":       allRepoBound,
	})
}

// handleGithubCredentialApprove previously proxied to Django saas-backend.
// OPT-052: Django retired 2026-07-30 — reimplemented in Go. 业务校验：
//  1. repo 必须属于任务关联项目（防任意仓库绑定）
//  2. github_user_id 必须是当前用户已连接的 GitHub 账号
//     通过后 upsert cloud_task_repo_github_bindings。
func handleGithubCredentialApprove(w http.ResponseWriter, r *http.Request, tenantID, workspaceID, taskID string) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	if taskID == "" {
		taskID = strings.TrimSpace(r.URL.Query().Get("task_id"))
	}
	if taskID == "" || tenantID == "" || workspaceID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"status": "error", "message": "缺少任务或租户上下文"})
		return
	}
	userID := getAuthUser(r)
	if userID == "" || userID == "<nil>" {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"status": "error", "message": "缺少用户上下文"})
		return
	}

	body, err := readJSONBody(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"status": "error", "message": "invalid json"})
		return
	}
	repoURL := strings.TrimSpace(strField(body, "repo_url"))
	repoSlug := strings.TrimSpace(strings.ToLower(strField(body, "repo_slug")))
	githubUserID := strings.TrimSpace(strField(body, "github_user_id"))
	if repoSlug == "" {
		repoSlug = githubRepoSlugFromURLStrict(repoURL)
	}
	if repoSlug == "" || githubUserID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"status": "error", "message": "缺少仓库或 GitHub 账号"})
		return
	}

	// 校验 1: repo 属于任务关联项目。
	// 收集失败（taskTaskService 不可达/5xx）≠「无匹配」：给 503 而非业务 400，
	// 避免误导排障为仓库归属问题（OPT-20260809-014）。
	githubRepos, err := collectTaskGithubRepos(r.Context(), tenantID, workspaceID, taskID)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]interface{}{
			"status": "error", "message": "任务关联仓库收集失败", "detail": truncateStr(err.Error(), 500),
		})
		return
	}
	repoOK := false
	var matchedRepoURL string
	for _, repo := range githubRepos {
		if repo.RepoSlug == repoSlug {
			repoOK = true
			matchedRepoURL = repo.RepoURL
			break
		}
	}
	if !repoOK {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"status": "error", "message": "该仓库不在任务关联项目中"})
		return
	}

	// 校验 2: github_user_id 是当前用户已连接的 GitHub 账号。
	summary, err := fetchGithubCredentialSummary(userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"status": "error", "message": "GitHub 账号状态查询失败", "detail": err.Error(),
		})
		return
	}
	conn, ok := summary.ConnectionByID[githubUserID]
	if !ok || !conn.Connected {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"status": "error", "message": "GitHub 账号未连接"})
		return
	}

	if err := upsertTaskRepoGithubBinding(taskID, matchedRepoURL, repoSlug, githubUserID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"status": "error", "message": "绑定保存失败", "detail": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"approved":       true,
		"repo_slug":      repoSlug,
		"github_user_id": githubUserID,
		"github_login":   conn.GithubLogin,
	})
}

// taskGithubRepo is a task-linked GitHub repo with its normalized slug.
type taskGithubRepo struct {
	RepoURL  string
	RepoSlug string
}

// collectTaskGithubRepos gathers the task's GitHub repos via taskTaskService
// (projects[].stored_repo_address), deduped and slug-sorted.
// 收集失败（taskTaskService 不可达/5xx）透出 error（OPT-20260809-014），
// 调用方区分「收集失败」与「无匹配」：approve 应 503，status 可降级为空列表。
func collectTaskGithubRepos(ctx context.Context, tenantID, workspaceID, taskID string) ([]taskGithubRepo, error) {
	githubRows, _, _, _, err := collectTaskGitReposForAuthContext(ctx, tenantID, workspaceID, taskID)
	if err != nil {
		return nil, err
	}
	out := make([]taskGithubRepo, 0, len(githubRows))
	seen := map[string]bool{}
	for _, row := range githubRows {
		slug := githubRepoSlugFromURLStrict(row["repo_url"])
		if slug == "" || seen[slug] {
			continue
		}
		seen[slug] = true
		out = append(out, taskGithubRepo{RepoURL: row["repo_url"], RepoSlug: slug})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].RepoSlug < out[j].RepoSlug })
	return out, nil
}

// nilIfEmpty maps empty strings to nil so the JSON response carries null
// (mirrors taskTaskService / gitOauth conventions for optional fields).
func nilIfEmpty(s string) interface{} {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}
