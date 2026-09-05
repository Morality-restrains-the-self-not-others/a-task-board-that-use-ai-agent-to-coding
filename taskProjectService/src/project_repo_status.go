package main

import (
	"fmt"
	"strings"
)

// enrichProjectGitReposStatus attaches per-repo OAuth token_status for project detail.
// Probe remote access so "已授权" means this repo is reachable with the user's token
// (not merely that a provider-level OAuth credential exists).
func enrichProjectGitReposStatus(detail map[string]interface{}, userID string, tenantID string, traceHeaders ...map[string]string) {
	if detail == nil || strings.TrimSpace(userID) == "" {
		return
	}
	repos := collectProjectGitRepoURLs(detail)
	if len(repos) == 0 {
		return
	}
	projectID := ""
	if raw, ok := detail["id"]; ok && raw != nil {
		projectID = strings.TrimSpace(fmt.Sprintf("%v", raw))
		if projectID == "<nil>" {
			projectID = ""
		}
	}
	validated := validateGitReposForUser(userID, repos, true, tenantID, projectID, outboundTrace(traceHeaders...))
	statuses := make([]map[string]interface{}, 0, len(validated))
	for _, item := range validated {
		statuses = append(statuses, map[string]interface{}{
			"repo_url":               item.URL,
			"token_status":           item.TokenStatus,
			"oauth_provider":         item.OAuthProvider,
			"oauth_service_provider": item.OAuthServiceProvider,
		})
	}
	detail["git_repos_status"] = statuses
}

// hydrateProjectGitReposStatusLocal upgrades git_repos_status placeholders
// (token_status=not_applicable) to token_available when the requesting user has a
// project-scoped Git OAuth grant for the repo's site. Local DB lookup only — no
// git-oauth exchange / remote probe — preserving the GET project db-only
// contract (B-086). The create-task OAuth gate uses the hydrated list/detail to
// skip the validate-git-repos POST for already-authorized repos (OPT-20260902-020).
func hydrateProjectGitReposStatusLocal(detail map[string]interface{}, userID string) {
	if detail == nil {
		return
	}
	uid := strings.TrimSpace(userID)
	if uid == "" {
		return
	}
	projectID := ""
	if raw, ok := detail["id"]; ok && raw != nil {
		projectID = strings.TrimSpace(fmt.Sprintf("%v", raw))
	}
	if projectID == "" || projectID == "<nil>" {
		return
	}
	for _, row := range collectGitReposStatusRows(detail) {
		repoURL := strings.TrimSpace(fmt.Sprintf("%v", row["repo_url"]))
		if repoURL == "" || repoURL == "<nil>" {
			continue
		}
		_, site := extractHostNetloc(repoURL)
		site = strings.ToLower(strings.TrimSpace(site))
		if site == "" {
			continue
		}
		if !hasProjectGitOAuthGrant(projectID, uid, site) {
			continue
		}
		row["token_status"] = tokenStatusAvailable
		if providerKey := resolveProviderKeyForRepoURL(repoURL); providerKey != "" {
			provider, serviceProvider := splitProviderKey(providerKey)
			row["oauth_provider"] = provider
			row["oauth_service_provider"] = serviceProvider
		}
	}
}

func collectGitReposStatusRows(detail map[string]interface{}) []map[string]interface{} {
	if detail == nil {
		return nil
	}
	switch rows := detail["git_repos_status"].(type) {
	case []map[string]interface{}:
		return rows
	case []interface{}:
		out := make([]map[string]interface{}, 0, len(rows))
		for _, item := range rows {
			if m, ok := item.(map[string]interface{}); ok {
				out = append(out, m)
			}
		}
		return out
	default:
		return nil
	}
}

func resolveProviderKeyForRepoURL(repoURL string) string {
	if providerResolver != nil {
		return providerResolver.ResolveProvider(repoURL)
	}
	return resolveProviderFallback(repoURL)
}

func collectProjectGitRepoURLs(detail map[string]interface{}) []string {
	raw, ok := detail["git_repos"]
	if !ok || raw == nil {
		return nil
	}
	switch repos := raw.(type) {
	case []string:
		out := make([]string, 0, len(repos))
		for _, u := range repos {
			s := strings.TrimSpace(u)
			if s != "" {
				out = append(out, s)
			}
		}
		return out
	case []interface{}:
		out := make([]string, 0, len(repos))
		for _, item := range repos {
			s := strings.TrimSpace(fmt.Sprintf("%v", item))
			if s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}
