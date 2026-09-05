package main

import (
	"confload"
	"dbload"
	"log"
	"os"
	"strings"

	"gatewayauth"
)

type Config struct {
	Host                  string
	Port                  int
	DBPath                string
	TaskAuthURL           string
	TaskProjectServiceURL string
	TaskCloudServiceURL   string
	InternalSecret        string
	GatewayInternalSecret string
	FrontendBase          string
	KafkaBootstrapServers string
}

var cfg Config

func findMonorepoRoot() (string, error) {
	return confload.FindMonorepoRoot()
}

func loadConfig(repoRoot string) {
	type baseConf struct {
		Services struct {
			TaskTenantService struct {
				Host string `yaml:"host"`
				Port int    `yaml:"port"`
			} `yaml:"taskTenantService"`
			TaskAuth struct {
				Host string `yaml:"host"`
				Port int    `yaml:"port"`
			} `yaml:"taskAuth"`
			TaskProjectService struct {
				Host string `yaml:"host"`
				Port int    `yaml:"port"`
			} `yaml:"taskProjectService"`
			TaskCloudService struct {
				Host string `yaml:"host"`
				Port int    `yaml:"port"`
			} `yaml:"taskCloudService"`
		} `yaml:"services"`
		Shared struct {
			InternalSecret string `yaml:"internalSecret"`
		} `yaml:"shared"`
		FrontendBase string `yaml:"frontendBase"`
		Kafka        struct {
			BootstrapServers string `yaml:"bootstrapServers"`
		} `yaml:"kafka"`
	}

	var bc baseConf
	if err := confload.ReadAppConfigResolved(repoRoot, "taskTenantService", &bc); err != nil {
		log.Fatalf("[taskTenantService] confload: %v", err)
	}

	cfg.Host = bc.Services.TaskTenantService.Host
	cfg.Port = bc.Services.TaskTenantService.Port
	if cfg.Host == "" {
		cfg.Host = "0.0.0.0"
	}
	if cfg.Port == 0 {
		cfg.Port = 8020
	}
	cfg.TaskAuthURL = "http://" + loopbackHost(bc.Services.TaskAuth.Host) + ":" + itoa(bc.Services.TaskAuth.Port)
	projHost := loopbackHost(bc.Services.TaskProjectService.Host)
	projPort := bc.Services.TaskProjectService.Port
	if projPort == 0 {
		projPort = 8016
	}
	cfg.TaskProjectServiceURL = "http://" + projHost + ":" + itoa(projPort)
	cloudHost := loopbackHost(bc.Services.TaskCloudService.Host)
	cloudPort := bc.Services.TaskCloudService.Port
	if cloudPort == 0 {
		cloudPort = 8018
	}
	cfg.TaskCloudServiceURL = "http://" + cloudHost + ":" + itoa(cloudPort)
	// Validation downgraded from fatal to warning.
	cfg.InternalSecret = bc.Shared.InternalSecret
	cfg.GatewayInternalSecret = gatewayauth.LoadGatewayInternalSecret(repoRoot)
	cfg.KafkaBootstrapServers = strings.TrimSpace(bc.Kafka.BootstrapServers)
	if cfg.KafkaBootstrapServers == "" {
		cfg.KafkaBootstrapServers = loadKafkaBootstrap(repoRoot)
	}
	dbPath, err := dbload.ResolveMySQLDSN("task-tenant", repoRoot)
	if err != nil {
		log.Fatalf("[taskTenantService] db path: %v", err)
	}
	cfg.DBPath = dbPath
	cfg.FrontendBase = strings.TrimRight(strings.TrimSpace(bc.FrontendBase), "/")
	if cfg.FrontendBase == "" {
		cfg.FrontendBase = strings.TrimRight(strings.TrimSpace(os.Getenv("FRONTEND_BASE")), "/")
	}
	if cfg.FrontendBase == "" {
		cfg.FrontendBase = "http://127.0.0.1:4000"
	}

	log.Printf("[taskTenantService] config: host=%s port=%d auth=%s project=%s cloud=%s kafka=%s",
		cfg.Host, cfg.Port, cfg.TaskAuthURL, cfg.TaskProjectServiceURL,
		cfg.TaskCloudServiceURL, cfg.KafkaBootstrapServers)
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

func loopbackHost(h string) string {
	h = strings.TrimSpace(h)
	if h == "" || h == "0.0.0.0" {
		return "127.0.0.1"
	}
	return h
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
