package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	gitlabNeedsAuthMessage = "无法获取 GitLab 分支：未检测到可用授权。请先在个人资料完成 GitLab 绑定，或先登录本地 GitLab（localhost）后重试。"
	githubNeedsAuthMessage = "无法获取 GitHub 分支：未检测到可用授权。请先在个人资料完成 GitHub 绑定后重试。"
	genericBranchError     = "Branches cannot be retrieved from generic Git repositories without credentials"
)

type branchListPayload struct {
	Branches []string          `json:"branches"`
	Error    string            `json:"error"`
	Gitlab   map[string]string `json:"gitlab,omitempty"`
	Github   map[string]string `json:"github,omitempty"`
}

func emptyBranchPayload() branchListPayload {
	return branchListPayload{Branches: []string{}}
}

func listProjectRepoBranches(userID string, repoURL string, gitlabSessionCookie string, tenantID string, traceHeaders ...map[string]string) branchListPayload {
	trace := outboundTrace(traceHeaders...)
	normalized := strings.TrimSpace(repoURL)
	if normalized == "" {
		p := emptyBranchPayload()
		p.Error = "No repository URL provided"
		return p
	}

	apiRepoURL, ok := normalizeGitRepoURLForBranchLookup(normalized)
	if !ok {
		p := emptyBranchPayload()
		p.Error = "Invalid Git repository URL format"
		return p
	}

	match := matchRepoProvider(apiRepoURL, tenantID, trace)
	providerKey := match.ProviderKey
	if strings.HasPrefix(providerKey, "gitlab:") || providerResolver.IsGitLabRepo(apiRepoURL) {
		return listGitLabBranches(userID, apiRepoURL, match.GitoauthBase, gitlabSessionCookie, trace)
	}
	if strings.HasPrefix(providerKey, "github:") || strings.Contains(strings.ToLower(apiRepoURL), "github.com") {
		return listGitHubBranches(userID, apiRepoURL, match.GitoauthBase, trace)
	}
	if strings.Contains(strings.ToLower(apiRepoURL), "bitbucket.org") {
		branches, err := fetchBitbucketBranches(apiRepoURL)
		p := emptyBranchPayload()
		p.Branches = branches
		p.Error = err
		return p
	}
	p := emptyBranchPayload()
	p.Error = genericBranchError
	return p
}

func listGitLabBranches(userID, repoURL, gitoauthBase, sessionCookie string, traceHeaders ...map[string]string) branchListPayload {
	payload := emptyBranchPayload()
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

	// The browser `_gitlab_session` belongs to the primary GitLab instance;
	// gitlabRESTGet only attaches it when the request host matches that
	// instance, so regional repos rely on their own OAuth Bearer.
	branches, err := fetchGitLabBranches(repoURL, token, sessionCookie, defaultGitLabHost(), outboundTrace(traceHeaders...))
	payload.Branches = branches
	if err == "" {
		payload.Gitlab = nil
		return payload
	}

	if strings.HasPrefix(err, "无法获取") {
		payload.Error = err
	} else if err == "GitLab repository not found" {
		if sessionCookie != "" && token == "" {
			payload.Error = "无法获取 GitLab 分支：GitLab 会话无效或无权访问该仓库，请重新登录 GitLab 或完成 OAuth 授权"
		} else if sessionCookie != "" {
			payload.Error = "无法获取 GitLab 分支：GitLab 会话无效或仓库不存在，请确认仓库地址并完成 OAuth 授权"
		} else {
			payload.Error = "无法获取 GitLab 分支：" + err
		}
	} else {
		payload.Error = "无法获取 GitLab 分支：" + err
	}
	return payload
}

func listGitHubBranches(userID, repoURL, gitoauthBase string, traceHeaders ...map[string]string) branchListPayload {
	payload := emptyBranchPayload()
	var token string
	var tokenErr string
	if uid, err := strconv.ParseInt(strings.TrimSpace(userID), 10, 64); err == nil && uid > 0 {
		token, tokenErr = fetchGitAccessToken(uid, repoURL, gitoauthBase, outboundTrace(traceHeaders...))
	}
	if tokenErr != "" {
		payload.Github = map[string]string{"resolve_error": tokenErr}
	}
	// Public GitHub repos are readable via api.github.com without a user token.
	// When OAuth refresh fails (e.g. github.com/login/oauth timeout), still try anonymous.
	if token == "" {
		branches, err := fetchGitHubBranches(repoURL, "")
		if err == "" {
			payload.Branches = branches
			payload.Github = nil
			return payload
		}
		payload.Error = githubBranchesAuthFailureMessage(tokenErr)
		return payload
	}

	branches, err := fetchGitHubBranches(repoURL, token)
	payload.Branches = branches
	if err != "" {
		if strings.HasPrefix(err, "无法获取") {
			payload.Error = err
		} else {
			payload.Error = "无法获取 GitHub 分支：" + err
		}
	} else {
		payload.Github = nil
	}
	return payload
}

type gitlabProjectParts struct {
	APIBase string
}

func parseGitLabProjectParts(repoURL string) (*gitlabProjectParts, error) {
	u, err := url.Parse(strings.TrimSpace(repoURL))
	if err != nil || u.Host == "" {
		return nil, fmt.Errorf("Invalid GitLab repository URL format")
	}
	path := strings.Trim(strings.TrimSpace(u.Path), "/")
	seg := strings.Split(path, "/")
	if len(seg) < 2 {
		return nil, fmt.Errorf("Invalid GitLab repository URL format")
	}
	repo := seg[len(seg)-1]
	owner := strings.Join(seg[:len(seg)-1], "/")
	if strings.HasSuffix(strings.ToLower(repo), ".git") {
		repo = repo[:len(repo)-4]
	}
	if owner == "" || repo == "" {
		return nil, fmt.Errorf("Invalid GitLab repository URL format")
	}
	scheme := strings.ToLower(strings.TrimSpace(u.Scheme))
	if scheme == "" {
		scheme = "https"
	}
	projectPath := owner + "/" + repo
	apiBase := fmt.Sprintf("%s://%s/api/v4/projects/%s", scheme, u.Host, url.PathEscape(projectPath))
	return &gitlabProjectParts{APIBase: apiBase}, nil
}

func gitlabRESTGet(apiURL, token, sessionCookie, cookieHost string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}
	// sessionCookie is scoped to cookieHost (the primary GitLab instance).
	// Sending it to a different host would 401 with a foreign session and
	// hide the correct OAuth Bearer, so only attach it on host match. When
	// cookieHost is empty (unknown/unresolved) keep the legacy permissive
	// behavior so existing cookie flows keep working.
	if sessionCookie != "" {
		if cookieHost == "" {
			req.Header.Set("Cookie", "_gitlab_session="+sessionCookie)
		} else if u, err := url.Parse(apiURL); err == nil && strings.EqualFold(u.Host, cookieHost) {
			req.Header.Set("Cookie", "_gitlab_session="+sessionCookie)
		}
	}
	if token != "" && req.Header.Get("Cookie") == "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := gitHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusUnauthorized && token != "" {
		resp.Body.Close()
		req2, err := http.NewRequest(http.MethodGet, apiURL, nil)
		if err != nil {
			return nil, err
		}
		req2.Header.Set("PRIVATE-TOKEN", token)
		return gitHTTPClient.Do(req2)
	}
	return resp, nil
}

// branchNamedTime carries an optional tip/update timestamp for sorting.
// Zero time means "unknown" — those entries keep relative input order after dated ones.
type branchNamedTime struct {
	Name      string
	UpdatedAt time.Time
	ord       int
}

// sortBranchNamesByUpdatedAtDesc returns names newest-first when UpdatedAt is set.
// Entries without timestamps keep their original relative order among themselves,
// and are placed after any dated entries (dated first, then undated in input order).
func sortBranchNamesByUpdatedAtDesc(rows []branchNamedTime) []string {
	if len(rows) == 0 {
		return []string{}
	}
	cp := make([]branchNamedTime, len(rows))
	copy(cp, rows)
	for i := range cp {
		cp[i].ord = i
	}
	sort.SliceStable(cp, func(i, j int) bool {
		ai, aj := cp[i].UpdatedAt, cp[j].UpdatedAt
		zi, zj := ai.IsZero(), aj.IsZero()
		if zi && zj {
			return cp[i].ord < cp[j].ord
		}
		if zi {
			return false
		}
		if zj {
			return true
		}
		if !ai.Equal(aj) {
			return ai.After(aj)
		}
		return cp[i].ord < cp[j].ord
	})
	names := make([]string, 0, len(cp))
	for _, row := range cp {
		if row.Name != "" {
			names = append(names, row.Name)
		}
	}
	return names
}

func parseFlexibleTime(raw string) time.Time {
	s := strings.TrimSpace(raw)
	if s == "" {
		return time.Time{}
	}
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.000Z",
		"2006-01-02 15:04:05 MST",
		"2006-01-02T15:04:05Z",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

func fetchGitLabBranches(repoURL, token, sessionCookie, cookieHost string, traceHeaders ...map[string]string) ([]string, string) {
	parts, err := parseGitLabProjectParts(repoURL)
	if err != nil {
		return []string{}, err.Error()
	}
	// GitLab Branches API sort enum: name_asc | updated_asc | updated_desc
	// (lib/api/branches.rb). order_by=updated_at&sort=desc is 400.
	apiURL := parts.APIBase + "/repository/branches?per_page=100&sort=updated_desc"
	resp, err := gitlabRESTGet(apiURL, token, sessionCookie, cookieHost)
	if err != nil {
		return []string{}, fmt.Sprintf("Error getting GitLab branches: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	switch resp.StatusCode {
	case http.StatusOK:
		var rows []struct {
			Name   string `json:"name"`
			Commit struct {
				CommittedDate string `json:"committed_date"`
				AuthoredDate  string `json:"authored_date"`
			} `json:"commit"`
		}
		if err := json.Unmarshal(body, &rows); err != nil {
			return []string{}, fmt.Sprintf("Error getting GitLab branches: %v", err)
		}
		timed := make([]branchNamedTime, 0, len(rows))
		for _, row := range rows {
			if row.Name == "" {
				continue
			}
			ts := parseFlexibleTime(row.Commit.CommittedDate)
			if ts.IsZero() {
				ts = parseFlexibleTime(row.Commit.AuthoredDate)
			}
			timed = append(timed, branchNamedTime{Name: row.Name, UpdatedAt: ts})
		}
		return sortBranchNamesByUpdatedAtDesc(timed), ""
	case http.StatusNotFound:
		return []string{}, "GitLab repository not found"
	default:
		logGitLabAPIFailure("list-branches", repoURL, resp.StatusCode, body, outboundTrace(traceHeaders...))
		return []string{}, formatGitLabAPIError(resp.StatusCode, body)
	}
}

var githubRepoPattern = regexp.MustCompile(`(?i)github\.com/([^/]+)/([^/?#]+)`)

func fetchGitHubBranches(repoURL, token string) ([]string, string) {
	match := githubRepoPattern.FindStringSubmatch(repoURL)
	if match == nil {
		return []string{}, "Invalid GitHub repository URL format"
	}
	owner, repo := match[1], strings.TrimSuffix(match[2], "/")
	if strings.HasSuffix(strings.ToLower(repo), ".git") {
		repo = repo[:len(repo)-4]
	}
	apiURL := fmt.Sprintf("%s/repos/%s/%s/branches", strings.TrimRight(githubAPIBase, "/"), owner, repo)
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return []string{}, fmt.Sprintf("Error getting GitHub branches: %v", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := gitHTTPClient.Do(req)
	if err != nil {
		return []string{}, fmt.Sprintf("Error getting GitHub branches: %v", err)
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
		var rows []struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(body, &rows); err != nil {
			return []string{}, fmt.Sprintf("Error getting GitHub branches: %v", err)
		}
		// GitHub list-branches payload has no tip commit timestamps; keep API order.
		timed := make([]branchNamedTime, 0, len(rows))
		for _, row := range rows {
			if row.Name != "" {
				timed = append(timed, branchNamedTime{Name: row.Name})
			}
		}
		return sortBranchNamesByUpdatedAtDesc(timed), ""
	default:
		logGitHubAPIFailure("list-branches", repoURL, resp.StatusCode, body)
		return []string{}, formatGitHubAPIError(resp.StatusCode, body)
	}
}

var bitbucketRepoPattern = regexp.MustCompile(`bitbucket\.org/([^/]+)/([^/.]+)`)

func fetchBitbucketBranches(repoURL string) ([]string, string) {
	match := bitbucketRepoPattern.FindStringSubmatch(repoURL)
	if match == nil {
		return []string{}, "Invalid Bitbucket repository URL format"
	}
	apiURL := fmt.Sprintf("https://api.bitbucket.org/2.0/repositories/%s/%s/refs/branches", match[1], match[2])
	resp, err := gitHTTPClient.Do(mustNewGet(apiURL))
	if err != nil {
		return []string{}, fmt.Sprintf("Error getting Bitbucket branches: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	switch resp.StatusCode {
	case http.StatusOK:
		var payload struct {
			Values []struct {
				Name   string `json:"name"`
				Target struct {
					Date string `json:"date"`
				} `json:"target"`
			} `json:"values"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			return []string{}, fmt.Sprintf("Error getting Bitbucket branches: %v", err)
		}
		timed := make([]branchNamedTime, 0, len(payload.Values))
		for _, row := range payload.Values {
			if row.Name == "" {
				continue
			}
			timed = append(timed, branchNamedTime{
				Name:      row.Name,
				UpdatedAt: parseFlexibleTime(row.Target.Date),
			})
		}
		return sortBranchNamesByUpdatedAtDesc(timed), ""
	case http.StatusNotFound:
		return []string{}, "Bitbucket repository not found"
	case 429:
		return []string{}, "Bitbucket API rate limit exceeded"
	default:
		return []string{}, fmt.Sprintf("Bitbucket API error: %d", resp.StatusCode)
	}
}

func mustNewGet(rawURL string) *http.Request {
	req, _ := http.NewRequest(http.MethodGet, rawURL, nil)
	return req
}
