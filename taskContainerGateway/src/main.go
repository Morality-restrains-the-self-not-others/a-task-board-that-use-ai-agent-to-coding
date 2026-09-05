package main

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"confload"
	"tracelog"
)

type serviceConfig struct {
	Host                   string
	Port                   int
	InternalSecret         string
	ForwardReadSec         float64
	ForwardConnectSec      float64
	RelayToTraeURL         string
	RelayToTraeSecret      string
	RelayReadSec           float64
	CredentialServiceURL   string
	RelayTaskAPIOrigin     string
	RelayBusinessAPIOrigin string
	KafkaBootstrapServers  string
	CloudServiceURL        string
	CloudInternalSecret    string
	TaskAuthURL            string
	TaskAuthInternalSecret string
}

var cfg serviceConfig

func initConfig(repoRoot string) {
	cfg = serviceConfig{
		Host:                   strings.TrimSpace(os.Getenv("TASK_CONTAINER_GATEWAY_HOST")),
		Port:                   envInt("TASK_CONTAINER_GATEWAY_PORT", 8014),
		InternalSecret:         strings.TrimSpace(os.Getenv("TASK_CONTAINER_GATEWAY_INTERNAL_SECRET")),
		CloudServiceURL:        strings.TrimRight(strings.TrimSpace(os.Getenv("TASK_CLOUD_SERVICE_BASE_URL")), "/"),
		CloudInternalSecret:    strings.TrimSpace(os.Getenv("TASK_CLOUD_INTERNAL_SECRET")),
		TaskAuthURL:            strings.TrimRight(strings.TrimSpace(os.Getenv("TASK_AUTH_BASE_URL")), "/"),
		TaskAuthInternalSecret: strings.TrimSpace(os.Getenv("TASK_AUTH_INTERNAL_SECRET")),
		ForwardReadSec:         120,
		ForwardConnectSec:      15,
	}
	if cfg.Host == "" {
		cfg.Host = "127.0.0.1"
	}
	if cfg.CloudServiceURL == "" {
		cfg.CloudServiceURL = "http://127.0.0.1:8018"
	}
	if cfg.TaskAuthURL == "" {
		cfg.TaskAuthURL = "http://127.0.0.1:8003"
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
		Host              string   `yaml:"host"`
		Port              int      `yaml:"port"`
		InternalSecret    *string  `yaml:"internalSecret"`
		ForwardReadSec    *float64 `yaml:"forwardReadSec"`
		ForwardConnectSec *float64 `yaml:"forwardConnectSec"`
		RelayToTrae       *struct {
			URL     string   `yaml:"url"`
			Secret  string   `yaml:"secret"`
			ReadSec *float64 `yaml:"readSec"`
		} `yaml:"relayToTrae"`
		CredentialService *struct {
			URL string `yaml:"url"`
		} `yaml:"credentialService"`
		RelayRuntime *struct {
			TaskAPIEndpointOrigin     string `yaml:"taskApiEndpointOrigin"`
			BusinessAPIEndpointOrigin string `yaml:"businessApiEndpointOrigin"`
		} `yaml:"relayRuntime"`
		DomainEvents *struct {
			KafkaBootstrapServers string `yaml:"kafkaBootstrapServers"`
		} `yaml:"domainEvents"`
		TaskCloudService *struct {
			URL            string `yaml:"url"`
			InternalSecret string `yaml:"internalSecret"`
		} `yaml:"taskCloudService"`
		TaskAuth *struct {
			URL            string `yaml:"url"`
			InternalSecret string `yaml:"internalSecret"`
		} `yaml:"taskAuth"`
	}
	if err := confload.ReadAppConfig(repoRoot, "gateway/task-container-gateway", &block); err != nil {
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
	if block.ForwardReadSec != nil {
		cfg.ForwardReadSec = *block.ForwardReadSec
	}
	if block.ForwardConnectSec != nil {
		cfg.ForwardConnectSec = *block.ForwardConnectSec
	}
	if block.RelayToTrae != nil {
		cfg.RelayToTraeURL = strings.TrimSpace(block.RelayToTrae.URL)
		cfg.RelayToTraeSecret = strings.TrimSpace(block.RelayToTrae.Secret)
		if block.RelayToTrae.ReadSec != nil {
			cfg.RelayReadSec = *block.RelayToTrae.ReadSec
		}
	}
	if block.CredentialService != nil {
		cfg.CredentialServiceURL = strings.TrimSpace(block.CredentialService.URL)
	}
	if block.RelayRuntime != nil {
		cfg.RelayTaskAPIOrigin = strings.TrimSpace(block.RelayRuntime.TaskAPIEndpointOrigin)
		cfg.RelayBusinessAPIOrigin = strings.TrimSpace(block.RelayRuntime.BusinessAPIEndpointOrigin)
	}
	if block.DomainEvents != nil {
		cfg.KafkaBootstrapServers = strings.TrimSpace(block.DomainEvents.KafkaBootstrapServers)
	}
	if block.TaskCloudService != nil {
		if u := strings.TrimSpace(block.TaskCloudService.URL); u != "" {
			cfg.CloudServiceURL = strings.TrimRight(u, "/")
		}
		if s := strings.TrimSpace(block.TaskCloudService.InternalSecret); s != "" {
			cfg.CloudInternalSecret = s
		}
	}
	if block.TaskAuth != nil {
		if u := strings.TrimSpace(block.TaskAuth.URL); u != "" {
			cfg.TaskAuthURL = strings.TrimRight(u, "/")
		}
		if s := strings.TrimSpace(block.TaskAuth.InternalSecret); s != "" {
			cfg.TaskAuthInternalSecret = s
		}
	}
}

func (c serviceConfig) RelayTimeout() time.Duration {
	sec := c.RelayReadSec
	if sec <= 0 {
		sec = 30
	}
	return time.Duration(sec * float64(time.Second))
}

func main() {
	repoRoot, err := findMonorepoRoot()
	if err != nil {
		log.Fatalf("[taskContainerGateway] %v", err)
	}
	initConfig(repoRoot)
	initJobStreamConfig()

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	mux := http.NewServeMux()
	mountRoutes(mux)

	tracelog.Init("task-container-gateway")
	slog.Info("gateway listening", "addr", addr, "cloud", cfg.CloudServiceURL)
	handler := tracelog.Middleware(corsMiddleware(mux))
	if err := tracelog.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("[taskContainerGateway] server error: %v", err)
	}
}

func findMonorepoRootFrom(start string) (string, error) {
	marker := filepath.Join("trae-agent", "onlineServiceJS", "run.sh")
	dir := filepath.Clean(start)
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(dir, marker)); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("trae-agent/onlineServiceJS/run.sh not found")
}

func findMonorepoRoot() (string, error) {
	if cwd, err := os.Getwd(); err == nil {
		if root, err := findMonorepoRootFrom(cwd); err == nil {
			return root, nil
		}
	}
	if execPath, err := os.Executable(); err == nil {
		if root, err := findMonorepoRootFrom(filepath.Dir(execPath)); err == nil {
			return root, nil
		}
	}
	return "", fmt.Errorf("trae-agent/onlineServiceJS/run.sh not found")
}
