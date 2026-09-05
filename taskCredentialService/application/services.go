// Package application contains application services that orchestrate domain
// services and port implementations. No business logic here — only workflow.
package application

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"taskCredentialService/domain"
	"taskCredentialService/ports"
)

// ─── Token Service ───────────────────────────────────────────────────────────

// TokenService orchestrates token lifecycle: issue, validate, exchange, refresh.
type TokenService struct {
	tokenRepo ports.ContainerTokenRepository
	auditRepo ports.TokenAuditEventRepository
}

func NewTokenService(tokenRepo ports.ContainerTokenRepository, auditRepo ports.TokenAuditEventRepository) *TokenService {
	return &TokenService{tokenRepo: tokenRepo, auditRepo: auditRepo}
}

// tokenReuseMinTTL is the minimum remaining lifetime for a token to be reused.
// Tokens with less than this duration left are treated as expired for reuse purposes,
// ensuring the Go relay has enough time to perform exchange-refresh + refresh-access.
const tokenReuseMinTTL = 5 * time.Minute

// IssueToken creates a new container access token for a task.
func (s *TokenService) IssueToken(scope domain.TaskScope, envOverrides map[string]string) (*domain.ContainerToken, error) {
	if err := scope.ValidateIssue(); err != nil {
		return nil, err
	}
	existing, _ := s.tokenRepo.FindByScope(scope.TenantID, scope.WorkspaceID, scope.TaskID, scope.CommentID)
	if existing != nil && existing.ContainerAccessToken != "" && existing.ContainerRefreshToken == "" && !s.isExpired(existing) && s.remainingTTL(existing) >= tokenReuseMinTTL {
		return existing, nil
	}
	accessToken := generateToken()
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	expiresAt := time.Now().Add(1 * time.Hour).UTC().Format("2006-01-02 15:04:05")
	if existing != nil {
		if err := s.tokenRepo.UpdateAccessToken(existing.ID, accessToken, expiresAt); err != nil {
			return nil, fmt.Errorf("issue token: update access: %w", err)
		}
		existing.ContainerAccessToken = accessToken
		existing.ContainerAccessTokenExpiresAt = expiresAt
		existing.UpdatedAt = now
		s.recordAudit(existing, "token_issued", "", nil)
		return existing, nil
	}
	token := &domain.ContainerToken{
		ID:                            generateSnowflake(),
		TaskID:                        scope.TaskID,
		CompanyID:                     scope.TenantID,
		WorkspaceID:                   scope.WorkspaceID,
		CommentID:                     scope.CommentID,
		ContainerAccessToken:          accessToken,
		ContainerAccessTokenExpiresAt: expiresAt,
		ContainerRefreshToken:         "",
		CreatedAt:                     now,
		UpdatedAt:                     now,
	}
	saved, err := s.tokenRepo.Save(token)
	if err != nil {
		return nil, fmt.Errorf("issue token: %w", err)
	}
	s.recordAudit(saved, "token_issued", "", nil)
	return saved, nil
}

// ValidateToken checks that an access token is valid for the given scope.
func (s *TokenService) ValidateToken(accessToken string, scope domain.TaskScope) (*domain.ContainerToken, error) {
	if err := scope.Validate(); err != nil {
		return nil, err
	}
	token, err := s.tokenRepo.FindByAccessToken(accessToken)
	if err != nil || token == nil {
		return nil, domain.ErrTokenNotFound
	}
	if token.CompanyID != scope.TenantID || token.WorkspaceID != scope.WorkspaceID || token.TaskID != scope.TaskID {
		return nil, domain.ErrScopeMismatch
	}
	if domain.CommentConflicts(token.CommentID, scope.CommentID) {
		return nil, domain.ErrScopeMismatch
	}
	if s.isExpired(token) {
		return nil, domain.ErrTokenExpired
	}
	return token, nil
}

// FindByAccessToken looks up a token without scope validation (for internal use).
func (s *TokenService) FindByAccessToken(accessToken string) (*domain.ContainerToken, error) {
	token, err := s.tokenRepo.FindByAccessToken(accessToken)
	if err != nil || token == nil {
		return nil, domain.ErrTokenNotFound
	}
	return token, nil
}

// IsExpired checks whether the token's access token has expired.
func (s *TokenService) IsExpired(token *domain.ContainerToken) bool {
	return s.isExpired(token)
}

// FindByScope returns the current token for a given scope without mutating state.
// Returns nil if no token record exists for the scope.
func (s *TokenService) FindByScope(scope domain.TaskScope) (*domain.ContainerToken, error) {
	if err := scope.Validate(); err != nil {
		return nil, err
	}
	token, err := s.tokenRepo.FindByScope(scope.TenantID, scope.WorkspaceID, scope.TaskID, scope.CommentID)
	if err != nil {
		return nil, err
	}
	return token, nil
}

func (s *TokenService) CountByTaskScope(scope domain.TaskScope) (int, error) {
	return s.tokenRepo.CountByTaskScope(scope.TenantID, scope.WorkspaceID, scope.TaskID)
}

// hasForwardableAccess reports whether the row holds a non-empty access_token that SaaS can
// present to the container. Container auth matches process.env.ACCESS_TOKEN by string equality
// and does NOT enforce platform expires_at — so an expired-but-non-empty access is still
// forwardable (and preferable to minting a new one that would desync env → 401).
// After exchange-refresh, access is cleared (expires_at also "") — that must NOT count.
func (s *TokenService) hasForwardableAccess(token *domain.ContainerToken) bool {
	return token != nil && strings.TrimSpace(token.ContainerAccessToken) != ""
}

// EnsureAccessByScope returns a token with a forwardable access_token for SaaS→container calls.
//
//  1. access 非空（含已过期）：原样返回，不单方 RefreshAccess（防 env 比对 401）。
//  2. access 被 exchange-refresh 清空且仍有 refresh：自动 RefreshAccess，避免 by-scope
//     200 空 token → 网关 409「缺少容器 access_token」。
//  3. access 空且无 refresh：TOKEN_NOT_FOUND。
func (s *TokenService) EnsureAccessByScope(scope domain.TaskScope) (*domain.ContainerToken, error) {
	if strings.TrimSpace(scope.CommentID) == "" {
		n, err := s.tokenRepo.CountByTaskScope(scope.TenantID, scope.WorkspaceID, scope.TaskID)
		if err != nil {
			return nil, err
		}
		if n > 1 {
			return nil, domain.ErrCommentIDRequired
		}
	}
	token, err := s.FindByScope(scope)
	if err != nil {
		return nil, err
	}
	if token == nil {
		return nil, domain.ErrTokenNotFound
	}
	if s.hasForwardableAccess(token) {
		return token, nil
	}
	refresh := strings.TrimSpace(token.ContainerRefreshToken)
	if refresh != "" {
		newAccess, expiresAt, err := s.RefreshAccess(refresh, scope)
		if err != nil {
			return nil, err
		}
		token.ContainerAccessToken = newAccess
		token.ContainerAccessTokenExpiresAt = expiresAt
		log.Printf("[task-credential-service] ensure-access-by-scope: filled cleared access task=%s", token.TaskID)
		return token, nil
	}
	return nil, domain.ErrTokenNotFound
}

func (s *TokenService) isExpired(token *domain.ContainerToken) bool {
	if token.ContainerAccessTokenExpiresAt == "" {
		return false
	}
	expiresAt, err := time.Parse("2006-01-02 15:04:05", token.ContainerAccessTokenExpiresAt)
	if err != nil {
		return true // unparseable expiry → treat as expired
	}
	return time.Now().After(expiresAt)
}

// remainingTTL returns the time until the token expires.
// Returns 0 if the expiry is empty or unparseable.
func (s *TokenService) remainingTTL(token *domain.ContainerToken) time.Duration {
	if token.ContainerAccessTokenExpiresAt == "" {
		return 0
	}
	expiresAt, err := time.Parse("2006-01-02 15:04:05", token.ContainerAccessTokenExpiresAt)
	if err != nil {
		return 0
	}
	return time.Until(expiresAt)
}

func (s *TokenService) recordAudit(token *domain.ContainerToken, eventType, errorCode string, errDetail *string) {
	detail := ""
	if errDetail != nil {
		detail = *errDetail
	}
	hash := sha256Hex(token.ContainerAccessToken)
	event := &domain.TokenAuditEvent{
		ID:                generateSnowflake(),
		TaskID:            token.TaskID,
		CommentID:         token.CommentID,
		EventType:         eventType,
		AccessTokenSHA256: hash,
		ErrorCode:         errorCode,
		ErrorDetail:       detail,
		CreatedAt:         time.Now().UTC().Format("2006-01-02 15:04:05"),
	}
	_ = s.auditRepo.Save(event) // best-effort audit
}

// AppendAuditEvent persists an external audit event (relay lifecycle consumer).
func (s *TokenService) AppendAuditEvent(event *domain.TokenAuditEvent) error {
	if event == nil {
		return fmt.Errorf("audit event required")
	}
	if strings.TrimSpace(event.TaskID) == "" || strings.TrimSpace(event.EventType) == "" {
		return fmt.Errorf("task_id and event_type required")
	}
	if strings.TrimSpace(event.ID) == "" {
		event.ID = generateSnowflake()
	}
	if strings.TrimSpace(event.CreatedAt) == "" {
		event.CreatedAt = time.Now().UTC().Format("2006-01-02 15:04:05")
	}
	return s.auditRepo.Save(event)
}

func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

// ─── Credential Service ──────────────────────────────────────────────────────

// CredentialService orchestrates repo clone credential building.
type CredentialService struct {
	tokenRepo     ports.ContainerTokenRepository
	businessRepo  ports.BusinessDataRepository
	gitoauth      ports.GitoauthClient
	nestedFetcher NestedGitReposFetcher
}

func NewCredentialService(
	tokenRepo ports.ContainerTokenRepository,
	businessRepo ports.BusinessDataRepository,
	gitoauth ports.GitoauthClient,
) *CredentialService {
	return &CredentialService{
		tokenRepo:    tokenRepo,
		businessRepo: businessRepo,
		gitoauth:     gitoauth,
	}
}

// WithNestedFetcher enables latest nested-git-repos enrich for clone credentials.
func (s *CredentialService) WithNestedFetcher(f NestedGitReposFetcher) *CredentialService {
	s.nestedFetcher = f
	return s
}

// BuildRepoCloneCredentials validates the token and builds per-repo credentials.
func (s *CredentialService) BuildRepoCloneCredentials(accessToken string, scope domain.TaskScope) (*domain.RepoCloneCredentialsResult, error) {
	// Validate token.
	token, err := s.ValidateToken(accessToken, scope)
	if err != nil {
		return nil, err
	}
	identities, err := s.businessRepo.FetchRepoIdentities(scope.TaskID, token.CommentID)
	if err != nil {
		return nil, fmt.Errorf("fetch identities: %w", err)
	}
	repos, err := s.businessRepo.FetchTaskRepos(scope.TaskID, token.CommentID)
	if err != nil {
		return nil, fmt.Errorf("fetch repos: %w", err)
	}
	authorID := lookupCommentAuthorUserID(s.businessRepo, token.CommentID)
	authorIdents := lookupUserGitIdentities(s.businessRepo, authorID)
	repos, identities, discoveryUserID := EnrichReposForCommentAuthor(
		authorID, authorIdents, identities, repos, scope.TenantID, s.nestedFetcher,
	)
	log.Printf("[task-credential-service] repo-clone nested enrich task=%s comment=%s author=%d discovery_user=%d repos=%d",
		scope.TaskID, token.CommentID, authorID, discoveryUserID, len(collectRepoURLs(repos)))
	return buildCredentials(collectRepoURLs(repos), identities, s.gitoauth), nil
}

func (s *CredentialService) ValidateToken(accessToken string, scope domain.TaskScope) (*domain.ContainerToken, error) {
	if accessToken == "" {
		return nil, domain.ErrTokenNotFound
	}
	token, err := s.tokenRepo.FindByAccessToken(accessToken)
	if err != nil || token == nil {
		return nil, domain.ErrTokenNotFound
	}
	if s.isExpired(token) {
		return nil, domain.ErrTokenExpired
	}
	return token, nil
}

func (s *CredentialService) isExpired(token *domain.ContainerToken) bool {
	if token.ContainerAccessTokenExpiresAt == "" {
		return false
	}
	expiresAt, err := time.Parse("2006-01-02 15:04:05", token.ContainerAccessTokenExpiresAt)
	if err != nil {
		return true
	}
	return time.Now().After(expiresAt)
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func collectRepoURLs(repos []domain.TaskRepoSnapshot) []string {
	seen := make(map[string]bool)
	var urls []string
	for _, r := range repos {
		for _, url := range r.RepoURLs {
			if !seen[url] {
				seen[url] = true
				urls = append(urls, url)
			}
		}
	}
	return urls
}

func generateToken() string {
	return "tok_" + generateSnowflake()
}

// buildCredentials iterates repos, resolves providers, and calls gitoauth for each.
// Mirrors Python _build_repo_clone_credentials_with_diagnostics.
//
// 同一 (userID, providerKey) 只换票一次并复用：GitLab/GitHub 的 refresh 会吊销
// 上一次 access_token；多仓任务若每仓各刷一次，先拿到的 token 在 clone 时已失效。
func buildCredentials(repoURLs []string, identities []domain.GitIdentitySnapshot, gitoauth ports.GitoauthClient) *domain.RepoCloneCredentialsResult {
	result := &domain.RepoCloneCredentialsResult{
		Credentials: make(map[string]domain.RepoCloneCredential),
	}

	tokenCache := make(map[string]string) // userID|providerKey → access_token
	tokenErrCache := make(map[string]error)

	for _, repoURL := range repoURLs {
		ident, ok := lookupIdentityForRepo(identities, repoURL, 0)
		if !ok {
			result.MissingIdentityRepos = append(result.MissingIdentityRepos, repoURL)
			continue
		}
		resolvedKey := gitoauth.ResolveProvider(repoURL)
		site := cloneProviderSite(repoURL)
		if site == "" {
			result.MissingIdentityRepos = append(result.MissingIdentityRepos, repoURL)
			continue
		}
		if ident.UserID <= 0 {
			result.MissingIdentityRepos = append(result.MissingIdentityRepos, repoURL)
			continue
		}
		if strings.TrimSpace(resolvedKey) == "" {
			log.Printf("[task-credential-service] clone provider yaml miss; gitsite path repo=%s site=%s user=%d",
				repoURL, site, ident.UserID)
		}
		cacheKey := fmt.Sprintf("%d|%s", ident.UserID, site)
		token, cached := tokenCache[cacheKey]
		if !cached {
			if prevErr, hadErr := tokenErrCache[cacheKey]; hadErr {
				result.TokenRefreshFailures = append(result.TokenRefreshFailures, domain.TokenRefreshFailure{
					RepoURL:   repoURL,
					ErrorCode: "GITOAUTH_UPSTREAM_ERROR",
					Detail:    prevErr.Error(),
				})
				continue
			}
			var err error
			token, err = gitoauth.FetchAccessToken(ident.UserID, site)
			if err != nil {
				log.Printf("[task-credential-service] WARN token refresh failed: repo=%s site=%s user=%d err=%v",
					repoURL, site, ident.UserID, err)
				tokenErrCache[cacheKey] = err
				result.TokenRefreshFailures = append(result.TokenRefreshFailures, domain.TokenRefreshFailure{
					RepoURL:   repoURL,
					ErrorCode: "GITOAUTH_UPSTREAM_ERROR",
					Detail:    err.Error(),
				})
				continue
			}
			tokenCache[cacheKey] = token
		}
		provider := cloneCredentialProvider(resolvedKey, repoURL)
		cred := domain.RepoCloneCredential{
			RepoURL:                   repoURL,
			EphemeralOAuthAccessToken: token,
			Provider:                  provider,
			GitHTTPUsername:           gitHTTPUsername(provider),
			HttpsCloneURL:             gitoauth.ResolveHttpsCloneURL(repoURL),
		}
		result.Credentials[repoURL] = cred
	}
	log.Printf("[task-credential-service] credentials built: total=%d ok=%d missing=%d failures=%d fetches=%d",
		len(repoURLs), len(result.Credentials), len(result.MissingIdentityRepos), len(result.TokenRefreshFailures), len(tokenCache))
	return result
}

// gitHTTPUsername returns the HTTP Basic username for git clone auth.
func gitHTTPUsername(provider string) string {
	switch provider {
	case "github":
		return "x-access-token"
	case "gitlab":
		return "oauth2"
	default:
		return ""
	}
}

// tokenIDMu guards the millisecond/sequence state below. Row IDs and tokens
// must stay unique even when calls land in the same millisecond — the previous
// timestamp-only generator returned identical values for rapid back-to-back
// calls (nightly flake: two exchange-refresh within one ms shared a refresh
// token, breaking per-comment refresh isolation).
var (
	tokenIDMu   sync.Mutex
	tokenIDLast int64 // last returned Unix millisecond
	tokenIDSeq  int64 // per-millisecond monotonic sequence
)

// generateSnowflake returns a process-unique, monotonic ID. The leading part is
// the Unix millisecond timestamp (readable, historically stable); a per-process
// sequence disambiguates calls within the same millisecond.
func generateSnowflake() string {
	tokenIDMu.Lock()
	defer tokenIDMu.Unlock()
	ms := time.Now().UnixNano() / 1e6
	if ms <= tokenIDLast {
		tokenIDSeq++
	} else {
		tokenIDLast = ms
		tokenIDSeq = 0
	}
	return fmt.Sprintf("%d%06d", tokenIDLast, tokenIDSeq)
}
