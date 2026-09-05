package application

import (
	"log"
	"strings"

	"taskCredentialService/domain"
	"taskCredentialService/ports"
)

// NestedRepoItem is one discovered nested git repository.
type NestedRepoItem struct {
	Path   string
	URL    string
	Source string
}

// NestedGitReposFetcher discovers nested repos under a parent URL for a user.
// tenantID (company_id) is forwarded to the internal discovery API so that
// Path A GitLab / region-hosted repos resolve the right OAuth provider.
type NestedGitReposFetcher interface {
	FetchNestedGitRepos(userID int64, tenantID string, parentRepoURL string) ([]NestedRepoItem, error)
}

// ResolveNestedDiscoveryUserID picks the git user for nested discovery.
// Comment author always wins. Task owner_id is never used.
func ResolveNestedDiscoveryUserID(commentAuthorUserID, identityUserID int64) int64 {
	if commentAuthorUserID > 0 {
		return commentAuthorUserID
	}
	return identityUserID
}

// SelectIdentitiesForCommentAuthor keeps git identities that belong to the
// comment author. Task bindings for other users are dropped (no owner fallback).
func SelectIdentitiesForCommentAuthor(
	taskIdents []domain.GitIdentitySnapshot,
	authorID int64,
	authorIdents []domain.GitIdentitySnapshot,
) []domain.GitIdentitySnapshot {
	if authorID <= 0 {
		return taskIdents
	}
	matched := make([]domain.GitIdentitySnapshot, 0, len(taskIdents))
	for _, id := range taskIdents {
		if id.UserID == authorID {
			matched = append(matched, id)
		}
	}
	if len(matched) > 0 {
		return matched
	}
	if len(authorIdents) > 0 {
		return authorIdents
	}
	return nil
}

// EnrichReposForCommentAuthor discovers nested repos as the comment author and
// attaches that user's git identities to every clone URL.
func EnrichReposForCommentAuthor(
	commentAuthorUserID int64,
	authorGitIdentities []domain.GitIdentitySnapshot,
	taskIdentities []domain.GitIdentitySnapshot,
	repos []domain.TaskRepoSnapshot,
	tenantID string,
	fetcher NestedGitReposFetcher,
) (outRepos []domain.TaskRepoSnapshot, outIdents []domain.GitIdentitySnapshot, discoveryUserID int64) {
	discoveryUserID = ResolveNestedDiscoveryUserID(commentAuthorUserID, FirstPositiveUserID(taskIdentities))
	outRepos = MergeNestedReposIntoSnapshots(repos, discoveryUserID, tenantID, fetcher)
	idents := taskIdentities
	if !anyCommentSelectedIdentity(taskIdentities) {
		idents = SelectIdentitiesForCommentAuthor(taskIdentities, commentAuthorUserID, authorGitIdentities)
	}
	outIdents = InheritIdentitiesForRepos(idents, collectRepoURLs(outRepos))
	return outRepos, outIdents, discoveryUserID
}

func anyCommentSelectedIdentity(idents []domain.GitIdentitySnapshot) bool {
	for _, id := range idents {
		if !id.FromComment {
			continue
		}
		if strings.TrimSpace(id.GitIdentityID) != "" || strings.TrimSpace(id.OauthGitsite) != "" {
			return true
		}
	}
	return false
}

func lookupCommentAuthorUserID(repo ports.BusinessDataRepository, commentID string) int64 {
	if repo == nil || strings.TrimSpace(commentID) == "" {
		return 0
	}
	uid, err := repo.FetchCommentCreatedByUserID(commentID)
	if err != nil {
		log.Printf("[task-credential-service] comment author lookup failed comment=%s err=%v", commentID, err)
		return 0
	}
	if uid <= 0 {
		log.Printf("[task-credential-service] comment author missing comment=%s", commentID)
	}
	return uid
}

func lookupUserGitIdentities(repo ports.BusinessDataRepository, userID int64) []domain.GitIdentitySnapshot {
	if repo == nil || userID <= 0 {
		return nil
	}
	idents, err := repo.FetchUserGitIdentities(userID)
	if err != nil {
		log.Printf("[task-credential-service] user git identities lookup failed user=%d err=%v", userID, err)
		return nil
	}
	return idents
}

// MergeNestedReposIntoSnapshots appends discovered nested repos into each project's
// git_repos / git_repo_entries (clone_alias = path). Existing URLs are not duplicated.
// Discovery failures are ignored (nested empty) so parent clones still proceed.
// Projects with AutoCloneNestedRepos=false are skipped (project-level opt-out).
// userID may be 0: public GitHub nested discovery is anonymous; do not skip enrich
// just because the task has no git identity binding.
func MergeNestedReposIntoSnapshots(
	repos []domain.TaskRepoSnapshot,
	userID int64,
	tenantID string,
	fetcher NestedGitReposFetcher,
) []domain.TaskRepoSnapshot {
	if fetcher == nil || len(repos) == 0 {
		return repos
	}
	out := make([]domain.TaskRepoSnapshot, len(repos))
	copy(out, repos)
	cache := map[string][]NestedRepoItem{}

	for i := range out {
		if !out[i].AutoCloneNestedRepos {
			log.Printf("[task-credential-service] skip nested enrich project=%s auto_clone_nested_repos=false", out[i].ProjectID)
			continue
		}
		parents := append([]string{}, out[i].RepoURLs...)
		if len(parents) == 0 {
			for _, e := range out[i].RepoEntries {
				if u := strings.TrimSpace(e.URL); u != "" {
					parents = append(parents, u)
				}
			}
		}
		seen := map[string]bool{}
		for _, u := range out[i].RepoURLs {
			seen[strings.TrimSpace(u)] = true
		}
		for _, e := range out[i].RepoEntries {
			seen[strings.TrimSpace(e.URL)] = true
		}
		for _, parent := range parents {
			parent = strings.TrimSpace(parent)
			if parent == "" {
				continue
			}
			nested, ok := cache[parent]
			if !ok {
				var err error
				nested, err = fetcher.FetchNestedGitRepos(userID, tenantID, parent)
				if err != nil {
					log.Printf("[task-credential-service] nested-git-repos fetch failed parent=%s user=%d tenant=%s err=%v", parent, userID, tenantID, err)
					nested = nil
				} else {
					log.Printf("[task-credential-service] nested-git-repos fetched parent=%s user=%d tenant=%s nested=%d", parent, userID, tenantID, len(nested))
				}
				cache[parent] = nested
			}
			for _, n := range nested {
				url := strings.TrimSpace(n.URL)
				if url == "" || seen[url] {
					continue
				}
				seen[url] = true
				alias := strings.TrimSpace(n.Path)
				out[i].RepoURLs = append(out[i].RepoURLs, url)
				out[i].RepoEntries = append(out[i].RepoEntries, domain.RepoCloneEntry{
					URL:           url,
					CloneAlias:    alias,
					ParentRepoURL: parent,
				})
			}
		}
	}
	return out
}

// InheritIdentitiesForRepos ensures every repo URL has a GitIdentitySnapshot by
// copying the first usable identity (UserID>0) onto URLs that lack a binding.
func InheritIdentitiesForRepos(idents []domain.GitIdentitySnapshot, repoURLs []string) []domain.GitIdentitySnapshot {
	byURL := map[string]domain.GitIdentitySnapshot{}
	var fallback *domain.GitIdentitySnapshot
	for i := range idents {
		u := strings.TrimSpace(idents[i].RepoURL)
		if u != "" {
			byURL[u] = idents[i]
		}
		if fallback == nil && idents[i].UserID > 0 {
			cp := idents[i]
			fallback = &cp
		}
	}
	if fallback == nil {
		return idents
	}
	out := append([]domain.GitIdentitySnapshot{}, idents...)
	for _, url := range repoURLs {
		url = strings.TrimSpace(url)
		if url == "" {
			continue
		}
		if _, ok := byURL[url]; ok {
			continue
		}
		cp := *fallback
		cp.RepoURL = url
		out = append(out, cp)
		byURL[url] = cp
	}
	return out
}

// FirstPositiveUserID returns the first identity user id > 0.
func FirstPositiveUserID(idents []domain.GitIdentitySnapshot) int64 {
	for _, id := range idents {
		if id.UserID > 0 {
			return id.UserID
		}
	}
	return 0
}
