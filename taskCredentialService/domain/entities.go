// Package domain defines the container runtime domain model.
// No infrastructure dependencies (no database/sql, no HTTP clients).
package domain

import "strings"

// ─── Entities ────────────────────────────────────────────────────────────────

// ContainerToken is the aggregate root for container runtime token lifecycle.
type ContainerToken struct {
	ID                            string
	TaskID                        string
	CompanyID                     string
	WorkspaceID                   string
	CommentID                     string
	ContainerAccessToken          string
	ContainerAccessTokenExpiresAt string // SQLite TEXT datetime, empty if not set
	ContainerRefreshToken         string
	ServerURL                     string
	BusinessAPIEndpoint           string
	ContainerVscodeURL            string
	AuthorizationID               string
	ImageID                       string
	InstanceType                  string
	RegionID                      string
	ZoneID                        string
	CreatedAt                     string
	UpdatedAt                     string
}

// TokenAuditEvent records a token lifecycle event for observability.
type TokenAuditEvent struct {
	ID                    string
	TaskID                string
	CommentID             string
	EventType             string
	AccessTokenSHA256     string
	PrevAccessTokenSHA256 string
	NewAccessTokenSHA256  string
	RefreshTokenSHA256    string
	SourceComponent       string
	TraceID               string
	ErrorCode             string
	ErrorDetail           string
	Seq                   int
	CreatedAt             string
}

// ─── Value Objects ───────────────────────────────────────────────────────────

// TaskScope identifies the tenant/workspace/task(/comment) context of a container.
type TaskScope struct {
	TenantID    string
	WorkspaceID string
	TaskID      string
	CommentID   string
}

func (s TaskScope) Validate() error {
	if s.TenantID == "" || s.WorkspaceID == "" || s.TaskID == "" {
		return ErrInvalidScope
	}
	return nil
}

// ValidateIssue requires comment_id for new token issuance (ADR-0005).
func (s TaskScope) ValidateIssue() error {
	if err := s.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(s.CommentID) == "" {
		return ErrCommentIDRequired
	}
	return nil
}

// CommentConflicts reports SCOPE_MISMATCH only when both sides set comment and they differ.
func CommentConflicts(tokenCommentID, scopeCommentID string) bool {
	t := strings.TrimSpace(tokenCommentID)
	s := strings.TrimSpace(scopeCommentID)
	return t != "" && s != "" && t != s
}

// AccessToken wraps a container access token string.
type AccessToken string

func (t AccessToken) IsEmpty() bool { return string(t) == "" }

// RefreshToken wraps a container refresh token string.
type RefreshToken string

func (t RefreshToken) IsEmpty() bool { return string(t) == "" }

// RepoCloneCredential is per-repo ephemeral HTTPS clone credentials.
// JSON 字段名须与 docs/skills/saas-container/saas-machine-container.md / onlineServiceJS bootstrap 契约一致（snake_case）。
type RepoCloneCredential struct {
	RepoURL                   string `json:"repo_url,omitempty"`
	EphemeralOAuthAccessToken string `json:"ephemeral_oauth_access_token"`
	Provider                  string `json:"provider,omitempty"`
	GitHTTPUsername           string `json:"git_http_username,omitempty"`
	// HttpsCloneURL is the OAuth-ready HTTP(S) clone URL when repo_url is SSH/SCP-style.
	HttpsCloneURL string `json:"https_clone_url,omitempty"`
}

// RepoCloneCredentialsResult is the outcome of a credential fetch.
type RepoCloneCredentialsResult struct {
	Credentials          map[string]RepoCloneCredential
	MissingIdentityRepos []string
	TokenRefreshFailures []TokenRefreshFailure
}

// TokenRefreshFailure records why a particular repo's token refresh failed.
type TokenRefreshFailure struct {
	RepoURL   string `json:"repo_url"`
	ErrorCode string `json:"error_code"`
	Detail    string `json:"detail"`
	GitHost   string `json:"git_host,omitempty"`
}

// GitIdentitySnapshot is a read-only view of a task's git identities.
type GitIdentitySnapshot struct {
	RepoURL           string
	GitIdentityID     string
	UserID            int64
	UserName          string
	UserEmail         string
	OauthGitsite      string
	OauthRemoteUserID string
	OauthGrantedAt    string
	// FromComment is true when the binding came from task_comments.repo_identities_json.
	FromComment bool
}

// RepoGitIdentityDTO is the container-facing resolved git author for one repo URL.
type RepoGitIdentityDTO struct {
	RepoURL    string `json:"repo_url"`
	UserName   string `json:"user_name"`
	UserEmail  string `json:"user_email"`
	IdentityID string `json:"identity_id,omitempty"`
}

// RepoBranchEntry is per-repo base/work branch metadata for container bootstrap checkout.
type RepoBranchEntry struct {
	GitRepo      string `json:"git_repo"`
	BaseBranch   string `json:"base_branch"`
	TargetBranch string `json:"target_branch"`
}

// BranchStrategySnapshot is the task-level branch naming strategy.
type BranchStrategySnapshot struct {
	WorkBranchName        string `json:"work_branch_name"`
	MergeTargetBranchName string `json:"merge_target_branch_name"`
	TargetBranchName      string `json:"target_branch_name"`
}

// RepoCloneEntry is a repo URL with optional clone directory alias.
// ParentRepoURL is set for nested (discovered) child repos so the container can
// relocate the clone under the parent working tree after git clone completes.
type RepoCloneEntry struct {
	URL           string `json:"url"`
	CloneAlias    string `json:"clone_alias,omitempty"`
	ParentRepoURL string `json:"parent_repo_url,omitempty"`
}

// TaskRepoSnapshot is a read-only view of a task's linked repos.
type TaskRepoSnapshot struct {
	ProjectID            string            `json:"project_id"`
	ProjectName          string            `json:"project_name"`
	RepoURLs             []string          `json:"git_repos"`
	RepoEntries          []RepoCloneEntry  `json:"git_repo_entries,omitempty"`
	RepoBranches         []RepoBranchEntry `json:"repo_branches,omitempty"`
	AutoCloneNestedRepos bool              `json:"auto_clone_nested_repos"`
}

// TaskSnapshot is a read-only view of a task for container consumption.
type TaskSnapshot struct {
	ID               string                  `json:"id"`
	Title            string                  `json:"title"`
	Description      string                  `json:"description"`
	TargetBranch     string                  `json:"target_branch"`
	AutoRun          bool                    `json:"auto_run"`
	InstalledImageID string                  `json:"installed_image_id,omitempty"`
	BranchStrategy   *BranchStrategySnapshot `json:"-"`
	CompanyID        string                  `json:"-"`
	WorkspaceID      string                  `json:"-"`
}

// InstructionIdlePolicy is the workspace idle-recycle view for containers.
type InstructionIdlePolicy struct {
	Enabled bool `json:"enabled"`
	Minutes int  `json:"minutes"`
}

// ContainerTaskDetail is the full task-detail response body for onlineServiceJS.
type ContainerTaskDetail struct {
	CompanyID          string                 `json:"company_id"`
	WorkspaceID        string                 `json:"workspace_id"`
	TaskID             string                 `json:"task_id"`
	Task               *TaskSnapshot          `json:"task"`
	ProjectRepos       []TaskRepoSnapshot     `json:"project_repos"`
	RepoGitIdentities  []RepoGitIdentityDTO   `json:"repo_git_identities"`
	IdleRecycleMinutes int                    `json:"idle_recycle_minutes"`
	InstructionIdle    InstructionIdlePolicy  `json:"instruction_idle"`
	MachineReleaseSTS  map[string]interface{} `json:"machine_release_sts,omitempty"`
}

// LayerOauthTokensResult is the outcome of layer-github-oauth-access-tokens.
// Aligns with Django container_layer_github_oauth_views response contract.
type LayerOauthTokensResult struct {
	OK                    bool
	GithubAuthByRepo      map[string]string // owner/repo → access_token
	GitAuthByRepoMatchKey map[string]string // host/path → access_token (optional)
	Detail                string
	ErrorCode             string
	FailedStage           string
	Retryable             bool
	DetailSafe            string
	PartialError          string
}

// ─── Domain Errors ───────────────────────────────────────────────────────────

var (
	ErrInvalidScope             = &DomainError{Code: "INVALID_SCOPE", Message: "tenant/workspace/task IDs must not be empty"}
	ErrCommentIDRequired        = &DomainError{Code: "COMMENT_ID_REQUIRED", Message: "comment_id must not be empty"}
	ErrTokenNotFound            = &DomainError{Code: "TOKEN_NOT_FOUND", Message: "access token not found"}
	ErrTokenExpired             = &DomainError{Code: "TOKEN_EXPIRED", Message: "access token has expired"}
	ErrScopeMismatch            = &DomainError{Code: "SCOPE_MISMATCH", Message: "URL scope does not match token scope"}
	ErrTokenExchangeInProgress  = &DomainError{Code: "TOKEN_EXCHANGE_IN_PROGRESS", Message: "token exchange is in progress, retry later"}
	ErrTokenExchangeAlreadyDone = &DomainError{Code: "TOKEN_EXCHANGE_ALREADY_DONE", Message: "预埋 AccessToken 仅可用于首次换取 RefreshToken"}
)

// DomainError is a domain-level error with a machine-readable code.
type DomainError struct {
	Code    string
	Message string
}

func (e *DomainError) Error() string { return e.Message }
