package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

var commitHashLikePattern = regexp.MustCompile(`(?i)^[0-9a-f]{7,40}$`)

func isCommitHashLike(ref string) bool {
	return commitHashLikePattern.MatchString(strings.TrimSpace(ref))
}

type resolveRefPayload struct {
	Exists bool              `json:"exists"`
	SHA    string            `json:"sha,omitempty"`
	Error  string            `json:"error,omitempty"`
	Gitlab map[string]string `json:"gitlab,omitempty"`
	Github map[string]string `json:"github,omitempty"`
}

func emptyResolveRefPayload() resolveRefPayload {
	return resolveRefPayload{Exists: false}
}

func resolveProjectRepoRef(userID string, repoURL string, ref string, gitlabSessionCookie string, tenantID string, traceHeaders ...map[string]string) resolveRefPayload {
	trace := outboundTrace(traceHeaders...)
	normalizedRef := strings.TrimSpace(ref)
	if !isCommitHashLike(normalizedRef) {
		p := emptyResolveRefPayload()
		p.Error = "ref must be a commit hash (7-40 hex chars)"
		return p
	}

	normalized := strings.TrimSpace(repoURL)
	if normalized == "" {
		p := emptyResolveRefPayload()
		p.Error = "No repository URL provided"
		return p
	}

	apiRepoURL, ok := normalizeGitRepoURLForBranchLookup(normalized)
	if !ok {
		p := emptyResolveRefPayload()
		p.Error = "Invalid Git repository URL format"
		return p
	}

	match := matchRepoProvider(apiRepoURL, tenantID, trace)
	providerKey := match.ProviderKey
	if strings.HasPrefix(providerKey, "gitlab:") || providerResolver.IsGitLabRepo(apiRepoURL) {
		return resolveGitLabCommit(userID, apiRepoURL, providerKey, match.GitoauthBase, gitlabSessionCookie, normalizedRef, trace)
	}
	if strings.HasPrefix(providerKey, "github:") || strings.Contains(strings.ToLower(apiRepoURL), "github.com") {
		if providerKey == "" {
			providerKey = "github:github-official"
		}
		return resolveGitHubCommit(userID, apiRepoURL, providerKey, match.GitoauthBase, normalizedRef, trace)
	}
	if strings.Contains(strings.ToLower(apiRepoURL), "bitbucket.org") {
		return resolveBitbucketCommit(apiRepoURL, normalizedRef)
	}
	p := emptyResolveRefPayload()
	p.Error = "Commit lookup is not supported for this repository host"
	return p
}

func resolveGitLabCommit(userID, repoURL, providerKey, gitoauthBase, sessionCookie, ref string, traceHeaders ...map[string]string) resolveRefPayload {
	payload := emptyResolveRefPayload()
	sessionCookie = strings.TrimSpace(sessionCookie)

	var token string
	var tokenErr string
	if uid, err := strconv.ParseInt(strings.TrimSpace(userID), 10, 64); err == nil && uid > 0 && strings.TrimSpace(repoURL) != "" {
		token, tokenErr = fetchGitAccessToken(uid, repoURL, gitoauthBase, outboundTrace(traceHeaders...))
	}
	if tokenErr != "" {
		payload.Gitlab = map[string]string{"resolve_error": tokenErr}
	}

	if token == "" && sessionCookie == "" {
		payload.Error = gitlabNeedsAuthMessage
		return payload
	}

	sha, errMsg := fetchGitLabCommit(repoURL, token, sessionCookie, ref, outboundTrace(traceHeaders...))
	if errMsg == "" {
		payload.Exists = true
		payload.SHA = sha
		payload.Gitlab = nil
		return payload
	}
	if errMsg == "not_found" {
		return payload
	}
	if strings.HasPrefix(errMsg, "无法获取") {
		payload.Error = errMsg
	} else {
		payload.Error = "无法校验 GitLab commit：" + errMsg
	}
	return payload
}

func resolveGitHubCommit(userID, repoURL, providerKey, gitoauthBase, ref string, traceHeaders ...map[string]string) resolveRefPayload {
	payload := emptyResolveRefPayload()
	var token string
	var tokenErr string
	if uid, err := strconv.ParseInt(strings.TrimSpace(userID), 10, 64); err == nil && uid > 0 {
		token, tokenErr = fetchGitAccessToken(uid, repoURL, gitoauthBase, outboundTrace(traceHeaders...))
	}
	if tokenErr != "" {
		payload.Github = map[string]string{"resolve_error": tokenErr}
	}
	// Public commits are readable anonymously via api.github.com.
	if token == "" {
		sha, errMsg := fetchGitHubCommit(repoURL, "", ref)
		if errMsg == "" {
			payload.Exists = true
			payload.SHA = sha
			payload.Github = nil
			return payload
		}
		if errMsg == "not_found" {
			return payload
		}
		payload.Error = githubBranchesAuthFailureMessage(tokenErr)
		return payload
	}

	sha, errMsg := fetchGitHubCommit(repoURL, token, ref)
	if errMsg == "" {
		payload.Exists = true
		payload.SHA = sha
		payload.Github = nil
		return payload
	}
	if errMsg == "not_found" {
		return payload
	}
	if strings.HasPrefix(errMsg, "无法获取") {
		payload.Error = errMsg
	} else {
		payload.Error = "无法校验 GitHub commit：" + errMsg
	}
	return payload
}

func fetchGitLabCommit(repoURL, token, sessionCookie, ref string, traceHeaders ...map[string]string) (string, string) {
	parts, err := parseGitLabProjectParts(repoURL)
	if err != nil {
		return "", err.Error()
	}
	apiURL := parts.APIBase + "/repository/commits/" + url.PathEscape(ref)
	resp, err := gitlabRESTGet(apiURL, token, sessionCookie, defaultGitLabHost())
	if err != nil {
		return "", fmt.Sprintf("Error getting GitLab commit: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	switch resp.StatusCode {
	case http.StatusOK:
		var row struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(body, &row); err != nil {
			return "", fmt.Sprintf("Error parsing GitLab commit: %v", err)
		}
		if strings.TrimSpace(row.ID) == "" {
			return "", "GitLab commit response missing id"
		}
		return row.ID, ""
	case http.StatusNotFound:
		return "", "not_found"
	default:
		logGitLabAPIFailure("resolve-commit", repoURL, resp.StatusCode, body, outboundTrace(traceHeaders...))
		return "", formatGitLabAPIError(resp.StatusCode, body)
	}
}

func fetchGitHubCommit(repoURL, token, ref string) (string, string) {
	match := githubRepoPattern.FindStringSubmatch(repoURL)
	if match == nil {
		return "", "Invalid GitHub repository URL format"
	}
	owner, repo := match[1], strings.TrimSuffix(match[2], "/")
	if strings.HasSuffix(strings.ToLower(repo), ".git") {
		repo = repo[:len(repo)-4]
	}
	apiURL := fmt.Sprintf("%s/repos/%s/%s/commits/%s", strings.TrimRight(githubAPIBase, "/"), owner, repo, url.PathEscape(ref))
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return "", fmt.Sprintf("Error getting GitHub commit: %v", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := gitHTTPClient.Do(req)
	if err != nil {
		return "", fmt.Sprintf("Error getting GitHub commit: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusUnauthorized && token != "" {
		req2, _ := http.NewRequest(http.MethodGet, apiURL, nil)
		req2.Header.Set("Accept", "application/vnd.github+json")
		req2.Header.Set("X-GitHub-Api-Version", "2022-11-28")
		req2.Header.Set("Authorization", "token "+token)
		resp2, err2 := gitHTTPClient.Do(req2)
		if err2 == nil {
			defer resp2.Body.Close()
			body, _ = io.ReadAll(resp2.Body)
			resp = resp2
		}
	}
	switch resp.StatusCode {
	case http.StatusOK:
		var row struct {
			SHA string `json:"sha"`
		}
		if err := json.Unmarshal(body, &row); err != nil {
			return "", fmt.Sprintf("Error parsing GitHub commit: %v", err)
		}
		if strings.TrimSpace(row.SHA) == "" {
			return "", "GitHub commit response missing sha"
		}
		return row.SHA, ""
	case http.StatusNotFound, http.StatusUnprocessableEntity:
		// Ambiguous or invalid SHA often returns 422
		return "", "not_found"
	default:
		logGitHubAPIFailure("resolve-commit", repoURL, resp.StatusCode, body)
		return "", formatGitHubAPIError(resp.StatusCode, body)
	}
}

func resolveBitbucketCommit(repoURL, ref string) resolveRefPayload {
	payload := emptyResolveRefPayload()
	match := bitbucketRepoPattern.FindStringSubmatch(repoURL)
	if match == nil {
		payload.Error = "Invalid Bitbucket repository URL format"
		return payload
	}
	apiURL := fmt.Sprintf("https://api.bitbucket.org/2.0/repositories/%s/%s/commit/%s", match[1], match[2], url.PathEscape(ref))
	resp, err := gitHTTPClient.Do(mustNewGet(apiURL))
	if err != nil {
		payload.Error = fmt.Sprintf("Error getting Bitbucket commit: %v", err)
		return payload
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	switch resp.StatusCode {
	case http.StatusOK:
		var row struct {
			Hash string `json:"hash"`
		}
		if err := json.Unmarshal(body, &row); err != nil {
			payload.Error = fmt.Sprintf("Error parsing Bitbucket commit: %v", err)
			return payload
		}
		if strings.TrimSpace(row.Hash) == "" {
			payload.Error = "Bitbucket commit response missing hash"
			return payload
		}
		payload.Exists = true
		payload.SHA = row.Hash
		return payload
	case http.StatusNotFound:
		return payload
	case 429:
		payload.Error = "Bitbucket API rate limit exceeded"
		return payload
	default:
		payload.Error = fmt.Sprintf("Bitbucket API error: %d", resp.StatusCode)
		return payload
	}
}
