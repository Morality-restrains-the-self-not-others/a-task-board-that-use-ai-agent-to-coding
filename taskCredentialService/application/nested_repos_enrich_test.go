package application_test

import (
	"errors"
	"testing"

	"taskCredentialService/application"
	"taskCredentialService/domain"
)

type stubNestedFetcher struct {
	byParent    map[string][]application.NestedRepoItem
	err         error
	calls       int
	lastUserID  int64
	lastTenant  string
}

func (s *stubNestedFetcher) FetchNestedGitRepos(userID int64, tenantID, parentRepoURL string) ([]application.NestedRepoItem, error) {
	s.calls++
	s.lastUserID = userID
	s.lastTenant = tenantID
	if s.err != nil {
		return nil, s.err
	}
	return s.byParent[parentRepoURL], nil
}

func TestMergeNestedReposIntoSnapshots(t *testing.T) {
	fetcher := &stubNestedFetcher{
		byParent: map[string][]application.NestedRepoItem{
			"https://gitlab.example/g/ram-work.git": {
				{Path: "task2app", URL: "https://gitlab.example/g/task2app.git", Source: "gitmodules"},
				{Path: "docs", URL: "https://gitlab.example/g/docs.git", Source: "gitmodules"},
			},
		},
	}
	repos := []domain.TaskRepoSnapshot{{
		ProjectID:            "p1",
		RepoURLs:             []string{"https://gitlab.example/g/ram-work.git"},
		RepoEntries:          []domain.RepoCloneEntry{{URL: "https://gitlab.example/g/ram-work.git"}},
		AutoCloneNestedRepos: true,
	}}
	got := application.MergeNestedReposIntoSnapshots(repos, 42, "", fetcher)
	if len(got[0].RepoURLs) != 3 {
		t.Fatalf("urls=%v", got[0].RepoURLs)
	}
	aliases := map[string]string{}
	parents := map[string]string{}
	for _, e := range got[0].RepoEntries {
		aliases[e.URL] = e.CloneAlias
		parents[e.URL] = e.ParentRepoURL
	}
	if aliases["https://gitlab.example/g/task2app.git"] != "task2app" {
		t.Fatalf("alias=%v", aliases)
	}
	if parents["https://gitlab.example/g/task2app.git"] != "https://gitlab.example/g/ram-work.git" {
		t.Fatalf("parent_repo_url=%v", parents)
	}
	if parents["https://gitlab.example/g/docs.git"] != "https://gitlab.example/g/ram-work.git" {
		t.Fatalf("docs parent=%v", parents)
	}
	if parents["https://gitlab.example/g/ram-work.git"] != "" {
		t.Fatalf("parent entry should not have parent_repo_url: %q", parents["https://gitlab.example/g/ram-work.git"])
	}
	if fetcher.calls != 1 {
		t.Fatalf("calls=%d", fetcher.calls)
	}
}

func TestMergeNestedReposIntoSnapshotsIgnoresFetchError(t *testing.T) {
	fetcher := &stubNestedFetcher{err: errors.New("oauth down")}
	repos := []domain.TaskRepoSnapshot{{
		RepoURLs:             []string{"https://gitlab.example/g/ram-work.git"},
		AutoCloneNestedRepos: true,
	}}
	got := application.MergeNestedReposIntoSnapshots(repos, 42, "", fetcher)
	if len(got[0].RepoURLs) != 1 {
		t.Fatalf("should keep parent only: %v", got[0].RepoURLs)
	}
}

func TestMergeNestedReposIntoSnapshotsFetchesWhenUserIDZero(t *testing.T) {
	parent := "https://github.com/task2money/ram-work.git"
	child := "https://github.com/task2money/AiMonitor.git"
	fetcher := &stubNestedFetcher{
		byParent: map[string][]application.NestedRepoItem{
			parent: {{Path: "AiMonitor", URL: child, Source: "gitmodules"}},
		},
	}
	repos := []domain.TaskRepoSnapshot{{
		ProjectID:            "p1",
		RepoURLs:             []string{parent},
		RepoEntries:          []domain.RepoCloneEntry{{URL: parent}},
		AutoCloneNestedRepos: true,
	}}
	got := application.MergeNestedReposIntoSnapshots(repos, 0, "", fetcher)
	if fetcher.calls != 1 {
		t.Fatalf("empty git identities must still discover nested repos, calls=%d", fetcher.calls)
	}
	if len(got[0].RepoURLs) != 2 {
		t.Fatalf("urls=%v want parent+child", got[0].RepoURLs)
	}
	found := false
	for _, e := range got[0].RepoEntries {
		if e.URL == child && e.ParentRepoURL == parent && e.CloneAlias == "AiMonitor" {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing nested entry: %+v", got[0].RepoEntries)
	}
}

func TestResolveNestedDiscoveryUserIDPrefersCommentAuthor(t *testing.T) {
	if got := application.ResolveNestedDiscoveryUserID(99, 42); got != 99 {
		t.Fatalf("comment author must win, got %d", got)
	}
	if got := application.ResolveNestedDiscoveryUserID(0, 42); got != 42 {
		t.Fatalf("identity user only when comment author missing, got %d", got)
	}
	if got := application.ResolveNestedDiscoveryUserID(0, 0); got != 0 {
		t.Fatalf("both missing → 0 (anonymous public), got %d", got)
	}
}

func TestSelectIdentitiesForCommentAuthorDropsOtherUsers(t *testing.T) {
	taskIdents := []domain.GitIdentitySnapshot{
		{RepoURL: "https://git.example/owner.git", UserID: 1, GitIdentityID: "owner-id"},
	}
	authorIdents := []domain.GitIdentitySnapshot{
		{UserID: 99, GitIdentityID: "cmt-id", UserName: "Ann", UserEmail: "ann@example.com"},
	}
	got := application.SelectIdentitiesForCommentAuthor(taskIdents, 99, authorIdents)
	if len(got) != 1 || got[0].UserID != 99 {
		t.Fatalf("must use comment author identities, got %#v", got)
	}
	got = application.SelectIdentitiesForCommentAuthor(taskIdents, 99, nil)
	if got != nil {
		t.Fatalf("must not fall back to another user's task bindings, got %#v", got)
	}
}

func TestEnrichReposForCommentAuthorUsesCommentUserNotOwnerBinding(t *testing.T) {
	parent := "https://github.com/task2money/private-meta.git"
	child := "https://github.com/task2money/private-sub.git"
	fetcher := &stubNestedFetcher{
		byParent: map[string][]application.NestedRepoItem{
			parent: {{Path: "private-sub", URL: child, Source: "gitmodules"}},
		},
	}
	repos := []domain.TaskRepoSnapshot{{
		ProjectID:            "p1",
		RepoURLs:             []string{parent},
		RepoEntries:          []domain.RepoCloneEntry{{URL: parent}},
		AutoCloneNestedRepos: true,
	}}
	taskIdents := []domain.GitIdentitySnapshot{
		{RepoURL: parent, UserID: 1, GitIdentityID: "owner-id"},
	}
	authorIdents := []domain.GitIdentitySnapshot{
		{UserID: 99, GitIdentityID: "cmt-id", UserName: "Ann", UserEmail: "ann@example.com"},
	}
	outRepos, outIdents, discoveryUserID := application.EnrichReposForCommentAuthor(
		99, authorIdents, taskIdents, repos, "tenant-1", fetcher,
	)
	if discoveryUserID != 99 {
		t.Fatalf("discovery user=%d want comment author 99", discoveryUserID)
	}
	if fetcher.lastUserID != 99 {
		t.Fatalf("nested fetch user=%d want 99", fetcher.lastUserID)
	}
	if fetcher.lastTenant != "tenant-1" {
		t.Fatalf("nested fetch tenant=%q want tenant-1", fetcher.lastTenant)
	}
	if len(outRepos[0].RepoURLs) != 2 {
		t.Fatalf("urls=%v want parent+child", outRepos[0].RepoURLs)
	}
	for _, id := range outIdents {
		if id.UserID != 99 {
			t.Fatalf("clone identity leaked non-author user: %#v", id)
		}
	}
}

func TestMergeNestedReposIntoSnapshotsSkipsWhenDisabled(t *testing.T) {
	fetcher := &stubNestedFetcher{
		byParent: map[string][]application.NestedRepoItem{
			"https://gitlab.example/g/ram-work.git": {
				{Path: "task2app", URL: "https://gitlab.example/g/task2app.git", Source: "gitmodules"},
			},
		},
	}
	repos := []domain.TaskRepoSnapshot{{
		ProjectID:            "p1",
		RepoURLs:             []string{"https://gitlab.example/g/ram-work.git"},
		RepoEntries:          []domain.RepoCloneEntry{{URL: "https://gitlab.example/g/ram-work.git"}},
		AutoCloneNestedRepos: false,
	}}
	got := application.MergeNestedReposIntoSnapshots(repos, 42, "", fetcher)
	if len(got[0].RepoURLs) != 1 {
		t.Fatalf("disabled should keep parent only: %v", got[0].RepoURLs)
	}
	if fetcher.calls != 0 {
		t.Fatalf("fetcher should not be called when disabled, calls=%d", fetcher.calls)
	}
}

func TestInheritIdentitiesForRepos(t *testing.T) {
	idents := []domain.GitIdentitySnapshot{{
		RepoURL: "https://gitlab.example/g/ram-work.git", UserID: 9, GitIdentityID: "id1",
	}}
	urls := []string{
		"https://gitlab.example/g/ram-work.git",
		"https://gitlab.example/g/task2app.git",
	}
	got := application.InheritIdentitiesForRepos(idents, urls)
	if len(got) != 2 {
		t.Fatalf("len=%d", len(got))
	}
	found := false
	for _, g := range got {
		if g.RepoURL == "https://gitlab.example/g/task2app.git" && g.UserID == 9 {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing inherited identity: %#v", got)
	}
}

func TestFirstPositiveUserID(t *testing.T) {
	if application.FirstPositiveUserID(nil) != 0 {
		t.Fatal()
	}
	got := application.FirstPositiveUserID([]domain.GitIdentitySnapshot{
		{UserID: 0}, {UserID: 7},
	})
	if got != 7 {
		t.Fatalf("got %d", got)
	}
}

func TestEnrichReposKeepsCommentSelectedIdentity(t *testing.T) {
	parent := "https://git.example/a.git"
	repos := []domain.TaskRepoSnapshot{{
		ProjectID:            "p1",
		RepoURLs:             []string{parent},
		RepoEntries:          []domain.RepoCloneEntry{{URL: parent}},
		AutoCloneNestedRepos: false,
	}}
	commentIdents := []domain.GitIdentitySnapshot{{
		RepoURL:       parent,
		UserID:        99,
		GitIdentityID: "gid-comment",
		UserName:      "Ann",
		UserEmail:     "ann@example.com",
		FromComment:   true,
	}}
	authorDump := []domain.GitIdentitySnapshot{
		{UserID: 99, GitIdentityID: "gid-other", UserName: "Other", UserEmail: "other@example.com"},
		{UserID: 99, GitIdentityID: "gid-comment", UserName: "Ann", UserEmail: "ann@example.com"},
	}
	_, outIdents, _ := application.EnrichReposForCommentAuthor(
		99, authorDump, commentIdents, repos, "", nil,
	)
	if len(outIdents) != 1 || outIdents[0].GitIdentityID != "gid-comment" || outIdents[0].UserName != "Ann" {
		t.Fatalf("comment selection must win over author dump, got %#v", outIdents)
	}
}
