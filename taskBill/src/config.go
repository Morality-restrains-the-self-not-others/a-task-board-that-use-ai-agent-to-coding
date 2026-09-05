package main

import (
	"log"
	"os"
	"strconv"
	"strings"

	"confload"
	dbload "dbload"
	"gatewayauth"
)

type Config struct {
	Host                       string
	Port                       int
	InternalSecret             string
	GatewayInternalSecret      string // 经 gatewayauth.LoadGatewayInternalSecret 从网关配置注入
	TaskAuthBaseURL            string
	TaskAuthInternalSecret     string
	KycGateMode                string // live | mock | off
	SmsGateMode                string // live | off (default: live)
	TaskProjectServiceBase     string
	TaskTaskServiceBase        string
	GitlabAdminPrivateToken    string
	GitlabAPIBase              string
	KafkaBootstrapServers      string
	TaskTenantServiceURL       string
	TaskReferralBaseURL        string
	TaskReferralInternalSecret string
	TaskSseURL                 string
	TaskSseSecret              string
	MySQLDSN                   string
	// OPT-20260823-013: 未映射 tenant-{id} 组的 Git 仓由 fail-open 改为可配置拒绝。
	// 默认 false 保持 fail-open（个人命名空间不受影响）；盘点+迁组完成后可置 true。
	TrafficGateUnmappedReject bool
}

var cfg Config

func loadConfig(repoRoot string) {
	mysqlDSN, err := dbload.ResolveMySQLDSN("task-bill", repoRoot)
	if err != nil {
		log.Fatalf("[taskBill] MySQL DSN: %v", err)
	}
	cfg = Config{
		Host:                       envOr("TASKBILL_HOST", "0.0.0.0"),
		Port:                       envIntOr("TASKBILL_PORT", 8004),
		InternalSecret:             strings.TrimSpace(os.Getenv("TASKBILL_INTERNAL_SECRET")),
		TaskAuthBaseURL:            envOr("TASKAUTH_BASE_URL", "http://127.0.0.1:8003"),
		TaskAuthInternalSecret:     strings.TrimSpace(os.Getenv("TASKAUTH_INTERNAL_SECRET")),
		KycGateMode:                strings.ToLower(envOr("TASKBILL_KYC_GATE", "live")),
		SmsGateMode:                strings.ToLower(envOr("TASKBILL_SMS_GATE", "off")),
		TaskProjectServiceBase:     envOr("TASKBILL_TASK_PROJECT_SERVICE_BASE", "http://127.0.0.1:8016"),
		TaskTaskServiceBase:        envOr("TASKTASK_SERVICE_URL", "http://127.0.0.1:8017"),
		GitlabAdminPrivateToken:    strings.TrimSpace(os.Getenv("GITLAB_ADMIN_PRIVATE_TOKEN")),
		GitlabAPIBase:              envOr("GITLAB_API_BASE", "http://127.0.0.1:8012"),
		KafkaBootstrapServers:      envOr("KAFKA_BOOTSTRAP_SERVERS", "localhost:9093"),
		TaskTenantServiceURL:       envOr("TASKBILL_TENANT_SERVICE_URL", "http://127.0.0.1:8020"),
		TaskReferralBaseURL:        envOr("TASKREFERRAL_BASE_URL", "http://127.0.0.1:8025"),
		TaskReferralInternalSecret: firstNonEmptyEnv("TASK_REFERRAL_INTERNAL_SECRET", "SHARED_INTERNAL_SECRET"),
		TaskSseURL:                 envOr("TASK_SSE_URL", ""),
		TaskSseSecret:              strings.TrimSpace(os.Getenv("TASK_SSE_SECRET")),
		MySQLDSN:                   mysqlDSN,
		TrafficGateUnmappedReject:  envBoolOr("TASKBILL_TRAFFIC_GATE_UNMAPPED_REJECT", false),
	}
	// 遵循项目标准模式：从网关配置读取 GatewayInternalSecret（env 优先，conf/gateway/task-gateway 兜底）。
	// taskCloudService / taskProjectService / taskTaskService / taskTenantService / taskAIComment 同理。
	cfg.GatewayInternalSecret = gatewayauth.LoadGatewayInternalSecret(repoRoot)

	var block struct {
		Host                       string `yaml:"host"`
		Port                       int    `yaml:"port"`
		InternalSecret             string `yaml:"internalSecret"`
		TaskAuthBaseURL            string `yaml:"taskAuthBaseUrl"`
		TaskAuthInternalSecret     string `yaml:"taskAuthInternalSecret"`
		KycGateMode                string `yaml:"kycGateMode"`
		SmsGateMode                string `yaml:"smsGateMode"`
		TaskProjectServiceBase     string `yaml:"taskProjectServiceBase"`
		TaskTaskServiceBase        string `yaml:"taskTaskServiceBase"`
		GitlabAdminPrivateToken    string `yaml:"gitlabAdminPrivateToken"`
		GitlabAPIBase              string `yaml:"gitlabApiBase"`
		KafkaBootstrapServers      string `yaml:"kafkaBootstrapServers"`
		TaskTenantServiceURL       string `yaml:"taskTenantServiceUrl"`
		TaskReferralBaseURL        string `yaml:"taskReferralBaseUrl"`
		TaskReferralInternalSecret string `yaml:"taskReferralInternalSecret"`
		TaskSseURL                 string `yaml:"taskSseUrl"`
		TaskSseSecret              string `yaml:"taskSseSecret"`
		TrafficGateUnmappedReject  bool   `yaml:"trafficGateUnmappedReject"`
	}
	if err := confload.ReadAppConfig(repoRoot, "task-bill", &block); err != nil {
		normalizeKycGateMode()
		return
	}
	if block.Host != "" {
		cfg.Host = block.Host
	}
	if block.Port != 0 {
		cfg.Port = block.Port
	}
	if block.InternalSecret != "" {
		cfg.InternalSecret = block.InternalSecret
	}
	if block.TaskAuthBaseURL != "" {
		cfg.TaskAuthBaseURL = block.TaskAuthBaseURL
	}
	if block.TaskAuthInternalSecret != "" {
		cfg.TaskAuthInternalSecret = block.TaskAuthInternalSecret
	}
	if block.KycGateMode != "" {
		cfg.KycGateMode = strings.ToLower(strings.TrimSpace(block.KycGateMode))
	}
	if block.SmsGateMode != "" {
		cfg.SmsGateMode = strings.ToLower(strings.TrimSpace(block.SmsGateMode))
	}
	if block.TaskProjectServiceBase != "" {
		cfg.TaskProjectServiceBase = block.TaskProjectServiceBase
	}
	if block.TaskTaskServiceBase != "" {
		cfg.TaskTaskServiceBase = block.TaskTaskServiceBase
	}
	if block.GitlabAdminPrivateToken != "" {
		cfg.GitlabAdminPrivateToken = block.GitlabAdminPrivateToken
	}
	if block.GitlabAPIBase != "" {
		cfg.GitlabAPIBase = block.GitlabAPIBase
	}
	if block.KafkaBootstrapServers != "" {
		cfg.KafkaBootstrapServers = block.KafkaBootstrapServers
	}
	if block.TaskTenantServiceURL != "" {
		cfg.TaskTenantServiceURL = block.TaskTenantServiceURL
	}
	if block.TaskReferralBaseURL != "" {
		cfg.TaskReferralBaseURL = block.TaskReferralBaseURL
	}
	if block.TaskReferralInternalSecret != "" {
		cfg.TaskReferralInternalSecret = block.TaskReferralInternalSecret
	}
	if block.TaskSseURL != "" {
		cfg.TaskSseURL = block.TaskSseURL
	}
	if block.TaskSseSecret != "" {
		cfg.TaskSseSecret = block.TaskSseSecret
	}
	cfg.TrafficGateUnmappedReject = block.TrafficGateUnmappedReject
	normalizeKycGateMode()
}

func normalizeKycGateMode() {
	switch strings.ToLower(strings.TrimSpace(cfg.KycGateMode)) {
	case "live", "mock", "off":
		cfg.KycGateMode = strings.ToLower(strings.TrimSpace(cfg.KycGateMode))
	default:
		cfg.KycGateMode = "live"
	}
}

func envOr(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func firstNonEmptyEnv(keys ...string) string {
	for _, key := range keys {
		if v := strings.TrimSpace(os.Getenv(key)); v != "" {
			return v
		}
	}
	return ""
}

func envIntOr(key string, def int) int {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func envBoolOr(key string, def bool) bool {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		switch strings.ToLower(v) {
		case "1", "true", "yes", "on":
			return true
		case "0", "false", "no", "off":
			return false
		}
	}
	return def
}

func findMonorepoRoot() (string, error) {
	return confload.FindMonorepoRoot()
}
