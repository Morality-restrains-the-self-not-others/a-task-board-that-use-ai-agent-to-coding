package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const (
	nestedNeedsAuthMessage          = "无法获取子 Git 仓库列表：未检测到可用授权。请先在个人资料完成 Git 网站绑定，或先登录本地 GitLab 后重试。"
	nestedParentInaccessibleMessage = "无法访问父仓库（远端返回不可见/无权限）。" +
		"请确认 GitHub App 已配置 Contents: Read，并已安装到该仓库所属组织/账号后重新 OAuth 授权；" +
		"若父仓在 GitLab，请重新完成对应站点绑定。"
)

type nestedGitReposPayload struct {
	ParentRepoURL string       `json:"parent_repo_url"`
	NestedRepos   []NestedRepo `json:"nested_repos"`
	Error         string       `json:"error"`
}

func emptyNestedPayload(parent string) nestedGitReposPayload {
	return nestedGitReposPayload{
		ParentRepoURL: parent,
		NestedRepos:   []NestedRepo{},
		Error:         "",
	}
}

func listNestedGitRepos(userID, repoURL, gitlabSessionCookie, tenantID string, traceHeaders ...map[string]string) nestedGitReposPayload {
	trace := outboundTrace(traceHeaders...)
	normalized := strings.TrimSpace(repoURL)
	payload := emptyNestedPayload(normalized)
	if normalized == "" {
		payload.Error = "No repository URL provided"
		return payload
	}

	apiRepoURL, ok := normalizeGitRepoURLForBranchLookup(normalized)
	if !ok {
		payload.Error = "Invalid Git repository URL format"
		return payload
	}
	payload.ParentRepoURL = apiRepoURL

	match := matchRepoProvider(apiRepoURL, tenantID, trace)
	providerKey := match.ProviderKey
	isGitHub := strings.HasPrefix(providerKey, "github:") || strings.Contains(strings.ToLower(apiRepoURL), "github.com")
	// Prefer GitHub when host/key says so; never route github.com/*.git through GitLab APIs.
	isGitLab := !isGitHub && (strings.HasPrefix(providerKey, "gitlab:") || providerResolver.IsGitLabRepo(apiRepoURL))

	if !isGitLab && !isGitHub {
		payload.Error = "当前仅支持从 GitLab / GitHub 仓库发现子 Git 仓库"
		return payload
	}
	if isGitHub && providerKey == "" {
		providerKey = "github:github-official"
	}

	sessionCookie := strings.TrimSpace(gitlabSessionCookie)
	hasGitLabSession := isGitLab && sessionCookie != ""

	// GitHub fast path: try anonymous Contents API first.
	// api.github.com is often reachable while github.com/login/oauth/access_token times out;
	// skipping OAuth refresh avoids a ~15s hard-fail that was mislabeled as "未检测到可用授权".
	if isGitHub {
		if body, err := fetchRepoRawFile(apiRepoURL, ".gitmodules", "", "", false, outboundTrace(traceHeaders...)); err == "" {
			payload.NestedRepos = mergeNestedRepos(parseGitmodules(body), apiRepoURL)
			return payload
		}
	}

	var token, tokenErr string
	if uid, err := strconv.ParseInt(strings.TrimSpace(userID), 10, 64); err == nil && uid > 0 &&
		strings.TrimSpace(apiRepoURL) != "" && (strings.TrimSpace(providerKey) != "" || strings.TrimSpace(tenantID) != "") {
		token, tokenErr = fetchGitAccessToken(uid, apiRepoURL, match.GitoauthBase, trace)
	}
	if token == "" && !hasGitLabSession && !isGitHub {
		payload.Error = nestedAuthFailureMessage(tokenErr)
		return payload
	}
	if isGitHub && token == "" {
		// Anonymous already failed above; surface refresh/network copy when available.
		payload.Error = nestedAuthFailureMessage(tokenErr)
		return payload
	}

	gitmodulesBody, gmErr := fetchRepoRawFile(apiRepoURL, ".gitmodules", token, sessionCookie, isGitLab, outboundTrace(traceHeaders...))
	if gmErr != "" {
		// Parent-inaccessible is a distinct UX from "no OAuth binding".
		if strings.Contains(gmErr, "无法访问父仓库") {
			if tokenErr != "" {
				payload.Error = nestedAuthFailureMessage(tokenErr)
				return payload
			}
			payload.Error = gmErr
			return payload
		}
		if strings.Contains(gmErr, "401") || strings.Contains(gmErr, "重新 OAuth 授权") || strings.Contains(gmErr, "拒绝访问") {
			payload.Error = nestedAuthFailureMessage(tokenErr)
			return payload
		}
		payload.Error = "无法读取父仓库 .gitmodules：" + gmErr
		return payload
	}

	// Missing .gitmodules (404 → empty body) yields an empty list — no .gitignore fallback.
	gm := parseGitmodules(gitmodulesBody)
	payload.NestedRepos = mergeNestedRepos(gm, apiRepoURL)
	return payload
}

// fetchRepoRawFile loads a root file; 404 → empty body + "".
// When every candidate ref is 404, probes whether the parent repo itself is visible:
// inaccessible parent → error (must not look like "未发现子仓库"); visible parent → empty file.
func fetchRepoRawFile(repoURL, filePath, token, sessionCookie string, isGitLab bool, traceHeaders ...map[string]string) (string, string) {
	refs := candidateRawFileRefs(repoURL, token, isGitLab)
	var lastErr string
	sawNotFound := false
	for _, ref := range refs {
		body, errMsg, notFound := fetchRepoRawFileAtRef(repoURL, filePath, ref, token, sessionCookie, isGitLab, outboundTrace(traceHeaders...))
		if errMsg == "" {
			return body, ""
		}
		if notFound {
			sawNotFound = true
			lastErr = errMsg
			continue
		}
		return "", errMsg
	}
	if sawNotFound {
		if err := nestedParentAccessError(repoURL, token, sessionCookie, isGitLab, outboundTrace(traceHeaders...)); err != "" {
			return "", err
		}
		return "", "" // parent visible, file missing → empty list
	}
	return "", lastErr
}

func candidateRawFileRefs(repoURL, token string, isGitLab bool) []string {
	refs := make([]string, 0, 3)
	if def := fetchRemoteDefaultBranch(repoURL, token, isGitLab); def != "" {
		refs = append(refs, def)
	}
	for _, r := range []string{"main", "master"} {
		dup := false
		for _, e := range refs {
			if e == r {
				dup = true
				break
			}
		}
		if !dup {
			refs = append(refs, r)
		}
	}
	return refs
}

func fetchRemoteDefaultBranch(repoURL, token string, isGitLab bool) string {
	if isGitLab {
		return fetchGitLabDefaultBranch(repoURL, token)
	}
	return fetchGitHubDefaultBranch(repoURL, token)
}

func fetchGitHubDefaultBranch(repoURL, token string) string {
	match := githubRepoPattern.FindStringSubmatch(repoURL)
	if match == nil {
		return ""
	}
	owner, repo := match[1], strings.TrimSuffix(match[2], "/")
	if strings.HasSuffix(strings.ToLower(repo), ".git") {
		repo = repo[:len(repo)-4]
	}
	apiURL := fmt.Sprintf("%s/repos/%s/%s", strings.TrimRight(githubAPIBase, "/"), owner, repo)
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := gitHTTPClient.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return ""
	}
	var meta struct {
		DefaultBranch string `json:"default_branch"`
	}
	if err := json.Unmarshal(raw, &meta); err != nil {
		return ""
	}
	return strings.TrimSpace(meta.DefaultBranch)
}

func fetchGitLabDefaultBranch(repoURL, token string) string {
	parts, err := parseGitLabProjectParts(repoURL)
	if err != nil {
		return ""
	}
	resp, err := gitlabRESTGet(parts.APIBase, token, "", "")
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return ""
	}
	var meta struct {
		DefaultBranch string `json:"default_branch"`
	}
	if err := json.Unmarshal(raw, &meta); err != nil {
		return ""
	}
	return strings.TrimSpace(meta.DefaultBranch)
}

func nestedParentAccessError(repoURL, token, sessionCookie string, isGitLab bool, traceHeaders ...map[string]string) string {
	if isGitLab {
		// Prefer token probe; when only session cookie is available, skip parent probe
		// (cookie auth is not applied by probeGitLabRepo) and keep empty-file semantics.
		if strings.TrimSpace(token) == "" && strings.TrimSpace(sessionCookie) != "" {
			return ""
		}
		ok, msg := probeGitLabRepo(repoURL, token, outboundTrace(traceHeaders...))
		if ok {
			return ""
		}
		if msg == "" {
			return nestedParentInaccessibleMessage
		}
		return nestedParentInaccessibleMessage + "（" + msg + "）"
	}
	ok, msg := probeGitHubRepo(repoURL, token)
	if ok {
		return ""
	}
	if msg == "" {
		return nestedParentInaccessibleMessage
	}
	return nestedParentInaccessibleMessage + "（" + msg + "）"
}

func fetchRepoRawFileAtRef(repoURL, filePath, ref, token, sessionCookie string, isGitLab bool, traceHeaders ...map[string]string) (body string, errMsg string, notFound bool) {
	if isGitLab {
		parts, err := parseGitLabProjectParts(repoURL)
		if err != nil {
			return "", err.Error(), false
		}
		apiURL := parts.APIBase + "/repository/files/" + url.PathEscape(filePath) + "/raw?ref=" + url.QueryEscape(ref)
		resp, err := gitlabRESTGet(apiURL, token, sessionCookie, defaultGitLabHost())
		if err != nil {
			return "", fmt.Sprintf("Error reading GitLab file: %v", err), false
		}
		defer resp.Body.Close()
		raw, _ := io.ReadAll(resp.Body)
		switch resp.StatusCode {
		case http.StatusOK:
			return string(raw), "", false
		case http.StatusNotFound:
			return "", "not found", true
		default:
			logGitLabAPIFailure("read-file", repoURL, resp.StatusCode, raw, outboundTrace(traceHeaders...))
			return "", formatGitLabAPIError(resp.StatusCode, raw), false
		}
	}

	match := githubRepoPattern.FindStringSubmatch(repoURL)
	if match == nil {
		return "", "Invalid GitHub repository URL format", false
	}
	owner, repo := match[1], strings.TrimSuffix(match[2], "/")
	if strings.HasSuffix(strings.ToLower(repo), ".git") {
		repo = repo[:len(repo)-4]
	}
	apiURL := fmt.Sprintf("%s/repos/%s/%s/contents/%s?ref=%s",
		strings.TrimRight(githubAPIBase, "/"), owner, repo, url.PathEscape(filePath), url.QueryEscape(ref))
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return "", err.Error(), false
	}
	req.Header.Set("Accept", "application/vnd.github.raw")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := gitHTTPClient.Do(req)
	if err != nil {
		return "", fmt.Sprintf("Error reading GitHub file: %v", err), false
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	switch resp.StatusCode {
	case http.StatusOK:
		return string(raw), "", false
	case http.StatusNotFound:
		return "", "not found", true
	default:
		logGitHubAPIFailure("read-file", repoURL, resp.StatusCode, raw)
		return "", formatGitHubAPIError(resp.StatusCode, raw), false
	}
}
