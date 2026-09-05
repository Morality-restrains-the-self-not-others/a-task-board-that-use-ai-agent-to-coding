// Package infrastructure contains adapter implementations of domain ports.
// This file is the composition root — where all dependencies are assembled.
package infrastructure

import (
	"fmt"
	"log"
	"os"
	"strings"

	"confload"

	"taskCredentialService/application"
	"taskCredentialService/ports"
)

// AppServices holds all assembled application services.
type AppServices struct {
	Token      *application.TokenService
	Credential *application.CredentialService
	TaskDetail *application.TaskDetailService
	LayerOauth *application.LayerOauthService
	AIComment  *AICommentClient
}

// Compose builds the full dependency graph and returns assembled services.
// This is the single place where concrete adapters are chosen and wired.
// Switching DB or HTTP client backends only requires changes here.
func Compose(monorepoRoot string, gitoauthBase string, gitoauthTimeout int) (*AppServices, error) {
	// ─── Data layer adapters ────────────────────────────────────────────
	tokenDB, err := OpenTokensDB(monorepoRoot)
	if err != nil {
		return nil, fmt.Errorf("open tokens db: %w", err)
	}

	tokenRepo := NewSQLiteTokenRepository(tokenDB)
	auditRepo := NewSQLiteAuditRepository(tokenDB)

	taskBase, projectBase, internalSecret := loadBusinessHTTPConfig(monorepoRoot)

	// Prefer direct MySQL reads (no HTTP) for business data.
	// Falls back to HTTPBusinessRepository when DB connections are unavailable.
	// OPT-052: saas DB removed — git_identities now in task_task DB.
	var businessRepo ports.BusinessDataRepository
	bizTaskDB, bizProjectDB, bizErr := OpenBusinessDBs(monorepoRoot)
	if bizErr == nil {
		businessRepo = NewSQLiteBusinessRepository(bizTaskDB, bizProjectDB)
		log.Printf("[task-credential-service] business data: direct MySQL (OPT-023 git-identities resolved without HTTP)")
	} else {
		log.Printf("[task-credential-service] WARN business DBs unavailable, falling back to HTTP: %v", bizErr)
		businessRepo = NewHTTPBusinessRepository(taskBase, internalSecret, 10)
	}
	nestedFetcher := NewNestedGitReposHTTPClient(projectBase, internalSecret, 15)
	aiCommentBase, aiCommentSecret := loadAICommentHTTPConfig(monorepoRoot, internalSecret)
	aiCommentClient := NewAICommentClient(aiCommentBase, aiCommentSecret)
	cloudBase := loadTaskCloudHTTPConfig(monorepoRoot)
	policyFetcher := NewCloudPolicyHTTPClient(cloudBase, internalSecret, 5)

	// ─── External service adapters ──────────────────────────────────────
	// Load provider configs for self-hosted GitLab resolution (non-fatal).
	resolver, resolverErr := LoadProviderConfigs(monorepoRoot)
	if resolverErr != nil {
		log.Printf("[task-credential-service] WARN provider config load failed (self-hosted GitLab may not resolve): %v", resolverErr)
		resolver = nil // gitoauth client will use basic fallback
	}
	bridgeSecret := strings.TrimSpace(os.Getenv("GITOAUTH_BRIDGE_JWT_SECRET"))
	if bridgeSecret == "" {
		bridgeSecret = strings.TrimSpace(os.Getenv("TASK2APP_SSO_JWT_SECRET"))
	}
	if bridgeSecret == "" {
		// conf 注入：container/task-credential-service/config.yaml gitoauth_bridge_secret
		// （SSOT conf/core/sso/config.yaml ssoJwtSecret，OPT-20260807-071）。
		var block struct {
			GitoauthBridgeSecret string `yaml:"gitoauth_bridge_secret"`
		}
		if err := confload.ReadAppConfig(monorepoRoot, "container/task-credential-service", &block); err == nil {
			bridgeSecret = strings.TrimSpace(block.GitoauthBridgeSecret)
		}
	}
	if bridgeSecret == "" {
		bridgeSecret = "task2app-local-sso-bridge-dev-do-not-use-in-prod"
	}
	gitoauthClient := NewGitoauthHTTPClient(gitoauthBase, gitoauthTimeout, resolver, bridgeSecret)

	// ─── Application services ───────────────────────────────────────────
	return &AppServices{
		Token:      application.NewTokenService(tokenRepo, auditRepo),
		Credential: application.NewCredentialService(tokenRepo, businessRepo, gitoauthClient).WithNestedFetcher(nestedFetcher),
		TaskDetail: application.NewTaskDetailService(businessRepo).WithNestedFetcher(nestedFetcher).WithPolicyFetcher(policyFetcher),
		LayerOauth: application.NewLayerOauthService(tokenRepo, businessRepo, gitoauthClient),
		AIComment:  aiCommentClient,
	}, nil
}

func loadBusinessHTTPConfig(monorepoRoot string) (taskBase, projectBase, internalSecret string) {
	taskBase = strings.TrimSpace(os.Getenv("TASK_TASK_SERVICE_BASE_URL"))
	projectBase = strings.TrimSpace(os.Getenv("TASK_PROJECT_SERVICE_BASE_URL"))
	internalSecret = strings.TrimSpace(os.Getenv("SAAS_INTERNAL_API_SECRET"))
	if internalSecret == "" {
		internalSecret = strings.TrimSpace(os.Getenv("X_INTERNAL_SECRET"))
	}

	var block struct {
		TaskTaskServiceBase    string `yaml:"task_task_service_base"`
		TaskProjectServiceBase string `yaml:"task_project_service_base"`
		InternalSecret         string `yaml:"internal_secret"`
	}
	if err := confload.ReadAppConfig(monorepoRoot, "container/task-credential-service", &block); err == nil {
		if taskBase == "" {
			taskBase = strings.TrimSpace(block.TaskTaskServiceBase)
		}
		if projectBase == "" {
			projectBase = strings.TrimSpace(block.TaskProjectServiceBase)
		}
		if internalSecret == "" {
			internalSecret = strings.TrimSpace(block.InternalSecret)
		}
	}

	// Fallback: resolve from peer service configs when not explicitly set.
	if taskBase == "" || projectBase == "" {
		var peer struct {
			Services struct {
				TaskTaskService struct {
					Host string `yaml:"host"`
					Port int    `yaml:"port"`
				} `yaml:"taskTaskService"`
				TaskProjectService struct {
					Host string `yaml:"host"`
					Port int    `yaml:"port"`
				} `yaml:"taskProjectService"`
			} `yaml:"services"`
			Shared struct {
				InternalSecret string `yaml:"internalSecret"`
			} `yaml:"shared"`
		}
		if err := confload.ReadAppConfigResolved(monorepoRoot, "taskTaskService", &peer); err == nil {
			if taskBase == "" {
				host := strings.TrimSpace(peer.Services.TaskTaskService.Host)
				if host == "" || host == "0.0.0.0" {
					host = "127.0.0.1"
				}
				port := peer.Services.TaskTaskService.Port
				if port == 0 {
					port = 8017
				}
				taskBase = fmt.Sprintf("http://%s:%d", host, port)
			}
			if projectBase == "" {
				host := strings.TrimSpace(peer.Services.TaskProjectService.Host)
				if host == "" || host == "0.0.0.0" {
					host = "127.0.0.1"
				}
				port := peer.Services.TaskProjectService.Port
				if port == 0 {
					port = 8016
				}
				projectBase = fmt.Sprintf("http://%s:%d", host, port)
			}
			if internalSecret == "" {
				internalSecret = strings.TrimSpace(peer.Shared.InternalSecret)
			}
		}
	}

	if taskBase == "" {
		taskBase = "http://127.0.0.1:8017"
	}
	if projectBase == "" {
		projectBase = "http://127.0.0.1:8016"
	}
	log.Printf("[task-credential-service] business HTTP: task=%s project=%s", taskBase, projectBase)
	return taskBase, projectBase, internalSecret
}

func loadAICommentHTTPConfig(monorepoRoot, fallbackSecret string) (baseURL, secret string) {
	baseURL = strings.TrimSpace(os.Getenv("TASK_AI_COMMENT_SERVICE_BASE_URL"))
	secret = strings.TrimSpace(os.Getenv("TASK_AI_COMMENT_INTERNAL_SECRET"))
	if secret == "" {
		secret = strings.TrimSpace(fallbackSecret)
	}
	var peer struct {
		Services struct {
			TaskAIComment struct {
				Host string `yaml:"host"`
				Port int    `yaml:"port"`
			} `yaml:"taskAIComment"`
		} `yaml:"services"`
		Shared struct {
			InternalSecret              string `yaml:"internalSecret"`
			TaskAICommentInternalSecret string `yaml:"taskAICommentInternalSecret"`
		} `yaml:"shared"`
	}
	if err := confload.ReadAppConfigResolved(monorepoRoot, "taskAIComment", &peer); err == nil {
		if baseURL == "" {
			host := strings.TrimSpace(peer.Services.TaskAIComment.Host)
			if host == "" || host == "0.0.0.0" {
				host = "127.0.0.1"
			}
			port := peer.Services.TaskAIComment.Port
			if port == 0 {
				port = 8019
			}
			baseURL = fmt.Sprintf("http://%s:%d", host, port)
		}
		if secret == "" {
			secret = strings.TrimSpace(peer.Shared.TaskAICommentInternalSecret)
		}
		if secret == "" {
			secret = strings.TrimSpace(peer.Shared.InternalSecret)
		}
	}
	if baseURL == "" {
		baseURL = "http://127.0.0.1:8019"
	}
	log.Printf("[task-credential-service] aiComment HTTP: base=%s", baseURL)
	return baseURL, secret
}

func loadTaskCloudHTTPConfig(monorepoRoot string) string {
	base := strings.TrimSpace(os.Getenv("TASK_CLOUD_SERVICE_BASE_URL"))
	var block struct {
		TaskCloudServiceBase string `yaml:"task_cloud_service_base"`
	}
	if err := confload.ReadAppConfig(monorepoRoot, "container/task-credential-service", &block); err == nil {
		if base == "" {
			base = strings.TrimSpace(block.TaskCloudServiceBase)
		}
	}
	if base == "" {
		// 同节点 loopback；拆分时改 conf/container/task-credential-service/config.yaml
		base = "http://127.0.0.1:8018"
	}
	log.Printf("[task-credential-service] taskCloud HTTP: base=%s", base)
	return strings.TrimRight(base, "/")
}

// Shutdown closes all database connections.
func (a *AppServices) Shutdown() {
	log.Println("[task-credential-service] shutting down database connections")
}

// NewTestTokenService creates a TokenService with injected repositories for testing.
func NewTestTokenService(tokenRepo ports.ContainerTokenRepository, auditRepo ports.TokenAuditEventRepository) *application.TokenService {
	return application.NewTokenService(tokenRepo, auditRepo)
}

// Ensure interface compliance at compile time.
var _ ports.ContainerTokenRepository = (*SQLiteTokenRepository)(nil)
var _ ports.TokenAuditEventRepository = (*SQLiteAuditRepository)(nil)
var _ ports.BusinessDataRepository = (*HTTPBusinessRepository)(nil)
var _ ports.GitoauthClient = (*GitoauthHTTPClient)(nil)
var _ ports.WorkspaceMachinePolicyFetcher = (*CloudPolicyHTTPClient)(nil)
