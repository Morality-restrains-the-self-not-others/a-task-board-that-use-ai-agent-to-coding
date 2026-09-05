package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// resolveLayerGitPushOauthMaps exchanges GitHub/GitLab tokens for all task repos.
// Same provider_key is refreshed once and reused (avoid revoking prior access_token).
func resolveLayerGitPushOauthMaps(
	ctx context.Context, tenantID, workspaceID, taskID, userID, repoURLHint string,
) (githubAuth map[string]string, oauthAuth map[string]any, errDetail string) {
	githubRows, gitlabRows, _, grants, _ := collectTaskGitReposForAuthContext(ctx, tenantID, workspaceID, taskID)
	githubAuth = map[string]string{}
	oauthAuth = map[string]any{}

	hintSlug := githubRepoSlugFromURL(repoURLHint)
	if hintSlug != "" {
		found := false
		for _, row := range githubRows {
			if row["repo_slug"] == hintSlug {
				found = true
				break
			}
		}
		if !found {
			githubRows = append(githubRows, map[string]string{
				"repo_url":  strings.TrimSpace(repoURLHint),
				"repo_slug": hintSlug,
			})
		}
	}
	if len(githubRows) == 0 && strings.TrimSpace(repoURLHint) != "" && hintSlug == "" {
		// Non-github hint may still be gitlab; collectTask may have missed if task service down.
		if slug := gitlabRepoSlugFromURL(repoURLHint); slug != "" {
			gitlabRows = append(gitlabRows, map[string]string{
				"repo_url":  strings.TrimSpace(repoURLHint),
				"repo_slug": slug,
			})
		}
	}

	tokenByProviderKey := map[string]string{}
	fetchCached := func(provider, providerKey string) (string, string) {
		providerKey = strings.TrimSpace(providerKey)
		if providerKey == "" {
			return "", "empty provider_key"
		}
		if tok, ok := tokenByProviderKey[providerKey]; ok {
			return tok, ""
		}
		tok, errDetail := fetchGitOauthAccessForUser(userID, provider, providerKey)
		if errDetail != "" {
			return "", errDetail
		}
		if tok == "" {
			return "", ""
		}
		tokenByProviderKey[providerKey] = tok
		return tok, ""
	}

	if len(githubRows) > 0 {
		var grantedGithub []map[string]string
		for _, row := range githubRows {
			site := hostnameFromRepoURL(row["repo_url"])
			if userHasCommentOAuthGrant(grants, userID, site) {
				grantedGithub = append(grantedGithub, row)
			}
		}
		if len(grantedGithub) == 0 {
			site := hostnameFromRepoURL(githubRows[0]["repo_url"])
			log.Printf("[taskCloudService] layer-git-push deny exchange without comment L2 user=%s gitsite=%s", userID, site)
			if errDetail == "" {
				errDetail = commentGrantMissingDetail(site)
			}
		} else {
			ghKey := defaultGithubProviderKey()
			tok, detail := fetchCached("github", ghKey)
			if detail != "" && errDetail == "" {
				errDetail = detail
			}
			if tok != "" {
				for _, row := range grantedGithub {
					slug := strings.TrimSpace(row["repo_slug"])
					if slug == "" {
						slug = githubRepoSlugFromURL(row["repo_url"])
					}
					if slug == "" {
						continue
					}
					githubAuth[slug] = tok
				}
				if len(githubAuth) == 0 && hintSlug != "" {
					githubAuth[hintSlug] = tok
				}
				if len(githubAuth) == 0 {
					githubAuth["default"] = tok
				}
			} else if errDetail == "" {
				errDetail = "gitOauth 未找到该用户的 GitHub 凭据，请重新完成授权"
			}
		}
	}

	for _, row := range gitlabRows {
		repoURL := strings.TrimSpace(row["repo_url"])
		if repoURL == "" {
			continue
		}
		site := hostnameFromRepoURL(repoURL)
		if !userHasCommentOAuthGrant(grants, userID, site) {
			log.Printf("[taskCloudService] layer-git-push deny gitlab exchange without comment L2 user=%s gitsite=%s", userID, site)
			if errDetail == "" {
				errDetail = commentGrantMissingDetail(site)
			}
			continue
		}
		var chosenKey, tok string
		var lastDetail string
		for _, providerKey := range gitlabProviderKeysToTry(repoURL) {
			// Same as GitHub: access-for-user is SSOT. Do not skip on summary 404 —
			// live taskGitOauth never registered user-credential/summary-for-user.
			got, detail := fetchCached("gitlab", providerKey)
			if detail != "" {
				lastDetail = detail
				continue
			}
			if got == "" {
				continue
			}
			chosenKey = providerKey
			tok = got
			break
		}
		if tok == "" {
			if lastDetail != "" && errDetail == "" {
				errDetail = "GitLab OAuth 换票失败：" + lastDetail
			}
			continue
		}
		oauthAuth[canonicalRepoKey(repoURL)] = map[string]any{
			"provider":     "gitlab",
			"access_token": tok,
			"provider_key": chosenKey,
		}
	}

	if len(githubAuth) == 0 {
		githubAuth = nil
	}
	if len(oauthAuth) == 0 {
		oauthAuth = nil
	}
	if len(githubAuth) > 0 || len(oauthAuth) > 0 {
		errDetail = ""
	}
	return githubAuth, oauthAuth, errDetail
}

func canonicalRepoKey(raw string) string {
	value := strings.TrimSpace(raw)
	value = strings.TrimRight(value, "/")
	if strings.HasSuffix(strings.ToLower(value), ".git") {
		value = value[:len(value)-4]
	}
	return strings.ToLower(value)
}

// gitlabProviderKeysToTry returns candidate provider_key values for a GitLab repo URL.
// Host-matched regional keys are exclusive: do not borrow another CE's token
// (conf/auth/git-oauth/ai.md 多区域 GitLab; ADR-0014). Unmatched hosts still
// try configured gitlab providers so aliases / legacy websites keep working.
func gitlabProviderKeysToTry(repoURL string) []string {
	seen := map[string]bool{}
	var keys []string
	add := func(k string) {
		k = strings.TrimSpace(k)
		if k == "" || seen[k] {
			return
		}
		seen[k] = true
		keys = append(keys, k)
	}
	resolved := resolveProviderKeyFromRepoURL(repoURL)
	if strings.HasPrefix(resolved, "gitlab:") && resolved != "gitlab:default" {
		add(resolved)
		return keys
	}
	for _, e := range loadGitOauthProviders() {
		if e.Provider == "gitlab" {
			add(e.ProviderKey)
		}
	}
	return keys
}

// writeLayerGitPushOAuthResult exchanges GitHub token and writes prepare response.
// Kept for narrow call sites / tests; prefer resolveLayerGitPushOauthMaps for new paths.
func writeLayerGitPushOAuthResult(w http.ResponseWriter, pushBody map[string]any, userID, repoURL string) {
	token, errDetail := fetchGitOauthAccessForUser(userID, "github", defaultGithubProviderKey())
	if errDetail != "" {
		status := http.StatusBadGateway
		writeJSON(w, status, map[string]any{
			"ok":     false,
			"status": status,
			"detail": errDetail,
		})
		return
	}
	if token == "" {
		writeJSON(w, http.StatusConflict, map[string]any{
			"ok":     false,
			"status": 409,
			"detail": "gitOauth 未找到该用户的 GitHub 凭据，请重新完成授权",
		})
		return
	}

	slug := githubRepoSlugFromURL(repoURL)
	if slug == "" {
		slug = "default"
	}
	pushBody["github_auth_by_repo"] = map[string]string{slug: token}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":                    true,
		"push_body":             pushBody,
		"use_oauth_access_push": true,
	})
}

func githubRepoSlugFromURL(raw string) string {
	u := strings.TrimSpace(raw)
	if u == "" {
		return ""
	}
	u = strings.TrimSuffix(u, ".git")
	u = strings.TrimRight(u, "/")
	// ssh: git@host:owner/repo
	if at := strings.Index(u, ":"); at > 0 && strings.Contains(u[:at], "@") {
		path := u[at+1:]
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) >= 2 {
			return strings.ToLower(parts[len(parts)-2] + "/" + parts[len(parts)-1])
		}
	}
	// https://host/owner/repo
	u = strings.TrimPrefix(u, "https://")
	u = strings.TrimPrefix(u, "http://")
	parts := strings.Split(u, "/")
	if len(parts) >= 3 {
		host := strings.ToLower(parts[0])
		if host != "github.com" && !strings.HasSuffix(host, ".github.com") {
			return ""
		}
		return strings.ToLower(parts[len(parts)-2] + "/" + parts[len(parts)-1])
	}
	return ""
}

func fetchGitOauthAccessForUser(userID, provider, providerKey string) (token string, errDetail string) {
	base := strings.TrimRight(strings.TrimSpace(cfg.GitOauthBaseURL), "/")
	if base == "" {
		return "", "gitOauth not configured"
	}
	provider = strings.TrimSpace(provider)
	if provider == "" {
		provider = "github"
	}
	providerKey = strings.TrimSpace(providerKey)
	if providerKey == "" {
		providerKey = provider
	}
	url := base + "/api/internal/" + provider + "/oauth/access-for-user/"
	payload, _ := json.Marshal(map[string]any{
		"user_id":      userID,
		"provider_key": providerKey,
	})
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Sprintf("gitOauth request error: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if secret := strings.TrimSpace(cfg.GitOauthBridgeSecret); secret != "" {
		req.Header.Set("X-GitOauth-Bridge-Secret", secret)
	}
	client := &http.Client{Transport: &http.Transport{Proxy: nil}, Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Sprintf("gitOauth unreachable: %v", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusNotFound {
		// OPT-20260821-037: 透传 taskGitOauth 404 的 errDetail（缺凭据/已失效），
		// 让调用方与前端看到具体原因而非「未能换取 Git OAuth 凭据」。
		if detail := gitOAuthErrDetailFromBody(raw); detail != "" {
			return "", detail
		}
		return "", "gitOauth 未找到该用户的 Git OAuth 授权，请重新完成授权"
	}
	if resp.StatusCode >= 400 {
		detail := strings.TrimSpace(string(raw))
		if detail == "" {
			detail = fmt.Sprintf("http %d", resp.StatusCode)
		}
		return "", fmt.Sprintf("gitOauth error: %s", detail)
	}
	var parsed struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", "invalid json from gitOauth"
	}
	tok := strings.TrimSpace(parsed.AccessToken)
	if tok == "" {
		return "", "no access_token in gitOauth response"
	}
	return tok, ""
}

// gitOAuthErrDetailFromBody 从 taskGitOauth 错误响应提取 errDetail（优先）或 detail。
// 不含 token；解析失败返回空串。
func gitOAuthErrDetailFromBody(raw []byte) string {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return ""
	}
	var out struct {
		Detail    string `json:"detail"`
		ErrDetail string `json:"errDetail"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return strings.TrimSpace(string(raw))
	}
	if d := strings.TrimSpace(out.ErrDetail); d != "" {
		return d
	}
	if d := strings.TrimSpace(out.Detail); d != "" {
		return d
	}
	return ""
}
