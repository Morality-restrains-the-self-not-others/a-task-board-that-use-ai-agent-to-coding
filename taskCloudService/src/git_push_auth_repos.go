package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"tracelog"
)

var (
	githubOwnerRepoRe = regexp.MustCompile(`(?i)github\.com[:/]([\w.-]+)/([\w.-]+?)(?:\.git)?/?$`)
	gitlabOwnerRepoRe = regexp.MustCompile(`(?i)gitlab[^:/]*[:/]([\w.-]+)/([\w.-]+?)(?:\.git)?/?$`)
	gitScpHostRe      = regexp.MustCompile(`(?i)^[^@]+@([^:/]+)(?::\d+)?[:/].+$`)
)

func collectTaskGitReposForAuthContext(ctx context.Context, tenantID, workspaceID, taskID string) (githubRows, gitlabRows []map[string]string, repos []authContextRepo, grants []commentOAuthGrant, err error) {
	projects, grants, err := collectTaskRelatedProjectsHTTP(ctx, tenantID, workspaceID, taskID)
	if err != nil || projects == nil {
		// No saas sqlite fallback — require task/project HTTP path.
		// 收集失败需要透出（OPT-20260809-014）：调用方区分「收集失败」与「无匹配」。
		return nil, nil, nil, grants, err
	}

	githubSeen := map[string]bool{}
	gitlabSeen := map[string]bool{}
	repoURLSeen := map[string]bool{}
	for _, p := range projects {
		for _, raw := range p.Repos {
			repoURL := strings.TrimSpace(raw)
			if repoURL == "" {
				continue
			}
			if slug := githubRepoSlugFromURLStrict(repoURL); slug != "" {
				if !githubSeen[slug] {
					githubSeen[slug] = true
					githubRows = append(githubRows, map[string]string{"repo_url": repoURL, "repo_slug": slug})
				}
				if !repoURLSeen[repoURL] {
					repoURLSeen[repoURL] = true
					repos = append(repos, authContextRepo{
						ProjectID: p.ID, ProjectName: p.Name, RepoURL: repoURL, RepoSlug: slug,
					})
				}
				continue
			}
			if slug := gitlabRepoSlugFromURL(repoURL); slug != "" {
				key := "gitlab::" + slug
				if !gitlabSeen[key] {
					gitlabSeen[key] = true
					gitlabRows = append(gitlabRows, map[string]string{"repo_url": repoURL, "repo_slug": slug})
				}
				if !repoURLSeen[repoURL] {
					repoURLSeen[repoURL] = true
					repos = append(repos, authContextRepo{
						ProjectID: p.ID, ProjectName: p.Name, RepoURL: repoURL, RepoSlug: slug,
					})
				}
			}
		}
	}
	return githubRows, gitlabRows, repos, grants, nil
}

func collectTaskRelatedProjectsHTTP(ctx context.Context, tenantID, workspaceID, taskID string) ([]relatedProject, []commentOAuthGrant, error) {
	taskID = strings.TrimSpace(taskID)
	tenantID = strings.TrimSpace(tenantID)
	if taskID == "" {
		return nil, nil, fmt.Errorf("empty task_id")
	}
	base := strings.TrimRight(strings.TrimSpace(cfg.TaskServiceURL), "/")
	if base == "" {
		return nil, nil, fmt.Errorf("task service url not configured")
	}
	u := base + "/api/tasks/" + url.PathEscape(taskID) + "/"
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, nil, err
	}
	if tenantID != "" {
		req.Header.Set("X-Auth-Tenant-Id", tenantID)
	}
	req.Header.Set("X-Auth-User-Id", "internal")
	client := &http.Client{Transport: &http.Transport{Proxy: nil}, Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("task service status=%d", resp.StatusCode)
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, nil, err
	}
	grants := commentOAuthGrantsFromTaskBody(body)

	// taskTaskService 当前返回 projects[]（含 stored_repo_address），未必有 project_ids。
	if embedded := relatedProjectsFromTaskProjectsField(body["projects"]); len(embedded) > 0 {
		return embedded, grants, nil
	}

	projectIDs := extractStringIDs(body["project_ids"])
	out := make([]relatedProject, 0, len(projectIDs))
	for _, pid := range projectIDs {
		proj, repos, err := fetchProjectReposHTTP(ctx, tenantID, pid)
		if err != nil || proj == nil {
			continue
		}
		if workspaceID != "" && len(proj.workspaceIDs) > 0 {
			found := false
			for _, wid := range proj.workspaceIDs {
				if wid == workspaceID {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		out = append(out, relatedProject{ID: pid, Name: proj.name, Repos: repos})
	}
	return out, grants, nil
}

// relatedProjectsFromTaskProjectsField parses task.projects[] entries into relatedProject
// rows keyed by project_id, collecting stored_repo_address / project_repo_url.
func relatedProjectsFromTaskProjectsField(raw any) []relatedProject {
	arr, ok := raw.([]any)
	if !ok || len(arr) == 0 {
		return nil
	}
	byID := map[string]*relatedProject{}
	var order []string
	repoSeen := map[string]map[string]bool{}
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		pid := strings.TrimSpace(fmt.Sprintf("%v", m["project_id"]))
		if pid == "" || pid == "<nil>" {
			pid = strings.TrimSpace(fmt.Sprintf("%v", m["id"]))
		}
		if pid == "" || pid == "<nil>" {
			continue
		}
		repo := strings.TrimSpace(fmt.Sprintf("%v", m["stored_repo_address"]))
		if repo == "" || repo == "<nil>" {
			repo = strings.TrimSpace(fmt.Sprintf("%v", m["project_repo_url"]))
		}
		if repo == "" || repo == "<nil>" {
			repo = strings.TrimSpace(fmt.Sprintf("%v", m["repo_url"]))
		}
		if repo == "" || repo == "<nil>" {
			continue
		}
		if _, ok := byID[pid]; !ok {
			name := strings.TrimSpace(fmt.Sprintf("%v", m["project_name"]))
			if name == "" || name == "<nil>" {
				name = pid
			}
			byID[pid] = &relatedProject{ID: pid, Name: name, Repos: nil}
			order = append(order, pid)
			repoSeen[pid] = map[string]bool{}
		}
		if repoSeen[pid][repo] {
			continue
		}
		repoSeen[pid][repo] = true
		byID[pid].Repos = append(byID[pid].Repos, repo)
	}
	if len(order) == 0 {
		return nil
	}
	out := make([]relatedProject, 0, len(order))
	for _, id := range order {
		out = append(out, *byID[id])
	}
	return out
}

type projectMeta struct {
	name         string
	workspaceIDs []string
}

func fetchProjectReposHTTP(ctx context.Context, tenantID, projectID string) (*projectMeta, []string, error) {
	base := strings.TrimRight(strings.TrimSpace(cfg.ProjectServiceURL), "/")
	if base == "" {
		return nil, nil, fmt.Errorf("project service url not configured")
	}
	u := base + "/api/projects/tenant_id/" + url.PathEscape(tenantID) + "/" + url.PathEscape(projectID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("X-Auth-Tenant-Id", tenantID)
	// OPT-20260821-012: 把入站 X-Trace-Id / span 透传到 taskProjectService。
	tracelog.ApplyOutboundHeaders(req, ctx)
	client := &http.Client{Transport: &http.Transport{Proxy: nil}, Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("project service status=%d", resp.StatusCode)
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, nil, err
	}
	meta := &projectMeta{name: strings.TrimSpace(fmt.Sprintf("%v", body["name"]))}
	if meta.name == "<nil>" {
		meta.name = ""
	}
	meta.workspaceIDs = extractWorkspaceIDs(body["workspaces"])
	repos := extractRepoURLs(body)
	return meta, repos, nil
}

func extractRepoURLs(body map[string]any) []string {
	candidates := []any{body["git_repos"], body["repos"], body["project_repos"], body["repo_urls"]}
	var out []string
	seen := map[string]bool{}
	for _, c := range candidates {
		for _, u := range extractStringIDs(c) {
			u = strings.TrimSpace(u)
			if u == "" || seen[u] {
				continue
			}
			seen[u] = true
			out = append(out, u)
		}
	}
	return out
}

func extractWorkspaceIDs(raw any) []string {
	switch v := raw.(type) {
	case []any:
		var out []string
		for _, item := range v {
			switch t := item.(type) {
			case map[string]any:
				id := strings.TrimSpace(fmt.Sprintf("%v", t["id"]))
				if id != "" && id != "<nil>" {
					out = append(out, id)
				}
			default:
				id := strings.TrimSpace(fmt.Sprintf("%v", t))
				if id != "" && id != "<nil>" {
					out = append(out, id)
				}
			}
		}
		return out
	default:
		return extractStringIDs(raw)
	}
}

func extractStringIDs(raw any) []string {
	switch v := raw.(type) {
	case nil:
		return nil
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			s := strings.TrimSpace(fmt.Sprintf("%v", item))
			if s == "" || s == "<nil>" {
				continue
			}
			out = append(out, s)
		}
		return out
	case []string:
		out := make([]string, 0, len(v))
		for _, item := range v {
			s := strings.TrimSpace(item)
			if s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func hostnameFromRepoURL(rawURL string) string {
	raw := strings.TrimSpace(rawURL)
	if raw == "" {
		return ""
	}
	if m := gitScpHostRe.FindStringSubmatch(raw); len(m) == 2 {
		return strings.ToLower(strings.TrimSpace(m[1]))
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(parsed.Hostname()))
}

func inferGitProviderFromRepoURL(repoURL string) string {
	host := hostnameFromRepoURL(repoURL)
	if host == "" {
		return ""
	}
	if host == "github.com" {
		return "github"
	}
	if strings.HasSuffix(host, "gitlab.com") || strings.Contains(host, "gitlab") {
		return "gitlab"
	}
	if host == "bitbucket.org" {
		return "bitbucket"
	}
	return ""
}

// resolveProviderFromRepoURL mirrors Django resolve_provider_from_repo_url without provider configs.
// Returns (provider, service_provider). service_provider may be "" (caller uses "default").
func resolveProviderFromRepoURL(repoURL string) (provider, serviceProvider string) {
	provider = inferGitProviderFromRepoURL(repoURL)
	if provider == "" {
		return "", ""
	}
	host := hostnameFromRepoURL(repoURL)
	serviceProvider = serviceProviderFromHost(host)
	return provider, serviceProvider
}

func serviceProviderFromHost(host string) string {
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" || host == "github.com" || host == "gitlab.com" || strings.HasSuffix(host, ".gitlab.com") {
		return "default"
	}
	if strings.Contains(host, "gitlab") {
		return "default"
	}
	// hostname without dots as key (e.g. self-hosted single-label host)
	if !strings.Contains(host, ".") {
		return host
	}
	return "default"
}

func githubRepoSlugFromURLStrict(repoURL string) string {
	raw := strings.TrimSpace(repoURL)
	if raw == "" {
		return ""
	}
	m := githubOwnerRepoRe.FindStringSubmatch(raw)
	if len(m) != 3 {
		return ""
	}
	owner := strings.TrimSpace(m[1])
	repo := strings.TrimSuffix(strings.TrimSpace(m[2]), ".git")
	if owner == "" || repo == "" {
		return ""
	}
	return strings.ToLower(owner + "/" + repo)
}

func gitlabRepoSlugFromURL(repoURL string) string {
	raw := strings.TrimSpace(repoURL)
	if raw == "" {
		return ""
	}
	if inferGitProviderFromRepoURL(raw) != "gitlab" {
		return ""
	}
	if m := gitlabOwnerRepoRe.FindStringSubmatch(raw); len(m) == 3 {
		owner := strings.TrimSpace(m[1])
		repo := strings.TrimSuffix(strings.TrimSpace(m[2]), ".git")
		if owner == "" || repo == "" {
			return ""
		}
		return strings.ToLower(owner + "/" + repo)
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	parts := []string{}
	for _, p := range strings.Split(parsed.Path, "/") {
		p = strings.TrimSpace(p)
		if p != "" {
			parts = append(parts, p)
		}
	}
	if len(parts) < 2 {
		return ""
	}
	owner := parts[len(parts)-2]
	repo := strings.TrimSuffix(parts[len(parts)-1], ".git")
	if owner == "" || repo == "" {
		return ""
	}
	return strings.ToLower(owner + "/" + repo)
}
