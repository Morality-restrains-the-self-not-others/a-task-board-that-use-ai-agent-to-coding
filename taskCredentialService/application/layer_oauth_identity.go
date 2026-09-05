package application

import (
	"log"
	"strings"

	"taskCredentialService/domain"
)

// lookupIdentityForRepo finds a git identity for a clone/push URL.
// Order: exact URL, repo match key (HTTPS vs SSH vs .git), then site-level
// comment L2 (empty repo_url + oauth_gitsite). UserID falls back to the
// comment author when the grant row has no git_identity_id.
func lookupIdentityForRepo(idents []domain.GitIdentitySnapshot, repoURL string, authorID int64) (domain.GitIdentitySnapshot, bool) {
	repoURL = strings.TrimSpace(repoURL)
	if repoURL == "" {
		return domain.GitIdentitySnapshot{}, false
	}
	site := cloneProviderSite(repoURL)
	matchKey := RepoMatchKeyFromURL(repoURL)

	var byURL, byKey, bySite *domain.GitIdentitySnapshot
	for i := range idents {
		id := &idents[i]
		identURL := strings.TrimSpace(id.RepoURL)
		if identURL != "" && identURL == repoURL {
			byURL = id
		}
		if identURL != "" && matchKey != "" && RepoMatchKeyFromURL(identURL) == matchKey {
			byKey = id
		}
		identSite := strings.TrimSpace(id.OauthGitsite)
		if identSite == "" && identURL != "" {
			identSite = cloneProviderSite(identURL)
		}
		if site != "" && identSite != "" && strings.EqualFold(identSite, site) {
			if identURL == "" || bySite == nil {
				bySite = id
			}
		}
	}

	base := byURL
	via := "exact_url"
	if base == nil && byKey != nil {
		base = byKey
		via = "match_key"
	}
	if base == nil && bySite != nil {
		base = bySite
		via = "gitsite"
	}
	if base == nil {
		return domain.GitIdentitySnapshot{}, false
	}

	out := *base
	if strings.TrimSpace(out.OauthGitsite) == "" && bySite != nil {
		out.OauthGitsite = strings.TrimSpace(bySite.OauthGitsite)
		out.OauthRemoteUserID = strings.TrimSpace(bySite.OauthRemoteUserID)
		out.OauthGrantedAt = strings.TrimSpace(bySite.OauthGrantedAt)
		if via == "exact_url" || via == "match_key" {
			via = via + "+gitsite"
		}
	}
	if out.UserID <= 0 && authorID > 0 {
		out.UserID = authorID
	}
	if out.UserID <= 0 {
		return domain.GitIdentitySnapshot{}, false
	}
	if strings.TrimSpace(out.RepoURL) == "" {
		out.RepoURL = repoURL
	}
	if strings.TrimSpace(out.OauthGitsite) == "" && site != "" {
		out.OauthGitsite = site
		via = via + "+implicit_site"
	}
	log.Printf("[task-credential-service] layer-oauth identity resolved repo=%s via=%s user=%d site=%s",
		repoURL, via, out.UserID, site)
	return out, true
}
