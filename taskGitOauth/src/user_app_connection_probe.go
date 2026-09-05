package main

import (
	"strings"
	"time"

	"taskGitOauth/domain"
	"taskGitOauth/infrastructure"
)

// probeIssueTokenTimeout bounds a single AccessToken probe. GitHub refresh can
// otherwise burn ~15s-100s dialing unreachable endpoints while the frontend has
// already given up at 6s (OPT-20260902-010). Overridable in tests.
var probeIssueTokenTimeout = 6 * time.Second

func queryBoolFlag(raw string) bool {
	v := strings.ToLower(strings.TrimSpace(raw))
	return v == "1" || v == "true" || v == "yes"
}

func pickActiveCredential(rows []infrastructure.CredentialRow) *infrastructure.CredentialRow {
	for i := range rows {
		row := &rows[i]
		if strings.TrimSpace(row.RefreshTokenCipher) != "" && row.BindStatus == "active" {
			return row
		}
	}
	return nil
}

// probeUserAppAccessToken checks whether a DB-active credential can still
// produce a usable access token (cache hit, stored GitHub PAT, or refresh).
// Never returns the token to callers; only a boolean. Refresh errors stay
// non-fatal for the GET connection API.
func (a *App) probeUserAppAccessToken(traceID, userID string, row *infrastructure.CredentialRow) (valid bool, cacheHit bool) {
	if a == nil || row == nil {
		return false, false
	}
	providerKey := credentialProviderKey(row)
	cachedBefore := ""
	if a.Cache != nil {
		// OPT-20260822-036: 与 issueAccessTokenFromCredential 同一 cache 键（含 remote_user_id）。
		cachedBefore = a.Cache.Get(cacheKeyID(providerKey, row.RemoteUserID), userID)
	}
	type issueOutcome struct {
		token string
		err   error
	}
	outcomeCh := make(chan issueOutcome, 1)
	go func() {
		token, err := a.issueAccessTokenFromCredential(userID, row)
		outcomeCh <- issueOutcome{token: token, err: err}
	}()
	var outcome issueOutcome
	select {
	case outcome = <-outcomeCh:
		// refresh 返回后继续正常判定
	case <-time.After(probeIssueTokenTimeout):
		// OPT-20260902-010：GitHub issue token 探测最坏会 dial 数十秒；探测失败快路径，
		// 避免前端 6s abort 后上游 handler 仍挂在 refresh 上（issue_failed 也照此快速返回）。
		logWarn("event=git_oauth_access_token_probe valid=false reason=issue_timeout provider=%s trace_id=%s",
			providerKey, traceID)
		return false, false
	}
	if outcome.err != nil || outcome.token == "" {
		logWarn("event=git_oauth_access_token_probe valid=false reason=issue_failed provider=%s trace_id=%s",
			providerKey, traceID)
		return false, false
	}
	hit := cachedBefore != ""
	logInfo("event=git_oauth_access_token_probe valid=true cache_hit=%t provider=%s trace_id=%s",
		hit, providerKey, traceID)
	return true, hit
}

// gitlabOAuthProbeNetwork classifies control-plane reachability for a GitLab repo.
// GitHub omits network_status. Intranet skips the live GET. Public GitLab is probed
// even when an AccessToken is already cached, so a down server is not shown as bound.
func (a *App) gitlabOAuthProbeNetwork(repoURL string) (networkStatus string, skipTokenProbe bool) {
	if a == nil {
		return "", false
	}
	host := hostnameOfRaw(repoURL)
	isGitLab := false
	intranet := false
	origin := domain.RepoHTTPOrigin(repoURL)

	if pc := a.providerConfigForRepoURL(repoURL); pc != nil && strings.EqualFold(strings.TrimSpace(pc.Provider), "gitlab") {
		isGitLab = true
		if o := domain.RepoHTTPOrigin(pc.Website); o != "" {
			origin = o
		}
	}
	if a.DB != nil && host != "" {
		row, err := a.DB.GetTenantGitLabConnectionByHost(host)
		if err != nil {
			logWarn("event=git_oauth_access_token_probe stage=tenant_host_lookup host=%s err=%v", host, err)
		} else if row != nil {
			isGitLab = true
			intranet = row.Intranet
			if o := domain.RepoHTTPOrigin(row.BaseURL); o != "" {
				origin = o
			}
		}
	}
	if !isGitLab {
		return "", false
	}
	if intranet {
		return domain.OAuthProbeNetworkStatus(true, true, false), true
	}
	if origin == "" {
		return domain.OAuthProbeNetworkStatus(true, false, true), true
	}
	result := a.probeStoredGitLab(origin)
	unreachable := result.status == domain.ReachabilityUnreachable
	return domain.OAuthProbeNetworkStatus(true, false, unreachable), unreachable
}
