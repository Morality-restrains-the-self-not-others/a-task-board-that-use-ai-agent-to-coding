// Package ports defines the abstract interfaces (ports) that the domain layer
// requires from the outside world. Infrastructure adapters implement these ports.
// Domain layer depends on these interfaces, NOT on concrete implementations.
package ports

import "taskCredentialService/domain"

// ─── Repository Ports ────────────────────────────────────────────────────────

// ContainerTokenRepository is the persistence port for ContainerToken aggregates.
// Implementations: infrastructure.SQLiteTokenRepository
type ContainerTokenRepository interface {
	// Save persists a container token, returning the saved state.
	Save(token *domain.ContainerToken) (*domain.ContainerToken, error)

	// FindByAccessToken looks up a token by its access token string.
	FindByAccessToken(accessToken string) (*domain.ContainerToken, error)

	// FindByTaskID finds the most recent token for a given task.
	FindByTaskID(companyID, taskID string) (*domain.ContainerToken, error)

	// FindByScope finds the most recent token matching full scope including comment_id.
	FindByScope(companyID, workspaceID, taskID, commentID string) (*domain.ContainerToken, error)

	// CountByTaskScope counts token rows for tenant/workspace/task (any comment).
	CountByTaskScope(companyID, workspaceID, taskID string) (int, error)

	// UpdateAccessToken atomically replaces the access token and expiry.
	UpdateAccessToken(id string, newAccessToken string, expiresAt string) error

	// UpdateRefreshToken atomically replaces the refresh token.
	UpdateRefreshToken(id string, newRefreshToken string) error

	// FindByRefreshToken looks up a token by its refresh token string.
	FindByRefreshToken(refreshToken string) (*domain.ContainerToken, error)

	// UpdateBusinessAPIEndpoint atomically sets the business API endpoint.
	UpdateBusinessAPIEndpoint(id string, endpoint string) error
}

// TokenAuditEventRepository is the persistence port for audit events.
type TokenAuditEventRepository interface {
	Save(event *domain.TokenAuditEvent) error
	FindByTaskID(taskID string, limit int) ([]domain.TokenAuditEvent, error)
}

// BusinessDataRepository provides read-only access to task/repo/git-identity data
// via owner-service HTTP APIs (task-task-service + saas-backend).
type BusinessDataRepository interface {
	// FetchTaskSnapshot returns task metadata for container consumption.
	// commentID scopes to the comment-level snapshot when non-empty (task-level fallback otherwise).
	FetchTaskSnapshot(taskID, commentID string) (*domain.TaskSnapshot, error)

	// FetchTaskRepos returns the repos linked to a task via its projects.
	// commentID scopes to the comment-level snapshot when non-empty (task-level fallback otherwise).
	FetchTaskRepos(taskID, commentID string) ([]domain.TaskRepoSnapshot, error)

	// FetchRepoIdentities returns git identities for clone/commit.
	// When commentID is non-empty and the comment has repo_identities_json, those
	// per-repo selections win; otherwise the task-level task_repo_identities table.
	FetchRepoIdentities(taskID, commentID string) ([]domain.GitIdentitySnapshot, error)

	// FetchCommentCreatedByUserID returns task_comments.created_by_id for a comment.
	FetchCommentCreatedByUserID(commentID string) (int64, error)

	// FetchUserGitIdentities returns git identities owned by a platform user.
	FetchUserGitIdentities(userID int64) ([]domain.GitIdentitySnapshot, error)
}

// ─── External Service Ports ──────────────────────────────────────────────────

// WorkspaceMachinePolicy is the Cloud-owned idle recycle view for a workspace.
type WorkspaceMachinePolicy struct {
	IdleRecycleMinutes int
	MachineReleaseSTS  map[string]interface{}
}

// WorkspaceMachinePolicyFetcher reads idle recycle policy from taskCloudService.
// Implementations must not connect to task_cloud MySQL.
type WorkspaceMachinePolicyFetcher interface {
	FetchWorkspaceMachinePolicy(companyID, workspaceID, taskID, commentID string) (WorkspaceMachinePolicy, error)
}

// GitoauthClient is the HTTP client port for gitOauth token exchange.
type GitoauthClient interface {
	// FetchAccessToken exchanges a user's gitOauth binding for an ephemeral token.
	// site is the Git site host from the repo URL (e.g. "github.com", "115.29.110.74").
	// Always routed through the gitsite internal path
	// (/api/internal/gitsite/{site}/oauth/access-for-user/) — gitOauth resolves the
	// site to a provider (OPT-20260827-036). No YAML provider-key fork.
	FetchAccessToken(userID int64, site string) (accessToken string, err error)

	// ResolveProvider identifies the full provider key (e.g. "gitlab:synology-gitlab")
	// from a repo URL. For self-hosted GitLab instances the URL may not contain
	// "gitlab" — this method matches against configured provider websites.
	// Kept only for git_http_username / logging; token exchange no longer branches on it.
	ResolveProvider(repoURL string) string

	// ResolveHttpsCloneURL converts SSH/SCP-style repo URLs to HTTP(S) clone URLs
	// using configured provider websites (OAuth HTTPS clone contract).
	ResolveHttpsCloneURL(repoURL string) string
}
