package infrastructure

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

func normalizeProviderConfigs(raw map[string]any) map[string][]ProviderConfig {
	out := map[string][]ProviderConfig{}
	if raw == nil {
		return out
	}
	for mapKey, value := range raw {
		item, ok := value.(map[string]any)
		if !ok {
			continue
		}
		target := item
		if t, ok := item["target"].(map[string]any); ok {
			target = t
		}
		service := item
		if s, ok := item["service"].(map[string]any); ok {
			service = s
		}
		provider := strings.TrimSpace(strings.ToLower(fmt.Sprint(item["provider"])))
		if provider == "" || provider == "<nil>" {
			provider = inferProviderFromWebsite(fmt.Sprint(target["website"]))
			if provider == "" {
				key := strings.ToLower(mapKey)
				if i := strings.Index(key, ":"); i > 0 {
					provider = key[:i]
				}
			}
		}
		if provider == "" {
			continue
		}
		sp := strings.TrimSpace(strings.ToLower(fmt.Sprint(item["service_provider"])))
		if sp == "" || sp == "<nil>" {
			sp = "default"
		}
		providerKey := strings.TrimSpace(strings.ToLower(fmt.Sprint(item["provider_key"])))
		if providerKey == "" || providerKey == "<nil>" {
			providerKey = provider + ":" + sp
		}
		website := strings.TrimSpace(fmt.Sprint(target["website"]))
		if website == "<nil>" {
			website = ""
		}
		matchOrigins := collectMatchOrigins(target, website)
		authorizeOrigin := strings.TrimRight(strings.TrimSpace(fmt.Sprint(target["authorize_origin"])), "/")
		if authorizeOrigin == "" || authorizeOrigin == "<nil>" {
			authorizeOrigin = OriginFromURL(website)
		}
		port := 0
		if p, ok := asInt(service["port"]); ok {
			port = p
		}
		outboundProxy := strings.TrimSpace(fmt.Sprint(service["outbound_proxy"]))
		if outboundProxy == "" || outboundProxy == "<nil>" {
			outboundProxy = strings.TrimSpace(fmt.Sprint(item["outbound_proxy"]))
		}
		if outboundProxy == "<nil>" {
			outboundProxy = ""
		}
		cfg := ProviderConfig{
			Provider:        provider,
			ServiceProvider: sp,
			ProviderKey:     providerKey,
			Website:         website,
			MatchOrigins:    matchOrigins,
			AuthorizeOrigin: authorizeOrigin,
			ClientID:        strings.TrimSpace(fmt.Sprint(target["client_id"])),
			ClientSecret:    strings.TrimSpace(fmt.Sprint(target["client_secret"])),
			RedirectURI:     strings.TrimSpace(fmt.Sprint(target["redirect_uri"])),
			Scope:           strings.TrimSpace(fmt.Sprint(target["scope"])),
			ServiceBase:     strings.TrimRight(strings.TrimSpace(fmt.Sprint(service["allowedHost"])), "/"),
			Host:            strings.TrimSpace(fmt.Sprint(service["host"])),
			Port:            port,
			OutboundProxy:   outboundProxy,
		}
		if cfg.ClientID == "<nil>" {
			cfg.ClientID = ""
		}
		if cfg.ClientSecret == "<nil>" {
			cfg.ClientSecret = ""
		}
		if cfg.RedirectURI == "<nil>" {
			cfg.RedirectURI = ""
		}
		if cfg.Scope == "<nil>" {
			cfg.Scope = ""
		}
		if aliases, ok := target["website_aliases"].([]any); ok {
			for _, a := range aliases {
				o := OriginFromURL(fmt.Sprint(a))
				if o != "" {
					cfg.WebsiteAliases = append(cfg.WebsiteAliases, o)
				}
			}
		}
		out[provider] = append(out[provider], cfg)
	}
	return out
}

func collectMatchOrigins(target map[string]any, website string) []string {
	var origins []string
	seen := map[string]bool{}
	add := func(o string) {
		o = strings.TrimRight(strings.TrimSpace(o), "/")
		if o == "" || seen[o] {
			return
		}
		seen[o] = true
		origins = append(origins, o)
	}
	add(OriginFromURL(website))
	if aliases, ok := target["website_aliases"].([]any); ok {
		for _, a := range aliases {
			add(OriginFromURL(fmt.Sprint(a)))
		}
	}
	return origins
}

// OriginFromURL returns scheme://host[:port] with no path. Empty / unparseable
// input is returned trimmed (caller decides whether to treat as invalid).
func OriginFromURL(raw string) string {
	raw = strings.TrimRight(strings.TrimSpace(raw), "/")
	if raw == "" || raw == "<nil>" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return raw
	}
	return strings.TrimRight(u.Scheme+"://"+u.Host, "/")
}

func inferProviderFromWebsite(website string) string {
	u, err := url.Parse(strings.TrimSpace(website))
	if err != nil {
		return ""
	}
	host := strings.ToLower(u.Hostname())
	switch {
	case host == "github.com":
		return "github"
	case strings.HasSuffix(host, "gitlab.com") || strings.Contains(host, "gitlab"):
		return "gitlab"
	case host == "bitbucket.org":
		return "bitbucket"
	default:
		return ""
	}
}

func asInt(v any) (int, bool) {
	switch t := v.(type) {
	case int:
		return t, true
	case int64:
		return int(t), true
	case float64:
		return int(t), true
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(t))
		return n, err == nil
	default:
		return 0, false
	}
}
