package infrastructure

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"confload"
	dbload "dbload"
	"gopkg.in/yaml.v3"
)

const DefaultSecretKey = "django-insecure-gitOauth-dev-change-for-production"

// ProviderConfig is one OAuth App entry (GitHub / GitLab / …).
type ProviderConfig struct {
	Provider        string
	ServiceProvider string
	ProviderKey     string
	Website         string
	WebsiteAliases  []string
	MatchOrigins    []string
	AuthorizeOrigin string
	ClientID        string
	ClientSecret    string
	RedirectURI     string
	Scope           string
	ServiceBase     string
	Host            string
	Port            int
	// OutboundProxy is optional explicit egress for provider HTTPS (e.g. socks5://127.0.0.1:1080).
	// Never populated from shell HTTP(S)_PROXY.
	OutboundProxy string
}

// Config is the runtime configuration for taskGitOauth.
type Config struct {
	SecretKey             string
	Port                  int
	Host                  string
	DatabasePath          string
	PublicBaseURL         string
	BridgeJWTSecret       string
	BridgeJWTIssuer       string
	BridgeJWTAudienceGH   string
	BridgeJWTAudienceGL   string
	RequireBridgeSecret   bool
	Task2appInternalAPI   string
	DjangoInternalSecret  string
	TaskTenantServiceURL  string
	TaskProjectServiceURL string
	TaskTaskServiceURL    string
	GatewayInternalSecret string
	KafkaBootstrapServers string
	TenantAuthBypass      bool
	FrontendBase          string
	PublicEntryOrigins    []string
	Providers             map[string][]ProviderConfig // provider -> configs
	GithubClientID        string
	GithubClientSecret    string
	GithubRedirectURI     string
	GithubScope           string
	// GithubOutboundProxy: optional socks5/http egress after direct GitHub dial fails.
	GithubOutboundProxy string
	Debug               bool
}

var domainPlaceholderRE = regexp.MustCompile(`\$\{(scheme|baseDomain|subdomains\.[A-Za-z][A-Za-z0-9]*)\}`)
var envDefaultRE = regexp.MustCompile(`\$\{([A-Z0-9_]+):-([^}]*)\}`)

// FindMonorepoRoot locates the deploy/repo root that contains conf/.
// OPT-20260901-019: delegate to confload so clone-run honors CONF_ROOT /
// DEPLOY_ROOT — a cwd walk stops at envs/current/<svc> whose conf/base.yaml
// has no sibling conf-local secrets (HTTP 200, empty key, provider AUTH fail).
func FindMonorepoRoot() (string, error) {
	return confload.FindConfigRoot()
}

func LoadConfig(root string) (*Config, error) {
	cfg := &Config{
		SecretKey:           DefaultSecretKey,
		Port:                8002,
		Host:                "0.0.0.0",
		BridgeJWTIssuer:     "task2app",
		BridgeJWTAudienceGH: "gitOauth-github-oauth",
		BridgeJWTAudienceGL: "gitOauth-gitlab-oauth",
		GithubScope:         "repo read:user",
		Providers:           map[string][]ProviderConfig{},
		Debug:               true,
		RequireBridgeSecret: true,
	}

	merged, err := loadMerged(root)
	if err != nil {
		return nil, err
	}

	if v, ok := merged["task2appSsoJwtSecret"].(string); ok && strings.TrimSpace(v) != "" {
		cfg.BridgeJWTSecret = strings.TrimSpace(v)
	}

	django, _ := merged["django"].(map[string]any)
	vue, _ := merged["vue"].(map[string]any)
	gw, _ := merged["taskGateway"].(map[string]any)
	if gw != nil {
		if s, ok := gw["gatewayInternalSecret"].(string); ok && strings.TrimSpace(s) != "" {
			cfg.GatewayInternalSecret = strings.TrimSpace(s)
		}
	}
	if s := strings.TrimSpace(os.Getenv("TASK_GATEWAY_INTERNAL_SECRET")); s != "" {
		cfg.GatewayInternalSecret = s
	}

	providersRaw, _ := merged["gitOauth"].(map[string]any)
	normalized := normalizeProviderConfigs(providersRaw)
	cfg.Providers = normalized

	ghList := normalized["github"]
	var gh ProviderConfig
	if len(ghList) > 0 {
		gh = ghList[0]
	}
	cfg.GithubClientID = gh.ClientID
	cfg.GithubClientSecret = gh.ClientSecret
	cfg.GithubRedirectURI = gh.RedirectURI
	if strings.TrimSpace(gh.Scope) != "" {
		cfg.GithubScope = gh.Scope
	}
	if gh.Port > 0 {
		cfg.Port = gh.Port
	}
	if strings.TrimSpace(gh.Host) != "" {
		cfg.Host = gh.Host
	}
	for _, p := range ghList {
		if s := strings.TrimSpace(p.OutboundProxy); s != "" {
			cfg.GithubOutboundProxy = s
			break
		}
	}
	if strings.TrimSpace(gh.ServiceBase) != "" {
		cfg.PublicBaseURL = strings.TrimRight(gh.ServiceBase, "/")
	} else if django != nil {
		if v, ok := django["gitoauth"].(string); ok && strings.TrimSpace(v) != "" {
			cfg.PublicBaseURL = strings.TrimRight(strings.TrimSpace(v), "/")
		}
	}
	if cfg.PublicBaseURL == "" {
		cfg.PublicBaseURL = "http://localhost:8002"
	}

	if django != nil {
		if v, ok := django["internalApiBase"].(string); ok && strings.TrimSpace(v) != "" {
			cfg.Task2appInternalAPI = strings.TrimRight(strings.TrimSpace(v), "/")
		} else {
			port := 8001
			if p, ok := asInt(django["port"]); ok {
				port = p
			}
			cfg.Task2appInternalAPI = fmt.Sprintf("http://127.0.0.1:%d", port)
		}
	} else {
		cfg.Task2appInternalAPI = "http://127.0.0.1:8001"
	}
	// Tenant service default (OPT-20260729-024 #8)
	cfg.TaskTenantServiceURL = "http://127.0.0.1:8020"
	// taskProjectService internal base (Path A GitLab conn cache invalidation)
	cfg.TaskProjectServiceURL = "http://127.0.0.1:8016"
	if s := strings.TrimSpace(os.Getenv("TASK_PROJECT_SERVICE_URL")); s != "" {
		cfg.TaskProjectServiceURL = strings.TrimRight(s, "/")
	}
	cfg.TaskTaskServiceURL = "http://127.0.0.1:8017"
	if s := strings.TrimSpace(os.Getenv("TASK_TASK_SERVICE_BASE_URL")); s != "" {
		cfg.TaskTaskServiceURL = strings.TrimRight(s, "/")
	}

	if vue != nil {
		if v, ok := vue["publicBaseUrl"].(string); ok {
			cfg.FrontendBase = strings.TrimRight(strings.TrimSpace(v), "/")
		}
		if raw, ok := vue["publicEntryOrigins"].([]any); ok {
			for _, item := range raw {
				if s, ok := item.(string); ok {
					o := strings.TrimRight(strings.TrimSpace(s), "/")
					if o != "" {
						cfg.PublicEntryOrigins = append(cfg.PublicEntryOrigins, o)
					}
				}
			}
		}
	}

	if s := strings.TrimSpace(os.Getenv("GITOAUTH_BRIDGE_JWT_SECRET")); s != "" {
		cfg.BridgeJWTSecret = s
	} else if s := strings.TrimSpace(os.Getenv("TASK2APP_SSO_JWT_SECRET")); s != "" {
		cfg.BridgeJWTSecret = s
	}
	if cfg.BridgeJWTSecret == "" {
		cfg.BridgeJWTSecret = cfg.SecretKey
	}
	if s := strings.TrimSpace(os.Getenv("GITOAUTH_BRIDGE_JWT_ISSUER")); s != "" {
		cfg.BridgeJWTIssuer = s
	}
	if s := strings.TrimSpace(os.Getenv("GITOAUTH_BRIDGE_JWT_AUDIENCE")); s != "" {
		cfg.BridgeJWTAudienceGH = s
	}
	if s := strings.TrimSpace(os.Getenv("GITOAUTH_BRIDGE_JWT_AUDIENCE_GITLAB")); s != "" {
		cfg.BridgeJWTAudienceGL = s
	}
	if s := strings.TrimSpace(os.Getenv("TASK2APP_INTERNAL_API_BASE")); s != "" {
		cfg.Task2appInternalAPI = strings.TrimRight(s, "/")
	}
	if s := strings.TrimSpace(os.Getenv("TASK_PROJECT_SERVICE_INTERNAL_SECRET")); s != "" {
		cfg.DjangoInternalSecret = s
	} else if s := strings.TrimSpace(os.Getenv("SAAS_INTERNAL_SECRET")); s != "" {
		cfg.DjangoInternalSecret = s
	} else if s := strings.TrimSpace(os.Getenv("INTERNAL_SECRET")); s != "" {
		cfg.DjangoInternalSecret = s
	} else {
		cfg.DjangoInternalSecret = cfg.BridgeJWTSecret
	}
	if s := strings.TrimSpace(os.Getenv("KAFKA_BOOTSTRAP_SERVERS")); s != "" {
		cfg.KafkaBootstrapServers = s
	}
	if s := strings.TrimSpace(os.Getenv("GITOAUTH_TENANT_AUTH_BYPASS")); s != "" {
		cfg.TenantAuthBypass = s == "1" || strings.EqualFold(s, "true") || strings.EqualFold(s, "yes")
	}
	if s := strings.TrimSpace(os.Getenv("TASK2APP_FRONTEND_BASE")); s != "" {
		cfg.FrontendBase = strings.TrimRight(s, "/")
	}
	if s := strings.TrimSpace(os.Getenv("GITHUB_APP_CLIENT_ID")); s != "" {
		cfg.GithubClientID = s
	}
	if s := strings.TrimSpace(os.Getenv("GITHUB_APP_CLIENT_SECRET")); s != "" {
		cfg.GithubClientSecret = s
	}
	if s := strings.TrimSpace(os.Getenv("GITHUB_APP_REDIRECT_URI")); s != "" {
		cfg.GithubRedirectURI = s
	}
	if s := strings.TrimSpace(os.Getenv("GITHUB_APP_SCOPE")); s != "" {
		cfg.GithubScope = s
	}
	if s := strings.TrimSpace(os.Getenv("GITOAUTH_GITHUB_OUTBOUND_PROXY")); s != "" {
		cfg.GithubOutboundProxy = s
	}
	if s := strings.TrimSpace(os.Getenv("GITOAUTH_GITHUB_DIAL_FALLBACK_IPS")); s != "" {
		var ips []string
		for _, p := range strings.Split(s, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				ips = append(ips, p)
			}
		}
		if len(ips) > 0 {
			githubDialFallbackIPs = ips
		}
	}
	if s := strings.TrimSpace(os.Getenv("GITOAUTH_PORT")); s != "" {
		if p, err := strconv.Atoi(s); err == nil && p > 0 {
			cfg.Port = p
		}
	}
	if s := strings.TrimSpace(os.Getenv("GITOAUTH_HOST")); s != "" {
		cfg.Host = s
	}
	if s := strings.TrimSpace(os.Getenv("GITOAUTH_REQUIRE_BRIDGE_SECRET")); s != "" {
		cfg.RequireBridgeSecret = s == "1" || strings.EqualFold(s, "true") || strings.EqualFold(s, "yes")
	}

	dbPath := strings.TrimSpace(os.Getenv("GITOAUTH_MYSQL_DSN"))
	if dbPath == "" {
		mysqlDSN, err := dbload.ResolveMySQLDSN("git-oauth", root)
		if err != nil {
			return nil, fmt.Errorf("git-oauth MySQL DSN: %w", err)
		}
		dbPath = mysqlDSN
	}
	cfg.DatabasePath = dbPath
	return cfg, nil
}

func loadMerged(root string) (map[string]any, error) {
	own := filepath.Join(root, "conf", "auth", "git-oauth")
	if st, err := os.Stat(own); err != nil || !st.IsDir() {
		old := filepath.Join(root, "conf", "git-oauth")
		if st2, err2 := os.Stat(old); err2 == nil && st2.IsDir() {
			own = old
		}
	}
	out := map[string]any{}

	if djangoCfg, err := readYAMLResolved(root, filepath.Join(own, "django.yaml")); err == nil {
		if sso, ok := djangoCfg["ssoJwtSecret"].(string); ok && strings.TrimSpace(sso) != "" {
			out["task2appSsoJwtSecret"] = strings.TrimSpace(sso)
		}
		djangoCopy := map[string]any{}
		for k, v := range djangoCfg {
			if k == "ssoJwtSecret" {
				continue
			}
			djangoCopy[k] = v
		}
		out["django"] = djangoCopy
	} else {
		out["django"] = map[string]any{}
	}

	if vueCfg, err := readYAMLResolved(root, filepath.Join(own, "vue.yaml")); err == nil {
		out["vue"] = vueCfg
	} else {
		out["vue"] = map[string]any{}
	}

	if gwCfg, err := readYAMLResolved(root, filepath.Join(own, "task-gateway.yaml")); err == nil {
		out["taskGateway"] = gwCfg
	} else {
		out["taskGateway"] = map[string]any{}
	}

	catalog := map[string]any{}
	provDir := filepath.Join(own, "providers")
	entries, err := os.ReadDir(provDir)
	if err == nil {
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
				continue
			}
			block, err := readYAMLResolved(root, filepath.Join(provDir, e.Name()))
			if err != nil {
				continue
			}
			provider := strings.TrimSpace(strings.ToLower(fmt.Sprint(block["provider"])))
			sp := strings.TrimSpace(strings.ToLower(fmt.Sprint(block["service_provider"])))
			var key string
			switch {
			case provider != "" && sp != "" && sp != "<nil>":
				key = provider + ":" + sp
			case provider != "":
				key = provider
			default:
				target, _ := block["target"].(map[string]any)
				if target != nil {
					key = strings.TrimSpace(fmt.Sprint(target["website"]))
				}
				if key == "" || key == "<nil>" {
					key = strings.TrimSuffix(e.Name(), ".yaml")
				}
			}
			catalog[key] = block
		}
	}
	out["gitOauth"] = catalog
	return out, nil
}

func readYAMLResolved(root, path string) (map[string]any, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw map[string]any
	if err := yaml.Unmarshal(b, &raw); err != nil {
		return nil, err
	}
	if raw == nil {
		raw = map[string]any{}
	}
	confRoot := filepath.Join(root, "conf")
	if rel, relErr := filepath.Rel(confRoot, path); relErr == nil && !strings.HasPrefix(rel, "..") {
		raw = confload.MergeConfLocal(root, filepath.ToSlash(rel), raw)
	}
	return resolveDomainPlaceholders(root, raw), nil
}

func buildDomainMap(root string) map[string]string {
	raw, err := confload.ReadYAMLMerged(root, "base.yaml")
	if err != nil || raw == nil {
		return nil
	}
	scheme := resolveEnvDefault(fmt.Sprint(raw["scheme"]), "PUBLIC_SCHEME")
	scheme = strings.TrimRight(strings.ToLower(strings.TrimSpace(scheme)), ":/")
	if scheme == "" {
		scheme = "https"
	}
	if scheme != "http" && scheme != "https" {
		scheme = "https"
	}
	baseDomain := resolveEnvDefault(fmt.Sprint(raw["baseDomain"]), "BASE_DOMAIN")
	baseDomain = strings.TrimSpace(baseDomain)
	if baseDomain == "" {
		return nil
	}
	dm := map[string]string{"scheme": scheme, "baseDomain": baseDomain}
	subs, _ := raw["subdomains"].(map[string]any)
	for k, v := range subs {
		val := strings.ReplaceAll(fmt.Sprint(v), "${scheme}", scheme)
		val = strings.ReplaceAll(val, "${baseDomain}", baseDomain)
		dm["subdomains."+k] = val
	}
	return dm
}

func resolveEnvDefault(value, envVar string) string {
	pattern := regexp.MustCompile(`\$\{` + regexp.QuoteMeta(envVar) + `:-([^}]*)\}`)
	envVal := strings.TrimSpace(os.Getenv(envVar))
	if envVal != "" {
		return pattern.ReplaceAllString(value, envVal)
	}
	return pattern.ReplaceAllString(value, "$1")
}

func resolveDomainPlaceholders(root string, config map[string]any) map[string]any {
	dm := buildDomainMap(root)
	if dm == nil {
		return config
	}
	var resolve func(any) any
	resolve = func(v any) any {
		switch t := v.(type) {
		case string:
			out := domainPlaceholderRE.ReplaceAllStringFunc(t, func(m string) string {
				key := domainPlaceholderRE.FindStringSubmatch(m)[1]
				if rep, ok := dm[key]; ok {
					return rep
				}
				return m
			})
			out = envDefaultRE.ReplaceAllStringFunc(out, func(m string) string {
				parts := envDefaultRE.FindStringSubmatch(m)
				if len(parts) != 3 {
					return m
				}
				if ev := strings.TrimSpace(os.Getenv(parts[1])); ev != "" {
					return ev
				}
				return parts[2]
			})
			return out
		case map[string]any:
			out := map[string]any{}
			for k, vv := range t {
				out[k] = resolve(vv)
			}
			return out
		case []any:
			out := make([]any, len(t))
			for i, vv := range t {
				out[i] = resolve(vv)
			}
			return out
		default:
			return v
		}
	}
	resolved, _ := resolve(config).(map[string]any)
	return resolved
}
