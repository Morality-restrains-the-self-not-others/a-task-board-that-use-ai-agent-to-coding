package application_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"taskCredentialService/application"
	"taskCredentialService/domain"
)

// ─── stubs ───────────────────────────────────────────────────────────────────

type layerTokenRepo struct {
	token *domain.ContainerToken
}

func (r *layerTokenRepo) Save(token *domain.ContainerToken) (*domain.ContainerToken, error) {
	r.token = token
	return token, nil
}
func (r *layerTokenRepo) FindByAccessToken(accessToken string) (*domain.ContainerToken, error) {
	if r.token != nil && r.token.ContainerAccessToken == accessToken {
		return r.token, nil
	}
	return nil, nil
}
func (r *layerTokenRepo) FindByTaskID(companyID, taskID string) (*domain.ContainerToken, error) {
	return r.token, nil
}
func (r *layerTokenRepo) FindByScope(companyID, workspaceID, taskID, commentID string) (*domain.ContainerToken, error) {
	return r.token, nil
}
func (r *layerTokenRepo) CountByTaskScope(companyID, workspaceID, taskID string) (int, error) {
	if r.token == nil {
		return 0, nil
	}
	return 1, nil
}
func (r *layerTokenRepo) UpdateAccessToken(id, newAccessToken, expiresAt string) error {
	return nil
}
func (r *layerTokenRepo) UpdateRefreshToken(id, newRefreshToken string) error { return nil }
func (r *layerTokenRepo) FindByRefreshToken(refreshToken string) (*domain.ContainerToken, error) {
	return nil, nil
}
func (r *layerTokenRepo) UpdateBusinessAPIEndpoint(id, endpoint string) error { return nil }

type layerBusinessRepo struct {
	repos           []domain.TaskRepoSnapshot
	identities      []domain.GitIdentitySnapshot
	commentAuthorID int64
	userIdentities  []domain.GitIdentitySnapshot
	reposErr        error
	identErr        error
}

func (r *layerBusinessRepo) FetchTaskSnapshot(taskID, commentID string) (*domain.TaskSnapshot, error) {
	return nil, nil
}
func (r *layerBusinessRepo) FetchTaskRepos(taskID, commentID string) ([]domain.TaskRepoSnapshot, error) {
	if r.reposErr != nil {
		return nil, r.reposErr
	}
	return r.repos, nil
}
func (r *layerBusinessRepo) FetchRepoIdentities(taskID, commentID string) ([]domain.GitIdentitySnapshot, error) {
	if r.identErr != nil {
		return nil, r.identErr
	}
	return r.identities, nil
}
func (r *layerBusinessRepo) FetchCommentCreatedByUserID(commentID string) (int64, error) {
	return r.commentAuthorID, nil
}
func (r *layerBusinessRepo) FetchUserGitIdentities(userID int64) ([]domain.GitIdentitySnapshot, error) {
	return r.userIdentities, nil
}

type layerGitoauth struct {
	providerByURL map[string]string
	tokenByUser   map[int64]string
	fetchErr      error
	lastKey       string
}

func (c *layerGitoauth) FetchAccessToken(userID int64, providerKey string) (string, error) {
	c.lastKey = providerKey
	if c.fetchErr != nil {
		return "", c.fetchErr
	}
	tok, ok := c.tokenByUser[userID]
	if !ok || tok == "" {
		return "", errors.New("user not bound")
	}
	return tok, nil
}
func (c *layerGitoauth) ResolveHttpsCloneURL(repoURL string) string {
	return repoURL
}

func (c *layerGitoauth) ResolveProvider(repoURL string) string {
	if c.providerByURL != nil {
		if k, ok := c.providerByURL[repoURL]; ok {
			return k
		}
	}
	low := strings.ToLower(repoURL)
	if strings.Contains(low, "github.com") {
		return "github:github-official"
	}
	if strings.Contains(low, "gitlab") {
		return "gitlab:default"
	}
	return ""
}

func validScope() domain.TaskScope {
	return domain.TaskScope{TenantID: "t1", WorkspaceID: "w1", TaskID: "task1"}
}

func validToken() *domain.ContainerToken {
	return &domain.ContainerToken{
		ID:                            "tok1",
		TaskID:                        "task1",
		CompanyID:                     "t1",
		WorkspaceID:                   "w1",
		ContainerAccessToken:          "access-valid",
		ContainerAccessTokenExpiresAt: time.Now().Add(time.Hour).UTC().Format("2006-01-02 15:04:05"),
	}
}

func newLayerSvc(tok *domain.ContainerToken, biz *layerBusinessRepo, git *layerGitoauth) *application.LayerOauthService {
	return application.NewLayerOauthService(&layerTokenRepo{token: tok}, biz, git)
}

// ─── helpers unit tests ──────────────────────────────────────────────────────

func TestRepoMatchKeyFromURL(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"https://github.com/acme/demo.git", "github.com/acme/demo"},
		{"http://localhost:8012/ljy/somanyad.git", "localhost:8012/ljy/somanyad"},
		{"git@github.com:acme/demo.git", "github.com/acme/demo"},
		{"ssh://git@github.com/acme/demo.git", "github.com/acme/demo"},
		{"ssh://git@gitlab.daydaymoney.com:2222/g/p.git", "gitlab.daydaymoney.com/g/p"},
	}
	for _, tc := range cases {
		got := application.RepoMatchKeyFromURL(tc.in)
		if got != tc.want {
			t.Errorf("RepoMatchKeyFromURL(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

func TestRepoSlugFromURL(t *testing.T) {
	got := application.RepoSlugFromURL("https://github.com/Acme/Demo.git")
	if got != "acme/demo" {
		t.Fatalf("slug=%q want acme/demo", got)
	}
}

// ─── ResolveLayerOauthTokens ─────────────────────────────────────────────────

func TestResolveLayerOauthTokens_SuccessBySlugs(t *testing.T) {
	repoURL := "https://github.com/acme/demo.git"
	svc := newLayerSvc(validToken(), &layerBusinessRepo{
		repos: []domain.TaskRepoSnapshot{
			{ProjectID: "p1", ProjectName: "p", RepoURLs: []string{repoURL}},
		},
		identities: []domain.GitIdentitySnapshot{
			{RepoURL: repoURL, GitIdentityID: "gid1", UserID: 42},
		},
	}, &layerGitoauth{
		tokenByUser: map[int64]string{42: "gh_pat_abc"},
	})

	result, err := svc.ResolveLayerOauthTokens("access-valid", validScope(), nil, []string{"acme/demo"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !result.OK {
		t.Fatalf("expected OK, got %+v", result)
	}
	if result.GithubAuthByRepo["acme/demo"] != "gh_pat_abc" {
		t.Fatalf("github_auth_by_repo=%v", result.GithubAuthByRepo)
	}
	if len(result.GitAuthByRepoMatchKey) != 0 {
		t.Fatalf("match key map should be empty without repo_match_keys, got %v", result.GitAuthByRepoMatchKey)
	}
}

func TestResolveLayerOauthTokens_SuccessByMatchKeys(t *testing.T) {
	repoURL := "https://github.com/acme/demo.git"
	svc := newLayerSvc(validToken(), &layerBusinessRepo{
		repos: []domain.TaskRepoSnapshot{
			{ProjectID: "p1", ProjectName: "p", RepoURLs: []string{repoURL}},
		},
		identities: []domain.GitIdentitySnapshot{
			{RepoURL: repoURL, GitIdentityID: "gid1", UserID: 7},
		},
	}, &layerGitoauth{
		tokenByUser: map[int64]string{7: "gh_tok_match"},
	})

	result, err := svc.ResolveLayerOauthTokens(
		"access-valid",
		validScope(),
		[]string{"github.com/acme/demo"},
		nil,
	)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !result.OK {
		t.Fatalf("expected OK, got %+v", result)
	}
	if result.GithubAuthByRepo["acme/demo"] != "gh_tok_match" {
		t.Fatalf("slug map=%v", result.GithubAuthByRepo)
	}
	if result.GitAuthByRepoMatchKey["github.com/acme/demo"] != "gh_tok_match" {
		t.Fatalf("match key map=%v", result.GitAuthByRepoMatchKey)
	}
}

func TestResolveLayerOauthTokens_MissingBinding409(t *testing.T) {
	repoURL := "https://github.com/acme/demo.git"
	svc := newLayerSvc(validToken(), &layerBusinessRepo{
		repos: []domain.TaskRepoSnapshot{
			{ProjectID: "p1", ProjectName: "p", RepoURLs: []string{repoURL}},
		},
		identities: nil, // no identity binding
	}, &layerGitoauth{
		tokenByUser: map[int64]string{42: "unused"},
	})

	result, err := svc.ResolveLayerOauthTokens("access-valid", validScope(), nil, []string{"acme/demo"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if result.OK {
		t.Fatal("expected conflict result")
	}
	if result.ErrorCode != "BINDING_MISSING" {
		t.Fatalf("error_code=%q want BINDING_MISSING", result.ErrorCode)
	}
	if result.FailedStage != "binding_check" {
		t.Fatalf("failed_stage=%q", result.FailedStage)
	}
	if result.Retryable {
		t.Fatal("retryable should be false")
	}
	if result.GithubAuthByRepo == nil || len(result.GithubAuthByRepo) != 0 {
		t.Fatalf("github_auth_by_repo should be empty map, got %v", result.GithubAuthByRepo)
	}
	if !strings.Contains(result.Detail, "缺少绑定") {
		t.Fatalf("detail=%q", result.Detail)
	}
	if result.DetailSafe == "" {
		t.Fatal("detail_safe required")
	}
	if strings.Contains(result.Detail, "关联项目") {
		t.Fatalf("detail must not send users to read-only 关联项目: %q", result.Detail)
	}
	if !strings.Contains(result.DetailSafe, "Git 授权") {
		t.Fatalf("detail_safe=%q want Git 授权", result.DetailSafe)
	}
	if strings.Contains(result.DetailSafe, "GitHub") && !strings.Contains(result.DetailSafe, "Git 授权") {
		t.Fatalf("detail_safe must not be GitHub-only: %q", result.DetailSafe)
	}
	if strings.Contains(result.DetailSafe, "GitHub") {
		t.Fatalf("detail_safe must not say GitHub for generic bind miss: %q", result.DetailSafe)
	}
}

func TestResolveLayerOauthTokens_InvalidToken401(t *testing.T) {
	svc := newLayerSvc(validToken(), &layerBusinessRepo{}, &layerGitoauth{})

	_, err := svc.ResolveLayerOauthTokens("access-bogus", validScope(), nil, nil)
	if err == nil {
		t.Fatal("expected domain error")
	}
	de, ok := err.(*domain.DomainError)
	if !ok {
		t.Fatalf("want DomainError, got %T %v", err, err)
	}
	if de.Code != "TOKEN_NOT_FOUND" {
		t.Fatalf("code=%q want TOKEN_NOT_FOUND", de.Code)
	}
}

func TestResolveLayerOauthTokens_ScopeMismatch(t *testing.T) {
	svc := newLayerSvc(validToken(), &layerBusinessRepo{}, &layerGitoauth{})
	wrong := domain.TaskScope{TenantID: "other", WorkspaceID: "w1", TaskID: "task1"}

	_, err := svc.ResolveLayerOauthTokens("access-valid", wrong, nil, nil)
	de, ok := err.(*domain.DomainError)
	if !ok || de.Code != "SCOPE_MISMATCH" {
		t.Fatalf("want SCOPE_MISMATCH, got %v", err)
	}
}

// OPT-20260816-004: when the token carries a comment_id and the task-repo
// identity belongs to another user, layer-oauth must exchange tokens for the
// comment author, not the task-owner.
func TestResolveLayerOauthTokens_CommentAuthorOverridesTaskOwnerIdentity(t *testing.T) {
	repoURL := "https://github.com/acme/demo.git"
	tok := validToken()
	tok.CommentID = "comment-1"

	svc := newLayerSvc(tok, &layerBusinessRepo{
		repos: []domain.TaskRepoSnapshot{
			{ProjectID: "p1", ProjectName: "p", RepoURLs: []string{repoURL}},
		},
		identities: []domain.GitIdentitySnapshot{
			// task-owner binding (someone else than the comment author)
			{RepoURL: repoURL, GitIdentityID: "gid-owner", UserID: 42, OauthGitsite: "github.com"},
		},
		commentAuthorID: 7,
		userIdentities: []domain.GitIdentitySnapshot{
			// comment author's own binding for the same repo
			{RepoURL: repoURL, GitIdentityID: "gid-author", UserID: 7, OauthGitsite: "github.com"},
		},
	}, &layerGitoauth{
		tokenByUser: map[int64]string{42: "gh_pat_owner", 7: "gh_pat_author"},
	})

	result, err := svc.ResolveLayerOauthTokens("access-valid", validScope(), nil, []string{"acme/demo"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !result.OK {
		t.Fatalf("expected OK, got %+v", result)
	}
	if got := result.GithubAuthByRepo["acme/demo"]; got != "gh_pat_author" {
		t.Fatalf("layer-oauth must use comment author token, got %q want %q", got, "gh_pat_author")
	}
}

// OPT-20260816-004: when the task-repo identity already belongs to the comment
// author, it wins over the author's separately-fetched identities.
func TestResolveLayerOauthTokens_CommentAuthorPrefersMatchedTaskIdentity(t *testing.T) {
	repoURL := "https://github.com/acme/demo.git"
	tok := validToken()
	tok.CommentID = "comment-1"

	svc := newLayerSvc(tok, &layerBusinessRepo{
		repos: []domain.TaskRepoSnapshot{
			{ProjectID: "p1", ProjectName: "p", RepoURLs: []string{repoURL}},
		},
		identities: []domain.GitIdentitySnapshot{
			{RepoURL: repoURL, GitIdentityID: "gid-task", UserID: 7, OauthGitsite: "github.com"},
		},
		commentAuthorID: 7,
		userIdentities: []domain.GitIdentitySnapshot{
			{RepoURL: repoURL, GitIdentityID: "gid-user", UserID: 7, OauthGitsite: "github.com"},
		},
	}, &layerGitoauth{
		tokenByUser: map[int64]string{7: "gh_pat_task"},
	})

	result, err := svc.ResolveLayerOauthTokens("access-valid", validScope(), nil, []string{"acme/demo"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !result.OK {
		t.Fatalf("expected OK, got %+v", result)
	}
	if got := result.GithubAuthByRepo["acme/demo"]; got != "gh_pat_task" {
		t.Fatalf("matched task identity should win, got %q want %q", got, "gh_pat_task")
	}
}

func TestResolveLayerOauthTokens_PathAIPGitLabYAMLMissStillFetches(t *testing.T) {
	repoURL := "http://115.29.110.74/example-user/somanyad.git"
	git := &layerGitoauth{
		tokenByUser: map[int64]string{42: "glpat-path-a"},
	}
	svc := newLayerSvc(validToken(), &layerBusinessRepo{
		repos: []domain.TaskRepoSnapshot{
			{ProjectID: "p1", ProjectName: "p", RepoURLs: []string{repoURL}},
		},
		identities: []domain.GitIdentitySnapshot{
			{RepoURL: repoURL, GitIdentityID: "gid1", UserID: 42},
		},
	}, git)

	result, err := svc.ResolveLayerOauthTokens("access-valid", validScope(), nil, []string{"example-user/somanyad"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !result.OK {
		t.Fatalf("expected OK for Path A IP GitLab with identity, got %+v", result)
	}
	if git.lastKey != "115.29.110.74" {
		t.Fatalf("site=%q want 115.29.110.74", git.lastKey)
	}
	if result.GithubAuthByRepo["example-user/somanyad"] != "glpat-path-a" {
		t.Fatalf("auth=%v", result.GithubAuthByRepo)
	}
}

const gitlabRamWorkHTTPS = "https://gitlab-tencent-sh-1.daydaymoney.com/example-user/ram-work.git"
const gitlabRamWorkSSH = "git@gitlab-tencent-sh-1.daydaymoney.com:example-user/ram-work.git"
const gitlabRamWorkMatchKey = "gitlab-tencent-sh-1.daydaymoney.com/example-user/ram-work"
const gitlabRamWorkSite = "gitlab-tencent-sh-1.daydaymoney.com"

func TestResolveLayerOauthTokens_GitLabMatchKeyIgnoresURLShape(t *testing.T) {
	tok := validToken()
	tok.CommentID = "cmt_882908904520970240"
	git := &layerGitoauth{
		tokenByUser: map[int64]string{42: "glpat-ram-work"},
	}
	svc := newLayerSvc(tok, &layerBusinessRepo{
		repos: []domain.TaskRepoSnapshot{
			{ProjectID: "p1", ProjectName: "ram-work", RepoURLs: []string{gitlabRamWorkHTTPS}},
		},
		identities: []domain.GitIdentitySnapshot{
			{
				RepoURL:       gitlabRamWorkSSH,
				GitIdentityID: "gid-gl",
				UserID:        42,
				OauthGitsite:  gitlabRamWorkSite,
				FromComment:   true,
			},
		},
		commentAuthorID: 42,
	}, git)

	result, err := svc.ResolveLayerOauthTokens(
		"access-valid",
		validScope(),
		[]string{gitlabRamWorkMatchKey},
		nil,
	)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !result.OK {
		t.Fatalf("HTTPS task URL + SSH identity must exchange, got %+v", result)
	}
	if result.GitAuthByRepoMatchKey[gitlabRamWorkMatchKey] != "glpat-ram-work" {
		t.Fatalf("match key auth=%v", result.GitAuthByRepoMatchKey)
	}
	if git.lastKey != gitlabRamWorkSite {
		t.Fatalf("site=%q want %s", git.lastKey, gitlabRamWorkSite)
	}
}

func TestResolveLayerOauthTokens_GitLabSiteLevelGrantUsesCommentAuthor(t *testing.T) {
	tok := validToken()
	tok.CommentID = "cmt_882908904520970240"
	git := &layerGitoauth{
		tokenByUser: map[int64]string{42: "glpat-site-grant"},
	}
	svc := newLayerSvc(tok, &layerBusinessRepo{
		repos: []domain.TaskRepoSnapshot{
			{ProjectID: "p1", ProjectName: "ram-work", RepoURLs: []string{gitlabRamWorkHTTPS}},
		},
		identities: []domain.GitIdentitySnapshot{
			{
				RepoURL:      "",
				OauthGitsite: gitlabRamWorkSite,
				FromComment:  true,
			},
		},
		commentAuthorID: 42,
		userIdentities: []domain.GitIdentitySnapshot{
			{GitIdentityID: "gid-author", UserID: 42, UserName: "ann"},
		},
	}, git)

	result, err := svc.ResolveLayerOauthTokens(
		"access-valid",
		validScope(),
		[]string{gitlabRamWorkMatchKey},
		nil,
	)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !result.OK {
		t.Fatalf("site-level comment L2 must exchange as comment author, got %+v", result)
	}
	if result.GitAuthByRepoMatchKey[gitlabRamWorkMatchKey] != "glpat-site-grant" {
		t.Fatalf("match key auth=%v", result.GitAuthByRepoMatchKey)
	}
	if git.lastKey != gitlabRamWorkSite {
		t.Fatalf("site=%q want %s", git.lastKey, gitlabRamWorkSite)
	}
}

// Production BINDING_MISSING: comment has git_identity_id but empty oauth_gitsite
// (fork/auto-run payload stripped L2). UserID + repo host must still exchange.
func TestResolveLayerOauthTokens_CommentGitIdentityWithoutOauthGitsiteUsesRepoHost(t *testing.T) {
	tok := validToken()
	tok.CommentID = "cmt_882943779273732096"
	git := &layerGitoauth{
		tokenByUser: map[int64]string{42: "glpat-implicit-site"},
	}
	svc := newLayerSvc(tok, &layerBusinessRepo{
		repos: []domain.TaskRepoSnapshot{
			{ProjectID: "p1", ProjectName: "ram-work", RepoURLs: []string{gitlabRamWorkHTTPS}},
		},
		identities: []domain.GitIdentitySnapshot{
			{
				RepoURL:       gitlabRamWorkHTTPS,
				GitIdentityID: "gid-gl",
				UserID:        42,
				FromComment:   true,
			},
		},
		commentAuthorID: 42,
	}, git)

	result, err := svc.ResolveLayerOauthTokens(
		"access-valid",
		validScope(),
		[]string{gitlabRamWorkMatchKey},
		nil,
	)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !result.OK {
		t.Fatalf("comment identity without oauth_gitsite must still exchange, got %+v", result)
	}
	if result.GitAuthByRepoMatchKey[gitlabRamWorkMatchKey] != "glpat-implicit-site" {
		t.Fatalf("match key auth=%v", result.GitAuthByRepoMatchKey)
	}
	if git.lastKey != gitlabRamWorkSite {
		t.Fatalf("site=%q want %s", git.lastKey, gitlabRamWorkSite)
	}
}
