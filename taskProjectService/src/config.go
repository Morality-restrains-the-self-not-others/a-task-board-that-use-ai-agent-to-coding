package main

import (
	"confload"
	"dbload"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"gatewayauth"
)

type Config struct {
	Host                    string
	Port                    int
	DBPath                  string
	TaskAuthURL             string
	TaskTenantURL           string
	InternalSecret          string
	GatewayInternalSecret   string
	GitoauthBaseURL         string
	GitoauthBridgeSecret    string
	GitoauthTimeoutSeconds  int
	GitlabAdminPrivateToken string
	GitlabAPIBase           string
	KafkaBootstrapServers   string
}

var (
	cfg              Config
	providerResolver *ProviderResolver
	monorepoRoot     string // 缓存用于运行时重载配置
)

var (
	defaultGitLabHostOnce sync.Once
	defaultGitLabHostVal  string
)

// defaultGitLabHost returns the host of the primary GitLab instance, i.e. the
// one whose `_gitlab_session` browser cookie the API receives. Regional /
// self-hosted instances have their own host, so their requests must never
// borrow this cookie (OPT-20260818-044). Empty when the template cannot be
// resolved (unit tests / missing base.yaml) — callers then skip cookie auth.
func defaultGitLabHost() string {
	defaultGitLabHostOnce.Do(func() {
		subs := confload.ResolveBaseYaml(monorepoRoot)
		origin := confload.ResolveTemplate("${scheme}://${subdomains.gitlab}", subs)
		if u, err := url.Parse(strings.TrimSpace(origin)); err == nil && u.Host != "" {
			defaultGitLabHostVal = u.Host
		}
	})
	return defaultGitLabHostVal
}

func findMonorepoRoot() (string, error) {
	return confload.FindMonorepoRoot()
}

func loadConfig(repoRoot string) {
	monorepoRoot = repoRoot
	type baseConf struct {
		Services struct {
			TaskProjectService struct {
				Host string `yaml:"host"`
				Port int    `yaml:"port"`
			} `yaml:"taskProjectService"`
			TaskAuth struct {
				Host string `yaml:"host"`
				Port int    `yaml:"port"`
			} `yaml:"taskAuth"`
			TaskTenantService struct {
				Host string `yaml:"host"`
				Port int    `yaml:"port"`
			} `yaml:"taskTenantService"`
		} `yaml:"services"`
		Shared struct {
			InternalSecret string `yaml:"internalSecret"`
		} `yaml:"shared"`
		GitoauthBridgeSecret   string `yaml:"gitoauth_bridge_secret"`
		GitoauthTimeoutSeconds int    `yaml:"gitoauth_timeout_seconds"`
	}

	var bc baseConf
	if err := confload.ReadAppConfigResolved(repoRoot, "taskProjectService", &bc); err != nil {
		log.Fatalf("[taskProjectService] confload: %v", err)
	}

	cfg.Host = bc.Services.TaskProjectService.Host
	cfg.Port = bc.Services.TaskProjectService.Port
	if cfg.Host == "" {
		cfg.Host = "0.0.0.0"
	}
	if cfg.Port == 0 {
		cfg.Port = 8016
	}
	cfg.TaskAuthURL = "http://" + bc.Services.TaskAuth.Host + ":" + itoa(bc.Services.TaskAuth.Port)
	tenantHost := strings.TrimSpace(bc.Services.TaskTenantService.Host)
	tenantPort := bc.Services.TaskTenantService.Port
	if tenantHost == "" {
		tenantHost = "127.0.0.1"
	}
	if tenantHost == "0.0.0.0" {
		tenantHost = "127.0.0.1"
	}
	if tenantPort == 0 {
		tenantPort = 8020
	}
	cfg.TaskTenantURL = "http://" + tenantHost + ":" + itoa(tenantPort)
	if v := strings.TrimSpace(os.Getenv("TASK_TENANT_SERVICE_URL")); v != "" {
		cfg.TaskTenantURL = strings.TrimRight(v, "/")
	}
	cfg.InternalSecret = bc.Shared.InternalSecret
	cfg.GatewayInternalSecret = gatewayauth.LoadGatewayInternalSecret(repoRoot)
	dbPath, err := dbload.ResolveMySQLDSN("task-project", repoRoot)
	if err != nil {
		log.Fatalf("[taskProjectService] db path: %v", err)
	}
	cfg.DBPath = dbPath
	cfg.GitoauthBaseURL = strings.TrimSpace(os.Getenv("GITOAUTH_BASE_URL"))
	if cfg.GitoauthBaseURL == "" {
		cfg.GitoauthBaseURL = "http://127.0.0.1:8002"
	}
	cfg.GitoauthBridgeSecret = strings.TrimSpace(os.Getenv("GITOAUTH_BRIDGE_JWT_SECRET"))
	if cfg.GitoauthBridgeSecret == "" {
		cfg.GitoauthBridgeSecret = strings.TrimSpace(os.Getenv("TASK2APP_SSO_JWT_SECRET"))
	}
	if cfg.GitoauthBridgeSecret == "" {
		// conf 注入：taskProjectService/config.yaml gitoauth_bridge_secret（SSOT
		// conf/core/sso/config.yaml ssoJwtSecret，OPT-20260807-071）。兜底 dev 密钥
		// 与 taskGitOauth 兜底不一致会致 access-for-user 401 → token_error。
		cfg.GitoauthBridgeSecret = strings.TrimSpace(bc.GitoauthBridgeSecret)
	}
	if cfg.GitoauthBridgeSecret == "" {
		cfg.GitoauthBridgeSecret = "task2app-local-sso-bridge-dev-do-not-use-in-prod"
	}
	cfg.GitlabAdminPrivateToken = strings.TrimSpace(os.Getenv("GITLAB_ADMIN_PRIVATE_TOKEN"))
	cfg.GitlabAPIBase = strings.TrimRight(strings.TrimSpace(os.Getenv("GITLAB_API_BASE")), "/")
	if cfg.GitlabAPIBase == "" {
		cfg.GitlabAPIBase = "http://127.0.0.1:8012"
	}
	cfg.GitoauthTimeoutSeconds = bc.GitoauthTimeoutSeconds
	applyGitoauthTimeout(cfg.GitoauthTimeoutSeconds)
	cfg.KafkaBootstrapServers = loadKafkaBootstrap(repoRoot)

	resolver, err := loadProviderConfigs(repoRoot)
	if err != nil {
		log.Printf("[taskProjectService] WARN provider configs: %v", err)
		providerResolver = &ProviderResolver{}
	} else {
		providerResolver = resolver
	}

	loadFanyiAgentConfig(repoRoot)

	log.Printf("[taskProjectService] config: host=%s port=%d auth=%s tenant=%s gitoauth=%s",
		cfg.Host, cfg.Port, cfg.TaskAuthURL, cfg.TaskTenantURL, cfg.GitoauthBaseURL)
}

// applyGitoauthTimeout wires conf/taskProjectService/config.yaml 的
// gitoauth_timeout_seconds 到共享 gitHTTPClient，使运维可改 conf 而非重编译
// （OPT-20260817-036）。0 表示未配置，保持默认不覆盖；低于下限 10s 时钳到 10s，
// 防止误配过短超时把 OAuth Refresh 正常往返打成 timeout。
func applyGitoauthTimeout(seconds int) {
	if seconds <= 0 {
		return
	}
	if seconds < 10 {
		seconds = 10
	}
	gitHTTPClient.Timeout = time.Duration(seconds) * time.Second
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

// assertDjangoInternalAPIIsLoopback 校验 Django 内部 API URL 仅指向本机回环地址。
// Django saas-backend 已于 2026-07-30 退役（OPT-049）；此守卫防止内部调用
// 误指向公网网关（api.daydaymoney.com 等），仅允许回环地址。
func assertDjangoInternalAPIIsLoopback(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid django internal api url %q: %v", raw, err)
	}
	host := u.Hostname()
	if host == "127.0.0.1" || host == "localhost" || host == "::1" {
		return nil
	}
	return fmt.Errorf("django internal api url %q must point to loopback address", raw)
}

func loadKafkaBootstrap(repoRoot string) string {
	type kafkaConf struct {
		Kafka struct {
			BootstrapServers string `yaml:"bootstrapServers"`
		} `yaml:"kafka"`
	}
	var kc kafkaConf
	if err := confload.ReadAppConfigResolved(repoRoot, "infra/docker-infra", &kc); err != nil {
		if err2 := confload.ReadAppConfigResolved(repoRoot, "events/domain-events", &kc); err2 != nil {
			return ""
		}
	}
	return kc.Kafka.BootstrapServers
}
