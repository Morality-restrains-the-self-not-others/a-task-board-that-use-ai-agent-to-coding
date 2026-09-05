package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

type relatedProject struct {
	ID    string
	Name  string
	Repos []string
}

type authContextRepo struct {
	ProjectID   string `json:"project_id"`
	ProjectName string `json:"project_name"`
	RepoURL     string `json:"repo_url"`
	RepoSlug    string `json:"repo_slug"`
}

type layerPushOauthReadiness struct {
	ProvidersRequired  []string
	ProvidersConnected []string
}

func (r layerPushOauthReadiness) missingProviders() []string {
	conn := map[string]bool{}
	for _, k := range r.ProvidersConnected {
		conn[k] = true
	}
	var missing []string
	for _, k := range r.ProvidersRequired {
		if !conn[k] {
			missing = append(missing, k)
		}
	}
	sort.Strings(missing)
	return missing
}

func (r layerPushOauthReadiness) allConnected() bool {
	return len(r.missingProviders()) == 0
}

func (r layerPushOauthReadiness) requiresOauth() bool {
	return len(r.ProvidersRequired) > 0
}

// handleInternalLayerGitPushAuthContext implements
// GET /api/internal/layer-git-push/auth-context
// Django-compatible readiness payload (simplified; no token plaintext).
func handleInternalLayerGitPushAuthContext(w http.ResponseWriter, r *http.Request) {
	if !requireInternalSecret(r) {
		writeJSON(w, http.StatusForbidden, map[string]any{"detail": "forbidden"})
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"detail": "method not allowed"})
		return
	}
	q := r.URL.Query()
	tenantID := strings.TrimSpace(q.Get("tenant_id"))
	workspaceID := strings.TrimSpace(q.Get("workspace_id"))
	taskID := strings.TrimSpace(q.Get("task_id"))
	userID := strings.TrimSpace(q.Get("user_id"))
	if tenantID == "" || taskID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "tenant_id and task_id required"})
		return
	}

	// readiness 端点对收集失败降级为空列表（同旧行为）；真实 5xx 由下游健康探测暴露。
	githubRows, gitlabRows, repos, _, _ := collectTaskGitReposForAuthContext(r.Context(), tenantID, workspaceID, taskID)
	hasGithub := len(githubRows) > 0
	hasGitlab := len(gitlabRows) > 0

	var readiness layerPushOauthReadiness
	githubConnected := false
	gitlabConnected := false
	if userID != "" {
		readiness = resolveLayerPushOauthReadiness(userID, githubRows, gitlabRows)
		githubKey := defaultGithubProviderKey()
		if hasGithub {
			for _, k := range readiness.ProvidersConnected {
				if k == githubKey {
					githubConnected = true
					break
				}
			}
		}
		gitlabKeys := []string{}
		for _, k := range readiness.ProvidersRequired {
			if strings.HasPrefix(k, "gitlab:") {
				gitlabKeys = append(gitlabKeys, k)
			}
		}
		if len(gitlabKeys) > 0 {
			conn := map[string]bool{}
			for _, k := range readiness.ProvidersConnected {
				conn[k] = true
			}
			gitlabConnected = true
			for _, k := range gitlabKeys {
				if !conn[k] {
					gitlabConnected = false
					break
				}
			}
		}
	} else if hasGithub || hasGitlab {
		// No user: still report required providers from repos; none connected.
		readiness = buildProvidersRequired(githubRows, gitlabRows)
	}

	pushRequiresGithub := hasGithub
	pushRequiresGit := readiness.requiresOauth()
	ready := true
	if userID != "" {
		ready = readiness.allConnected()
	} else if pushRequiresGit {
		ready = false
	}

	message := layerPushOauthGuidanceMessage(readiness)
	if !hasGithub && !hasGitlab {
		message = "当前工作空间未发现 GitHub/GitLab 仓库项目"
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"has_github_repo_projects":        hasGithub,
		"has_gitlab_repo_projects":        hasGitlab,
		"push_requires_github_oauth":      pushRequiresGithub,
		"push_requires_git_oauth":         pushRequiresGit,
		"git_oauth_providers_required":    readiness.ProvidersRequired,
		"git_oauth_ready_for_push":        ready,
		"github_app_user_oauth_connected": githubConnected,
		"gitlab_app_user_oauth_connected": gitlabConnected,
		"repos":                           repos,
		"message":                         message,
	})
}

func buildProvidersRequired(githubRows, gitlabRows []map[string]string) layerPushOauthReadiness {
	required := map[string]bool{}
	if len(githubRows) > 0 {
		required[defaultGithubProviderKey()] = true
	}
	for _, row := range gitlabRows {
		repoURL := strings.TrimSpace(row["repo_url"])
		if repoURL == "" {
			continue
		}
		if key := resolveProviderKeyFromRepoURL(repoURL); strings.HasPrefix(key, "gitlab:") {
			required[key] = true
			continue
		}
		_, sp := resolveProviderFromRepoURL(repoURL)
		if sp == "" {
			sp = "default"
		}
		required["gitlab:"+sp] = true
	}
	keys := make([]string, 0, len(required))
	for k := range required {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return layerPushOauthReadiness{ProvidersRequired: keys}
}

func resolveLayerPushOauthReadiness(userID string, githubRows, gitlabRows []map[string]string) layerPushOauthReadiness {
	base := buildProvidersRequired(githubRows, gitlabRows)
	connected := map[string]bool{}
	for _, key := range base.ProvidersRequired {
		provider := "github"
		if strings.HasPrefix(key, "gitlab:") {
			provider = "gitlab"
		} else if i := strings.Index(key, ":"); i > 0 {
			provider = key[:i]
		}
		ok, _ := fetchGitOauthCredentialSummary(userID, provider, key)
		if ok {
			connected[key] = true
		}
	}
	connKeys := make([]string, 0, len(connected))
	for k := range connected {
		connKeys = append(connKeys, k)
	}
	sort.Strings(connKeys)
	base.ProvidersConnected = connKeys
	return base
}

func layerPushOauthGuidanceMessage(readiness layerPushOauthReadiness) string {
	if !readiness.requiresOauth() {
		return "当前任务未发现需要 OAuth 的 Git 远程仓库。"
	}
	missing := readiness.missingProviders()
	if len(missing) == 0 {
		return "推送请在任务详情选择「克隆所用 Git 身份」；" +
			"平台将按 Git 站点 OAuth 换发 access_token 并由容器通过 HTTPS 推送。"
	}
	hasGithubMissing := false
	hasGitlabMissing := false
	for _, item := range missing {
		if strings.HasPrefix(item, "github:") {
			hasGithubMissing = true
		}
		if strings.HasPrefix(item, "gitlab:") {
			hasGitlabMissing = true
		}
	}
	if hasGithubMissing && hasGitlabMissing {
		return "推送前请在个人资料完成 GitHub 与 GitLab 网站 OAuth 授权，" +
			"并在任务详情选择「克隆所用 Git 身份」。"
	}
	if hasGitlabMissing {
		return "推送 GitLab 仓库请在任务详情选择「克隆所用 Git 身份」，" +
			"并完成个人资料中对应 GitLab 站点的 OAuth 授权，以便平台换发 access_token 并由容器 HTTPS 推送。"
	}
	return "推送 GitHub 仓库请在任务详情选择「克隆所用 Git 身份」，" +
		"并完成个人资料中的 GitHub 网站 OAuth 授权（OAuth refresh_token），" +
		"以便平台换发 access_token 并由容器通过 HTTPS 推送及自动创建 PR。"
}

// fetchGitOauthCredentialSummary calls gitOauth credential-summary.
// Canonical path is /api/internal/git-oauth/{provider}-credential-summary/
// (docs/superpowers/specs/2026-08-04-api-path-convention-audit.md). The old
// /api/internal/{provider}/oauth/user-credential/summary-for-user/ is not
// registered on taskGitOauth and 404s, which made GitLab push skip access-for-user.
func fetchGitOauthCredentialSummary(userID, provider, providerKey string) (connected bool, errDetail string) {
	base := strings.TrimRight(strings.TrimSpace(cfg.GitOauthBaseURL), "/")
	if base == "" {
		return false, "gitOauth not configured"
	}
	provider = strings.TrimSpace(provider)
	if provider == "" {
		provider = "github"
	}
	providerKey = strings.TrimSpace(providerKey)
	if providerKey == "" {
		providerKey = provider
	}
	urlStr := base + "/api/internal/git-oauth/" + provider + "-credential-summary/"
	payload, _ := json.Marshal(map[string]any{
		"user_id":      userID,
		"provider_key": providerKey,
	})
	req, err := http.NewRequest(http.MethodPost, urlStr, bytes.NewReader(payload))
	if err != nil {
		return false, fmt.Sprintf("gitOauth request error: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if secret := strings.TrimSpace(cfg.GitOauthBridgeSecret); secret != "" {
		req.Header.Set("X-GitOauth-Bridge-Secret", secret)
	}
	client := &http.Client{Transport: &http.Transport{Proxy: nil}, Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Sprintf("gitOauth unreachable: %v", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusNotFound {
		return false, ""
	}
	if resp.StatusCode >= 400 {
		detail := strings.TrimSpace(string(raw))
		if detail == "" {
			detail = fmt.Sprintf("http %d", resp.StatusCode)
		}
		return false, fmt.Sprintf("gitOauth error: %s", detail)
	}
	var parsed struct {
		Connected bool `json:"connected"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return false, "invalid json from gitOauth"
	}
	return parsed.Connected, ""
}
