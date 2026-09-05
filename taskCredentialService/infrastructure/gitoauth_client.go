package infrastructure

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// GitoauthHTTPClient implements ports.GitoauthClient via HTTP calls to gitOauth.
type GitoauthHTTPClient struct {
	baseURL      string
	bridgeSecret string
	client       *http.Client
	resolver     *ProviderResolver
}

// NewGitoauthHTTPClient creates a gitOauth HTTP client with timeout and no proxy.
func NewGitoauthHTTPClient(baseURL string, timeoutSeconds int, resolver *ProviderResolver, bridgeSecret string) *GitoauthHTTPClient {
	return &GitoauthHTTPClient{
		baseURL:      baseURL,
		bridgeSecret: strings.TrimSpace(bridgeSecret),
		resolver:     resolver,
		client: &http.Client{
			Timeout: time.Duration(timeoutSeconds) * time.Second,
			Transport: &http.Transport{
				Proxy: nil, // trust_env=false equivalent — no system proxy
			},
		},
	}
}

// ResolveProvider identifies the full provider key (e.g. "gitlab:synology-gitlab")
// from a repo URL. Delegates to ProviderResolver which matches against configured
// provider websites.
func (c *GitoauthHTTPClient) ResolveProvider(repoURL string) string {
	if c.resolver != nil {
		return c.resolver.ResolveProvider(repoURL)
	}
	// Fallback when no resolver configured: basic substring checks only.
	s := strings.ToLower(repoURL)
	if strings.Contains(s, "github.com") {
		return "github:github-official"
	}
	return ""
}

// ResolveHttpsCloneURL converts SSH/SCP repo URLs to HTTP(S) for OAuth clone.
func (c *GitoauthHTTPClient) ResolveHttpsCloneURL(repoURL string) string {
	if c.resolver != nil {
		return c.resolver.ResolveHttpsCloneURL(repoURL)
	}
	return (&ProviderResolver{}).ResolveHttpsCloneURL(repoURL)
}

// FetchAccessToken calls gitOauth to exchange a user binding for an ephemeral token.
// site is the Git site host from the repo URL (e.g. "github.com" or "115.29.110.74").
// Unified path (OPT-20260827-036): always POST /api/internal/gitsite/{site}/oauth/access-for-user/
// so gitOauth resolves the site to a provider via YAML website / tenant Path A rows.
// The old YAML key fork (/api/internal/{provider}/oauth/access-for-user/ + provider_key)
// is removed — token exchange never branches on a YAML provider key anymore.
func (c *GitoauthHTTPClient) FetchAccessToken(userID int64, site string) (string, error) {
	site = strings.ToLower(strings.TrimSpace(site))
	if site == "" {
		return "", fmt.Errorf("empty gitsite")
	}
	body := map[string]interface{}{
		"user_id": userID,
	}
	endpoint := fmt.Sprintf("%s/api/internal/gitsite/%s/oauth/access-for-user/", c.baseURL, url.PathEscape(site))
	bodyJSON, _ := json.Marshal(body)

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(bodyJSON))
	if err != nil {
		return "", fmt.Errorf("build gitoauth request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.bridgeSecret != "" {
		req.Header.Set("X-GitOauth-Bridge-Secret", c.bridgeSecret)
	}

	start := time.Now()
	log.Printf("[task-credential-service] gitoauth request: site=%s user=%d", site, userID)

	resp, err := c.client.Do(req)
	duration := time.Since(start).Milliseconds()
	if err != nil {
		log.Printf("[task-credential-service] WARN gitoauth request failed: site=%s user=%d duration=%dms err=%v",
			site, userID, duration, err)
		return "", fmt.Errorf("gitoauth request: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		AccessToken  string `json:"access_token"`
		ExpiresIn    int    `json:"expires_in"`
		GithubUserID string `json:"github_user_id"`
		Detail       string `json:"detail"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("[task-credential-service] WARN gitoauth response decode failed: status=%d duration=%dms err=%v",
			resp.StatusCode, duration, err)
		return "", fmt.Errorf("decode gitoauth response: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		log.Printf("[task-credential-service] WARN gitoauth user not bound: site=%s user=%d status=%d duration=%dms",
			site, userID, resp.StatusCode, duration)
		return "", fmt.Errorf("user %d not bound to %s", userID, site)
	}
	if resp.StatusCode >= 400 {
		log.Printf("[task-credential-service] WARN gitoauth upstream error: site=%s user=%d status=%d detail=%s duration=%dms",
			site, userID, resp.StatusCode, result.Detail, duration)
		return "", fmt.Errorf("gitoauth returned %d: %s", resp.StatusCode, result.Detail)
	}

	log.Printf("[task-credential-service] gitoauth token fetched: site=%s user=%d expires_in=%d status=%d duration=%dms",
		site, userID, result.ExpiresIn, resp.StatusCode, duration)
	return result.AccessToken, nil
}
