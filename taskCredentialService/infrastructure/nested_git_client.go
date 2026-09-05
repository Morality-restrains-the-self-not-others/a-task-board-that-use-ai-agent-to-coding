package infrastructure

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"taskCredentialService/application"
)

// NestedGitReposHTTPClient calls taskProjectService internal nested discovery.
type NestedGitReposHTTPClient struct {
	baseURL        string
	internalSecret string
	client         *http.Client
}

func NewNestedGitReposHTTPClient(projectServiceBase, internalSecret string, timeoutSeconds int) *NestedGitReposHTTPClient {
	if timeoutSeconds <= 0 {
		timeoutSeconds = 15
	}
	return &NestedGitReposHTTPClient{
		baseURL:        strings.TrimRight(strings.TrimSpace(projectServiceBase), "/"),
		internalSecret: strings.TrimSpace(internalSecret),
		client: &http.Client{
			Timeout: time.Duration(timeoutSeconds) * time.Second,
			Transport: &http.Transport{
				Proxy: nil,
			},
		},
	}
}

func (c *NestedGitReposHTTPClient) FetchNestedGitRepos(userID int64, tenantID, parentRepoURL string) ([]application.NestedRepoItem, error) {
	if c == nil || c.baseURL == "" {
		return nil, fmt.Errorf("project service base not configured")
	}
	parentRepoURL = strings.TrimSpace(parentRepoURL)
	if parentRepoURL == "" {
		return nil, nil
	}
	q := url.Values{}
	q.Set("repo_url", parentRepoURL)
	// userID=0：任务未绑定 git identity 时仍走匿名发现（GitHub Contents API）。
	if userID > 0 {
		q.Set("user_id", fmt.Sprintf("%d", userID))
	}
	// OPT-20260827-019：透传租户 ID（company_id），Path A GitLab 子仓发现
	// 才能匹配到租户 OAuth provider，避免「未检测到可用授权」误报。
	if strings.TrimSpace(tenantID) != "" {
		q.Set("tenant_id", tenantID)
	}
	reqURL := c.baseURL + "/api/internal/nested-git-repos/?" + q.Encode()
	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	if c.internalSecret != "" {
		req.Header.Set("X-Internal-Secret", c.internalSecret)
	}
	if strings.TrimSpace(tenantID) != "" {
		req.Header.Set("X-Auth-Tenant-Id", tenantID)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("nested-git-repos HTTP %d: %s", resp.StatusCode, truncate(string(raw), 200))
	}
	var payload struct {
		NestedRepos []struct {
			Path   string `json:"path"`
			URL    string `json:"url"`
			Source string `json:"source"`
		} `json:"nested_repos"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	out := make([]application.NestedRepoItem, 0, len(payload.NestedRepos))
	for _, n := range payload.NestedRepos {
		u := strings.TrimSpace(n.URL)
		if u == "" {
			continue
		}
		out = append(out, application.NestedRepoItem{
			Path:   strings.TrimSpace(n.Path),
			URL:    u,
			Source: strings.TrimSpace(n.Source),
		})
	}
	return out, nil
}
