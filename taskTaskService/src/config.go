package main

import (
	"confload"
	"dbload"
	"gatewayauth"
	"log"
	"strings"
)

type Config struct {
	Host                        string
	Port                        int
	DBPath                      string
	TaskAuthURL                 string
	ProjectServiceURL           string
	TaskBillURL                 string
	AICommentServiceURL         string
	TaskTenantServiceURL        string
	CloudServiceURL             string
	GitOAuthURL                 string
	InternalSecret              string
	TaskBillInternalSecret      string
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
			TaskTaskService struct {
				Host string `yaml:"host"`
				Port int    `yaml:"port"`
			} `yaml:"taskTaskService"`
			TaskAuth struct {
				Host string `yaml:"host"`
				Port int    `yaml:"port"`
			} `yaml:"taskAuth"`
			TaskProjectService struct {
				Host string `yaml:"host"`
				Port int    `yaml:"port"`
			} `yaml:"taskProjectService"`
			TaskBill struct {
				Host string `yaml:"host"`
				Port int    `yaml:"port"`
			} `yaml:"taskBill"`
			TaskAIComment struct {
				Host string `yaml:"host"`
				Port int    `yaml:"port"`
			} `yaml:"taskAIComment"`
			TaskCloudService struct {
				Host string `yaml:"host"`
				Port int    `yaml:"port"`
			} `yaml:"taskCloudService"`
			TaskTenantService struct {
				Host string `yaml:"host"`
				Port int    `yaml:"port"`
			} `yaml:"taskTenantService"`
			TaskGitOauth struct {
				Host string `yaml:"host"`
				Port int    `yaml:"port"`
			} `yaml:"taskGitOauth"`
		} `yaml:"services"`
		Shared struct {
			InternalSecret              string `yaml:"internalSecret"`
			TaskBillInternalSecret      string `yaml:"taskBillInternalSecret"`
			TaskAICommentInternalSecret string `yaml:"taskAICommentInternalSecret"`
		} `yaml:"shared"`
		Kafka struct {
			BootstrapServers string `yaml:"bootstrapServers"`
		} `yaml:"kafka"`
	}

	var bc baseConf
	if err := confload.ReadAppConfigResolved(repoRoot, "taskTaskService", &bc); err != nil {
		log.Fatalf("[taskTaskService] confload: %v", err)
	}

	cfg.Host = bc.Services.TaskTaskService.Host
	cfg.Port = bc.Services.TaskTaskService.Port
	if cfg.Host == "" {
		cfg.Host = "0.0.0.0"
	}
	if cfg.Port == 0 {
		cfg.Port = 8017
	}
	cfg.TaskAuthURL = "http://" + bc.Services.TaskAuth.Host + ":" + itoa(bc.Services.TaskAuth.Port)
	cfg.ProjectServiceURL = "http://" + bc.Services.TaskProjectService.Host + ":" + itoa(bc.Services.TaskProjectService.Port)
	if bc.Services.TaskBill.Port != 0 {
		host := bc.Services.TaskBill.Host
		if host == "" {
			host = "127.0.0.1"
		}
		cfg.TaskBillURL = "http://" + host + ":" + itoa(bc.Services.TaskBill.Port)
	}
	cfg.AICommentServiceURL = resolveLoopbackServiceURL(bc.Services.TaskAIComment.Host, bc.Services.TaskAIComment.Port, 8019)
	{
		host := bc.Services.TaskCloudService.Host
		if host == "" || host == "0.0.0.0" {
			host = "127.0.0.1"
		}
		port := bc.Services.TaskCloudService.Port
		if port == 0 {
			port = 8018
		}
		cfg.CloudServiceURL = "http://" + host + ":" + itoa(port)
	}
	// OPT-20260726-021: taskTenantService URL for group membership checks
	{
		host := bc.Services.TaskTenantService.Host
		if host == "" || host == "0.0.0.0" {
			host = "127.0.0.1"
		}
		port := bc.Services.TaskTenantService.Port
		if port == 0 {
			port = 8020
		}
		cfg.TaskTenantServiceURL = "http://" + host + ":" + itoa(port)
	}
	// OPT-20260821-036: taskGitOauth URL for comment-create Git OAuth binding check
	{
		host := bc.Services.TaskGitOauth.Host
		if host == "" || host == "0.0.0.0" {
			host = "127.0.0.1"
		}
		port := bc.Services.TaskGitOauth.Port
		if port == 0 {
			port = 8002
		}
		cfg.GitOAuthURL = "http://" + host + ":" + itoa(port)
	}
	cfg.InternalSecret = bc.Shared.InternalSecret
	cfg.TaskBillInternalSecret = bc.Shared.TaskBillInternalSecret
	cfg.TaskAICommentInternalSecret = bc.Shared.TaskAICommentInternalSecret
	if cfg.TaskAICommentInternalSecret == "" {
		cfg.TaskAICommentInternalSecret = cfg.InternalSecret
	}
	if cfg.TaskBillInternalSecret == "" {
		cfg.TaskBillInternalSecret = loadTaskBillSecret(repoRoot)
	}
	cfg.TaskGatewayInternalSecret = gatewayauth.LoadGatewayInternalSecret(repoRoot)
	dbPath, err := dbload.ResolveMySQLDSN("task-task", repoRoot)
	if err != nil {
		log.Fatalf("[taskTaskService] db path: %v", err)
	}
	cfg.DBPath = dbPath
	cfg.KafkaBootstrapServers = bc.Kafka.BootstrapServers
	if cfg.KafkaBootstrapServers == "" {
		cfg.KafkaBootstrapServers = loadKafkaBootstrap(repoRoot)
	}

	log.Printf("[taskTaskService] config: host=%s port=%d auth=%s project=%s bill=%s aicomment=%s cloud=%s kafka=%s",
		cfg.Host, cfg.Port, cfg.TaskAuthURL, cfg.ProjectServiceURL, cfg.TaskBillURL, cfg.AICommentServiceURL, cfg.CloudServiceURL, cfg.KafkaBootstrapServers)
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

func loadTaskBillSecret(repoRoot string) string {
	type billConf struct {
		InternalSecret string `yaml:"internalSecret"`
	}
	var bc billConf
	if err := confload.ReadAppConfigResolved(repoRoot, "billing/task-bill", &bc); err != nil {
		return ""
	}
	return bc.InternalSecret
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

// resolveLoopbackServiceURL builds a same-node client URL.
// Listen hosts (empty / 0.0.0.0) are rewritten to 127.0.0.1; port 0 uses defaultPort.
func resolveLoopbackServiceURL(host string, port, defaultPort int) string {
	h := strings.TrimSpace(host)
	if h == "" || h == "0.0.0.0" {
		h = "127.0.0.1"
	}
	p := port
	if p == 0 {
		p = defaultPort
	}
	return "http://" + h + ":" + itoa(p)
}
