package main

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"confload"
	dbload "dbload"
)

// OidcBootstrapClient represents a single OIDC bootstrap client configuration.
type OidcBootstrapClient struct {
	ClientID     string `yaml:"clientId"`
	ClientSecret string `yaml:"clientSecret"`
	RedirectURI  string `yaml:"redirectUri"`
}

type Config struct {
	Host                  string
	Port                  int
	InternalSecret        string
	UnsubscribeHmacSecret string
	SSOJwtSecret          string
	MySQLDSN              string
	UserContentTypeID     int64
	GatewayPublicBase     string
	FrontendBase          string
	AiProviderPublicBase  string
	OidcSigningKeyPath    string
	OidcAccessTokenTTL    int64
	OidcIDTokenTTL        int64
	OidcRefreshTokenTTL   int64
	OidcIssuer            string
	// Bootstrap clients (supports multiple; backward-compatible with legacy single-client)
	OidcBootstrapClients            []OidcBootstrapClient
	GitServicePublicBase            string // GitLab external URL for SLO redirect (e.g. http://<gitlab-host>:8012)
	PostLogoutOrigins               []string
	KafkaBootstrapServers           string
	TenantServiceURL                string // taskTenantService internal API base URL
	BillServiceURL                  string // taskBill internal API base URL
	CloudServiceURL                 string // taskCloudService internal API base URL
	ReferralServiceURL              string // taskReferral internal API base URL
	GitOauthServiceURL              string // taskGitOauth internal API（同节点 Path A 连接查询）
	GitOauthBridgeSecret            string // X-GitOauth-Bridge-Secret
	BillInternalSecret              string // X-TaskBill-Internal-Secret when calling taskBill
	ReferralInternalSecret          string
	AccountDeletionCooldownDays     int
	PersonalDataExportRetentionDays int // 导出快照有效天数（默认 7）
	// SMS：本目录 sync 产物 conf/auth/task-auth/sms.yaml（源 conf/core/sms/config.yaml；OPT-20260806-057: django 目录退役）
	SMSProvider                       string
	SMSAliyunAccessKeyID              string
	SMSAliyunAccessKeySecret          string
	SMSAliyunRegionID                 string
	SMSAliyunSignName                 string
	SMSAliyunTemplateCodeNotification string
	SMSTemplateCodeVerification       string
	SMSTemplateCodePasswordReset      string
	SMSTemplateCodeNotification       string
	// Email SMTP：本目录 sync 产物 conf/auth/task-auth/email.yaml（源 conf/core/email/config.yaml）
	// Kafka EMAIL_SENT 失败时 publishEmailSent → sendEmailSMTP 回退使用；EMAIL_* 环境变量仍优先。
	EmailHost         string
	EmailPort         int
	EmailHostUser     string
	EmailHostPassword string
	EmailDefaultFrom  string
	EmailUseSSL       bool
	EmailTimeoutSec   int
	nowUnix           func() int64
}

var cfg Config

func loadConfig(repoRoot string) {
	mysqlDSN, err := dbload.ResolveMySQLDSN("task-auth", repoRoot)
	if err != nil {
		log.Fatalf("[taskAuth] MySQL DSN: %v", err)
	}
	cfg = Config{
		Host:                 envOr("TASKAUTH_HOST", "127.0.0.1"),
		Port:                 envIntOr("TASKAUTH_PORT", 8003),
		InternalSecret:       strings.TrimSpace(os.Getenv("TASKAUTH_INTERNAL_SECRET")),
		SSOJwtSecret:         resolveSSOJwtSecret(repoRoot),
		MySQLDSN:             mysqlDSN,
		OidcAccessTokenTTL:   3600,
		OidcIDTokenTTL:       3600,
		OidcRefreshTokenTTL:  2592000, // 30 天；offline_access 场景 refresh token 长于 access token
		OidcSigningKeyPath:   "",
		GitServicePublicBase: envOr("TASKAUTH_GIT_SERVICE_PUBLIC_BASE", ""),
		nowUnix:              func() int64 { return time.Now().Unix() },
	}

	// 验证码 SMS 已迁至 taskAuth：仅读本目录 sync 片段 sms.yaml（由 conf-sync 从 core/sms 同步）
	loadSMSConfigFromSyncedFragment(repoRoot)
	// EMAIL_SENT SMTP 回退：仅读本目录 sync 片段 email.yaml（由 conf-sync 从 core/email 同步）
	loadEmailConfigFromSyncedFragment(repoRoot)

	// Top-level service config block (auto-resolves ${subdomains.xxx} templates)
	var block struct {
		Host                  string `yaml:"host"`
		Port                  int    `yaml:"port"`
		InternalSecret        string `yaml:"internalSecret"`
		UnsubscribeHmacSecret string `yaml:"unsubscribeHmacSecret"`
		GatewayPublicBase     string `yaml:"gatewayPublicBase"`
		FrontendBase          string `yaml:"frontendBase"`
		AiProviderPublicBase  string `yaml:"aiProviderPublicBase"`
		KafkaBootstrapServers string `yaml:"kafkaBootstrapServers"`
	}
	if err := confload.ReadAppConfig(repoRoot, "auth/task-auth", &block); err != nil {
		return
	}
	if block.Host != "" && strings.TrimSpace(os.Getenv("TASKAUTH_HOST")) == "" {
		cfg.Host = block.Host
	}
	// 环境变量显式设置优先于 yaml（OPT-20260808-022：dev 实例 TASKAUTH_PORT=8004 与
	// 生产 8003 并存；yaml port 仅在 env 未设置时兜底）
	if block.Port != 0 && strings.TrimSpace(os.Getenv("TASKAUTH_PORT")) == "" {
		cfg.Port = block.Port
	}
	if block.InternalSecret != "" {
		cfg.InternalSecret = block.InternalSecret
	}
	if strings.TrimSpace(block.UnsubscribeHmacSecret) != "" {
		cfg.UnsubscribeHmacSecret = strings.TrimSpace(block.UnsubscribeHmacSecret)
	}
	// Gateway public base from own config (uses ${subdomains.gateway} — no cross-service read)
	if block.GatewayPublicBase != "" {
		cfg.GatewayPublicBase = strings.TrimRight(strings.TrimSpace(block.GatewayPublicBase), "/")
	}
	// Frontend base for user-facing links (invite emails etc.) — human-friendly domain like www.daydaymoney.com
	if block.FrontendBase != "" {
		cfg.FrontendBase = strings.TrimRight(strings.TrimSpace(block.FrontendBase), "/")
	}
	// AI Provider public base for SSO bridge redirect (uses ${subdomains.provider} — bypasses APISIX)
	if block.AiProviderPublicBase != "" {
		cfg.AiProviderPublicBase = strings.TrimRight(strings.TrimSpace(block.AiProviderPublicBase), "/")
	}
	// Kafka from own config (uses ${subdomains.kafka} — no cross-service read of events/domain-events)
	if strings.TrimSpace(block.KafkaBootstrapServers) != "" {
		cfg.KafkaBootstrapServers = strings.TrimSpace(block.KafkaBootstrapServers)
	}

	// Gateway public base is now read from auth/task-auth's own config block above
	// (gatewayPublicBase: ${subdomains.gateway}) — no cross-service read needed.
	// Fallback: if not set, try task-gateway config for backward compatibility.
	if cfg.GatewayPublicBase == "" {
		var gw struct {
			PublicBase string `yaml:"publicBase"`
		}
		if err := confload.ReadAppConfig(repoRoot, "task-gateway", &gw); err == nil && gw.PublicBase != "" {
			cfg.GatewayPublicBase = strings.TrimRight(strings.TrimSpace(gw.PublicBase), "/")
		}
	}

	// OIDC config block (nested under oidc: key)
	var oidcWrapper struct {
		Oidc *struct {
			SigningKeyPath        string `yaml:"signingKeyPath"`
			AccessTokenTTL        int64  `yaml:"accessTokenTTL"`
			IDTokenTTL            int64  `yaml:"idTokenTTL"`
			RefreshTokenTTL       int64  `yaml:"refreshTokenTTL"`
			Issuer                string `yaml:"issuer"`
			BootstrapClientID     string `yaml:"bootstrapClientId"`
			BootstrapClientSecret string `yaml:"bootstrapClientSecret"`
			BootstrapRedirectURI  string `yaml:"bootstrapRedirectUri"`
			// New: multi-client list (takes precedence over single-client fields)
			BootstrapClients []struct {
				ClientID     string `yaml:"clientId"`
				ClientSecret string `yaml:"clientSecret"`
				RedirectURI  string `yaml:"redirectUri"`
			} `yaml:"bootstrapClients"`
			GitServicePublicBase      string   `yaml:"gitServicePublicBase"`
			PostLogoutRedirectOrigins []string `yaml:"postLogoutRedirectOrigins"`
		} `yaml:"oidc"`
	}
	if err := confload.ReadAppConfig(repoRoot, "auth/task-auth", &oidcWrapper); err == nil && oidcWrapper.Oidc != nil {
		o := oidcWrapper.Oidc
		if o.SigningKeyPath != "" {
			cfg.OidcSigningKeyPath = o.SigningKeyPath
		}
		if o.AccessTokenTTL > 0 {
			cfg.OidcAccessTokenTTL = o.AccessTokenTTL
		}
		if o.IDTokenTTL > 0 {
			cfg.OidcIDTokenTTL = o.IDTokenTTL
		}
		if o.RefreshTokenTTL > 0 {
			cfg.OidcRefreshTokenTTL = o.RefreshTokenTTL
		}
		if o.Issuer != "" {
			cfg.OidcIssuer = o.Issuer
		}
		// Populate bootstrap clients (new list format takes precedence)
		if len(o.BootstrapClients) > 0 {
			for _, bc := range o.BootstrapClients {
				if bc.ClientID != "" && bc.ClientSecret != "" && bc.RedirectURI != "" {
					cfg.OidcBootstrapClients = append(cfg.OidcBootstrapClients, OidcBootstrapClient{
						ClientID:     bc.ClientID,
						ClientSecret: bc.ClientSecret,
						RedirectURI:  bc.RedirectURI,
					})
				}
			}
		}
		// Legacy single-client backward compatibility
		if len(cfg.OidcBootstrapClients) == 0 && o.BootstrapClientID != "" {
			cfg.OidcBootstrapClients = append(cfg.OidcBootstrapClients, OidcBootstrapClient{
				ClientID:     o.BootstrapClientID,
				ClientSecret: o.BootstrapClientSecret,
				RedirectURI:  o.BootstrapRedirectURI,
			})
		}
		if o.GitServicePublicBase != "" {
			cfg.GitServicePublicBase = o.GitServicePublicBase
		}
		for _, origin := range o.PostLogoutRedirectOrigins {
			origin = strings.TrimRight(strings.TrimSpace(origin), "/")
			if origin != "" {
				cfg.PostLogoutOrigins = append(cfg.PostLogoutOrigins, origin)
			}
		}

		// Template variables (${subdomains.xxx}) are resolved automatically by
		// confload.ReadAppConfig — no manual resolution needed.
		log.Printf("[taskAuth] OIDC config loaded: issuer=%s clients=%d postLogoutOrigins=%d", cfg.OidcIssuer, len(cfg.OidcBootstrapClients), len(cfg.PostLogoutOrigins))
	}

	// WeChat 开放平台 OAuth 配置（可选，未配置时不启用微信登录）
	loadWeChatConfig(repoRoot)

	// Kafka bootstrap servers — prefer own config (kafkaBootstrapServers: ${subdomains.kafka}:9092).
	// Falls back to events/domain-events cross-read only if not in own config.
	if cfg.KafkaBootstrapServers == "" {
		var de struct {
			Kafka struct {
				BootstrapServers string `yaml:"bootstrapServers"`
			} `yaml:"kafka"`
		}
		if err := confload.ReadAppConfig(repoRoot, "events/domain-events", &de); err == nil {
			if strings.TrimSpace(de.Kafka.BootstrapServers) != "" {
				cfg.KafkaBootstrapServers = strings.TrimSpace(de.Kafka.BootstrapServers)
			}
		}
		// docker-infra fragment as additional fallback
		if cfg.KafkaBootstrapServers == "" {
			var infra struct {
				Kafka struct {
					BootstrapServers string `yaml:"bootstrapServers"`
				} `yaml:"kafka"`
			}
			if err := confload.ReadAppConfig(repoRoot, "events/domain-events/docker-infra", &infra); err == nil {
				if strings.TrimSpace(infra.Kafka.BootstrapServers) != "" {
					cfg.KafkaBootstrapServers = strings.TrimSpace(infra.Kafka.BootstrapServers)
				}
			}
		}
	}
	// Env var always wins (highest priority).
	if v := strings.TrimSpace(os.Getenv("KAFKA_BOOTSTRAP_SERVERS")); v != "" {
		cfg.KafkaBootstrapServers = v
	}

	// Tenant / bill / cloud / referral downstream URLs (同节点 loopback，拆分时改 ${subdomains.xxx})
	loadTenantServiceURL(repoRoot)
	loadBillAndCloudServiceURLs(repoRoot)
	loadReferralServiceURL(repoRoot)
	loadGitOauthServiceURL(repoRoot)
	loadAccountDeletionConfig(repoRoot)
}

// resolveSSOJwtSecret 解析 SSO bridge 共享密钥（契约 secret=ssoJwtSecret，见 sso_bridge.go）：
// TASK2APP_SSO_JWT_SECRET env → conf/core/sso/config.yaml（真源，与 taskAiProvider 解析链一致）
// → 空（由 ssoBridgeSecret 兜底本地开发默认值）。
func resolveSSOJwtSecret(repoRoot string) string {
	if s := strings.TrimSpace(os.Getenv("TASK2APP_SSO_JWT_SECRET")); s != "" {
		return s
	}
	var sso struct {
		SSOJwtSecret string `yaml:"ssoJwtSecret"`
	}
	if err := confload.ReadAppConfig(repoRoot, "core/sso", &sso); err == nil && strings.TrimSpace(sso.SSOJwtSecret) != "" {
		return strings.TrimSpace(sso.SSOJwtSecret)
	}
	return ""
}

// loadSMSConfigFromSyncedFragment 读取 conf/auth/task-auth/sms.yaml（sync 产物）。
// 环境变量仍优先于 YAML（与 Django settings._env_then_port_str 一致）。
func loadSMSConfigFromSyncedFragment(repoRoot string) {
	var wrap struct {
		SMS *struct {
			Provider string `yaml:"provider"`
			Aliyun   *struct {
				AccessKeyID              string `yaml:"access_key_id"`
				AccessKeySecret          string `yaml:"access_key_secret"`
				RegionID                 string `yaml:"region_id"`
				SignName                 string `yaml:"sign_name"`
				TemplateCodeNotification string `yaml:"template_code_notification"`
			} `yaml:"aliyun"`
			Templates *struct {
				Verification  string `yaml:"verification"`
				PasswordReset string `yaml:"password_reset"`
				Notification  string `yaml:"notification"`
			} `yaml:"templates"`
		} `yaml:"sms"`
	}
	if err := confload.ReadAppFragment(repoRoot, "auth/task-auth", "sms.yaml", &wrap); err != nil || wrap.SMS == nil {
		return
	}
	s := wrap.SMS
	cfg.SMSProvider = strings.TrimSpace(s.Provider)
	if s.Aliyun != nil {
		cfg.SMSAliyunAccessKeyID = strings.TrimSpace(s.Aliyun.AccessKeyID)
		cfg.SMSAliyunAccessKeySecret = strings.TrimSpace(s.Aliyun.AccessKeySecret)
		cfg.SMSAliyunRegionID = strings.TrimSpace(s.Aliyun.RegionID)
		cfg.SMSAliyunSignName = strings.TrimSpace(s.Aliyun.SignName)
		cfg.SMSAliyunTemplateCodeNotification = strings.TrimSpace(s.Aliyun.TemplateCodeNotification)
	}
	if s.Templates != nil {
		cfg.SMSTemplateCodeVerification = strings.TrimSpace(s.Templates.Verification)
		cfg.SMSTemplateCodePasswordReset = strings.TrimSpace(s.Templates.PasswordReset)
		cfg.SMSTemplateCodeNotification = strings.TrimSpace(s.Templates.Notification)
	}
	// 与 Django settings 一致：业务模板为空时回退阿里云 notification 模板
	aliyunTpl := cfg.SMSAliyunTemplateCodeNotification
	if aliyunTpl != "" {
		if cfg.SMSTemplateCodeVerification == "" {
			cfg.SMSTemplateCodeVerification = aliyunTpl
		}
		if cfg.SMSTemplateCodePasswordReset == "" {
			cfg.SMSTemplateCodePasswordReset = aliyunTpl
		}
		if cfg.SMSTemplateCodeNotification == "" {
			cfg.SMSTemplateCodeNotification = aliyunTpl
		}
	}
	log.Printf("[taskAuth] SMS config from auth/task-auth/sms.yaml: provider=%s sign=%q verificationTpl=%q",
		cfg.SMSProvider, cfg.SMSAliyunSignName, cfg.SMSTemplateCodeVerification)
}

// loadEmailConfigFromSyncedFragment 读取 conf/auth/task-auth/email.yaml（sync 产物）。
// 环境变量 EMAIL_* 仍优先于 YAML（与 loadSMTPConfig 一致）。
func loadEmailConfigFromSyncedFragment(repoRoot string) {
	var wrap struct {
		Email *struct {
			Host         string `yaml:"host"`
			Port         int    `yaml:"port"`
			HostUser     string `yaml:"host_user"`
			HostPassword string `yaml:"host_password"`
			DefaultFrom  string `yaml:"default_from"`
			UseSSL       bool   `yaml:"use_ssl"`
			Timeout      int    `yaml:"timeout"`
		} `yaml:"email"`
	}
	if err := confload.ReadAppFragment(repoRoot, "auth/task-auth", "email.yaml", &wrap); err != nil || wrap.Email == nil {
		return
	}
	e := wrap.Email
	cfg.EmailHost = strings.TrimSpace(e.Host)
	cfg.EmailPort = e.Port
	cfg.EmailHostUser = strings.TrimSpace(e.HostUser)
	cfg.EmailHostPassword = strings.TrimSpace(e.HostPassword)
	cfg.EmailDefaultFrom = strings.TrimSpace(e.DefaultFrom)
	cfg.EmailUseSSL = e.UseSSL
	cfg.EmailTimeoutSec = e.Timeout
	if cfg.EmailDefaultFrom == "" {
		cfg.EmailDefaultFrom = cfg.EmailHostUser
	}
	log.Printf("[taskAuth] Email SMTP from auth/task-auth/email.yaml: host=%s user=%q from=%q password_configured=%v",
		cfg.EmailHost, cfg.EmailHostUser, cfg.EmailDefaultFrom, cfg.EmailHostPassword != "")
}

func envOr(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func envIntOr(key string, def int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return def
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return def
	}
	return n
}

func findMonorepoRoot() (string, error) {
	return confload.FindMonorepoRoot()
}
