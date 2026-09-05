package main

import (
	"strings"
)

const (
	accessStatusAccessible    = "accessible"
	accessStatusNeedsAuth     = "needs_auth"
	accessStatusNotAccessible = "not_accessible"
	accessStatusNotConfigured = "not_configured"
	accessStatusUncheckable   = "uncheckable"
)

var repoAccessAuthErrorHints = []string{
	"401", "403", "未绑定", "未检测到可用授权", "access_token", "凭据", "授权",
	"oauth", "authentication", "unauthorized", "denied", "拒绝访问",
}

const gitlabNeedsAuthMessageCN = "此仓库需要 GitLab 授权后才能访问，请点击下方「OAuth 授权」按钮完成授权"

type projectRepoAccessResponse struct {
	IsAccessible         bool
	AccessStatus         string
	Message              string
	TokenStatus          string
	OAuthProvider        string
	OAuthServiceProvider string
}

func (r projectRepoAccessResponse) toMap() map[string]interface{} {
	out := map[string]interface{}{
		"is_accessible": r.IsAccessible,
		"access_status": r.AccessStatus,
		"message":       r.Message,
		"token_status":  r.TokenStatus,
	}
	if r.OAuthProvider != "" {
		out["oauth_provider"] = r.OAuthProvider
	}
	if r.OAuthServiceProvider != "" {
		out["oauth_service_provider"] = r.OAuthServiceProvider
	}
	return out
}

func looksLikeRepoAuthError(message string) bool {
	lower := strings.ToLower(message)
	for _, hint := range repoAccessAuthErrorHints {
		if strings.Contains(lower, strings.ToLower(hint)) {
			return true
		}
	}
	return false
}

func classifyRepoAccessAfterProbe(
	isAccessible bool,
	errorMessage string,
	oauthRequired bool,
	hasOAuthToken bool,
	tokenResolveError string,
	tokenStatus string,
	oauthProvider string,
	oauthServiceProvider string,
) projectRepoAccessResponse {
	if isAccessible {
		return projectRepoAccessResponse{
			IsAccessible:         true,
			AccessStatus:         accessStatusAccessible,
			Message:              "",
			TokenStatus:          tokenStatus,
			OAuthProvider:        oauthProvider,
			OAuthServiceProvider: oauthServiceProvider,
		}
	}

	err := strings.TrimSpace(errorMessage)
	resolveErr := strings.TrimSpace(tokenResolveError)

	if oauthRequired && (resolveErr != "" || !hasOAuthToken) {
		if looksLikeRepoAuthError(err) || !hasOAuthToken {
			msg := resolveErr
			if msg == "" {
				msg = err
			}
			if msg == "" {
				msg = "需要完成 Git 托管站 OAuth 授权后才能访问该仓库"
			}
			return projectRepoAccessResponse{
				IsAccessible:         false,
				AccessStatus:         accessStatusNeedsAuth,
				Message:              msg,
				TokenStatus:          tokenStatus,
				OAuthProvider:        oauthProvider,
				OAuthServiceProvider: oauthServiceProvider,
			}
		}
	}

	if looksLikeRepoAuthError(err) && !hasOAuthToken {
		msg := err
		if msg == "" {
			msg = "需要完成 Git 托管站 OAuth 授权后才能访问该仓库"
		}
		return projectRepoAccessResponse{
			IsAccessible:         false,
			AccessStatus:         accessStatusNeedsAuth,
			Message:              msg,
			TokenStatus:          tokenStatus,
			OAuthProvider:        oauthProvider,
			OAuthServiceProvider: oauthServiceProvider,
		}
	}

	msg := err
	if msg == "" {
		msg = "仓库不可访问"
	}
	return projectRepoAccessResponse{
		IsAccessible:         false,
		AccessStatus:         accessStatusNotAccessible,
		Message:              msg,
		TokenStatus:          tokenStatus,
		OAuthProvider:        oauthProvider,
		OAuthServiceProvider: oauthServiceProvider,
	}
}

// checkProjectPrimaryRepoAccess mirrors Django ProjectRepoAccessCheckService using Go-native probe.
func checkProjectPrimaryRepoAccess(userID string, repoURLs []string, tenantID, grantProjectID string, traceHeaders ...map[string]string) projectRepoAccessResponse {
	trace := outboundTrace(traceHeaders...)
	urls := make([]string, 0, len(repoURLs))
	for _, u := range repoURLs {
		s := strings.TrimSpace(u)
		if s != "" {
			urls = append(urls, s)
		}
	}
	if len(urls) == 0 {
		return projectRepoAccessResponse{
			IsAccessible: false,
			AccessStatus: accessStatusNotConfigured,
			Message:      "未设置Git仓库",
			TokenStatus:  tokenStatusNotApplicable,
		}
	}
	primary := urls[0]

	lookup, ok := normalizeGitRepoURLForBranchLookup(primary)
	if !ok || (isSSHGitRemote(primary) && isSSHGitRemote(lookup)) {
		return projectRepoAccessResponse{
			IsAccessible: false,
			AccessStatus: accessStatusNotAccessible,
			Message:      "无效的 Git 仓库 SSH 地址格式",
			TokenStatus:  tokenStatusNotApplicable,
		}
	}

	// Token-only first: GitLab without token short-circuits (Django parity; skip remote probe).
	tokenOnly := validateGitRepoForUser(userID, primary, false, tenantID, grantProjectID, trace)
	hasToken := tokenOnly.TokenStatus == tokenStatusAvailable
	if tokenOnly.OAuthProvider == "gitlab" && !hasToken {
		msg := gitlabNeedsAuthMessageCN
		if tokenOnly.TokenStatus == tokenStatusTokenError {
			// Prefer resolve error when present; validate path often leaves Message empty on token-only.
			if m := strings.TrimSpace(tokenOnly.Message); m != "" {
				msg = m
			}
		}
		return classifyRepoAccessAfterProbe(
			false,
			msg,
			true,
			false,
			"",
			tokenOnly.TokenStatus,
			tokenOnly.OAuthProvider,
			tokenOnly.OAuthServiceProvider,
		)
	}

	probed := validateGitRepoForUser(userID, primary, true, tenantID, grantProjectID, trace)
	oauthRequired := probed.OAuthProvider == "github" || probed.OAuthProvider == "gitlab"
	hasOAuthToken := probed.TokenStatus == tokenStatusAvailable
	tokenResolveError := ""
	if probed.TokenStatus == tokenStatusTokenError {
		tokenResolveError = strings.TrimSpace(probed.Message)
	}
	return classifyRepoAccessAfterProbe(
		probed.IsAccessible,
		probed.Message,
		oauthRequired,
		hasOAuthToken,
		tokenResolveError,
		probed.TokenStatus,
		probed.OAuthProvider,
		probed.OAuthServiceProvider,
	)
}
