package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"confload"
)

type providerEntry struct {
	Provider        string
	ServiceProvider string
	ProviderKey     string
	Host            string
	Netloc          string
	WebsiteOrigin   string
	GitoauthBase    string
}

type providerMatch struct {
	ProviderKey  string
	GitoauthBase string
}

type ProviderResolver struct {
	entries []providerEntry
}

func loadProviderConfigs(repoRoot string) (*ProviderResolver, error) {
	dir := filepath.Join(repoRoot, "conf", "auth", "task-credential", "git-oauth-providers")
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read provider configs dir %s: %w", dir, err)
	}

	subs := confload.ResolveBaseYaml(repoRoot)
	var resolver ProviderResolver
	for _, entry := range files {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		record := parseProviderYAML(path, subs)
		if record.Provider == "" || len(record.Websites) == 0 {
			continue
		}
		providerKey := record.Provider + ":" + record.ServiceProvider
		gitoauthBase := strings.TrimRight(strings.TrimSpace(record.GitoauthBase), "/")
		if gitoauthBase == "" {
			gitoauthBase = strings.TrimRight(strings.TrimSpace(cfg.GitoauthBaseURL), "/")
		}
		for _, website := range record.Websites {
			if strings.Contains(website, "${") {
				log.Printf("[taskProjectService] WARN skip unresolved website template key=%s website=%s", providerKey, website)
				continue
			}
			host, netloc := extractHostNetloc(website)
			resolver.entries = append(resolver.entries, providerEntry{
				Provider:        record.Provider,
				ServiceProvider: record.ServiceProvider,
				ProviderKey:     providerKey,
				Host:            host,
				Netloc:          netloc,
				WebsiteOrigin:   strings.TrimRight(strings.TrimSpace(website), "/"),
				GitoauthBase:    gitoauthBase,
			})
			log.Printf("[taskProjectService] provider config loaded: key=%s website=%s gitoauth=%s", providerKey, website, gitoauthBase)
		}
	}
	log.Printf("[taskProjectService] provider configs loaded: %d entries", len(resolver.entries))
	return &resolver, nil
}

type parsedProviderYAML struct {
	Provider        string
	ServiceProvider string
	Websites        []string
	GitoauthBase    string
}

func parseProviderYAML(path string, subs map[string]string) parsedProviderYAML {
	data, err := confload.ReadOverlaidYAML(path)
	if err != nil {
		log.Printf("[taskProjectService] WARN skip provider config %s: %v", filepath.Base(path), err)
		return parsedProviderYAML{}
	}

	var out parsedProviderYAML
	section := ""
	website := ""
	var aliases []string

	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			switch trimmed {
			case "target:", "service:":
				section = strings.TrimSuffix(trimmed, ":")
			default:
				if strings.HasPrefix(trimmed, "provider:") {
					out.Provider = strings.ToLower(strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmed, "provider:")), "'\""))
				}
				if strings.HasPrefix(trimmed, "service_provider:") {
					out.ServiceProvider = strings.ToLower(strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmed, "service_provider:")), "'\""))
				}
			}
			continue
		}

		switch section {
		case "target":
			if strings.HasPrefix(trimmed, "website:") && !strings.HasPrefix(trimmed, "website_aliases:") {
				website = strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmed, "website:")), "'\"")
				continue
			}
			if strings.HasPrefix(trimmed, "- ") && strings.Contains(trimmed, "://") {
				alias := strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmed, "- ")), "'\"")
				if alias != "" {
					aliases = append(aliases, alias)
				}
			}
		case "service":
			// gitoauth_base 显式内网地址（如 http://127.0.0.1:8002）优先；
			// allowedHost 仅作兼容兜底（历史文件无 gitoauth_base 字段）。
			// 误用 allowedHost（公网域名）会让 access-for-user 走公网被网关
			// deny-internal 拦截（token_error 根因，OPT-20260807-071 复盘）。
			if strings.HasPrefix(trimmed, "gitoauth_base:") {
				out.GitoauthBase = strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmed, "gitoauth_base:")), "'\"")
			} else if strings.HasPrefix(trimmed, "allowedHost:") && out.GitoauthBase == "" {
				out.GitoauthBase = strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmed, "allowedHost:")), "'\"")
			}
		}
	}

	if out.ServiceProvider == "" {
		out.ServiceProvider = "default"
	}
	if website != "" {
		out.Websites = append(out.Websites, confload.ResolveTemplate(website, subs))
	}
	for _, alias := range aliases {
		resolved := confload.ResolveTemplate(alias, subs)
		dup := false
		for _, existing := range out.Websites {
			if existing == resolved {
				dup = true
				break
			}
		}
		if !dup {
			out.Websites = append(out.Websites, resolved)
		}
	}
	out.GitoauthBase = confload.ResolveTemplate(out.GitoauthBase, subs)
	return out
}

func (r *ProviderResolver) matchProvider(repoURL string) providerMatch {
	if r == nil {
		return providerMatch{ProviderKey: resolveProviderFallback(repoURL), GitoauthBase: cfg.GitoauthBaseURL}
	}
	host, netloc := extractHostNetloc(repoURL)
	if host == "" {
		return providerMatch{}
	}

	var hostFallback *providerEntry
	for i := range r.entries {
		e := &r.entries[i]
		if e.Netloc != "" && netloc != "" && e.Netloc == netloc {
			return providerMatch{ProviderKey: e.ProviderKey, GitoauthBase: e.GitoauthBase}
		}
		if e.Host == host && hostFallback == nil {
			hostFallback = e
		}
	}
	if hostFallback != nil {
		return providerMatch{ProviderKey: hostFallback.ProviderKey, GitoauthBase: hostFallback.GitoauthBase}
	}

	if host == "github.com" {
		return providerMatch{ProviderKey: defaultGithubProviderKey(r), GitoauthBase: cfg.GitoauthBaseURL}
	}
	if strings.HasSuffix(host, "gitlab.com") {
		return providerMatch{ProviderKey: "gitlab:default", GitoauthBase: cfg.GitoauthBaseURL}
	}
	// Unmatched self-hosted hosts (regional gitlab-*, Path A IPs, unknown *.git)
	// must not borrow gitlab:default — that token belongs to another CE and
	// surfaces as「未检测到可用授权」/ 401. Callers attach tenant Path A via
	// matchRepoProvider when company_id is known.
	if strings.Contains(host, "gitlab") || strings.HasSuffix(strings.ToLower(strings.TrimSpace(repoURL)), ".git") {
		return providerMatch{GitoauthBase: cfg.GitoauthBaseURL}
	}
	return providerMatch{}
}

func (r *ProviderResolver) ResolveProvider(repoURL string) string {
	return r.matchProvider(repoURL).ProviderKey
}

func (r *ProviderResolver) IsGitLabRepo(repoURL string) bool {
	key := r.ResolveProvider(repoURL)
	// github.com/.../*.git must NOT be treated as GitLab; the trailing ".git"
	// heuristic is only for unknown hosts (self-hosted remotes without "gitlab" in hostname).
	if strings.HasPrefix(key, "github:") {
		return false
	}
	if strings.HasPrefix(key, "gitlab:") {
		return true
	}
	host, _ := extractHostNetloc(repoURL)
	if host == "github.com" || strings.HasSuffix(host, ".github.com") {
		return false
	}
	return strings.Contains(host, "gitlab") || strings.HasSuffix(strings.ToLower(strings.TrimSpace(repoURL)), ".git")
}

func (r *ProviderResolver) IsGitHubRepo(repoURL string) bool {
	return strings.HasPrefix(r.ResolveProvider(repoURL), "github:")
}

func defaultGithubProviderKey(r *ProviderResolver) string {
	if r != nil {
		for _, e := range r.entries {
			if e.Provider == "github" {
				return e.ProviderKey
			}
		}
	}
	return "github:github-official"
}

func resolveProviderFallback(repoURL string) string {
	host, _ := extractHostNetloc(repoURL)
	if host == "github.com" {
		return "github:github-official"
	}
	if host == "gitlab.com" || strings.HasSuffix(host, ".gitlab.com") {
		return "gitlab:default"
	}
	// Regional / self-hosted hosts that contain "gitlab" must not borrow the
	// default instance token (cross-CE OAuth → GitLab API 401).
	if strings.Contains(host, "gitlab") {
		return ""
	}
	if strings.HasSuffix(strings.ToLower(strings.TrimSpace(repoURL)), ".git") {
		return ""
	}
	return ""
}

func extractHostNetloc(raw string) (host, netloc string) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", ""
	}
	if strings.HasPrefix(s, "git@") {
		if m := gitScpURLPattern.FindStringSubmatch(s); len(m) >= 2 {
			h := strings.ToLower(strings.TrimSpace(m[1]))
			return h, h
		}
		return "", ""
	}
	if !strings.Contains(s, "://") {
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
