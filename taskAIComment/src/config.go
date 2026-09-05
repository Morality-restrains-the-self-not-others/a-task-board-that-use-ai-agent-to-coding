package main

import (
	"confload"
	"dbload"
	"fmt"
	"gatewayauth"
	"log"
	"os"
	"strings"
)

type Config struct {
	Host                        string
	Port                        int
	MySQLDSN                    string
	TaskAuthURL                 string
	TaskServiceURL              string
	TaskCloudServiceURL         string
	TaskSseURL                  string
	TaskSseSecret               string
	CredentialServiceURL        string
	InternalSecret              string
	TaskAICommentInternalSecret string
	TaskGatewayInternalSecret   string
	KafkaBootstrapServers       string
}

var cfg Config

func findMonorepoRoot() (string, error) {
	return confload.FindMonorepoRoot()
}

func loadConfig(repoRoot string) {
	type baseConf struct {
		Services struct {
			TaskAIComment struct {
				Host string `yaml:"host"`
				Port int    `yaml:"port"`
			} `yaml:"taskAIComment"`
			TaskAuth struct {
				Host string `yaml:"host"`
				Port int    `yaml:"port"`
			} `yaml:"taskAuth"`
			TaskTaskService struct {
				Host string `yaml:"host"`
				Port int    `yaml:"port"`
			} `yaml:"taskTaskService"`
			TaskCloudService struct {
				Host string `yaml:"host"`
				Port int    `yaml:"port"`
			} `yaml:"taskCloudService"`
			TaskSse struct {
				Host string `yaml:"host"`
				Port int    `yaml:"port"`
			} `yaml:"taskSse"`
		} `yaml:"services"`
		Shared struct {
			InternalSecret              string `yaml:"internalSecret"`
			TaskAICommentInternalSecret string `yaml:"taskAICommentInternalSecret"`
		} `yaml:"shared"`
		Kafka struct {
			BootstrapServers string `yaml:"bootstrapServers"`
		} `yaml:"kafka"`
	}

	var bc baseConf
	if err := confload.ReadAppConfigResolved(repoRoot, "taskAIComment", &bc); err != nil {
		log.Fatalf("[taskAIComment] confload: %v", err)
	}

	cfg.Host = bc.Services.TaskAIComment.Host
	cfg.Port = bc.Services.TaskAIComment.Port
	if cfg.Host == "" {
		cfg.Host = "0.0.0.0"
	}
	if cfg.Port == 0 {
		cfg.Port = 8019
	}
	cfg.TaskAuthURL = "http://" + bc.Services.TaskAuth.Host + ":" + fmt.Sprintf("%d", bc.Services.TaskAuth.Port)
	cfg.TaskServiceURL = "http://" + bc.Services.TaskTaskService.Host + ":" + fmt.Sprintf("%d", bc.Services.TaskTaskService.Port)
	tcsHost := bc.Services.TaskCloudService.Host
	tcsPort := bc.Services.TaskCloudService.Port
	if tcsHost == "" {
		tcsHost = "127.0.0.1"
	}
	if tcsPort == 0 {
		tcsPort = 8018
	}
	cfg.TaskCloudServiceURL = "http://" + tcsHost + ":" + fmt.Sprintf("%d", tcsPort)
	if bc.Services.TaskSse.Port != 0 {
		host := bc.Services.TaskSse.Host
		if host == "" {
			host = "127.0.0.1"
		}
		cfg.TaskSseURL = "http://" + host + ":" + fmt.Sprintf("%d", bc.Services.TaskSse.Port)
	}
	if v := os.Getenv("TASK_SSE_SECRET"); v != "" {
		cfg.TaskSseSecret = strings.TrimSpace(v)
	} else {
		cfg.TaskSseSecret = "dev-secret"
	}
	type infraServices struct {
		Services struct {
			TaskCredentialService struct {
				Host string `yaml:"host"`
				Port int    `yaml:"port"`
			} `yaml:"taskCredentialService"`
		} `yaml:"services"`
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
	cfg.InternalSecret = bc.Shared.InternalSecret
	cfg.TaskAICommentInternalSecret = bc.Shared.TaskAICommentInternalSecret
	if cfg.TaskAICommentInternalSecret == "" {
		cfg.TaskAICommentInternalSecret = cfg.InternalSecret
	}
	cfg.TaskGatewayInternalSecret = gatewayauth.LoadGatewayInternalSecret(repoRoot)
	cfg.KafkaBootstrapServers = bc.Kafka.BootstrapServers
	if cfg.KafkaBootstrapServers == "" {
		cfg.KafkaBootstrapServers = loadKafkaBootstrap(repoRoot)
	}
	mysqlDSN, err := dbload.ResolveMySQLDSN("task-ai-comment", repoRoot)
	if err != nil {
		log.Fatalf("[taskAIComment] MySQL DSN: %v", err)
	}
	cfg.MySQLDSN = mysqlDSN

	log.Printf("[taskAIComment] config: host=%s port=%d auth=%s task=%s cloud=%s sse=%s kafka=%s",
		cfg.Host, cfg.Port, cfg.TaskAuthURL, cfg.TaskServiceURL, cfg.TaskCloudServiceURL, cfg.TaskSseURL, cfg.KafkaBootstrapServers)
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
