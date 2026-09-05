// Package infrastructure — provider config loader for git provider resolution.
// Reads conf/auth/git-oauth/providers/*.yaml at startup to match self-hosted
// GitLab instances against configured provider websites (same source as Python's
// _infer_provider_from_configured_allowed_hosts).
//
// Uses a minimal YAML parser (no external deps) — the provider config files have
// a flat structure: provider, service_provider, target { website, ... }, service { ... }.
package infrastructure

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"confload"
)

// providerEntry is a flattened, searchable provider record.
type providerEntry struct {
	Provider        string // "github" or "gitlab"
	ServiceProvider string // e.g. "synology-gitlab" or "default"
	ProviderKey     string // e.g. "gitlab:synology-gitlab"
	Website         string // e.g. "http://<gitlab-host>:8012"
	Host            string // e.g. "<gitlab-host>"
	Netloc          string // e.g. "<gitlab-host>:8012"
}

// ProviderResolver resolves git providers from repo URLs using configured provider files.
type ProviderResolver struct {
	entries []providerEntry
}

// LoadProviderConfigs reads all provider YAML files from the sync-generated
// conf/auth/task-credential/git-oauth-providers/ directory (copied from git-oauth by sync.sh).
func LoadProviderConfigs(monorepoRoot string) (*ProviderResolver, error) {
	dir := filepath.Join(monorepoRoot, "conf", "auth", "task-credential", "git-oauth-providers")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read provider configs dir %s: %w", dir, err)
	}

	subs := confload.ResolveBaseYaml(monorepoRoot)
	var resolver ProviderResolver
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		prov, serviceProvider, websites := parseProviderYAML(path)
		if prov == "" || len(websites) == 0 {
			continue
		}
		for _, website := range websites {
			website = strings.TrimSpace(confload.ResolveTemplate(website, subs))
			host, netloc := extractHostNetloc(website)
			if host == "" {
				log.Printf("[task-credential-service] WARN skip provider with unresolved website provider=%s service_provider=%s website=%s file=%s",
					prov, serviceProvider, website, entry.Name())
				continue
			}
			providerKey := prov + ":" + serviceProvider
			resolver.entries = append(resolver.entries, providerEntry{
				Provider:        prov,
				ServiceProvider: serviceProvider,
				ProviderKey:     providerKey,
				Website:         strings.TrimRight(website, "/"),
				Host:            host,
				Netloc:          netloc,
			})
			log.Printf("[task-credential-service] provider config loaded: provider=%s service_provider=%s provider_key=%s website=%s host=%s netloc=%s file=%s",
				prov, serviceProvider, providerKey, website, host, netloc, entry.Name())
		}
	}
	log.Printf("[task-credential-service] provider configs loaded: %d entries from %s", len(resolver.entries), dir)
	return &resolver, nil
}

// parseProviderYAML extracts provider name, service_provider and match websites from YAML.
func parseProviderYAML(path string) (provider, serviceProvider string, websites []string) {
	data, err := confload.ReadOverlaidYAML(path)
	if err != nil {
		log.Printf("[task-credential-service] WARN skip provider config %s: %v", filepath.Base(path), err)
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
				provider = strings.TrimSpace(strings.TrimPrefix(trimmed, "provider:"))
				provider = strings.Trim(provider, "'\"")
				provider = strings.ToLower(provider)
			}
			if strings.HasPrefix(trimmed, "service_provider:") {
				serviceProvider = strings.TrimSpace(strings.TrimPrefix(trimmed, "service_provider:"))
				serviceProvider = strings.Trim(serviceProvider, "'\"")
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
			website = strings.TrimSpace(strings.TrimPrefix(trimmed, "website:"))
			website = strings.Trim(website, "'\"")
			continue
		}
		if strings.HasPrefix(trimmed, "- ") && strings.Contains(trimmed, "://") {
			alias := strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))
			alias = strings.Trim(alias, "'\"")
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

// DefaultGithubProviderKey returns the first configured github provider key,
// or "github:github-official" when no github entries are loaded.
func (r *ProviderResolver) DefaultGithubProviderKey() string {
	if r != nil {
		for _, e := range r.entries {
			if e.Provider == "github" {
				return e.ProviderKey
			}
		}
	}
	return "github:github-official"
}

// ResolveHttpsCloneURL converts SSH/SCP-style repo URLs to an HTTPS (or HTTP) clone URL
// using configured provider websites. Already-HTTP(S) URLs are returned trimmed.
// Mirrors Python projects.git_utils._normalize_repo_url_for_branch_lookup.
func (r *ProviderResolver) ResolveHttpsCloneURL(repoURL string) string {
	raw := strings.TrimSpace(repoURL)
	if raw == "" {
		return ""
	}
	lower := strings.ToLower(raw)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		return strings.TrimRight(raw, "/")
	}
	path := extractRepoPath(raw)
	if path == "" {
		return ""
	}
	host, _ := extractHostNetloc(raw)
	if host == "" {
		return ""
	}
	switch host {
	case "github.com":
		return "https://github.com/" + path
	case "gitlab.com":
		return "https://gitlab.com/" + path
	case "bitbucket.org":
		return "https://bitbucket.org/" + path
	}
	if website := r.resolveWebsiteForSSHHost(host); website != "" {
		return website + "/" + path
	}
	return "https://" + host + "/" + path
}

// resolveWebsiteForSSHHost picks a configured website for an SSH host.
// Prefer a website whose hostname equals the SSH host (e.g. <gitlab-host>:8012
// over 127.0.0.1:8012 when cloning git@<gitlab-host>:...).
func (r *ProviderResolver) resolveWebsiteForSSHHost(sshHost string) string {
	if r == nil || sshHost == "" {
		return ""
	}
	var hostOnlyMatch string
	for _, e := range r.entries {
		if e.Host != sshHost && !strings.EqualFold(e.Host, sshHost) {
			// Also match when provider website host differs but SSH host equals
			// an alias entry's host — entries are per-website already.
			continue
		}
		if e.Website == "" {
			continue
		}
		wh, _ := extractHostNetloc(e.Website)
		if wh == sshHost {
			return e.Website
		}
		if hostOnlyMatch == "" {
			hostOnlyMatch = e.Website
		}
	}
	// SSH host may only appear as website host while provider Host was set from
	// a different alias (e.g. website 127.0.0.1 + alias 183.x). Scan all entries
	// for website hostname == sshHost.
	for _, e := range r.entries {
		if e.Website == "" {
			continue
		}
		wh, _ := extractHostNetloc(e.Website)
		if wh == sshHost {
			return e.Website
		}
	}
	return hostOnlyMatch
}

// extractRepoPath returns owner/group/repo path without leading slash or .git suffix.
func extractRepoPath(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	if !strings.Contains(s, "://") {
		if at := strings.Index(s, "@"); at >= 0 {
			rest := s[at+1:]
			if colon := strings.Index(rest, ":"); colon >= 0 {
				p := strings.ReplaceAll(rest[colon+1:], "\\", "/")
				p = strings.Trim(p, "/")
				p = strings.TrimSuffix(p, ".git")
				return p
			}
		}
	}
	u, err := url.Parse(s)
	if err != nil {
		return ""
	}
	p := strings.Trim(strings.ReplaceAll(u.Path, "\\", "/"), "/")
	p = strings.TrimSuffix(p, ".git")
	return p
}

// ResolveProvider identifies the git provider key (e.g. "gitlab:synology-gitlab") from a repo URL.
// Strategy:
//  1. Match against configured provider websites by host:port (incl. github.com)
//  2. github.com → DefaultGithubProviderKey
//     Unmatched GitLab / self-hosted hosts return "" — never gitlab:default
//     (that key is another CE; access-for-user refuses prefix fallback).
func (r *ProviderResolver) ResolveProvider(repoURL string) string {
	host, netloc := extractHostNetloc(repoURL)
	if host == "" {
		return ""
	}
	// Step 1: match against configured provider websites first
	// Prefer exact netloc match (host:port); fall back to host-only match
	var fallbackKey string
	if r != nil {
		for _, e := range r.entries {
			if e.Netloc != "" && netloc != "" && e.Netloc == netloc {
				return e.ProviderKey
			}
			if e.Host == host && fallbackKey == "" {
				fallbackKey = e.ProviderKey
			}
		}
	}
	if fallbackKey != "" {
		return fallbackKey
	}
	// Step 2: known public GitHub only. Unmatched GitLab hosts must not collapse
	// to gitlab:default — that key belongs to another CE and access-for-user
	// correctly refuses prefix fallback (cross-instance token → 401/404).
	if host == "github.com" {
		return r.DefaultGithubProviderKey()
	}
	return ""
}

// extractHostNetloc returns (hostname, hostname:port) from a raw URL.
// Supports https/http/ssh URLs and SCP-style git@host:path forms.
func extractHostNetloc(raw string) (host, netloc string) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", ""
	}
	// SCP-style: git@host:path/to/repo.git (colon is path separator, not port)
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
