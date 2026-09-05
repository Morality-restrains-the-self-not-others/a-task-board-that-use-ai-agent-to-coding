package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// TenantDiskRepoUsage is one gitlab-local repo contributing to tenant disk usage.
type TenantDiskRepoUsage struct {
	RepoURL   string `json:"repo_url"`
	SizeBytes int64  `json:"size_bytes"`
	OK        bool   `json:"ok"`
}

// TenantDiskUsageResult is the aggregate disk usage for a tenant's internal repos.
type TenantDiskUsageResult struct {
	TenantID      string                `json:"tenant_id"`
	DiskUsedBytes int64                 `json:"disk_used_bytes"`
	RepoCount     int                   `json:"repo_count"`
	MeasuredCount int                   `json:"measured_count"`
	Repos         []TenantDiskRepoUsage `json:"repos"`
}

func listTenantGitlabLocalRepoURLs(tenantID string) ([]string, error) {
	tid := strings.TrimSpace(tenantID)
	if tid == "" {
		return nil, fmt.Errorf("tenant_id required")
	}
	rows, err := db.Query(`
		SELECT DISTINCT pr.repo_url
		FROM project_repos pr
		INNER JOIN project_entries p ON p.id = pr.project_id
		WHERE p.company_id = ?
		ORDER BY pr.repo_url`, tid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]string, 0)
	for rows.Next() {
		var u string
		if err := rows.Scan(&u); err != nil {
			return nil, err
		}
		u = strings.TrimSpace(u)
		if u == "" {
			continue
		}
		apiURL := u
		if normalized, ok := normalizeGitRepoURLForBranchLookup(u); ok {
			apiURL = normalized
		}
		if providerResolver == nil || !providerResolver.IsInternalRepo(apiURL) {
			continue
		}
		out = append(out, apiURL)
	}
	return out, rows.Err()
}

func fetchInternalGitlabRepositorySizeBytes(repoURL, token string) (int64, bool) {
	parts, err := parseGitLabProjectParts(repoURL)
	if err != nil {
		return 0, false
	}
	// Prefer loopback Omnibus API — public gitlab.* host may be unreachable from app hosts.
	apiBase := strings.TrimRight(strings.TrimSpace(cfg.GitlabAPIBase), "/")
	if apiBase == "" {
		apiBase = "http://127.0.0.1:8012"
	}
	// parts.APIBase is like https://host/api/v4/projects/owner%2Frepo — extract encoded path
	marker := "/api/v4/projects/"
	idx := strings.Index(parts.APIBase, marker)
	if idx < 0 {
		return 0, false
	}
	encodedPath := parts.APIBase[idx+len(marker):]
	apiURL := apiBase + marker + encodedPath + "?statistics=true"
	if size, ok := getCachedDiskSize(apiURL); ok {
		return size, true
	}
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return 0, false
	}
	req.Header.Set("PRIVATE-TOKEN", token)
	req.Header.Set("Accept", "application/json")
	resp, err := gitHTTPClient.Do(req)
	if err != nil {
		log.Printf("[taskProjectService] disk-usage gitlab request failed repo=%s err=%v", redactRepoURLForLog(repoURL), err)
		return 0, false
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			// OPT-045: 404 表示 GitLab 中该仓库已不存在（可能被删除或移动）。
			// 该 repo_url 仍在 project_repos 表中引用的项目下。
			// 清理策略：在 project_repos 中标记 deleted_at 或由管理员人工审核后移除关联。
			// 当前仅记录日志供排查；未来可扩展为定期任务扫描 404 并自动清理。
			log.Printf("[taskProjectService] disk-usage gitlab 404 repo=%s (project_missing/orphaned)", redactRepoURLForLog(repoURL))
		} else {
			log.Printf("[taskProjectService] disk-usage gitlab status=%d repo=%s", resp.StatusCode, redactRepoURLForLog(repoURL))
		}
		return 0, false
	}
	size, ok := parseGitLabRepositorySizeBytes(body)
	if !ok {
		return 0, false
	}
	putCachedDiskSize(apiURL, size)
	return size, true
}

// getGitlabAPIBase returns the configured GitLab API base URL (loopback preferred).
func getGitlabAPIBase() string {
	apiBase := strings.TrimRight(strings.TrimSpace(cfg.GitlabAPIBase), "/")
	if apiBase == "" {
		apiBase = "http://127.0.0.1:8012"
	}
	return apiBase
}

// measureTenantGitlabGroupDiskUsage discovers all repos under the tenant's GitLab
// group (tenant-{tenantID}) and returns their disk usage via the GitLab projects API
// with ?statistics=true.  This path does NOT depend on the project_repos DB table —
// it queries GitLab directly, so repos created outside the SaaS platform are also counted.
// Returns empty slice (not error) when the group doesn't exist or the token is missing.
func measureTenantGitlabGroupDiskUsage(tenantID, token string) ([]TenantDiskRepoUsage, error) {
	tid := strings.TrimSpace(tenantID)
	if tid == "" {
		return nil, nil
	}
	token = strings.TrimSpace(token)
	if token == "" {
		token = resolveGitlabAdminPrivateToken()
	}
	if token == "" {
		log.Printf("[taskProjectService] disk-usage group skip tenant=%s (no admin token)", tid)
		return nil, nil
	}

	apiBase := getGitlabAPIBase()
	groupPath := fmt.Sprintf("tenant-%s", tid)
	encodedPath := url.PathEscape(groupPath)

	// Step 1: find the group
	getURL := fmt.Sprintf("%s/api/v4/groups/%s", apiBase, encodedPath)
	req, err := http.NewRequest(http.MethodGet, getURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("PRIVATE-TOKEN", token)
	req.Header.Set("Accept", "application/json")
	resp, err := gitHTTPClient.Do(req)
	if err != nil {
		log.Printf("[taskProjectService] disk-usage group request failed tenant=%s err=%v", tid, err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		log.Printf("[taskProjectService] disk-usage group not found tenant=%s path=%s", tid, groupPath)
		return nil, nil // group doesn't exist yet — no repos
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("[taskProjectService] disk-usage group status=%d tenant=%s body=%s", resp.StatusCode, tid, truncateString(string(body), 200))
		return nil, fmt.Errorf("gitlab group %s: status=%d", groupPath, resp.StatusCode)
	}

	var group struct {
		ID int64 `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&group); err != nil {
		return nil, err
	}
	resp.Body.Close()

	// Step 2: list all projects with statistics (paginated)
	var repos []TenantDiskRepoUsage
	page := 1
	for {
		listURL := fmt.Sprintf("%s/api/v4/groups/%d/projects?statistics=true&per_page=100&page=%d&include_subgroups=true",
			apiBase, group.ID, page)
		req, err := http.NewRequest(http.MethodGet, listURL, nil)
		if err != nil {
			return repos, err
		}
		req.Header.Set("PRIVATE-TOKEN", token)
		req.Header.Set("Accept", "application/json")
		resp, err := gitHTTPClient.Do(req)
		if err != nil {
			log.Printf("[taskProjectService] disk-usage list projects failed group=%d page=%d err=%v", group.ID, page, err)
			return repos, err
		}

		var projects []struct {
			PathWithNamespace string `json:"path_with_namespace"`
			WebURL            string `json:"web_url"`
			Statistics        *struct {
				RepositorySize int64 `json:"repository_size"`
			} `json:"statistics"`
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Printf("[taskProjectService] disk-usage list projects status=%d group=%d", resp.StatusCode, group.ID)
			return repos, fmt.Errorf("list group %d projects: status=%d", group.ID, resp.StatusCode)
		}

		if err := json.Unmarshal(body, &projects); err != nil {
			return repos, err
		}

		if len(projects) == 0 {
			break
		}

		for _, p := range projects {
			size := int64(0)
			ok := false
			if p.Statistics != nil {
				size = p.Statistics.RepositorySize
				ok = true
			}
			repos = append(repos, TenantDiskRepoUsage{
				RepoURL:   p.WebURL,
				SizeBytes: size,
				OK:        ok,
			})
		}

		nextPage := resp.Header.Get("X-Next-Page")
		if nextPage == "" {
			break
		}
		page++
	}

	log.Printf("[taskProjectService] disk-usage group tenant=%s group_id=%d repos=%d", tid, group.ID, len(repos))
	return repos, nil
}

func truncateString(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// resolveGitlabUserByExternUID searches GitLab users by OIDC extern_uid (SaaS user ID).
// Returns the GitLab username and user ID, or empty if not found.
func resolveGitlabUserByExternUID(externUID, token string) (string, int, error) {
	externUID = strings.TrimSpace(externUID)
	if externUID == "" || token == "" {
		return "", 0, nil
	}
	apiBase := getGitlabAPIBase()
	searchURL := fmt.Sprintf("%s/api/v4/users?extern_uid=%s&provider=openid_connect",
		apiBase, url.QueryEscape(externUID))
	req, err := http.NewRequest(http.MethodGet, searchURL, nil)
	if err != nil {
		return "", 0, err
	}
	req.Header.Set("PRIVATE-TOKEN", token)
	req.Header.Set("Accept", "application/json")
	resp, err := gitHTTPClient.Do(req)
	if err != nil {
		log.Printf("[taskProjectService] disk-usage gitlab user search failed extern_uid=%s err=%v", externUID, err)
		return "", 0, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		log.Printf("[taskProjectService] disk-usage gitlab user search status=%d extern_uid=%s", resp.StatusCode, externUID)
		return "", 0, nil
	}
	var users []struct {
		ID       int    `json:"id"`
		Username string `json:"username"`
	}
	if err := json.Unmarshal(body, &users); err != nil {
		return "", 0, err
	}
	if len(users) == 0 {
		return "", 0, nil
	}
	return users[0].Username, users[0].ID, nil
}

// listUserPersonalProjects returns repos under a GitLab user's personal namespace.
// Only returns repos whose namespace is exactly the user's username (not group repos).
func listUserPersonalProjects(gitlabUserID int, username, token string) ([]TenantDiskRepoUsage, error) {
	if gitlabUserID <= 0 || username == "" || token == "" {
		return nil, nil
	}
	apiBase := getGitlabAPIBase()
	var repos []TenantDiskRepoUsage
	page := 1
	for {
		listURL := fmt.Sprintf("%s/api/v4/users/%d/projects?statistics=true&per_page=100&page=%d",
			apiBase, gitlabUserID, page)
		req, err := http.NewRequest(http.MethodGet, listURL, nil)
		if err != nil {
			return repos, err
		}
		req.Header.Set("PRIVATE-TOKEN", token)
		req.Header.Set("Accept", "application/json")
		resp, err := gitHTTPClient.Do(req)
		if err != nil {
			log.Printf("[taskProjectService] disk-usage user projects failed user=%s err=%v", username, err)
			return repos, err
		}

		var projects []struct {
			ID                int    `json:"id"`
			PathWithNamespace string `json:"path_with_namespace"`
			WebURL            string `json:"web_url"`
			Statistics        *struct {
				RepositorySize int64 `json:"repository_size"`
			} `json:"statistics"`
			Namespace struct {
				FullPath string `json:"full_path"`
			} `json:"namespace"`
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Printf("[taskProjectService] disk-usage user projects status=%d user=%s", resp.StatusCode, username)
			return repos, nil
		}

		if err := json.Unmarshal(body, &projects); err != nil {
			return repos, err
		}

		if len(projects) == 0 {
			break
		}

		for _, p := range projects {
			// Only include repos directly under the user's personal namespace
			// (exclude repos from groups the user has joined)
			ns := strings.TrimSpace(p.Namespace.FullPath)
			if ns != username {
				continue
			}
			size := int64(0)
			ok := false
			if p.Statistics != nil {
				size = p.Statistics.RepositorySize
				ok = true
			}
			repos = append(repos, TenantDiskRepoUsage{
				RepoURL:   p.WebURL,
				SizeBytes: size,
				OK:        ok,
			})
		}

		nextPage := resp.Header.Get("X-Next-Page")
		if nextPage == "" {
			break
		}
		page++
	}

	log.Printf("[taskProjectService] disk-usage user personal user=%s user_id=%d repos=%d",
		username, gitlabUserID, len(repos))
	return repos, nil
}

// measureTenantMemberPersonalDiskUsage finds GitLab repos under personal namespaces
// of all members of a tenant (mapped via OIDC extern_uid).
// Only counts repos whose namespace == the user's GitLab username — excludes group repos.
func measureTenantMemberPersonalDiskUsage(tenantID, token string) ([]TenantDiskRepoUsage, error) {
	tid := strings.TrimSpace(tenantID)
	if tid == "" {
		return nil, nil
	}
	token = strings.TrimSpace(token)
	if token == "" {
		token = resolveGitlabAdminPrivateToken()
	}
	if token == "" {
		log.Printf("[taskProjectService] disk-usage personal skip tenant=%s (no admin token)", tid)
		return nil, nil
	}

	// Get all SaaS user IDs that are members of this tenant.
	members, err := tenantListMembers(tid)
	if err != nil {
		log.Printf("[taskProjectService] disk-usage personal tenant members failed tenant=%s err=%v", tid, err)
		return nil, nil
	}
	if len(members) == 0 {
		return nil, nil
	}

	// Collect SaaS user IDs from tenant members.
	// tenantListMembers returns records with "user_id" or "id" field.
	memberIDs := map[string]bool{}
	for _, m := range members {
		for _, key := range []string{"user_id", "id", "member_id"} {
			if v := strings.TrimSpace(fmt.Sprintf("%v", m[key])); v != "" && v != "<nil>" && v != "0" {
				memberIDs[v] = true
				break
			}
		}
	}

	var allRepos []TenantDiskRepoUsage
	seenGLUser := map[string]bool{}
	for uid := range memberIDs {
		// Find GitLab user by OIDC extern_uid (SaaS user ID).
		glUsername, glUserID, err := resolveGitlabUserByExternUID(uid, token)
		if err != nil {
			log.Printf("[taskProjectService] disk-usage personal resolve gitlab user failed saas_user=%s err=%v", uid, err)
			continue
		}
		if glUsername == "" {
			log.Printf("[taskProjectService] disk-usage personal no gitlab user saas_user=%s", uid)
			continue
		}
		// Avoid duplicate fetches for the same GitLab user.
		if seenGLUser[glUsername] {
			continue
		}
		seenGLUser[glUsername] = true

		repos, err := listUserPersonalProjects(glUserID, glUsername, token)
		if err != nil {
			log.Printf("[taskProjectService] disk-usage personal list projects failed user=%s err=%v", glUsername, err)
			continue
		}
		allRepos = append(allRepos, repos...)
	}

	log.Printf("[taskProjectService] disk-usage personal tenant=%s members=%d gitlab_users=%d repos=%d",
		tid, len(memberIDs), len(seenGLUser), len(allRepos))
	return allRepos, nil
}

func resolveGitlabAdminPrivateToken() string {
	if t := strings.TrimSpace(os.Getenv("GITLAB_ADMIN_PRIVATE_TOKEN")); t != "" {
		return t
	}
	if t := strings.TrimSpace(cfg.GitlabAdminPrivateToken); t != "" {
		return t
	}
	// Convention: gitService script writes PAT here (gitlab_home is gitignored).
	candidates := []string{
		filepath.Join("gitService", "gitlab_home", ".taskbill_admin_pat"),
	}
	if root, err := findMonorepoRoot(); err == nil {
		candidates = append([]string{
			filepath.Join(root, "gitService", "gitlab_home", ".taskbill_admin_pat"),
		}, candidates...)
	}
	for _, p := range candidates {
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		if t := strings.TrimSpace(string(b)); t != "" {
			return t
		}
	}
	return ""
}

func measureTenantGitlabLocalDiskUsage(tenantID, adminToken string) (TenantDiskUsageResult, error) {
	result := TenantDiskUsageResult{
		TenantID: strings.TrimSpace(tenantID),
		Repos:    []TenantDiskRepoUsage{},
	}
	token := strings.TrimSpace(adminToken)
	if token == "" {
		token = resolveGitlabAdminPrivateToken()
	}

	// Path 1: GitLab Group API — covers repos under the tenant's GitLab group.
	groupRepos, err := measureTenantGitlabGroupDiskUsage(tenantID, token)
	if err != nil {
		log.Printf("[taskProjectService] disk-usage group list failed tenant=%s err=%v", tenantID, err)
	}

	// Path 2: Tenant member personal namespaces — covers repos under users' personal
	// GitLab namespaces (mapped via OIDC extern_uid = SaaS user ID). Excludes repos
	// from other groups the user has joined (namespace != username filter).
	personalRepos, err := measureTenantMemberPersonalDiskUsage(tenantID, token)
	if err != nil {
		log.Printf("[taskProjectService] disk-usage personal list failed tenant=%s err=%v", tenantID, err)
	}

	// Merge: deduplicate by web_url. Priority: group repos (pre-measured) > personal repos.
	seen := map[string]bool{}
	for _, r := range groupRepos {
		u := strings.TrimSpace(r.RepoURL)
		if u == "" || seen[u] {
			continue
		}
		seen[u] = true
		result.Repos = append(result.Repos, r)
	}
	for _, r := range personalRepos {
		u := strings.TrimSpace(r.RepoURL)
		if u == "" || seen[u] {
			continue
		}
		seen[u] = true
		result.Repos = append(result.Repos, r)
	}

	result.RepoCount = len(result.Repos)
	if result.RepoCount == 0 {
		return result, nil
	}

	// Measure sizes for repos that don't have a pre-fetched size (DB-only repos).
	type job struct {
		idx  int
		size int64
		ok   bool
	}
	jobs := make([]job, 0, len(result.Repos))
	for i, r := range result.Repos {
		if r.OK {
			continue // already measured via GitLab group API
		}
		jobs = append(jobs, job{idx: i})
	}
	if len(jobs) > 0 && token != "" {
		var wg sync.WaitGroup
		for i := range jobs {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				size, ok := fetchInternalGitlabRepositorySizeBytes(result.Repos[jobs[i].idx].RepoURL, token)
				jobs[i].size = size
				jobs[i].ok = ok
			}(i)
		}
		wg.Wait()
		for _, j := range jobs {
			result.Repos[j.idx].SizeBytes = j.size
			result.Repos[j.idx].OK = j.ok
		}
	}

	var total int64
	for _, r := range result.Repos {
		if r.OK {
			total += r.SizeBytes
			result.MeasuredCount++
		}
	}
	result.DiskUsedBytes = total
	return result, nil
}

func requireProjectInternalSecret(r *http.Request) bool {
	if strings.TrimSpace(cfg.InternalSecret) == "" {
		return true
	}
	got := r.Header.Get("X-Internal-Secret")
	if got == "" {
		got = r.Header.Get("X-TaskProjectService-Internal-Secret")
	}
	return got == cfg.InternalSecret
}

func listCompanyIDsWithGitlabLocalRepos() ([]string, error) {
	rows, err := db.Query(`
		SELECT DISTINCT p.company_id, pr.repo_url
		FROM project_repos pr
		INNER JOIN project_entries p ON p.id = pr.project_id
		ORDER BY p.company_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	seen := map[string]struct{}{}
	out := make([]string, 0)
	for rows.Next() {
		var companyID, repoURL string
		if err := rows.Scan(&companyID, &repoURL); err != nil {
			return nil, err
		}
		companyID = strings.TrimSpace(companyID)
		apiURL := strings.TrimSpace(repoURL)
		if normalized, ok := normalizeGitRepoURLForBranchLookup(apiURL); ok {
			apiURL = normalized
		}
		if companyID == "" || providerResolver == nil || !providerResolver.IsInternalRepo(apiURL) {
			continue
		}
		if _, ok := seen[companyID]; ok {
			continue
		}
		seen[companyID] = struct{}{}
		out = append(out, companyID)
	}
	return out, rows.Err()
}

func handleInternalTenantGitlabLocalDiskUsage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !requireProjectInternalSecret(r) {
		writeError(w, r, http.StatusForbidden, "forbidden")
		return
	}
	parts := cleanPath(r, "/api/internal/tenants/")
	// GET /api/internal/tenants/with-gitlab-local-repos/
	if len(parts) == 1 && parts[0] == "with-gitlab-local-repos" {
		ids, err := listCompanyIDsWithGitlabLocalRepos()
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"tenant_ids": ids, "total": len(ids)})
		return
	}
	// tenants/{tid}/gitlab-local-disk-usage
	if len(parts) < 2 || parts[1] != "gitlab-local-disk-usage" {
		writeError(w, r, http.StatusNotFound, "not found")
		return
	}
	tenantID := strings.TrimSpace(parts[0])
	if tenantID == "" {
		writeError(w, r, http.StatusBadRequest, "tenant_id required")
		return
	}
	result, err := measureTenantGitlabLocalDiskUsage(tenantID, r.Header.Get("X-Gitlab-Admin-Token"))
	if err != nil {
		log.Printf("[taskProjectService] tenant disk usage failed tenant=%s err=%v", tenantID, err)
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}
