package application

import (
	"sync/atomic"
	"testing"

	"taskCredentialService/domain"
	"taskCredentialService/ports"
)

type countingGitoauth struct {
	fetchCount   atomic.Int64
	token        string
	lastKey      string
	emptyResolve bool
}

func (c *countingGitoauth) FetchAccessToken(userID int64, providerKey string) (string, error) {
	c.fetchCount.Add(1)
	c.lastKey = providerKey
	return c.token, nil
}

func (c *countingGitoauth) ResolveProvider(repoURL string) string {
	if c.emptyResolve {
		return ""
	}
	return "gitlab:gitlab-local"
}

func (c *countingGitoauth) ResolveHttpsCloneURL(repoURL string) string {
	return "http://127.0.0.1:8012/demo/repo"
}

var _ ports.GitoauthClient = (*countingGitoauth)(nil)

func TestBuildCredentialsReusesAccessTokenPerUserProvider(t *testing.T) {
	client := &countingGitoauth{token: "tok-shared"}
	repoA := "git@127.0.0.1:ljy/repo-a.git"
	repoB := "git@127.0.0.1:ljy/repo-b.git"
	identities := []domain.GitIdentitySnapshot{
		{RepoURL: repoA, UserID: 42, GitIdentityID: "g1"},
		{RepoURL: repoB, UserID: 42, GitIdentityID: "g2"},
	}

	got := buildCredentials([]string{repoA, repoB}, identities, client)
	if got == nil {
		t.Fatal("expected credentials result")
	}
	if len(got.Credentials) != 2 {
		t.Fatalf("credentials=%d want 2 missing=%v failures=%v",
			len(got.Credentials), got.MissingIdentityRepos, got.TokenRefreshFailures)
	}
	if client.fetchCount.Load() != 1 {
		t.Fatalf("FetchAccessToken calls=%d want 1 (same user+provider must reuse token)", client.fetchCount.Load())
	}
	a := got.Credentials[repoA].EphemeralOAuthAccessToken
	b := got.Credentials[repoB].EphemeralOAuthAccessToken
	if a != "tok-shared" || b != "tok-shared" {
		t.Fatalf("tokens not shared: a=%q b=%q", a, b)
	}
	if got.Credentials[repoA].GitHTTPUsername != "oauth2" {
		t.Fatalf("git_http_username=%q want oauth2", got.Credentials[repoA].GitHTTPUsername)
	}
}

func TestBuildCredentialsFetchesOncePerDistinctUserProvider(t *testing.T) {
	client := &countingGitoauth{token: "tok-x"}
	repoA := "git@127.0.0.1:u1/a.git"
	repoB := "git@127.0.0.1:u2/b.git"
	identities := []domain.GitIdentitySnapshot{
		{RepoURL: repoA, UserID: 1, GitIdentityID: "g1"},
		{RepoURL: repoB, UserID: 2, GitIdentityID: "g2"},
	}

	got := buildCredentials([]string{repoA, repoB}, identities, client)
	if len(got.Credentials) != 2 {
		t.Fatalf("credentials=%d want 2", len(got.Credentials))
	}
	if client.fetchCount.Load() != 2 {
		t.Fatalf("FetchAccessToken calls=%d want 2 for distinct users", client.fetchCount.Load())
	}
}

// YAML website 不含租户 Path A IP 时，不得把已有 identity 的仓标成 missing。
// 须用 gitsite:{host} 向 gitOauth 换票（与 fork「已绑定 Git OAuth」同一套 Path A 解析）。
func TestBuildCredentialsPathAIPGitLabYAMLMissStillFetches(t *testing.T) {
	client := &countingGitoauth{token: "tok-path-a", emptyResolve: true}
	repo := "http://115.29.110.74/example-user/somanyad.git"
	identities := []domain.GitIdentitySnapshot{
		{RepoURL: repo, UserID: 42, GitIdentityID: "g1"},
	}

	got := buildCredentials([]string{repo}, identities, client)
	if len(got.MissingIdentityRepos) != 0 {
		t.Fatalf("missing=%v want none (identity exists; YAML miss is not unbound)", got.MissingIdentityRepos)
	}
	if client.fetchCount.Load() != 1 {
		t.Fatalf("FetchAccessToken calls=%d want 1", client.fetchCount.Load())
	}
	if client.lastKey != "115.29.110.74" {
		t.Fatalf("site=%q want 115.29.110.74", client.lastKey)
	}
	cred, ok := got.Credentials[repo]
	if !ok {
		t.Fatalf("credentials missing for %s failures=%v", repo, got.TokenRefreshFailures)
	}
	if cred.EphemeralOAuthAccessToken != "tok-path-a" {
		t.Fatalf("token=%q", cred.EphemeralOAuthAccessToken)
	}
	if cred.Provider != "gitlab" {
		t.Fatalf("provider=%q want gitlab", cred.Provider)
	}
	if cred.GitHTTPUsername != "oauth2" {
		t.Fatalf("git_http_username=%q want oauth2", cred.GitHTTPUsername)
	}
}
