package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// gitOAuthHTTP 探测 taskGitOauth user-app-connection 的超时客户端。
var gitOAuthHTTP = &http.Client{Timeout: 8 * time.Second}

// gitOAuthUserAppConnection queries taskGitOauth for the user's Git app-connection
// status for a specific repo URL. Returns connected bool. Network/5xx errors surface
// as error so callers can fail closed (OPT-20260821-036). Package var seam for tests.
var gitOAuthUserAppConnection = gitOAuthUserAppConnectionLive

func gitOAuthUserAppConnectionLive(tenantID, userID, repoURL string) (bool, error) {
	base := strings.TrimRight(strings.TrimSpace(cfg.GitOAuthURL), "/")
	if base == "" {
		return false, fmt.Errorf("git-oauth url not configured")
	}
	u, err := url.Parse(base + "/api/git-oauth/user-app-connection/")
	if err != nil {
		return false, err
	}
	q := u.Query()
	if strings.TrimSpace(repoURL) != "" {
		q.Set("repo_url", repoURL)
	}
	u.RawQuery = q.Encode()
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("X-User-Id", userID)
	req.Header.Set("X-Tenant-Id", tenantID)
	resp, err := gitOAuthHTTP.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		return false, fmt.Errorf("git-oauth status %d", resp.StatusCode)
	}
	var out struct {
		Connected bool `json:"connected"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return false, err
	}
	return out.Connected, nil
}

// isHTTPGitRepoURL reports whether repoURL is an HTTP(S) remote that requires
// Git OAuth binding (SSH remotes and local paths do not).
func isHTTPGitRepoURL(repoURL string) bool {
	l := strings.ToLower(strings.TrimSpace(repoURL))
	return strings.HasPrefix(l, "https://") || strings.HasPrefix(l, "http://")
}
