package main

import (
	"os"
	"strconv"
	"strings"

	"confload"
)

type timeoutConfig struct {
	HeartbeatSec        float64
	ExchangeRefreshSec  float64
	RefreshAccessSec    float64
	DefaultSec          float64
}

type serviceConfig struct {
	Host                      string
	Port                      int
	TaskCredentialServiceURL  string
	TaskCloudServiceURL       string
	InternalSecret            string
	Timeouts                  timeoutConfig
}

var cfg serviceConfig

func initConfig(repoRoot string) {
	cfg = serviceConfig{
		Host:              strings.TrimSpace(os.Getenv("TASK_AGENT_SUPPORT_HOST")),
		Port:              envInt("TASK_AGENT_SUPPORT_PORT", 8011),
		TaskCredentialServiceURL: strings.TrimRight(strings.TrimSpace(os.Getenv("TASK_AGENT_SUPPORT_CREDENTIAL_SERVICE_URL")), "/"),
		TaskCloudServiceURL:      strings.TrimRight(strings.TrimSpace(os.Getenv("TASK_AGENT_SUPPORT_CLOUD_SERVICE_URL")), "/"),
		InternalSecret:    strings.TrimSpace(os.Getenv("TASK_AGENT_SUPPORT_INTERNAL_SECRET")),
		Timeouts: timeoutConfig{
			// 须覆盖 Cloud 下行 probe（~2.5s）+ Kafka SSE；过短会导致心跳 502 且前端永久 idle
			HeartbeatSec:       15,
			ExchangeRefreshSec: 15,
			RefreshAccessSec:   15,
			DefaultSec:         30,
		},
	}
	if cfg.Host == "" {
		cfg.Host = "127.0.0.1"
	}
	if cfg.TaskCredentialServiceURL == "" {
		cfg.TaskCredentialServiceURL = "http://127.0.0.1:8015"
	}
	if cfg.TaskCloudServiceURL == "" {
		cfg.TaskCloudServiceURL = "http://127.0.0.1:8018"
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

func applyPortConfig(repoRoot string) {
	var block struct {
		Host                  string  `yaml:"host"`
		Port                  int     `yaml:"port"`
		TaskCredentialServiceURL *string `yaml:"taskCredentialServiceUrl"`
		InternalSecret        *string `yaml:"internalSecret"`
		Timeouts              struct {
			HeartbeatSec       *float64 `yaml:"heartbeatSec"`
			ExchangeRefreshSec *float64 `yaml:"exchangeRefreshSec"`
			RefreshAccessSec   *float64 `yaml:"refreshAccessSec"`
			DefaultSec         *float64 `yaml:"defaultSec"`
		} `yaml:"timeouts"`
	}
	if err := confload.ReadAppConfig(repoRoot, "task-agent-support", &block); err != nil {
		return
	}
	if block.Host != "" {
		cfg.Host = block.Host
	}
	if block.Port > 0 {
		cfg.Port = block.Port
	}
	if block.TaskCredentialServiceURL != nil {
		cfg.TaskCredentialServiceURL = strings.TrimRight(strings.TrimSpace(*block.TaskCredentialServiceURL), "/")
	}
	if block.InternalSecret != nil {
		cfg.InternalSecret = strings.TrimSpace(*block.InternalSecret)
	}
	to := block.Timeouts
	if to.HeartbeatSec != nil {
		cfg.Timeouts.HeartbeatSec = *to.HeartbeatSec
	}
	if to.ExchangeRefreshSec != nil {
		cfg.Timeouts.ExchangeRefreshSec = *to.ExchangeRefreshSec
	}
	if to.RefreshAccessSec != nil {
		cfg.Timeouts.RefreshAccessSec = *to.RefreshAccessSec
	}
	if to.DefaultSec != nil {
		cfg.Timeouts.DefaultSec = *to.DefaultSec
	}
}

func timeoutForAction(action string) float64 {
	switch action {
	case "heartbeat":
		return cfg.Timeouts.HeartbeatSec
	case "exchange-refresh":
		return cfg.Timeouts.ExchangeRefreshSec
	case "refresh-access":
		return cfg.Timeouts.RefreshAccessSec
	default:
		return cfg.Timeouts.DefaultSec
	}
}
