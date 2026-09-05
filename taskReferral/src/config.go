package main

import (
	"log"
	"os"
	"strconv"
	"strings"

	"confload"
	dbload "dbload"
)

type serviceConfig struct {
	Host                 string
	Port                 int
	MySQLDSN             string
	AuthMySQLDSN         string
	BillMySQLDSN         string
	BillServiceURL       string
	TenantServiceURL     string
	BillInternalSecret   string
	TenantInternalSecret string
}

var cfg serviceConfig

func loadConfig(repoRoot string) {
	var err error
	cfg = serviceConfig{
		Host: "0.0.0.0",
		Port: 8025,
	}

	cfg.MySQLDSN, err = dbload.ResolveMySQLDSN("task-referral", repoRoot)
	if err != nil {
		log.Fatalf("[taskReferral] resolve MySQL DSN: %v", err)
	}
	cfg.AuthMySQLDSN, err = dbload.ResolveMySQLDSN("task-auth", repoRoot)
	if err != nil {
		log.Fatalf("[taskReferral] resolve auth MySQL DSN: %v", err)
	}
	cfg.BillMySQLDSN, err = dbload.ResolveMySQLDSN("task-bill", repoRoot)
	if err != nil {
		log.Fatalf("[taskReferral] resolve bill MySQL DSN: %v", err)
	}

	if v := strings.TrimSpace(os.Getenv("TASK_REFERRAL_HOST")); v != "" {
		cfg.Host = v
	}
	if v := strings.TrimSpace(os.Getenv("TASK_REFERRAL_PORT")); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			cfg.Port = p
		}
	}
	if v := strings.TrimSpace(os.Getenv("TASK_REFERRAL_MYSQL_DSN")); v != "" {
		cfg.MySQLDSN = v
	}
	if v := strings.TrimSpace(os.Getenv("TASK_REFERRAL_AUTH_MYSQL_DSN")); v != "" {
		cfg.AuthMySQLDSN = v
	}

	var block struct {
		Host             string `yaml:"host"`
		Port             int    `yaml:"port"`
		BillServiceURL   string `yaml:"billServiceUrl"`
		TenantServiceURL string `yaml:"tenantServiceUrl"`
	}
	if err := confload.ReadAppConfig(repoRoot, "task-referral", &block); err == nil {
		if block.Host != "" {
			cfg.Host = block.Host
		}
		if block.Port > 0 {
			cfg.Port = block.Port
		}
		if strings.TrimSpace(block.BillServiceURL) != "" {
			cfg.BillServiceURL = strings.TrimRight(strings.TrimSpace(block.BillServiceURL), "/")
		}
		if strings.TrimSpace(block.TenantServiceURL) != "" {
			cfg.TenantServiceURL = strings.TrimRight(strings.TrimSpace(block.TenantServiceURL), "/")
		}
	}
	if v := strings.TrimSpace(os.Getenv("TASK_REFERRAL_BILL_URL")); v != "" {
		cfg.BillServiceURL = strings.TrimRight(v, "/")
	}
	if v := strings.TrimSpace(os.Getenv("TASK_REFERRAL_TENANT_URL")); v != "" {
		cfg.TenantServiceURL = strings.TrimRight(v, "/")
	}
	if cfg.BillServiceURL == "" {
		cfg.BillServiceURL = "http://127.0.0.1:8004"
	}
	if cfg.TenantServiceURL == "" {
		cfg.TenantServiceURL = "http://127.0.0.1:8020"
	}
	cfg.BillInternalSecret = loadBillInternalSecret(repoRoot)
	if cfg.BillInternalSecret == "" {
		log.Printf("[taskReferral] WARN bill internal secret empty; taskBill sync-edge will 403 when internalSecret is set")
	}
	cfg.TenantInternalSecret = strings.TrimSpace(os.Getenv("TASK_TENANT_INTERNAL_SECRET"))
	if cfg.TenantInternalSecret == "" {
		cfg.TenantInternalSecret = strings.TrimSpace(os.Getenv("SHARED_INTERNAL_SECRET"))
	}
}

// loadBillInternalSecret 优先环境变量，否则读本目录 conf-sync 片段 task-bill.yaml
// （规则 29：禁止运行时直读 conf/billing/task-bill）。
func loadBillInternalSecret(repoRoot string) string {
	if v := strings.TrimSpace(os.Getenv("TASKBILL_INTERNAL_SECRET")); v != "" {
		return v
	}
	var frag struct {
		InternalSecret string `yaml:"internalSecret"`
	}
	if err := confload.ReadAppFragment(repoRoot, "task-referral", "task-bill.yaml", &frag); err != nil {
		return ""
	}
	return strings.TrimSpace(frag.InternalSecret)
}
