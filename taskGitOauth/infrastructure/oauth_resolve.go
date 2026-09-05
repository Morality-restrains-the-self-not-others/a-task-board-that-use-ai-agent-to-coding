package infrastructure

import (
	"strings"

	"taskGitOauth/domain"
)

// ResolveGitLabAuthorizeContext mirrors Python _resolve_start_authorize_context.
// Optional tenantPC is used when YAML has no match for tenant-{company_id} service providers.
func ResolveGitLabAuthorizeContext(cfg *Config, serviceProvider, allowedHost string, tenantPC ...*ProviderConfig) (map[string]string, string) {
	preferred := strings.TrimSpace(strings.ToLower(serviceProvider))
	if preferred == "" {
		preferred = "default"
	}
	var pc *ProviderConfig
	if allowedHost != "" {
		pc = cfg.ResolveProviderConfig("gitlab", allowedHost)
	}
	if pc == nil {
		pc, _ = cfg.ResolveByServiceProvider(preferred)
	}
	if pc == nil && len(tenantPC) > 0 && tenantPC[0] != nil {
		pc = tenantPC[0]
	}
	if pc == nil {
		// try first gitlab config when preferred is default
		rows := cfg.GetProviderConfigs("gitlab")
		if preferred == "default" && len(rows) == 1 {
			host := hostnameOf(allowedHost)
			if host == "" || hostnameOf(rows[0].Website) == host {
				cp := rows[0]
				pc = &cp
			}
		}
	}
	if pc == nil || strings.TrimSpace(pc.Website) == "" {
		return nil, "missing_allowed_host"
	}
	repoURL := strings.TrimSpace(pc.Website)
	matched := matchGitLabRoute(cfg, repoURL, preferred)
	if matched == nil && len(tenantPC) > 0 && tenantPC[0] != nil {
		matched = tenantPC[0]
	}
	if matched == nil {
		// tenant-only provider: use pc directly when it matches preferred SP
		if pc.ServiceProvider == preferred || strings.HasPrefix(preferred, "tenant-") {
			matched = pc
		}
	}
	if matched == nil {
		return nil, "no_matching_provider_route"
	}
	origin := strings.TrimRight(matched.AuthorizeOrigin, "/")
	if origin == "" {
		origin = OriginFromURL(matched.Website)
	}
	cid := strings.TrimSpace(matched.ClientID)
	redirect := strings.TrimSpace(matched.RedirectURI)
	scope := strings.TrimSpace(matched.Scope)
	if scope == "" {
		scope = domain.DefaultTenantGitLabScope
	}
	if cid == "" || redirect == "" || origin == "" {
		return nil, "missing_route_params"
	}
	return map[string]string{
		"service_provider": matched.ServiceProvider,
		"origin":           origin,
		"client_id":        cid,
		"redirect_uri":     redirect,
		"scope":            scope,
	}, ""
}

func matchGitLabRoute(cfg *Config, repoURL, preferred string) *ProviderConfig {
	origin := OriginFromURL(repoURL)
	host := hostnameOf(origin)
	rows := cfg.GetProviderConfigs("gitlab")
	var byPreferred *ProviderConfig
	var byOrigin *ProviderConfig
	for i := range rows {
		row := &rows[i]
		matches := false
		for _, o := range row.MatchOrigins {
			if hostnameOf(o) == host || strings.EqualFold(strings.TrimRight(o, "/"), origin) {
				matches = true
				break
			}
		}
		if !matches && hostnameOf(row.Website) == host {
			matches = true
		}
		if !matches {
			continue
		}
		cp := *row
		if preferred != "" && row.ServiceProvider == preferred {
			byPreferred = &cp
		}
		if byOrigin == nil {
			byOrigin = &cp
		}
	}
	if byPreferred != nil {
		return byPreferred
	}
	return byOrigin
}

func ProviderConfigByKey(cfg *Config, providerKey string) *ProviderConfig {
	p, sp := splitProviderKey(providerKey, "github")
	rows := cfg.GetProviderConfigs(p)
	for i := range rows {
		if rows[i].ServiceProvider == sp || rows[i].ProviderKey == providerKey {
			cp := rows[i]
			return &cp
		}
	}
	if p == "github" && len(rows) > 0 {
		cp := rows[0]
		return &cp
	}
	return nil
}

// AllowGitCredentialPrefixFallback reports whether access-for-user may reuse a
// credential stored under a different provider_key that shares the same prefix.
// GitHub keeps the alias behavior (one github.com). GitLab is multi-instance:
// only the same website host may share a token (otherwise API 401 on the wrong CE).
func AllowGitCredentialPrefixFallback(cfg *Config, requestedKey, storedKey string) bool {
	reqP, _ := splitProviderKey(requestedKey, "github")
	if reqP != "gitlab" {
		return true
	}
	reqHost := GitLabProviderWebsiteHost(cfg, requestedKey)
	storedHost := GitLabProviderWebsiteHost(cfg, storedKey)
	if reqHost == "" || storedHost == "" {
		return false
	}
	return reqHost == storedHost
}

func GitLabProviderWebsiteHost(cfg *Config, providerKey string) string {
	pc := ProviderConfigByKey(cfg, providerKey)
	if pc == nil {
		return ""
	}
	return hostnameOf(pc.Website)
}

func splitProviderKey(providerKey, fallback string) (string, string) {
	raw := strings.TrimSpace(strings.ToLower(providerKey))
	fb := strings.TrimSpace(strings.ToLower(fallback))
	if fb == "" {
		fb = "github"
	}
	if i := strings.Index(raw, ":"); i >= 0 {
		p := strings.TrimSpace(raw[:i])
		sp := strings.TrimSpace(raw[i+1:])
		if p == "" {
			p = fb
		}
		if sp == "" {
			sp = "default"
		}
		return p, sp
	}
	if raw == "" {
		return fb, "default"
	}
	return raw, "default"
}

func OAuthCallbackPath(serviceProvider string) string {
	sp := strings.TrimSpace(strings.ToLower(serviceProvider))
	if sp == "" {
		sp = "default"
	}
	return "/api/accounts/" + sp + "/oauth/callback/"
}

func ResolveOAuthRedirectURI(primary, serviceProvider, frontendBase string, publicOrigins []string) string {
	// SSOT provider YAML target.redirect_uri 为权威值：v2 契约
	// ${scheme}://${subdomains.base}/redirect/gitsite/<gitsite>/oauth/callback/。
	// 其余候选（publicOrigins/frontendBase + 旧 /api/accounts/<sp>/... 路径）仅作
	// 未配置 redirect_uri 时的兼容回退（历史 primary 为空场景）。
	if p := strings.TrimSpace(primary); p != "" {
		return p
	}
	path := OAuthCallbackPath(serviceProvider)
	allowed := []string{}
	seen := map[string]bool{}
	add := func(u string) {
		u = strings.TrimSpace(u)
		if u == "" || seen[u] {
			return
		}
		seen[u] = true
		allowed = append(allowed, u)
	}
	for _, o := range publicOrigins {
		add(strings.TrimRight(o, "/") + path)
	}
	feb := strings.TrimRight(strings.TrimSpace(frontendBase), "/")
	if feb != "" {
		candidate := feb + path
		if seen[candidate] {
			return candidate
		}
	}
	if len(allowed) > 0 {
		return allowed[0]
	}
	return ""
}
