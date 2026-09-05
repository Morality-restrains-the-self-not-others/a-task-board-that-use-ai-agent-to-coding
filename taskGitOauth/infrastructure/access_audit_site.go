package infrastructure

import (
	"net/url"
	"strings"
)

// AuditSiteHostPort returns the Git site as host[:port] (url.Host, lowercased).
// Empty website or parse failure returns "". Never returns a provider_key.
func AuditSiteHostPort(website string) string {
	raw := strings.TrimSpace(website)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	host := ""
	if err == nil {
		host = strings.TrimSpace(u.Host)
	}
	if host == "" && !strings.Contains(raw, "://") {
		u2, err2 := url.Parse("http://" + raw)
		if err2 == nil {
			host = strings.TrimSpace(u2.Host)
		}
	}
	return strings.ToLower(host)
}

// AuditSiteForProviderKey resolves provider_key → configured website → host[:port].
func AuditSiteForProviderKey(cfg *Config, providerKey string) string {
	if cfg == nil {
		return ""
	}
	pc := ProviderConfigByKey(cfg, providerKey)
	if pc == nil {
		return ""
	}
	return AuditSiteHostPort(pc.Website)
}
