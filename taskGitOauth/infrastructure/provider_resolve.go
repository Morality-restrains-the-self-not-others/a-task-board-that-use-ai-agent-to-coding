package infrastructure

import (
	"fmt"
	"net/url"
	"strings"

	"taskGitOauth/domain"
)

func (c *Config) GetProviderConfigs(provider string) []ProviderConfig {
	p := strings.TrimSpace(strings.ToLower(provider))
	return c.Providers[p]
}

// GetAllProviderConfigs returns all provider configs across all providers as a flat list.
func (c *Config) GetAllProviderConfigs() []ProviderConfig {
	var all []ProviderConfig
	for _, rows := range c.Providers {
		all = append(all, rows...)
	}
	return all
}

func (c *Config) ResolveByServiceProvider(sp string) (*ProviderConfig, error) {
	sp = strings.TrimSpace(strings.ToLower(sp))
	if sp == "" {
		return nil, nil
	}
	var matches []ProviderConfig
	for _, rows := range c.Providers {
		for _, row := range rows {
			if row.ServiceProvider == sp {
				matches = append(matches, row)
			}
		}
	}
	if len(matches) == 0 {
		return nil, nil
	}
	if len(matches) > 1 {
		return nil, fmt.Errorf("ambiguous")
	}
	m := matches[0]
	return &m, nil
}

// ResolveByServiceProviderWithDB checks YAML first, then git_oauth_tenant_gitlab_oauth_connections for tenant-{company_id}.
func (c *Config) ResolveByServiceProviderWithDB(db *DB, fernet *Fernet, sp string) (*ProviderConfig, error) {
	pc, err := c.ResolveByServiceProvider(sp)
	if err != nil {
		return nil, err
	}
	if pc != nil {
		return pc, nil
	}
	if db == nil {
		return nil, nil
	}
	companyID, ok := parseTenantCompanyIDFromSP(sp)
	if !ok {
		return nil, nil
	}
	row, err := db.GetTenantGitLabConnection(companyID)
	if err != nil || row == nil || !row.Active {
		return nil, err
	}
	return TenantRowToProviderConfig(row, fernet)
}

func parseTenantCompanyIDFromSP(serviceProvider string) (string, bool) {
	sp := strings.TrimSpace(serviceProvider)
	lower := strings.ToLower(sp)
	if lower == "" || lower == "tenant-gitlab" {
		return "", false
	}
	const prefix = "tenant-"
	if !strings.HasPrefix(lower, prefix) {
		return "", false
	}
	id := strings.TrimSpace(sp[len(prefix):])
	if id == "" {
		return "", false
	}
	return id, true
}

// TenantRowToProviderConfig maps a DB row to runtime ProviderConfig (decrypts secret when fernet set).
func TenantRowToProviderConfig(row *TenantGitLabOAuthConnectionRow, fernet *Fernet) (*ProviderConfig, error) {
	if row == nil {
		return nil, nil
	}
	secret := ""
	if fernet != nil && strings.TrimSpace(row.ClientSecretEnc) != "" {
		plain, err := fernet.Decrypt(row.ClientSecretEnc)
		if err != nil {
			return nil, err
		}
		secret = plain
	}
	companyID := strings.TrimSpace(row.CompanyID)
	sp := "tenant-" + companyID
	base := strings.TrimRight(strings.TrimSpace(row.BaseURL), "/")
	scope := strings.TrimSpace(row.Scope)
	if scope == "" {
		scope = "read_repository write_repository api read_user"
	}
	return &ProviderConfig{
		Provider:        "gitlab",
		ServiceProvider: strings.ToLower(sp),
		ProviderKey:     "gitlab:" + strings.ToLower(sp),
		Website:         base,
		MatchOrigins:    []string{base},
		AuthorizeOrigin: base,
		ClientID:        row.ClientID,
		ClientSecret:    secret,
		// Rewrite legacy shared tenant-gitlab callback to tenant-{companyID}.
		// 单一实现：domain.CanonicalTenantRedirectURI（OPT-20260812-036 去重）。
		RedirectURI: domain.CanonicalTenantRedirectURI(row.RedirectURI, companyID),
		Scope:       scope,
	}, nil
}

func (c *Config) ResolveProviderConfig(provider, allowedHost string) *ProviderConfig {
	rows := c.GetProviderConfigs(provider)
	host := hostnameOf(allowedHost)
	if host == "" {
		return nil
	}
	for i := range rows {
		row := &rows[i]
		for _, origin := range row.MatchOrigins {
			if hostnameOf(origin) == host {
				cp := *row
				return &cp
			}
		}
		if hostnameOf(row.Website) == host {
			cp := *row
			return &cp
		}
	}
	return nil
}

// ResolveProviderByGitsite finds the provider config whose website / match origin
// hostname equals gitsite (e.g. "github.com", "gitlab.daydaymoney.com"). Used by the
// v2 browser callback contract /redirect/gitsite/<gitsite>/oauth/callback/ — the
// gitsite is the target Git 站点主机名 from provider YAML (never hardcoded in app).
// ResolveProviderByGitsiteWithDB tries YAML first, then tenant Path A rows by host.
func (c *Config) ResolveProviderByGitsiteWithDB(db *DB, fernet *Fernet, gitsite string) (*ProviderConfig, error) {
	if pc := c.ResolveProviderByGitsite(gitsite); pc != nil {
		return pc, nil
	}
	if db == nil {
		return nil, nil
	}
	row, err := db.GetTenantGitLabConnectionByHost(gitsite)
	if err != nil || row == nil {
		return nil, err
	}
	return TenantRowToProviderConfig(row, fernet)
}

func (c *Config) ResolveProviderByGitsite(gitsite string) *ProviderConfig {
	g := strings.TrimSpace(strings.ToLower(gitsite))
	if g == "" {
		return nil
	}
	for _, rows := range c.Providers {
		for i := range rows {
			row := &rows[i]
			if hostnameOf(row.Website) == g {
				cp := *row
				return &cp
			}
			for _, origin := range row.MatchOrigins {
				if hostnameOf(origin) == g {
					cp := *row
					return &cp
				}
			}
		}
	}
	return nil
}

func hostnameOf(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	if u.Hostname() != "" {
		return strings.ToLower(u.Hostname())
	}
	// bare host
	if !strings.Contains(raw, "://") {
		u2, err2 := url.Parse("http://" + raw)
		if err2 == nil {
			return strings.ToLower(u2.Hostname())
		}
	}
	return ""
}
