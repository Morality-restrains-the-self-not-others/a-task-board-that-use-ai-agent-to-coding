package main

import (
	"net/url"
	"regexp"
	"strings"
)

var gitScpURLPattern = regexp.MustCompile(`(?i)^git@([^:]+):(.+?)(?:\.git)?/?$`)

// normalizeGitRepoURLForBranchLookup converts SSH remotes (git@host:path and
// ssh://[user@]host[:port]/path) to HTTP(S) so OAuth-backed provider APIs can
// list branches. http(s) URLs are returned unchanged.
func normalizeGitRepoURLForBranchLookup(repoURL string) (string, bool) {
	raw := strings.TrimSpace(repoURL)
	if raw == "" {
		return "", false
	}

	host, path, ok := parseSSHGitRemote(raw)
	if !ok {
		if strings.HasPrefix(strings.ToLower(raw), "ssh://") {
			return "", false
		}
		if strings.HasPrefix(raw, "git@") {
			return "", false
		}
		return raw, true
	}
	return httpURLForGitHostPath(host, path)
}

func parseSSHGitRemote(raw string) (host, path string, ok bool) {
	lower := strings.ToLower(raw)
	if strings.HasPrefix(lower, "ssh://") {
		u, err := url.Parse(raw)
		if err != nil {
			return "", "", false
		}
		host = strings.ToLower(strings.TrimSpace(u.Hostname()))
		path = strings.Trim(strings.TrimSpace(u.Path), "/")
		path = strings.TrimSuffix(path, ".git")
		if host == "" || path == "" {
			return "", "", false
		}
		return host, path, true
	}
	if !strings.HasPrefix(raw, "git@") {
		return "", "", false
	}
	m := gitScpURLPattern.FindStringSubmatch(raw)
	if len(m) < 3 {
		return "", "", false
	}
	host = strings.ToLower(strings.TrimSpace(m[1]))
	path = strings.Trim(strings.TrimSpace(m[2]), "/")
	if host == "" || path == "" {
		return "", "", false
	}
	return host, path, true
}

func httpURLForGitHostPath(host, path string) (string, bool) {
	path = strings.Trim(strings.TrimSuffix(strings.TrimSpace(path), ".git"), "/")
	if host == "" || path == "" {
		return "", false
	}
	switch host {
	case "github.com":
		return "https://github.com/" + path, true
	case "gitlab.com":
		return "https://gitlab.com/" + path, true
	case "bitbucket.org":
		return "https://bitbucket.org/" + path, true
	}
	if origin := resolveWebsiteOriginForHost(host); origin != "" {
		return origin + "/" + path, true
	}
	return "https://" + host + "/" + path, true
}

func isSSHGitRemote(raw string) bool {
	s := strings.ToLower(strings.TrimSpace(raw))
	return strings.HasPrefix(s, "git@") || strings.HasPrefix(s, "ssh://")
}

func resolveWebsiteOriginForHost(host string) string {
	if providerResolver == nil {
		return ""
	}
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" {
		return ""
	}
	for _, entry := range providerResolver.entries {
		if entry.Host == host && entry.WebsiteOrigin != "" {
			return entry.WebsiteOrigin
		}
	}
	return ""
}
