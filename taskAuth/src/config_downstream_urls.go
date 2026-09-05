package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"confload"
)

func loadTenantServiceURL(repoRoot string) {
	if v := strings.TrimSpace(os.Getenv("TASKAUTH_TENANT_SERVICE_URL")); v != "" {
		cfg.TenantServiceURL = v
	}
	if cfg.TenantServiceURL == "" {
		var tenantConf struct {
			Services struct {
				TaskTenantService struct {
					Host string `yaml:"host"`
					Port int    `yaml:"port"`
				} `yaml:"taskTenantService"`
			} `yaml:"services"`
		}
		if err := confload.ReadAppConfig(repoRoot, "taskTenantService", &tenantConf); err == nil {
			host := tenantConf.Services.TaskTenantService.Host
			if host == "" || host == "0.0.0.0" {
				host = "127.0.0.1"
			}
			port := tenantConf.Services.TaskTenantService.Port
			if port == 0 {
				port = 8020
			}
			cfg.TenantServiceURL = "http://" + host + ":" + strconv.Itoa(port)
		}
	}
	if cfg.TenantServiceURL == "" {
		cfg.TenantServiceURL = "http://127.0.0.1:8020"
	}
}

func loadBillAndCloudServiceURLs(repoRoot string) {
	if v := strings.TrimSpace(os.Getenv("TASKAUTH_BILL_SERVICE_URL")); v != "" {
		cfg.BillServiceURL = strings.TrimRight(v, "/")
	}
	if v := strings.TrimSpace(os.Getenv("TASKAUTH_CLOUD_SERVICE_URL")); v != "" {
		cfg.CloudServiceURL = strings.TrimRight(v, "/")
	}
	if cfg.BillServiceURL == "" {
		var billConf struct {
			Host string `yaml:"host"`
			Port int    `yaml:"port"`
		}
		if err := confload.ReadAppConfig(repoRoot, "billing/task-bill", &billConf); err == nil && billConf.Port != 0 {
			host := billConf.Host
			if host == "" || host == "0.0.0.0" {
				host = "127.0.0.1"
			}
			cfg.BillServiceURL = fmt.Sprintf("http://%s:%d", host, billConf.Port)
		}
	}
	if cfg.BillServiceURL == "" {
		cfg.BillServiceURL = "http://127.0.0.1:8004"
	}
	if cfg.CloudServiceURL == "" {
		var cloudConf struct {
			Host string `yaml:"host"`
			Port int    `yaml:"port"`
		}
		if err := confload.ReadAppConfig(repoRoot, "cloud/task-cloud-service", &cloudConf); err == nil && cloudConf.Port != 0 {
			host := cloudConf.Host
			if host == "" || host == "0.0.0.0" {
				host = "127.0.0.1"
			}
			cfg.CloudServiceURL = fmt.Sprintf("http://%s:%d", host, cloudConf.Port)
		}
	}
	if cfg.CloudServiceURL == "" {
		cfg.CloudServiceURL = "http://127.0.0.1:8018"
	}
	if v := strings.TrimSpace(os.Getenv("TASKBILL_INTERNAL_SECRET")); v != "" {
		cfg.BillInternalSecret = v
	}
	if cfg.BillInternalSecret == "" {
		// OPT-20260821-026: 规则 29 — 只读本目录 conf-sync 片段 task-bill.yaml，
		// 禁止直读 conf/billing/task-bill（拆分/同步遗漏时静默丢密钥）。
		var billSec struct {
			InternalSecret string `yaml:"internalSecret"`
		}
		if err := confload.ReadAppFragment(repoRoot, "auth/task-auth", "task-bill.yaml", &billSec); err == nil {
			cfg.BillInternalSecret = strings.TrimSpace(billSec.InternalSecret)
		}
	}
}

func loadReferralServiceURL(repoRoot string) {
	if v := strings.TrimSpace(os.Getenv("TASKAUTH_REFERRAL_SERVICE_URL")); v != "" {
		cfg.ReferralServiceURL = strings.TrimRight(v, "/")
	}
	if cfg.ReferralServiceURL == "" {
		var wrap struct {
			ReferralServiceURL string `yaml:"referralServiceUrl"`
		}
		if err := confload.ReadAppConfig(repoRoot, "auth/task-auth", &wrap); err == nil {
			cfg.ReferralServiceURL = strings.TrimRight(strings.TrimSpace(wrap.ReferralServiceURL), "/")
		}
	}
	if cfg.ReferralServiceURL == "" {
		cfg.ReferralServiceURL = "http://127.0.0.1:8025"
	}
	if v := strings.TrimSpace(os.Getenv("TASK_REFERRAL_INTERNAL_SECRET")); v != "" {
		cfg.ReferralInternalSecret = v
	}
	if cfg.ReferralInternalSecret == "" {
		cfg.ReferralInternalSecret = strings.TrimSpace(os.Getenv("SHARED_INTERNAL_SECRET"))
	}
}

func loadGitOauthServiceURL(repoRoot string) {
	if v := strings.TrimSpace(os.Getenv("TASKAUTH_GITOAUTH_SERVICE_URL")); v != "" {
		cfg.GitOauthServiceURL = strings.TrimRight(v, "/")
	}
	if cfg.GitOauthServiceURL == "" {
		var wrap struct {
			GitOauthServiceURL string `yaml:"gitOauthServiceUrl"`
		}
		if err := confload.ReadAppConfig(repoRoot, "auth/task-auth", &wrap); err == nil {
			cfg.GitOauthServiceURL = strings.TrimRight(strings.TrimSpace(wrap.GitOauthServiceURL), "/")
		}
	}
	if cfg.GitOauthServiceURL == "" {
		cfg.GitOauthServiceURL = "http://127.0.0.1:8002"
	}
	if v := strings.TrimSpace(os.Getenv("TASKAUTH_GITOAUTH_BRIDGE_SECRET")); v != "" {
		cfg.GitOauthBridgeSecret = v
	}
	if cfg.GitOauthBridgeSecret == "" {
		var frag struct {
			SSOJwtSecret string `yaml:"ssoJwtSecret"`
		}
		if err := confload.ReadAppFragment(repoRoot, "auth/task-auth", "git-oauth.yaml", &frag); err == nil {
			cfg.GitOauthBridgeSecret = strings.TrimSpace(frag.SSOJwtSecret)
		}
	}
}

func loadAccountDeletionConfig(repoRoot string) {
	var wrap struct {
		AccountDeletion *struct {
			CooldownDays int `yaml:"cooldownDays"`
		} `yaml:"accountDeletion"`
		PersonalDataExport *struct {
			RetentionDays int `yaml:"retentionDays"`
		} `yaml:"personalDataExport"`
	}
	if err := confload.ReadAppConfig(repoRoot, "auth/task-auth", &wrap); err == nil {
		if wrap.AccountDeletion != nil && wrap.AccountDeletion.CooldownDays > 0 {
			cfg.AccountDeletionCooldownDays = wrap.AccountDeletion.CooldownDays
		}
		if wrap.PersonalDataExport != nil && wrap.PersonalDataExport.RetentionDays > 0 {
			cfg.PersonalDataExportRetentionDays = wrap.PersonalDataExport.RetentionDays
		}
	}
	if cfg.AccountDeletionCooldownDays <= 0 {
		cfg.AccountDeletionCooldownDays = 15
	}
	if cfg.PersonalDataExportRetentionDays <= 0 {
		cfg.PersonalDataExportRetentionDays = 7
	}
}
