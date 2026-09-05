package main

import (
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"confload"
)

// gitOauthProviderEntry is a flattened provider record from YAML configs.
type gitOauthProviderEntry struct {
	Provider        string
	ServiceProvider string
	ProviderKey     string
	Host            string
	Netloc          string
}

var (
	gitOauthProviderOnce sync.Once
	gitOauthProviders    []gitOauthProviderEntry
)

// loadGitOauthProviders loads provider YAML once from monorepo conf
// (task-credential first, then django fallback).
func loadGitOauthProviders() []gitOauthProviderEntry {
	gitOauthProviderOnce.Do(func() {
		root, err := findMonorepoRoot()
		if err != nil || strings.TrimSpace(root) == "" {
			return
		}
		candidates := []string{
			filepath.Join(root, "conf", "auth", "task-credential", "git-oauth-providers"),
			filepath.Join(root, "conf", "core", "django", "git-oauth-providers"),
		}
		var dir string
		for _, c := range candidates {
			if st, err := os.Stat(c); err == nil && st.IsDir() {
				dir = c
				break
			}
		}
		if dir == "" {
			return
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		subs := confload.ResolveBaseYaml(root)
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
				continue
			}
			path := filepath.Join(dir, entry.Name())
			prov, serviceProvider, websites := parseGitOauthProviderYAML(path)
			if prov == "" || len(websites) == 0 {
				continue
			}
			if serviceProvider == "" {
				serviceProvider = "default"
			}
			providerKey := prov + ":" + serviceProvider
			for _, website := range websites {
				website = strings.TrimSpace(confload.ResolveTemplate(website, subs))
				host, netloc := extractProviderHostNetloc(website)
				if host == "" {
					log.Printf("[taskCloudService] skip unresolved git oauth website provider_key=%s website=%s file=%s",
						providerKey, website, entry.Name())
					continue
				}
				gitOauthProviders = append(gitOauthProviders, gitOauthProviderEntry{
					Provider:        prov,
					ServiceProvider: serviceProvider,
					ProviderKey:     providerKey,
					Host:            host,
					Netloc:          netloc,
				})
			}
		}
	})
	return gitOauthProviders
}

// parseGitOauthProviderYAML extracts provider, service_provider and websites
// (mirrors taskCredentialService parseProviderYAML).
func parseGitOauthProviderYAML(path string) (provider, serviceProvider string, websites []string) {
	data, err := confload.ReadOverlaidYAML(path)
	if err != nil {
		return "", "", nil
	}
	lines := strings.Split(string(data), "\n")
	inTarget := false
	website := ""
	var aliases []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if !inTarget {
			if strings.HasPrefix(trimmed, "provider:") {
				provider = strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmed, "provider:")), "'\"")
				provider = strings.ToLower(provider)
			}
			if strings.HasPrefix(trimmed, "service_provider:") {
				serviceProvider = strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmed, "service_provider:")), "'\"")
				serviceProvider = strings.ToLower(serviceProvider)
			}
			if trimmed == "target:" {
				inTarget = true
			}
			continue
		}
		if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			inTarget = false
			continue
		}
		if strings.HasPrefix(trimmed, "website:") {
			website = strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmed, "website:")), "'\"")
			continue
		}
		if strings.HasPrefix(trimmed, "- ") && strings.Contains(trimmed, "://") {
			alias := strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmed, "- ")), "'\"")
			if alias != "" {
				aliases = append(aliases, alias)
			}
		}
	}
	if serviceProvider == "" {
		serviceProvider = "default"
	}
	if website != "" {
		websites = append(websites, website)
	}
	for _, alias := range aliases {
		found := false
		for _, existing := range websites {
			if existing == alias {
				found = true
				break
			}
		}
		if !found {
			websites = append(websites, alias)
		}
	}
	return provider, serviceProvider, websites
}

func extractProviderHostNetloc(raw string) (host, netloc string) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", ""
	}
	if !strings.Contains(s, "://") {
		if at := strings.Index(s, "@"); at >= 0 {
			rest := s[at+1:]
			if colon := strings.Index(rest, ":"); colon >= 0 {
				h := strings.ToLower(strings.TrimSpace(rest[:colon]))
				if h != "" && !strings.Contains(h, "/") {
					return h, h
				}
			}
		}
		s = "https://" + s
	}
	u, err := url.Parse(s)
	if err != nil {
		return "", ""
	}
	h := strings.ToLower(strings.TrimSpace(u.Hostname()))
	n := strings.ToLower(strings.TrimSpace(u.Host))
	return h, n
}

// defaultGithubProviderKey returns the first configured github provider key,
// or "github:github-official" when YAML is unavailable / empty.
func defaultGithubProviderKey() string {
	for _, e := range loadGitOauthProviders() {
		if e.Provider == "github" {
			return e.ProviderKey
		}
	}
	return "github:github-official"
}

// resolveProviderKeyFromRepoURL matches a repo URL against loaded provider
// websites; falls back to github-official / gitlab:default heuristics.
func resolveProviderKeyFromRepoURL(repoURL string) string {
	host, netloc := extractProviderHostNetloc(repoURL)
	if host == "" {
		return ""
	}
	var hostFallback string
	for _, e := range loadGitOauthProviders() {
		if e.Netloc != "" && netloc != "" && e.Netloc == netloc {
			return e.ProviderKey
		}
		if e.Host == host && hostFallback == "" {
			hostFallback = e.ProviderKey
		}
	}
	if hostFallback != "" {
		return hostFallback
	}
	if host == "github.com" {
		return defaultGithubProviderKey()
	}
	// Unmatched GitLab hosts must stay empty. Collapsing to gitlab:default lets
	// gitlabProviderKeysToTry borrow another CE's token (ADR-0014).
	return ""
}
