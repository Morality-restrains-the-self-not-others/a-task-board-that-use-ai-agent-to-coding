package application

import (
	"net/url"
	"strings"
)

// cloneProviderSite maps a repo URL to the gitOauth exchange site (host).
// OPT-20260827-036: token exchange always goes through /api/internal/gitsite/{site}/oauth/access-for-user/
// — the YAML provider-key fork and "gitsite:" sentinel are removed. gitOauth owns
// site→provider resolution (YAML website / tenant Path A rows), including github.com.
func cloneProviderSite(repoURL string) string {
	return cloneRepoHost(repoURL)
}

// cloneCredentialProvider returns github|gitlab for clone HTTP Basic username
// and the repo_clone_credentials.provider field. YAML-resolved key wins when
// present (GitHub Enterprise stays "github"); otherwise derive from host.
func cloneCredentialProvider(resolvedKey, repoURL string) string {
	if key := strings.TrimSpace(resolvedKey); key != "" {
		if p := strings.ToLower(strings.SplitN(key, ":", 2)[0]); p == "github" {
			return "github"
		}
		return "gitlab"
	}
	if strings.EqualFold(strings.TrimSpace(cloneRepoHost(repoURL)), "github.com") {
		return "github"
	}
	return "gitlab"
}

func cloneRepoHost(repoURL string) string {
	raw := strings.TrimSpace(repoURL)
	if raw == "" {
		return ""
	}
	if !strings.Contains(raw, "://") {
		if at := strings.Index(raw, "@"); at >= 0 {
			rest := raw[at+1:]
			if colon := strings.Index(rest, ":"); colon >= 0 {
				h := strings.ToLower(strings.TrimSpace(rest[:colon]))
				if h != "" && !strings.Contains(h, "/") {
					return h
				}
			}
		}
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(u.Hostname()))
}
