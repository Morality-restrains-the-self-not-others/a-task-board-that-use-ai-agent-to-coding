package application

import (
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strings"
	"time"

	"taskCredentialService/domain"
	"taskCredentialService/ports"
)

// LayerOauthService resolves per-repo OAuth access tokens for container layer push.
// Mirrors Django cloud.services.layer_github_oauth_tokens (simplified: no PR metadata).
type LayerOauthService struct {
	tokenRepo    ports.ContainerTokenRepository
	businessRepo ports.BusinessDataRepository
	gitoauth     ports.GitoauthClient
}

func NewLayerOauthService(
	tokenRepo ports.ContainerTokenRepository,
	businessRepo ports.BusinessDataRepository,
	gitoauth ports.GitoauthClient,
) *LayerOauthService {
	return &LayerOauthService{
		tokenRepo:    tokenRepo,
		businessRepo: businessRepo,
		gitoauth:     gitoauth,
	}
}

// ResolveLayerOauthTokens validates the container access token and fetches
// gitOauth access tokens for the task's linked repos.
//
// Auth failures (invalid/expired/scope mismatch) return a *domain.DomainError.
// Business failures (missing binding, no repos, upstream empty) return a result
// with OK=false and structured error fields (HTTP 409 at the handler).
func (s *LayerOauthService) ResolveLayerOauthTokens(
	accessToken string,
	scope domain.TaskScope,
	repoMatchKeys []string,
	repoSlugs []string,
) (*domain.LayerOauthTokensResult, error) {
	if err := scope.Validate(); err != nil {
		return nil, err
	}
	token, err := s.validateAccessToken(accessToken, scope)
	if err != nil {
		return nil, err
	}

	repos, err := s.businessRepo.FetchTaskRepos(scope.TaskID, token.CommentID)
	if err != nil {
		return nil, fmt.Errorf("fetch task repos: %w", err)
	}
	identities, err := s.businessRepo.FetchRepoIdentities(scope.TaskID, token.CommentID)
	if err != nil {
		return nil, fmt.Errorf("fetch repo identities: %w", err)
	}
	// Comment JSON selections (FromComment) must not be replaced by the author's
	// full identity list. Task-table / empty JSON still uses author-scoped fallback.
	authorID := lookupCommentAuthorUserID(s.businessRepo, token.CommentID)
	authorIdents := lookupUserGitIdentities(s.businessRepo, authorID)
	if !anyCommentSelectedIdentity(identities) {
		identities = SelectIdentitiesForCommentAuthor(identities, authorID, authorIdents)
	}

	allURLs := collectRepoURLs(repos)
	if len(allURLs) == 0 {
		return oauthConflict(
			"任务未关联 Git 仓库",
			"binding_check",
		), nil
	}

	matchKeys := normalizeStringList(repoMatchKeys)
	slugs := normalizeStringList(repoSlugs)

	var selected []selectedRepo
	if len(matchKeys) > 0 {
		selected = selectReposByMatchKeys(allURLs, matchKeys)
		if len(selected) == 0 {
			return oauthConflict(
				missingBindingDetail(matchKeys),
				"binding_check",
			), nil
		}
	} else if len(slugs) > 0 {
		selected = selectReposBySlugs(allURLs, slugs)
		if len(selected) == 0 {
			return oauthConflict(
				missingBindingDetail(slugs),
				"binding_check",
			), nil
		}
	} else {
		selected = selectAllRepos(allURLs)
	}

	githubAuth := make(map[string]string)
	matchKeyAuth := make(map[string]string)
	tokenCache := make(map[string]string) // userID|providerKey → token
	var missing []string
	var fetchErr string

	for _, sel := range selected {
		ident, ok := lookupIdentityForRepo(identities, sel.URL, authorID)
		if !ok || ident.UserID <= 0 {
			missing = append(missing, sel.Label)
			continue
		}
		if strings.TrimSpace(token.CommentID) != "" && token.CommentID != "-" && strings.TrimSpace(ident.OauthGitsite) == "" {
			missing = append(missing, sel.Label)
			continue
		}
		resolvedKey := s.gitoauth.ResolveProvider(sel.URL)
		site := cloneProviderSite(sel.URL)
		if site == "" {
			missing = append(missing, sel.Label)
			continue
		}
		if strings.TrimSpace(resolvedKey) == "" {
			log.Printf("[task-credential-service] layer-oauth provider yaml miss; gitsite path repo=%s site=%s user=%d",
				sel.URL, site, ident.UserID)
		}
		cacheKey := fmt.Sprintf("%d|%s", ident.UserID, site)
		access, cached := tokenCache[cacheKey]
		if !cached {
			access, err = s.gitoauth.FetchAccessToken(ident.UserID, site)
			if err != nil {
				fetchErr = err.Error()
				log.Printf("[task-credential-service] layer-oauth fetch failed: repo=%s user=%d site=%s err=%v",
					sel.URL, ident.UserID, site, err)
				break
			}
			access = strings.TrimSpace(access)
			if access == "" {
				fetchErr = "gitOauth 未找到该用户的 Git 凭据，请重新完成授权"
				break
			}
			tokenCache[cacheKey] = access
		}
		if sel.Slug != "" {
			githubAuth[sel.Slug] = access
		}
		if sel.MatchKey != "" {
			matchKeyAuth[sel.MatchKey] = access
		}
	}

	if fetchErr != "" && len(githubAuth) == 0 && len(matchKeyAuth) == 0 {
		stage := inferOauthFailedStage(fetchErr)
		return oauthConflict(fetchErr, stage), nil
	}
	if len(githubAuth) == 0 && len(matchKeyAuth) == 0 {
		labels := missing
		if len(labels) == 0 {
			labels = matchKeys
			if len(labels) == 0 {
				labels = slugs
			}
		}
		return oauthConflict(missingBindingDetail(labels), "binding_check"), nil
	}

	result := &domain.LayerOauthTokensResult{
		OK:               true,
		GithubAuthByRepo: githubAuth,
	}
	if len(matchKeys) > 0 && len(matchKeyAuth) > 0 {
		result.GitAuthByRepoMatchKey = matchKeyAuth
	}
	if fetchErr != "" {
		result.PartialError = fetchErr
	} else {
		resolved := len(githubAuth)
		if len(matchKeyAuth) > resolved {
			resolved = len(matchKeyAuth)
		}
		if skipped := len(selected) - resolved; skipped > 0 {
			result.PartialError = fmt.Sprintf("有 %d 个仓库未能换发 token", skipped)
		}
	}
	return result, nil
}

func (s *LayerOauthService) validateAccessToken(accessToken string, scope domain.TaskScope) (*domain.ContainerToken, error) {
	if strings.TrimSpace(accessToken) == "" {
		return nil, domain.ErrTokenNotFound
	}
	token, err := s.tokenRepo.FindByAccessToken(accessToken)
	if err != nil || token == nil {
		return nil, domain.ErrTokenNotFound
	}
	if token.CompanyID != scope.TenantID || token.WorkspaceID != scope.WorkspaceID || token.TaskID != scope.TaskID {
		return nil, domain.ErrScopeMismatch
	}
	if token.ContainerAccessTokenExpiresAt != "" {
		expiresAt, parseErr := time.Parse("2006-01-02 15:04:05", token.ContainerAccessTokenExpiresAt)
		if parseErr != nil || time.Now().After(expiresAt) {
			return nil, domain.ErrTokenExpired
		}
	}
	return token, nil
}

type selectedRepo struct {
	URL      string
	Slug     string
	MatchKey string
	Label    string // for error previews (slug or match key)
}

func selectAllRepos(urls []string) []selectedRepo {
	out := make([]selectedRepo, 0, len(urls))
	for _, u := range urls {
		slug := RepoSlugFromURL(u)
		mk := RepoMatchKeyFromURL(u)
		label := slug
		if label == "" {
			label = mk
		}
		out = append(out, selectedRepo{URL: u, Slug: slug, MatchKey: mk, Label: label})
	}
	return out
}

func selectReposByMatchKeys(urls []string, wantKeys []string) []selectedRepo {
	var out []selectedRepo
	seen := make(map[string]bool)
	for _, wantKey := range wantKeys {
		matchedURL := resolveURLForMatchKey(urls, wantKey)
		if matchedURL == "" {
			continue
		}
		if seen[wantKey] {
			continue
		}
		seen[wantKey] = true
		slug := RepoSlugFromURL(matchedURL)
		out = append(out, selectedRepo{
			URL:      matchedURL,
			Slug:     slug,
			MatchKey: wantKey,
			Label:    wantKey,
		})
	}
	return out
}

// resolveURLForMatchKey mirrors Django resolve_task_repo_url_for_match_key:
// prefer exact host/path match, else unique path-only match.
func resolveURLForMatchKey(urls []string, wantKey string) string {
	want := strings.ToLower(strings.TrimSpace(wantKey))
	if want == "" {
		return ""
	}
	wantPath := repoPathKey(want)
	var exact, pathOnly []string
	for _, u := range urls {
		u = strings.TrimSpace(u)
		if u == "" {
			continue
		}
		mk := RepoMatchKeyFromURL(u)
		if mk == want {
			exact = append(exact, u)
			continue
		}
		if wantPath != "" && repoPathKey(mk) == wantPath {
			pathOnly = append(pathOnly, u)
		}
	}
	if len(exact) >= 1 {
		return exact[0]
	}
	if len(pathOnly) == 1 {
		return pathOnly[0]
	}
	return ""
}

func selectReposBySlugs(urls []string, wantSlugs []string) []selectedRepo {
	want := make(map[string]bool, len(wantSlugs))
	for _, s := range wantSlugs {
		want[s] = true
	}
	var out []selectedRepo
	seen := make(map[string]bool)
	for _, u := range urls {
		slug := RepoSlugFromURL(u)
		if slug == "" || !want[slug] {
			continue
		}
		if seen[slug] {
			continue
		}
		seen[slug] = true
		out = append(out, selectedRepo{
			URL:      u,
			Slug:     slug,
			MatchKey: RepoMatchKeyFromURL(u),
			Label:    slug,
		})
	}
	return out
}

func normalizeStringList(raw []string) []string {
	if len(raw) == 0 {
		return nil
	}
	out := make([]string, 0, len(raw))
	seen := make(map[string]bool)
	for _, x := range raw {
		v := strings.ToLower(strings.TrimSpace(x))
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}

func missingBindingDetail(labels []string) string {
	n := 3
	if len(labels) < n {
		n = len(labels)
	}
	preview := strings.Join(labels[:n], ", ")
	if len(labels) > 3 {
		preview += " ..."
	}
	if preview == "" {
		preview = "(none)"
	}
	return "请先在创建或编辑任务、或评论「提交并运行」时为每个仓库完成 Git 授权绑定。" +
		"（缺少绑定: " + preview + "）"
}

func oauthConflict(detail, failedStage string) *domain.LayerOauthTokensResult {
	code, retryable, detailSafe := classifyOauthError(failedStage, detail)
	return &domain.LayerOauthTokensResult{
		OK:               false,
		GithubAuthByRepo: map[string]string{},
		Detail:           detail,
		ErrorCode:        code,
		FailedStage:      failedStage,
		Retryable:        retryable,
		DetailSafe:       detailSafe,
	}
}

func inferOauthFailedStage(detail string) string {
	text := strings.ToLower(strings.TrimSpace(detail))
	bindingMarkers := []string{
		"缺少绑定", "关联项目", "为每个仓库选择并保存", "多个 github 授权", "repo binding",
	}
	for _, m := range bindingMarkers {
		if strings.Contains(text, m) {
			return "binding_check"
		}
	}
	if strings.Contains(text, "access_token") && (strings.Contains(text, "无效") || strings.Contains(text, "过期")) {
		return "token_check"
	}
	if strings.Contains(text, "任务不存在") || strings.Contains(text, "task_id 无效") {
		return "entry"
	}
	return "gitoauth_summary"
}

func classifyOauthError(failedStage, detail string) (code string, retryable bool, detailSafe string) {
	text := strings.ToLower(detail)
	switch {
	case containsAny(text, "timed out", "timeout", "read timeout", "connect timeout", "aborted"):
		return "UPSTREAM_GITOAUTH_TIMEOUT", true, "访问 gitOauth 超时，请稍后重试"
	case containsAny(text, "connectionerror", "connection reset", "max retries exceeded",
		"failed to establish a new connection", "name or service not known"):
		return "UPSTREAM_GITOAUTH_NETWORK", true, "访问 gitOauth 网络异常，请检查网络或稍后重试"
	case containsAny(text, "server error", "bad gateway", "http 500", "http 502", "http 503",
		"invalid json", "bad response"):
		return "UPSTREAM_GITOAUTH_BAD_RESPONSE", false, "gitOauth 返回异常，请稍后重试"
	case failedStage == "binding_check":
		return "BINDING_MISSING", false, "请先完成该仓库的 Git 授权绑定"
	case failedStage == "token_check":
		return "TOKEN_INVALID", false, "access_token 无效或已过期"
	default:
		return "UNKNOWN_ERROR", false, "访问 gitOauth 失败，请稍后重试"
	}
}

func containsAny(text string, keywords ...string) bool {
	for _, k := range keywords {
		if strings.Contains(text, k) {
			return true
		}
	}
	return false
}

var gitSCPRe = regexp.MustCompile(`(?i)^git@([^:]+):(.+?)(?:\.git)?/?$`)

// RepoMatchKeyFromURL builds host/path (lowercase, no .git), aligned with Django repo_match_key_from_url.
func RepoMatchKeyFromURL(raw string) string {
	u := strings.TrimSpace(raw)
	if u == "" {
		return ""
	}
	if m := gitSCPRe.FindStringSubmatch(u); m != nil {
		host := strings.ToLower(m[1])
		path := strings.ReplaceAll(m[2], "\\", "/")
		path = strings.TrimRight(path, "/")
		if strings.HasSuffix(strings.ToLower(path), ".git") {
			path = path[:len(path)-4]
		}
		return strings.ToLower(host + "/" + path)
	}
	parsed, err := url.Parse(u)
	if err == nil && parsed.Host != "" {
		host := strings.ToLower(parsed.Host)
		if strings.EqualFold(parsed.Scheme, "ssh") {
			host = strings.ToLower(parsed.Hostname())
		}
		path := strings.TrimRight(parsed.Path, "/")
		if strings.HasSuffix(strings.ToLower(path), ".git") {
			path = path[:len(path)-4]
		}
		path = strings.TrimPrefix(path, "/")
		return strings.ToLower(host + "/" + path)
	}
	normalized := strings.ToLower(strings.TrimRight(u, "/"))
	if strings.HasSuffix(normalized, ".git") {
		normalized = normalized[:len(normalized)-4]
	}
	return normalized
}

func repoPathKey(matchKey string) string {
	if !strings.Contains(matchKey, "/") {
		return matchKey
	}
	parts := strings.SplitN(matchKey, "/", 2)
	if len(parts) < 2 {
		return matchKey
	}
	return parts[1]
}

// RepoSlugFromURL returns owner/repo from the last two path segments (lowercase).
func RepoSlugFromURL(raw string) string {
	key := RepoMatchKeyFromURL(raw)
	if key == "" {
		return ""
	}
	path := repoPathKey(key)
	parts := strings.Split(path, "/")
	var segs []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			segs = append(segs, p)
		}
	}
	if len(segs) < 2 {
		return ""
	}
	owner := segs[len(segs)-2]
	repo := segs[len(segs)-1]
	if strings.HasSuffix(strings.ToLower(repo), ".git") {
		repo = repo[:len(repo)-4]
	}
	if owner == "" || repo == "" {
		return ""
	}
	return strings.ToLower(owner + "/" + repo)
}
