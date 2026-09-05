package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
)

const (
	tokenStatusAvailable     = "token_available"
	tokenStatusNotBound      = "not_bound"
	tokenStatusTokenError    = "token_error"
	tokenStatusNotApplicable = "not_applicable"
)

type gitRepoValidateResult struct {
	URL                  string
	IsAccessible         bool
	Message              string
	TokenStatus          string
	OAuthProvider        string
	OAuthServiceProvider string
}

func (r gitRepoValidateResult) toMap() map[string]interface{} {
	out := map[string]interface{}{
		"url":                    r.URL,
		"repo_url":               r.URL,
		"is_accessible":          r.IsAccessible,
		"message":                r.Message,
		"token_status":           r.TokenStatus,
		"oauth_provider":         r.OAuthProvider,
		"oauth_service_provider": r.OAuthServiceProvider,
	}
	return out
}

func parseUserIDInt(userID string) int64 {
	uid, err := strconv.ParseInt(strings.TrimSpace(userID), 10, 64)
	if err != nil {
		return 0
	}
	return uid
}

func splitProviderKey(providerKey string) (provider, serviceProvider string) {
	providerKey = strings.TrimSpace(providerKey)
	if providerKey == "" {
		return "", ""
	}
	parts := strings.SplitN(providerKey, ":", 2)
	provider = parts[0]
	if len(parts) > 1 {
		serviceProvider = parts[1]
	} else {
		serviceProvider = "default"
	}
	return provider, serviceProvider
}

// validateGitRepoForUser resolves OAuth token_status via gitOauth and optionally probes remote accessibility.
// No Django dependency.
func validateGitRepoForUser(userID, repoURL string, probeAccess bool, tenantID, grantProjectID string, traceHeaders ...map[string]string) gitRepoValidateResult {
	trace := outboundTrace(traceHeaders...)
	raw := strings.TrimSpace(repoURL)
	result := gitRepoValidateResult{
		URL:         raw,
		TokenStatus: tokenStatusNotApplicable,
		Message:     "",
	}
	if raw == "" {
		result.IsAccessible = false
		result.Message = "请输入有效的 Git 仓库地址"
		return result
	}

	apiRepoURL, ok := normalizeGitRepoURLForBranchLookup(raw)
	if !ok {
		result.IsAccessible = false
		result.Message = "Invalid Git repository URL format"
		return result
	}
	result.URL = strings.TrimSpace(repoURL) // keep caller URL as key; probe uses apiRepoURL

	match := matchRepoProvider(apiRepoURL, tenantID, trace)
	providerKey := match.ProviderKey
	gitoauthBase := match.GitoauthBase
	lower := strings.ToLower(apiRepoURL)

	isGitHub := strings.HasPrefix(providerKey, "github:") || strings.Contains(lower, "github.com")
	host, _ := extractHostNetloc(apiRepoURL)
	isGitLab := strings.HasPrefix(providerKey, "gitlab:") ||
		strings.Contains(strings.ToLower(host), "gitlab") ||
		(providerResolver != nil && !isGitHub && providerResolver.IsGitLabRepo(apiRepoURL))
	isBitbucket := strings.Contains(lower, "bitbucket.org")

	uid := parseUserIDInt(userID)
	var token string
	var tokenErr string
	skipExchange := false
	if grantID := sanitizeResourceID(grantProjectID); grantID != "" {
		_, site := extractHostNetloc(apiRepoURL)
		if site == "" {
			site = strings.ToLower(host)
		}
		if !hasProjectGitOAuthGrant(grantID, userID, site) {
			skipExchange = true
		}
	}

	switch {
	case isGitHub:
		if providerKey == "" {
			providerKey = "github:github-official"
		}
		result.OAuthProvider, result.OAuthServiceProvider = splitProviderKey(providerKey)
		if uid > 0 && !skipExchange {
			token, tokenErr = fetchGitAccessToken(uid, apiRepoURL, gitoauthBase, trace)
		}
		result.TokenStatus = resolveTokenStatus(token, tokenErr)
	case isGitLab:
		result.OAuthProvider, result.OAuthServiceProvider = splitProviderKey(providerKey)
		if uid > 0 && strings.TrimSpace(apiRepoURL) != "" && !skipExchange {
			token, tokenErr = fetchGitAccessToken(uid, apiRepoURL, gitoauthBase, trace)
		}
		result.TokenStatus = resolveTokenStatus(token, tokenErr)
	default:
		result.TokenStatus = tokenStatusNotApplicable
	}

	if !probeAccess {
		// token-only path (autorun / nested oauth badges): do not hit remote
		result.IsAccessible = result.TokenStatus == tokenStatusAvailable || result.TokenStatus == tokenStatusNotApplicable
		if result.TokenStatus == tokenStatusNotBound || result.TokenStatus == tokenStatusTokenError {
			result.IsAccessible = false
		}
		applyProjectGrantGate(&result, grantProjectID, userID, apiRepoURL)
		if result.TokenStatus == tokenStatusNotBound || result.TokenStatus == tokenStatusTokenError {
			result.IsAccessible = false
		}
		return result
	}

	accessible, msg := probeGitRepoAccessible(apiRepoURL, token, isGitHub, isGitLab, isBitbucket, outboundTrace(traceHeaders...))
	result.IsAccessible = accessible
	result.Message = msg
	applyRepoProbeToTokenStatus(&result, traceIDFromOutbound(trace))
	applyProjectGrantGate(&result, grantProjectID, userID, apiRepoURL)
	return result
}

func applyProjectGrantGate(result *gitRepoValidateResult, grantProjectID, userID, repoURL string) {
	if result == nil || sanitizeResourceID(grantProjectID) == "" {
		return
	}
	_, site := extractHostNetloc(repoURL)
	if site == "" {
		site = strings.ToLower(strings.TrimSpace(result.URL))
	}
	result.TokenStatus = applyResourceGrantGate(result.TokenStatus, hasProjectGitOAuthGrant(grantProjectID, userID, site))
}

func resolveTokenStatus(token, tokenErr string) string {
	if strings.TrimSpace(token) != "" {
		return tokenStatusAvailable
	}
	if strings.TrimSpace(tokenErr) != "" {
		return tokenStatusTokenError
	}
	return tokenStatusNotBound
}

func traceIDFromOutbound(h map[string]string) string {
	if h == nil {
		return ""
	}
	return strings.TrimSpace(h["X-Trace-Id"])
}

// shouldDowngradeTokenAvailableOnInaccessibleProbe is true when a provider token
// exists but this specific repo is 401/403/404 (typical after switching the
// project URL to another owner's private repo). Rate-limit / 5xx stay available.
func shouldDowngradeTokenAvailableOnInaccessibleProbe(accessible bool, tokenStatus, message string) bool {
	if accessible || tokenStatus != tokenStatusAvailable {
		return false
	}
	lower := strings.ToLower(strings.TrimSpace(message))
	if lower == "" {
		return false
	}
	if strings.Contains(lower, "rate limit") || strings.Contains(lower, "限流") {
		return false
	}
	if strings.Contains(lower, "not found") || strings.Contains(lower, "404") {
		return true
	}
	if strings.Contains(lower, "401") || strings.Contains(lower, "unauthorized") {
		return true
	}
	if strings.Contains(lower, "403") || strings.Contains(lower, "forbidden") ||
		strings.Contains(lower, "拒绝访问") || strings.Contains(lower, "denied") {
		return true
	}
	if strings.Contains(lower, "push") || strings.Contains(lower, "写入") {
		return true
	}
	return false
}

func applyRepoProbeToTokenStatus(result *gitRepoValidateResult, traceID string) {
	if result == nil {
		return
	}
	if !shouldDowngradeTokenAvailableOnInaccessibleProbe(result.IsAccessible, result.TokenStatus, result.Message) {
		return
	}
	result.TokenStatus = tokenStatusTokenError
	logInfo(
		fmt.Sprintf("git-repo-validate token_status=token_error reason=probe_inaccessible repo=%s", redactRepoURLForLog(result.URL)),
		traceID,
	)
}

func probeGitRepoAccessible(apiRepoURL, token string, isGitHub, isGitLab, isBitbucket bool, traceHeaders ...map[string]string) (bool, string) {
	switch {
	case isGitHub:
		return probeGitHubRepo(apiRepoURL, token)
	case isGitLab:
		return probeGitLabRepo(apiRepoURL, token, outboundTrace(traceHeaders...))
	case isBitbucket:
		return probeBitbucketRepo(apiRepoURL)
	default:
		return probeGenericGitRepo(apiRepoURL)
	}
}

var githubAPIBase = "https://api.github.com"

func probeGitHubRepo(repoURL, token string) (bool, string) {
	match := githubRepoPattern.FindStringSubmatch(repoURL)
	if match == nil {
		return false, "Invalid GitHub repository URL format"
	}
	owner, repo := match[1], strings.TrimSuffix(match[2], "/")
	if strings.HasSuffix(strings.ToLower(repo), ".git") {
		repo = repo[:len(repo)-4]
	}
	apiURL := fmt.Sprintf("%s/repos/%s/%s", strings.TrimRight(githubAPIBase, "/"), owner, repo)
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return false, fmt.Sprintf("Error checking GitHub repository: %v", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := gitHTTPClient.Do(req)
	if err != nil {
		return false, fmt.Sprintf("Error checking GitHub repository: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	switch resp.StatusCode {
	case http.StatusOK:
		// GitHub GET /repos includes permissions.{admin,maintain,push,pull} when authenticated
		// (https://docs.github.com/en/rest/repos/repos#get-a-repository). Pull-only on a
		// public third-party repo must not count as 已授权 for 云端开发 (needs push).
		if strings.TrimSpace(token) != "" && !githubRepoResponseAllowsPush(body) {
			return false, "当前 GitHub 授权对该仓库没有 push 权限"
		}
		return true, ""
	default:
		logGitHubAPIFailure("probe-repo", repoURL, resp.StatusCode, body)
		return false, formatGitHubAPIError(resp.StatusCode, body)
	}
}

func probeGitLabRepo(repoURL, token string, traceHeaders ...map[string]string) (bool, string) {
	parts, err := parseGitLabProjectParts(repoURL)
	if err != nil {
		return false, err.Error()
	}
	resp, err := gitlabRESTGet(parts.APIBase, token, "", "")
	if err != nil {
		return false, fmt.Sprintf("Error checking GitLab repository: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	switch resp.StatusCode {
	case http.StatusOK:
		if strings.TrimSpace(token) != "" && !gitlabProjectResponseAllowsPush(body) {
			return false, "当前 GitLab 授权对该仓库没有 Developer 及以上写入权限"
		}
		return true, ""
	case http.StatusNotFound:
		return false, "GitLab repository not found"
	default:
		logGitLabAPIFailure("probe-repo", repoURL, resp.StatusCode, body, outboundTrace(traceHeaders...))
		return false, formatGitLabAPIError(resp.StatusCode, body)
	}
}

func probeBitbucketRepo(repoURL string) (bool, string) {
	match := bitbucketRepoPattern.FindStringSubmatch(repoURL)
	if match == nil {
		return false, "Invalid Bitbucket repository URL format"
	}
	apiURL := fmt.Sprintf("https://api.bitbucket.org/2.0/repositories/%s/%s", match[1], match[2])
	resp, err := gitHTTPClient.Do(mustNewGet(apiURL))
	if err != nil {
		return false, fmt.Sprintf("Error checking Bitbucket repository: %v", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	switch resp.StatusCode {
	case http.StatusOK:
		return true, ""
	case http.StatusNotFound:
		return false, "Bitbucket repository not found"
	case 429:
		return false, "Bitbucket API rate limit exceeded"
	default:
		return false, fmt.Sprintf("Bitbucket API error: %d", resp.StatusCode)
	}
}

func probeGenericGitRepo(repoURL string) (bool, string) {
	u, err := url.Parse(repoURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return false, "Invalid Git repository URL format"
	}
	// Lightweight probe: HEAD/GET the URL without following into git protocol.
	req, err := http.NewRequest(http.MethodHead, repoURL, nil)
	if err != nil {
		return false, fmt.Sprintf("Error checking repository: %v", err)
	}
	resp, err := gitHTTPClient.Do(req)
	if err != nil {
		// Some hosts reject HEAD; try GET
		req2, err2 := http.NewRequest(http.MethodGet, repoURL, nil)
		if err2 != nil {
			return false, fmt.Sprintf("Error checking repository: %v", err)
		}
		resp2, err2 := gitHTTPClient.Do(req2)
		if err2 != nil {
			return false, fmt.Sprintf("Error checking repository: %v", err2)
		}
		defer resp2.Body.Close()
		_, _ = io.Copy(io.Discard, resp2.Body)
		if resp2.StatusCode >= 200 && resp2.StatusCode < 400 {
			return true, ""
		}
		return false, fmt.Sprintf("Repository HTTP error: %d", resp2.StatusCode)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return true, ""
	}
	if resp.StatusCode == http.StatusMethodNotAllowed {
		return probeGenericGitRepoGET(repoURL)
	}
	return false, fmt.Sprintf("Repository HTTP error: %d", resp.StatusCode)
}

func probeGenericGitRepoGET(repoURL string) (bool, string) {
	req, err := http.NewRequest(http.MethodGet, repoURL, nil)
	if err != nil {
		return false, fmt.Sprintf("Error checking repository: %v", err)
	}
	resp, err := gitHTTPClient.Do(req)
	if err != nil {
		return false, fmt.Sprintf("Error checking repository: %v", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return true, ""
	}
	return false, fmt.Sprintf("Repository HTTP error: %d", resp.StatusCode)
}

const validateGitReposConcurrency = 8

func validateGitReposForUser(userID string, urls []string, probeAccess bool, tenantID, grantProjectID string, traceHeaders ...map[string]string) []gitRepoValidateResult {
	trace := outboundTrace(traceHeaders...)
	out := make([]gitRepoValidateResult, len(urls))
	if len(urls) == 0 {
		return out
	}
	workers := validateGitReposConcurrency
	if workers > len(urls) {
		workers = len(urls)
	}
	type job struct {
		idx int
		url string
	}
	jobs := make(chan job, len(urls))
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				out[j.idx] = validateGitRepoForUser(userID, j.url, probeAccess, tenantID, grantProjectID, trace)
			}
		}()
	}
	for i, u := range urls {
		jobs <- job{idx: i, url: u}
	}
	close(jobs)
	wg.Wait()
	return out
}
