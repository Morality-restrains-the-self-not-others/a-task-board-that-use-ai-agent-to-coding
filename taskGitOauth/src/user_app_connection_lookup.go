package main

import (
	"net/http"
	"net/url"
	"strings"

	"taskGitOauth/domain"
	"taskGitOauth/infrastructure"
)

// userAppConnectionLookupKeys returns the credential lookup keys for the GET
// status check. When repo_url matches a configured website/alias, only that
// instance's keys are used. Otherwise the caller-resolved provider:sp plus
// every configured storage key for that provider (githubStoredProviderKey
// mirrors the GitHub callback storage; gitlab covers all configured gitlab
// rows so localhost vs 127.0.0.1 still finds a bind). DELETE keeps the exact
// caller-resolved key only.
func (a *App) userAppConnectionLookupKeys(providerKey, repoURL string) []string {
	if pc := a.providerConfigForRepoURL(repoURL); pc != nil {
		return uniqueNonEmptyStrings([]string{
			strings.ToLower(strings.TrimSpace(pc.ProviderKey)),
			domain.ProviderKey(pc.Provider, pc.ServiceProvider),
		})
	}
	keys := []string{providerKey}
	provider, _ := domainParseProviderKey(providerKey, "github")
	switch provider {
	case "github":
		keys = append(keys, a.githubStoredProviderKeys()...)
	case "gitlab":
		for _, row := range a.Cfg.GetProviderConfigs("gitlab") {
			if k := strings.TrimSpace(strings.ToLower(row.ProviderKey)); k != "" {
				keys = append(keys, k)
			}
			keys = append(keys, domain.ProviderKey(gitlabProvider, row.ServiceProvider))
		}
	}
	return uniqueNonEmptyStrings(keys)
}

func uniqueNonEmptyStrings(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

// resolveUserAppConnectionProvider returns the provider part of the credential key:
// query `provider` → service_provider config → repo_url host match → "github" default.
func (a *App) resolveUserAppConnectionProvider(r *http.Request, sp string) string {
	provider := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("provider")))
	if provider != "" {
		return provider
	}
	if pc, err := a.resolveProviderByServiceProvider(sp); err == nil && pc != nil {
		return pc.Provider
	}
	if repoURL := strings.TrimSpace(r.URL.Query().Get("repo_url")); repoURL != "" {
		if p := a.providerForRepoURL(repoURL); p != "" {
			return p
		}
	}
	return "github"
}

// providerForRepoURL matches the repo URL host against configured provider websites /
// match origins (github.com → github, gitlab.com → gitlab, self-hosted → configured).
func (a *App) providerForRepoURL(repoURL string) string {
	if pc := a.providerConfigForRepoURL(repoURL); pc != nil {
		return pc.Provider
	}
	return ""
}

func (a *App) providerConfigForRepoURL(repoURL string) *infrastructure.ProviderConfig {
	host := hostnameOfRaw(repoURL)
	if host == "" {
		return nil
	}
	if pc, err := a.Cfg.ResolveProviderByGitsiteWithDB(a.DB, a.Fernet, host); err == nil && pc != nil {
		return pc
	}
	for _, pc := range a.Cfg.GetAllProviderConfigs() {
		if hostnameOfRaw(pc.Website) == host {
			cp := pc
			return &cp
		}
		for _, o := range pc.MatchOrigins {
			if hostnameOfRaw(o) == host {
				cp := pc
				return &cp
			}
		}
		for _, o := range pc.WebsiteAliases {
			if hostnameOfRaw(o) == host {
				cp := pc
				return &cp
			}
		}
	}
	return nil
}

// hostnameOfRaw extracts the lowercased hostname of a URL; empty when unparseable.
func hostnameOfRaw(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return ""
	}
	return strings.ToLower(u.Hostname())
}
