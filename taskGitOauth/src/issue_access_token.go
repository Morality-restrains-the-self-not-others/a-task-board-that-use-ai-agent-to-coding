package main

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"taskGitOauth/infrastructure"
)

var issueAccessLocks sync.Map

func lockIssueAccess(key string) func() {
	v, _ := issueAccessLocks.LoadOrStore(key, &sync.Mutex{})
	mu := v.(*sync.Mutex)
	mu.Lock()
	return mu.Unlock
}

func parseOAuthExpiresIn(raw any) int {
	if raw == nil {
		return 0
	}
	switch t := raw.(type) {
	case float64:
		return int(t)
	case string:
		n, _ := strconv.Atoi(t)
		return n
	case int:
		return t
	default:
		return 0
	}
}

func credentialProviderKey(row *infrastructure.CredentialRow) string {
	if row == nil {
		return ""
	}
	return strings.TrimSpace(strings.ToLower(row.Provider))
}

// issueAccessTokenFromCredential returns the user-level Git OAuth access token
// shared by comments, probe, clone, and merge. Cache hit must not refresh:
// GitLab refresh rotates and invalidates both access and refresh tokens.
func (a *App) issueAccessTokenFromCredential(userID string, row *infrastructure.CredentialRow) (string, error) {
	if a == nil || row == nil || strings.TrimSpace(row.RefreshTokenCipher) == "" {
		return "", errGitOAuthNotConnected
	}
	if a.Cache == nil {
		return "", fmt.Errorf("access token cache unavailable")
	}
	providerKey := credentialProviderKey(row)
	if providerKey == "" {
		return "", fmt.Errorf("unknown git provider")
	}
	uid := strings.TrimSpace(userID)
	// OPT-20260822-036: 缓存/并发锁键含 remote_user_id，同一 provider+user 下多远端账号互不共享。
	cacheID := cacheKeyID(providerKey, row.RemoteUserID)
	unlock := lockIssueAccess(cacheID + "|" + uid)
	defer unlock()
	if cached := a.Cache.Get(cacheID, uid); cached != "" {
		logInfo("event=git_oauth_access_token_issue cache_hit=true provider=%s", providerKey)
		return cached, nil
	}
	if a.DB == nil || a.DB.DB == nil {
		return "", fmt.Errorf("db unavailable")
	}

	tx, err := infrastructure.BeginImmediate(a.DB.DB)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback() }()

	locked, err := a.DB.GetCredentialByIDForUpdate(tx, row.ID)
	if err != nil || locked == nil || strings.TrimSpace(locked.RefreshTokenCipher) == "" {
		return "", errGitOAuthNotConnected
	}
	if cached := a.Cache.Get(cacheID, uid); cached != "" {
		return cached, nil
	}

	plain, _ := a.Fernet.Decrypt(locked.RefreshTokenCipher)
	if plain == "" {
		logWarn("event=git_oauth_access_token_issue reason=decrypt_failed provider=%s", providerKey)
		return "", fmt.Errorf("decrypt_failed")
	}

	access := ""
	expiresIn := 0
	if isDirectGitHubAccessToken(plain) {
		access = plain
		expiresIn = infrastructure.AccessCacheDefaultTTL
		logInfo("event=git_oauth_access_token_issue cache_hit=false kind=direct_github provider=%s", providerKey)
	} else {
		out, err := a.refreshAccessToken(providerKey, plain)
		if err != nil {
			a.Cache.Clear(cacheID, uid)
			logWarn("event=git_oauth_access_token_issue reason=refresh_failed provider=%s err=%v", providerKey, err)
			return "", err
		}
		access = strings.TrimSpace(fmt.Sprint(out["access_token"]))
		if access == "" || access == "<nil>" {
			return "", fmt.Errorf("no access_token")
		}
		if nr := strings.TrimSpace(fmt.Sprint(out["refresh_token"])); nr != "" && nr != "<nil>" {
			cipher, encErr := a.Fernet.Encrypt(nr)
			if encErr != nil || cipher == "" {
				return "", fmt.Errorf("encrypt rotated refresh: %w", encErr)
			}
			if err := a.DB.UpdateRefreshCipher(tx, locked.ID, cipher); err != nil {
				return "", err
			}
		} else if strings.HasPrefix(providerKey, "gitlab:") {
			return "", fmt.Errorf("gitlab refresh missing refresh_token")
		}
		expiresIn = parseOAuthExpiresIn(out["expires_in"])
		logInfo("event=git_oauth_access_token_issue cache_hit=false kind=refresh provider=%s", providerKey)
	}
	a.Cache.Put(cacheID, uid, access, expiresIn)
	if err := tx.Commit(); err != nil {
		return "", err
	}
	return access, nil
}
