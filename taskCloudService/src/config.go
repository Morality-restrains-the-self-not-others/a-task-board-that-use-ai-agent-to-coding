package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"confload"
	"dbload"
	"gatewayauth"
)

type Config struct {
	Host         string
	Port         int
	DBPath       string
	BudgetDBPath string
	TaskAuthURL  string
	// PublicTaskAPIBase is the ECS-reachable API origin (gateway publicBase), used for
	// which is often loopback-only.
	PublicTaskAPIBase string
	ProjectServiceURL string
	TaskServiceURL    string
	// TaskTenantURL is the taskTenantService base (tenant_company SSOT).
	TaskTenantURL          string
	CredentialServiceURL   string
	ContainerGatewayURL    string
	AIProviderBaseURL      string
	TaskBillURL            string
	TaskBillInternalSecret string
	InternalSecret         string
	GatewayInternalSecret  string
	// ContainerGatewayInternalSecret authenticates Cloud → Django
	// /api/internal/task-container-gateway/* (header X-TaskContainerGateway-Internal-Secret).
	ContainerGatewayInternalSecret string
	KafkaBootstrapServers          string
	RedisHost                      string
	RedisPort                      int
	RedisDB                        int
	RelaySessionTTLSec             int
	// Task AI endpoint proxy (FeatureParams sub-token rewrite). Env overrides conf.
	TaskAIEndpointEnabled    bool
	TaskAIEndpointPublicBase string
	// GitOauthBaseURL is the gitOauth service base for layer-git-push prepare (Phase C).
	GitOauthBaseURL      string
	GitOauthBridgeSecret string
	// Aliyun OAuth (RAM) integration for cloud platform authorization.
	AliyunOAuthClientID     string
	AliyunOAuthClientSecret string
	AliyunOAuthRedirectURI  string
	// Userdata container env agent (Phase F4): OpenAI-compatible chat/completions.
	UserdataEnvAgentAPIKey      string
	UserdataEnvAgentBaseURL     string
	UserdataEnvAgentModel       string
	UserdataEnvAgentMaxTokens   int
	UserdataEnvAgentTemperature float64
}

var cfg Config

func findMonorepoRoot() (string, error) {
	return confload.FindMonorepoRoot()
}

func loadConfig(repoRoot string) {
	type baseConf struct {
		Services struct {
			TaskCloudService struct {
				Host string `yaml:"host"`
				Port int    `yaml:"port"`
			} `yaml:"taskCloudService"`
			TaskAuth struct {
				Host string `yaml:"host"`
				Port int    `yaml:"port"`
			} `yaml:"taskAuth"`
			TaskProjectService struct {
				Host string `yaml:"host"`
				Port int    `yaml:"port"`
			} `yaml:"taskProjectService"`
			TaskTaskService struct {
				Host string `yaml:"host"`
				Port int    `yaml:"port"`
			} `yaml:"taskTaskService"`
			TaskTenantService struct {
				Host string `yaml:"host"`
				Port int    `yaml:"port"`
			} `yaml:"taskTenantService"`
			TaskContainerGateway struct {
				Host string `yaml:"host"`
				Port int    `yaml:"port"`
			} `yaml:"taskContainerGateway"`
			TaskBill struct {
				Host string `yaml:"host"`
				Port int    `yaml:"port"`
			} `yaml:"taskBill"`
		} `yaml:"services"`
		Shared struct {
			InternalSecret         string `yaml:"internalSecret"`
			TaskBillInternalSecret string `yaml:"taskBillInternalSecret"`
		} `yaml:"shared"`
		Kafka struct {
			BootstrapServers string `yaml:"bootstrapServers"`
		} `yaml:"kafka"`
	}

	type infraServices struct {
		Services struct {
			TaskCredentialService struct {
				Host string `yaml:"host"`
				Port int    `yaml:"port"`
			} `yaml:"taskCredentialService"`
		} `yaml:"services"`
	}

	var bc baseConf
	if err := confload.ReadAppConfigResolved(repoRoot, "taskCloudService", &bc); err != nil {
		log.Fatalf("[taskCloudService] confload: %v", err)
	}

	cfg.Host = bc.Services.TaskCloudService.Host
	cfg.Port = bc.Services.TaskCloudService.Port
	if cfg.Host == "" {
		cfg.Host = "0.0.0.0"
	}
	if cfg.Port == 0 {
		cfg.Port = 8018
	}
	cfg.TaskAuthURL = "http://" + bc.Services.TaskAuth.Host + ":" + fmt.Sprintf("%d", bc.Services.TaskAuth.Port)
	cfg.ProjectServiceURL = "http://" + bc.Services.TaskProjectService.Host + ":" + fmt.Sprintf("%d", bc.Services.TaskProjectService.Port)
	cfg.TaskServiceURL = "http://" + bc.Services.TaskTaskService.Host + ":" + fmt.Sprintf("%d", bc.Services.TaskTaskService.Port)
	tenantHost := strings.TrimSpace(bc.Services.TaskTenantService.Host)
	tenantPort := bc.Services.TaskTenantService.Port
	if tenantHost == "" || tenantHost == "0.0.0.0" {
		tenantHost = "127.0.0.1"
	}
	if tenantPort == 0 {
		tenantPort = 8020
	}
	cfg.TaskTenantURL = "http://" + tenantHost + ":" + fmt.Sprintf("%d", tenantPort)
	if v := strings.TrimSpace(os.Getenv("TASK_TENANT_SERVICE_URL")); v != "" {
		cfg.TaskTenantURL = strings.TrimRight(v, "/")
	}
	if bc.Services.TaskBill.Port != 0 {
		billHost := bc.Services.TaskBill.Host
		if billHost == "" {
			billHost = "127.0.0.1"
		}
		cfg.TaskBillURL = "http://" + billHost + ":" + fmt.Sprintf("%d", bc.Services.TaskBill.Port)
	}
	cfg.TaskBillInternalSecret = bc.Shared.TaskBillInternalSecret
	if cfg.TaskBillInternalSecret == "" {
		cfg.TaskBillInternalSecret = loadTaskBillSecret(repoRoot)
	}
	tcgHost := bc.Services.TaskContainerGateway.Host
	tcgPort := bc.Services.TaskContainerGateway.Port
	if tcgHost == "" {
		tcgHost = "127.0.0.1"
	}
	if tcgPort == 0 {
		tcgPort = 8014
	}
	cfg.ContainerGatewayURL = "http://" + tcgHost + ":" + fmt.Sprintf("%d", tcgPort)
	cfg.PublicTaskAPIBase = loadPublicTaskAPIBase(repoRoot)
	cfg.AIProviderBaseURL = resolveAIProviderBaseURL(repoRoot)
	cfg.InternalSecret = bc.Shared.InternalSecret
	cfg.GatewayInternalSecret = gatewayauth.LoadGatewayInternalSecret(repoRoot)
	cfg.ContainerGatewayInternalSecret = loadContainerGatewayInternalSecret(repoRoot)
	cfg.KafkaBootstrapServers = bc.Kafka.BootstrapServers
	if cfg.KafkaBootstrapServers == "" {
		cfg.KafkaBootstrapServers = loadKafkaBootstrap(repoRoot)
	}

	var cred infraServices
	if err := confload.ReadAppConfigResolved(repoRoot, "taskCredentialService", &cred); err == nil {
		host := cred.Services.TaskCredentialService.Host
		port := cred.Services.TaskCredentialService.Port
		if host == "" {
			host = "127.0.0.1"
		}
		if port == 0 {
			port = 8015
		}
		cfg.CredentialServiceURL = "http://" + host + ":" + fmt.Sprintf("%d", port)
	} else {
		cfg.CredentialServiceURL = "http://127.0.0.1:8015"
	}
	if v := strings.TrimSpace(os.Getenv("TASK_CREDENTIAL_SERVICE_BASE_URL")); v != "" {
		cfg.CredentialServiceURL = strings.TrimRight(v, "/")
	}
	dbPath, err := dbload.ResolveMySQLDSN("task-cloud", repoRoot)
	if err != nil {
		log.Fatalf("[taskCloudService] db path: %v", err)
	}
	cfg.DBPath = dbPath
	cfg.BudgetDBPath = resolveBudgetDBDSN(repoRoot)
	loadTaskAIEndpointConf(repoRoot)

	cfg.GitOauthBaseURL = strings.TrimRight(strings.TrimSpace(os.Getenv("GITOAUTH_BASE_URL")), "/")
	if cfg.GitOauthBaseURL == "" {
		cfg.GitOauthBaseURL = "http://127.0.0.1:8002"
	}
	// Align with taskProjectService / Django: empty bridge secret → gitOauth access-for-user returns 401 unauthorized
	// while UI summary (via Django) still shows「OAuth 已授权」.
	cfg.GitOauthBridgeSecret = strings.TrimSpace(os.Getenv("GITOAUTH_BRIDGE_JWT_SECRET"))
	if cfg.GitOauthBridgeSecret == "" {
		cfg.GitOauthBridgeSecret = strings.TrimSpace(os.Getenv("TASK2APP_SSO_JWT_SECRET"))
	}
	type gitoauthConf struct {
		GitOauth *struct {
			URL          string `yaml:"url"`
			BridgeSecret string `yaml:"bridgeSecret"`
		} `yaml:"gitOauth"`
	}
	var gc gitoauthConf
	if err := confload.ReadAppConfigResolved(repoRoot, "taskCloudService", &gc); err == nil && gc.GitOauth != nil {
		if u := strings.TrimSpace(gc.GitOauth.URL); u != "" {
			cfg.GitOauthBaseURL = strings.TrimRight(u, "/")
		}
		if s := strings.TrimSpace(gc.GitOauth.BridgeSecret); s != "" {
			cfg.GitOauthBridgeSecret = s
		}
	}
	// OPT-20260809-015 防漂移：taskGitOauth BridgeJWTSecret 的实际来源是
	// conf/auth/git-oauth/django.yaml ssoJwtSecret（SSOT，OPT-20260806-062 轮换）。
	// taskCloudService 自持一份 bridgeSecret，轮换漏改一边即静默 401 —— 启动时比对并告警，
	// 配置为本地占位符或空时回退到 SSOT 值。
	cfg.GitOauthBridgeSecret = resolveGitOauthBridgeSecret(cfg.GitOauthBridgeSecret, loadGitOauthSSOSecret(repoRoot))

	type cloudRedisConf struct {
		Redis struct {
			Host string `yaml:"host"`
			Port int    `yaml:"port"`
			DB   int    `yaml:"db"`
		} `yaml:"redis"`
	}
	var rc cloudRedisConf
	if err := confload.ReadAppConfigResolved(repoRoot, "taskCloudService", &rc); err == nil {
		if h := strings.TrimSpace(rc.Redis.Host); h != "" {
			cfg.RedisHost = h
		}
		if rc.Redis.Port != 0 {
			cfg.RedisPort = rc.Redis.Port
		}
		cfg.RedisDB = rc.Redis.DB
	}

	loadAliyunOAuthConf(repoRoot)
	loadUserdataEnvAgentConf(repoRoot)

	log.Printf("[taskCloudService] config: host=%s port=%d auth=%s tenant=%s project=%s task=%s kafka=%s aiProvider=%s budgetDB=%s",
		cfg.Host, cfg.Port, cfg.TaskAuthURL, cfg.TaskTenantURL, cfg.ProjectServiceURL, cfg.TaskServiceURL, cfg.KafkaBootstrapServers, cfg.AIProviderBaseURL, cfg.BudgetDBPath)
}

// localBridgePlaceholder is the dev-only default bridge secret; keeping it in
// production yields 401 from taskGitOauth requireBridgeSecret (OPT-20260809-015).
const localBridgePlaceholder = "task2app-local-sso-bridge-dev-do-not-use-in-prod"

// resolveGitOauthBridgeSecret aligns the configured gitOauth.bridgeSecret with the
// SSOT value from conf/auth/git-oauth/django.yaml ssoJwtSecret: placeholder/empty
// falls back to SSOT, drift is logged as a 401-risk warning (never the secret value).
func resolveGitOauthBridgeSecret(current, ssoSecret string) string {
	if ssoSecret != "" {
		if current == "" || current == localBridgePlaceholder {
			if current == localBridgePlaceholder {
				log.Printf("[taskCloudService] WARN gitOauth.bridgeSecret is local placeholder; falling back to conf/auth/git-oauth/django.yaml ssoJwtSecret")
			} else {
				log.Printf("[taskCloudService] WARN gitOauth.bridgeSecret empty; using conf/auth/git-oauth/django.yaml ssoJwtSecret")
			}
			return ssoSecret
		}
		if current != ssoSecret {
			log.Printf("[taskCloudService] WARN gitOauth.bridgeSecret differs from git-oauth SSOT ssoJwtSecret — 401 risk; keep both in sync on rotation (OPT-20260809-015)")
		}
	}
	if current == "" {
		current = localBridgePlaceholder
		log.Printf("[taskCloudService] WARN gitOauth.bridgeSecret empty; using local SSO bridge default (set GITOAUTH_BRIDGE_JWT_SECRET in prod)")
	}
	return current
}

// loadGitOauthSSOSecret reads conf/auth/git-oauth/django.yaml's ssoJwtSecret — the
// authoritative source for taskGitOauth BridgeJWTSecret (OPT-20260809-015 防漂移)。
// Returns "" when the file or key is absent so callers fall back to their own default.
func loadGitOauthSSOSecret(repoRoot string) string {
	var doc struct {
		SSOJwtSecret string `yaml:"ssoJwtSecret"`
	}
	if err := confload.ReadAppFragment(repoRoot, "git-oauth", "django.yaml", &doc); err != nil {
		return ""
	}
	return strings.TrimSpace(doc.SSOJwtSecret)
}

func loadAliyunOAuthConf(repoRoot string) {
	// Env vars take precedence over YAML config.
	cfg.AliyunOAuthClientID = strings.TrimSpace(os.Getenv("ALIYUN_OAUTH_CLIENT_ID"))
	cfg.AliyunOAuthClientSecret = strings.TrimSpace(os.Getenv("ALIYUN_OAUTH_CLIENT_SECRET"))
	cfg.AliyunOAuthRedirectURI = strings.TrimSpace(os.Getenv("ALIYUN_OAUTH_REDIRECT_URI"))

	if cfg.AliyunOAuthClientID == "" || cfg.AliyunOAuthClientSecret == "" {
		type aliyunOAuthConf struct {
			ClientID     string `yaml:"clientId"`
			ClientSecret string `yaml:"clientSecret"`
			RedirectURI  string `yaml:"redirectUri"`
		}
		type yamlConf struct {
			AliyunOAuth *aliyunOAuthConf `yaml:"aliyunOAuth"`
		}
		var yc yamlConf
		if err := confload.ReadAppConfigResolved(repoRoot, "taskCloudService", &yc); err == nil && yc.AliyunOAuth != nil {
			if cfg.AliyunOAuthClientID == "" {
				cfg.AliyunOAuthClientID = strings.TrimSpace(yc.AliyunOAuth.ClientID)
			}
			if cfg.AliyunOAuthClientSecret == "" {
				cfg.AliyunOAuthClientSecret = strings.TrimSpace(yc.AliyunOAuth.ClientSecret)
			}
			if cfg.AliyunOAuthRedirectURI == "" {
				cfg.AliyunOAuthRedirectURI = strings.TrimSpace(yc.AliyunOAuth.RedirectURI)
			}
		}
	}

	if cfg.AliyunOAuthRedirectURI == "" {
		cfg.AliyunOAuthRedirectURI = cfg.PublicTaskAPIBase + "/callback/cloudplatform/oauth2.0/aliyun"
	}
}

// loadUserdataEnvAgentConf loads USERDATA_CONTAINER_ENV_AGENT from yaml / JSON env / individual envs.
// Precedence: individual envs > USERDATA_CONTAINER_ENV_AGENT_JSON > conf yaml.
func loadUserdataEnvAgentConf(repoRoot string) {
	cfg.UserdataEnvAgentMaxTokens = 1200
	cfg.UserdataEnvAgentTemperature = 0

	type userdataAgentBlock struct {
		APIKey      string  `yaml:"api_key"`
		BaseURL     string  `yaml:"base_url"`
		Model       string  `yaml:"model"`
		MaxTokens   int     `yaml:"max_tokens"`
		Temperature float64 `yaml:"temperature"`
	}
	type agentYAML struct {
		UserdataContainerEnvAgent      *userdataAgentBlock `yaml:"userdataContainerEnvAgent"`
		UserdataContainerEnvAgentSnake *userdataAgentBlock `yaml:"userdata_container_env_agent"`
	}
	var yc agentYAML
	if err := confload.ReadAppConfigResolved(repoRoot, "taskCloudService", &yc); err == nil {
		block := yc.UserdataContainerEnvAgent
		if block == nil {
			block = yc.UserdataContainerEnvAgentSnake
		}
		if block != nil {
			applyUserdataEnvAgentFields(block.APIKey, block.BaseURL, block.Model, block.MaxTokens, block.Temperature)
		}
	}

	if raw := strings.TrimSpace(os.Getenv("USERDATA_CONTAINER_ENV_AGENT_JSON")); raw != "" {
		var obj map[string]any
		if err := json.Unmarshal([]byte(raw), &obj); err == nil {
			apiKey, _ := obj["api_key"].(string)
			baseURL, _ := obj["base_url"].(string)
			model, _ := obj["model"].(string)
			maxTokens := cfg.UserdataEnvAgentMaxTokens
			temp := cfg.UserdataEnvAgentTemperature
			switch v := obj["max_tokens"].(type) {
			case float64:
				maxTokens = int(v)
			case string:
				if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
					maxTokens = n
				}
			}
			switch v := obj["temperature"].(type) {
			case float64:
				temp = v
			case string:
				if f, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
					temp = f
				}
			}
			applyUserdataEnvAgentFields(apiKey, baseURL, model, maxTokens, temp)
		}
	}

	if v := strings.TrimSpace(os.Getenv("USERDATA_CONTAINER_ENV_AGENT_API_KEY")); v != "" {
		cfg.UserdataEnvAgentAPIKey = v
	}
	if v := strings.TrimSpace(os.Getenv("USERDATA_CONTAINER_ENV_AGENT_BASE_URL")); v != "" {
		cfg.UserdataEnvAgentBaseURL = v
	}
	if v := strings.TrimSpace(os.Getenv("USERDATA_CONTAINER_ENV_AGENT_MODEL")); v != "" {
		cfg.UserdataEnvAgentModel = v
	}
	if v := strings.TrimSpace(os.Getenv("USERDATA_CONTAINER_ENV_AGENT_MAX_TOKENS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.UserdataEnvAgentMaxTokens = n
		}
	}
	if v := strings.TrimSpace(os.Getenv("USERDATA_CONTAINER_ENV_AGENT_TEMPERATURE")); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			cfg.UserdataEnvAgentTemperature = f
		}
	}
}

func applyUserdataEnvAgentFields(apiKey, baseURL, model string, maxTokens int, temperature float64) {
	if s := strings.TrimSpace(apiKey); s != "" {
		cfg.UserdataEnvAgentAPIKey = s
	}
	if s := strings.TrimSpace(baseURL); s != "" {
		cfg.UserdataEnvAgentBaseURL = strings.TrimRight(s, "/")
	}
	if s := strings.TrimSpace(model); s != "" {
		cfg.UserdataEnvAgentModel = s
	}
	if maxTokens > 0 {
		cfg.UserdataEnvAgentMaxTokens = maxTokens
	}
	cfg.UserdataEnvAgentTemperature = temperature
}

func resolveAIProviderBaseURL(repoRoot string) string {
	if env := strings.TrimSpace(os.Getenv("TASK2APP_AI_PROVIDER_BASE_URL")); env != "" {
		return strings.TrimRight(env, "/")
	}
	type aiProviderConf struct {
		Host        string `yaml:"host"`
		Port        int    `yaml:"port"`
		IsDev       bool   `yaml:"isDev"`
		AllowedHost string `yaml:"allowedHost"`
		BaseURL     string `yaml:"baseUrl"`
		Scheme      string `yaml:"scheme"`
	}
	var ap aiProviderConf
	if err := confload.ReadAppConfigResolved(repoRoot, "ai/ai-provider", &ap); err != nil {
		return "http://127.0.0.1:8010"
	}
	if base := strings.TrimSpace(ap.BaseURL); base != "" {
		return strings.TrimRight(base, "/")
	}
	// allowedHost 优先：开发模式下若已解析为有效 URL（不含未解析模板变量 ${），则使用它；
	// 非开发模式下 always use allowedHost。这样在生产环境无需修改 isDev 即可连通 AI provider。
	if raw := strings.TrimSpace(ap.AllowedHost); raw != "" {
		if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
			if !ap.IsDev || !strings.Contains(raw, "${") {
				return strings.TrimRight(raw, "/")
			}
		}
	}
	host := strings.TrimSpace(ap.Host)
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	scheme := strings.TrimSpace(ap.Scheme)
	if scheme == "" {
		scheme = "http"
	}
	port := ap.Port
	if port == 0 {
		port = 8010
	}
	if (scheme == "http" && port == 80) || (scheme == "https" && port == 443) {
		return scheme + "://" + host
	}
	return fmt.Sprintf("%s://%s:%d", scheme, host, port)
}

func loadKafkaBootstrap(repoRoot string) string {
	type infraConf struct {
		Kafka struct {
			BootstrapServers string `yaml:"bootstrapServers"`
		} `yaml:"kafka"`
	}
	var ic infraConf
	if err := confload.ReadAppConfigResolved(repoRoot, "domain-events", &ic); err != nil {
		return "localhost:9093"
	}
	if ic.Kafka.BootstrapServers != "" {
		return ic.Kafka.BootstrapServers
	}
	return "localhost:9093"
}

func loadTaskBillSecret(repoRoot string) string {
	type billConf struct {
		InternalSecret string `yaml:"internalSecret"`
	}
	var bc billConf
	if err := confload.ReadAppConfigResolved(repoRoot, "billing/task-bill", &bc); err != nil {
		return ""
	}
	return strings.TrimSpace(bc.InternalSecret)
}

func loadContainerGatewayInternalSecret(repoRoot string) string {
	if v := strings.TrimSpace(os.Getenv("TASK_CONTAINER_GATEWAY_INTERNAL_SECRET")); v != "" {
		return v
	}
	type tcgConf struct {
		InternalSecret string `yaml:"internalSecret"`
	}
	var tc tcgConf
	if err := confload.ReadAppConfigResolved(repoRoot, "gateway/task-container-gateway", &tc); err != nil {
		return ""
	}
	return strings.TrimSpace(tc.InternalSecret)
}

// loadPublicTaskAPIBase returns the ECS-reachable task API origin (gateway publicBase).
// Empty when unset; callers should fall back carefully (not to loopback Django).
func loadPublicTaskAPIBase(repoRoot string) string {
	if v := strings.TrimSpace(os.Getenv("TASK2APP_PUBLIC_TASK_API_BASE")); v != "" {
		return strings.TrimRight(v, "/")
	}
	type gwConf struct {
		PublicBase string `yaml:"publicBase"`
	}
	var gw gwConf
	if err := confload.ReadAppConfigResolved(repoRoot, "gateway/task-gateway", &gw); err != nil {
		return ""
	}
	return strings.TrimRight(strings.TrimSpace(gw.PublicBase), "/")
}

// loadTaskAIEndpointConf mirrors Django settings: env > conf/ai/task-ai-endpoint > defaults.
func loadTaskAIEndpointConf(repoRoot string) {
	type aiEPConf struct {
		Enabled       bool   `yaml:"enabled"`
		PublicBaseURL string `yaml:"publicBaseUrl"`
		Host          string `yaml:"host"`
		Port          int    `yaml:"port"`
	}
	var ep aiEPConf
	_ = confload.ReadAppConfigResolved(repoRoot, "ai/task-ai-endpoint", &ep)

	cfg.TaskAIEndpointEnabled = ep.Enabled
	if env := strings.TrimSpace(os.Getenv("TASK_AI_ENDPOINT_ENABLED")); env != "" {
		cfg.TaskAIEndpointEnabled = env == "1" || strings.EqualFold(env, "true") || strings.EqualFold(env, "yes")
	}

	base := strings.TrimSpace(ep.PublicBaseURL)
	if base == "" {
		host := strings.TrimSpace(ep.Host)
		if host == "" || host == "0.0.0.0" {
			host = "127.0.0.1"
		}
		port := ep.Port
		if port == 0 {
			port = 8013
		}
		base = fmt.Sprintf("http://%s:%d", host, port)
	}
	if env := strings.TrimSpace(os.Getenv("TASK_AI_ENDPOINT_PUBLIC_BASE")); env != "" {
		base = env
	}
	cfg.TaskAIEndpointPublicBase = strings.TrimRight(base, "/")
}
