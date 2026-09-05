package main

import (
	"os"
	"strconv"
	"strings"

	"confload"
)

type serviceConfig struct {
	Host                string
	Port                int
	InternalSecret      string // deprecated Django AI-endpoint secret
	UpstreamTimeoutSec  float64
	CloudServiceURL     string
	CloudInternalSecret string
	CredentialServiceURL string
}

var cfg serviceConfig

func initConfig(repoRoot string) {
	cfg = serviceConfig{
		Host:                 strings.TrimSpace(os.Getenv("TASK_AI_ENDPOINT_HOST")),
		Port:                 envInt("TASK_AI_ENDPOINT_PORT", 8013),
		InternalSecret:       strings.TrimSpace(os.Getenv("TASK_AI_ENDPOINT_INTERNAL_SECRET")),
		UpstreamTimeoutSec:   envFloat("TASK_AI_ENDPOINT_UPSTREAM_TIMEOUT_SEC", 300),
		CloudServiceURL:      strings.TrimRight(strings.TrimSpace(os.Getenv("TASK_CLOUD_SERVICE_BASE_URL")), "/"),
		CloudInternalSecret:  strings.TrimSpace(os.Getenv("TASK_CLOUD_INTERNAL_SECRET")),
		CredentialServiceURL: strings.TrimRight(strings.TrimSpace(os.Getenv("TASK_CREDENTIAL_SERVICE_BASE_URL")), "/"),
	}
	if cfg.Host == "" {
		cfg.Host = "0.0.0.0"
	}
	if cfg.CloudServiceURL == "" {
		cfg.CloudServiceURL = "http://127.0.0.1:8018"
	}
	if cfg.CredentialServiceURL == "" {
		cfg.CredentialServiceURL = "http://127.0.0.1:8015"
	}
	applyPortConfig(repoRoot)
}

func envInt(key string, defaultVal int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(raw)
	if err != nil || val <= 0 {
		return defaultVal
	}
	return val
}

func envFloat(key string, defaultVal float64) float64 {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return defaultVal
	}
	val, err := strconv.ParseFloat(raw, 64)
	if err != nil || val <= 0 {
		return defaultVal
	}
	return val
}

func applyPortConfig(repoRoot string) {
	var block struct {
		Host                  string   `yaml:"host"`
		Port                  int      `yaml:"port"`
		InternalSecret        *string  `yaml:"internalSecret"`
		UpstreamTimeoutSec    *float64 `yaml:"upstreamTimeoutSec"`
		TaskCloudService      *struct {
			URL            string `yaml:"url"`
			InternalSecret string `yaml:"internalSecret"`
		} `yaml:"taskCloudService"`
		TaskCredentialService *struct {
			URL string `yaml:"url"`
		} `yaml:"taskCredentialService"`
	}
	if err := confload.ReadAppConfig(repoRoot, "task-ai-endpoint", &block); err != nil {
		return
	}
	if block.Host != "" {
		cfg.Host = block.Host
	}
	if block.Port > 0 {
		cfg.Port = block.Port
	}
	if block.InternalSecret != nil {
		cfg.InternalSecret = strings.TrimSpace(*block.InternalSecret)
	}
	if block.UpstreamTimeoutSec != nil && *block.UpstreamTimeoutSec > 0 {
		cfg.UpstreamTimeoutSec = *block.UpstreamTimeoutSec
	}
	if block.TaskCloudService != nil {
		if u := strings.TrimSpace(block.TaskCloudService.URL); u != "" {
			cfg.CloudServiceURL = strings.TrimRight(u, "/")
		}
		// Allow empty secret (matches Cloud when shared InternalSecret unset).
		if os.Getenv("TASK_CLOUD_INTERNAL_SECRET") == "" {
			cfg.CloudInternalSecret = strings.TrimSpace(block.TaskCloudService.InternalSecret)
		}
	}
	if block.TaskCredentialService != nil {
		if u := strings.TrimSpace(block.TaskCredentialService.URL); u != "" && os.Getenv("TASK_CREDENTIAL_SERVICE_BASE_URL") == "" {
			cfg.CredentialServiceURL = strings.TrimRight(u, "/")
		}
	}
}
