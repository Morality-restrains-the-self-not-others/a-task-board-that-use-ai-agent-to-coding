package main

import (
	"net/url"
	"regexp"
	"strings"
)

var gitSCPRe = regexp.MustCompile(`(?i)^git@([^:]+):(.+?)(?:\.git)?/?$`)

// repoMatchKeyFromURL builds host/path (lowercase, no .git), aligned with
// Django projects.services.repo_match_key.repo_match_key_from_url and
// onlineServiceJS repoMatchKeyFromUrl.
func repoMatchKeyFromURL(raw string) string {
	u := strings.TrimSpace(raw)
	if u == "" {
		return ""
	}
	if m := gitSCPRe.FindStringSubmatch(u); m != nil {
		host := strings.ToLower(m[1])
		path := strings.ReplaceAll(m[2], "\\", "/")
		path = strings.TrimRight(path, "/")
		if strings.HasSuffix(strings.ToLower(path), ".git") {
			path = path[:len(path)-4]
		}
		return strings.ToLower(host + "/" + path)
	}
	parsed, err := url.Parse(u)
	if err == nil {
		netloc := strings.ToLower(strings.TrimSpace(parsed.Host))
		if strings.EqualFold(parsed.Scheme, "ssh") {
			netloc = strings.ToLower(strings.TrimSpace(parsed.Hostname()))
		}
		path := strings.TrimRight(parsed.Path, "/")
		if strings.HasSuffix(strings.ToLower(path), ".git") {
			path = path[:len(path)-4]
		}
		path = strings.TrimPrefix(path, "/")
		if netloc != "" {
			return strings.ToLower(netloc + "/" + path)
		}
	}
	normalized := strings.ToLower(strings.TrimRight(u, "/"))
	if strings.HasSuffix(normalized, ".git") {
		normalized = normalized[:len(normalized)-4]
	}
	return normalized
}
