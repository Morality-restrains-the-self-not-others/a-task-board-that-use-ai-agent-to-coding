package application_test

import (
	"testing"

	"taskCredentialService/application"
	"taskCredentialService/domain"
	"taskCredentialService/ports"
)

type stubBusinessRepo struct {
	snap            *domain.TaskSnapshot
	repos           []domain.TaskRepoSnapshot
	identities      []domain.GitIdentitySnapshot
	commentAuthorID int64
	userIdentities  []domain.GitIdentitySnapshot
	err             error
	lastIdentTask   string
	lastIdentCmt    string
}

func (s *stubBusinessRepo) FetchTaskSnapshot(taskID, commentID string) (*domain.TaskSnapshot, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.snap, nil
}

func (s *stubBusinessRepo) FetchTaskRepos(taskID, commentID string) ([]domain.TaskRepoSnapshot, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.repos, nil
}

func (s *stubBusinessRepo) FetchRepoIdentities(taskID, commentID string) ([]domain.GitIdentitySnapshot, error) {
	s.lastIdentTask = taskID
	s.lastIdentCmt = commentID
	if s.err != nil {
		return nil, s.err
	}
	if s.identities == nil {
		return nil, nil
	}
	return s.identities, nil
}

func (s *stubBusinessRepo) FetchCommentCreatedByUserID(commentID string) (int64, error) {
	if s.err != nil {
		return 0, s.err
	}
	return s.commentAuthorID, nil
}

func (s *stubBusinessRepo) FetchUserGitIdentities(userID int64) ([]domain.GitIdentitySnapshot, error) {
	if s.err != nil {
		return nil, s.err
	}
	if userID <= 0 || s.userIdentities == nil {
		return nil, nil
	}
	return s.userIdentities, nil
}

func TestFetchTaskDetailIncludesProjectRepos(t *testing.T) {
	repo := &stubBusinessRepo{
		snap: &domain.TaskSnapshot{
			ID:           "task_1",
			Title:        "demo",
			Description:  "d",
			CompanyID:    "t1",
			WorkspaceID:  "w1",
			TargetBranch: "feature/demo",
			AutoRun:      true,
			BranchStrategy: &domain.BranchStrategySnapshot{
				WorkBranchName:        "feature/demo",
				MergeTargetBranchName: "develop",
			},
		},
		repos: []domain.TaskRepoSnapshot{
			{
				ProjectID:   "proj_1",
				ProjectName: "demo-proj",
				RepoURLs:    []string{"git@host:group/a.git", "git@host:group/b.git"},
				RepoBranches: []domain.RepoBranchEntry{
					{GitRepo: "git@host:group/a.git", BaseBranch: "main", TargetBranch: "feature/demo"},
					{GitRepo: "git@host:group/b.git", BaseBranch: "develop", TargetBranch: "feature/demo"},
				},
			},
		},
		identities: []domain.GitIdentitySnapshot{
			{
				RepoURL:       "git@host:group/a.git",
				GitIdentityID: "id_1",
				UserName:      "Alice",
				UserEmail:     "alice@example.com",
			},
			{
				RepoURL:       "git@host:group/b.git",
				GitIdentityID: "id_2",
				UserName:      "",
				UserEmail:     "",
			},
		},
	}
	svc := application.NewTaskDetailService(repo)
	detail, err := svc.FetchTaskDetail("task_1", "")
	if err != nil {
		t.Fatalf("FetchTaskDetail: %v", err)
	}
	if detail.Task == nil || detail.Task.ID != "task_1" {
		t.Fatalf("unexpected task: %+v", detail.Task)
	}
	if !detail.Task.AutoRun {
		t.Fatalf("auto_run=%v want true", detail.Task.AutoRun)
	}
	if detail.Task.TargetBranch != "feature/demo" {
		t.Fatalf("target_branch=%q want feature/demo", detail.Task.TargetBranch)
	}
	if detail.Task.BranchStrategy == nil || detail.Task.BranchStrategy.WorkBranchName != "feature/demo" {
		t.Fatalf("branch_strategy=%+v", detail.Task.BranchStrategy)
	}
	if len(detail.ProjectRepos) != 1 {
		t.Fatalf("project_repos len=%d want 1", len(detail.ProjectRepos))
	}
	if len(detail.ProjectRepos[0].RepoURLs) != 2 {
		t.Fatalf("git_repos len=%d want 2", len(detail.ProjectRepos[0].RepoURLs))
	}
	if len(detail.ProjectRepos[0].RepoBranches) != 2 {
		t.Fatalf("repo_branches len=%d want 2", len(detail.ProjectRepos[0].RepoBranches))
	}
	if detail.ProjectRepos[0].RepoBranches[0].BaseBranch != "main" {
		t.Fatalf("base_branch=%q want main", detail.ProjectRepos[0].RepoBranches[0].BaseBranch)
	}
	if len(detail.RepoGitIdentities) != 1 {
		t.Fatalf("repo_git_identities len=%d want 1 (incomplete skipped)", len(detail.RepoGitIdentities))
	}
	if detail.RepoGitIdentities[0].UserName != "Alice" || detail.RepoGitIdentities[0].UserEmail != "alice@example.com" {
		t.Fatalf("repo_git_identities[0]=%+v", detail.RepoGitIdentities[0])
	}
	if detail.IdleRecycleMinutes != 5 {
		t.Fatalf("idle_recycle_minutes=%d want default 5", detail.IdleRecycleMinutes)
	}
	if !detail.InstructionIdle.Enabled || detail.InstructionIdle.Minutes != 5 {
		t.Fatalf("instruction_idle=%+v", detail.InstructionIdle)
	}
}

func TestFetchTaskDetailRepoGitIdentitiesEmptyWhenNone(t *testing.T) {
	repo := &stubBusinessRepo{
		snap: &domain.TaskSnapshot{
			ID:          "task_2",
			Title:       "no-id",
			Description: "d",
			CompanyID:   "t1",
			WorkspaceID: "w1",
			AutoRun:     false,
		},
		repos: []domain.TaskRepoSnapshot{},
	}
	svc := application.NewTaskDetailService(repo)
	detail, err := svc.FetchTaskDetail("task_2", "")
	if err != nil {
		t.Fatalf("FetchTaskDetail: %v", err)
	}
	if detail.Task.AutoRun {
		t.Fatalf("auto_run=%v want false", detail.Task.AutoRun)
	}
	if detail.RepoGitIdentities == nil {
		t.Fatal("repo_git_identities must be non-nil empty slice")
	}
	if len(detail.RepoGitIdentities) != 0 {
		t.Fatalf("repo_git_identities len=%d want 0", len(detail.RepoGitIdentities))
	}
}

func TestFetchTaskDetailMergesNestedAndInheritsIdentities(t *testing.T) {
	parent := "https://gitlab.example/g/ram-work.git"
	child := "https://gitlab.example/g/task2app.git"
	repo := &stubBusinessRepo{
		snap: &domain.TaskSnapshot{
			ID: "task_nested", Title: "n", Description: "d",
			CompanyID: "t1", WorkspaceID: "w1",
		},
		repos: []domain.TaskRepoSnapshot{{
			ProjectID: "p1", ProjectName: "meta",
			RepoURLs:             []string{parent},
			RepoEntries:          []domain.RepoCloneEntry{{URL: parent}},
			AutoCloneNestedRepos: true,
		}},
		identities: []domain.GitIdentitySnapshot{{
			RepoURL: parent, UserID: 42, GitIdentityID: "id1",
			UserName: "Alice", UserEmail: "alice@example.com",
		}},
	}
	fetcher := &stubNestedFetcher{
		byParent: map[string][]application.NestedRepoItem{
			parent: {{Path: "task2app", URL: child, Source: "gitmodules"}},
		},
	}
	svc := application.NewTaskDetailService(repo).WithNestedFetcher(fetcher)
	detail, err := svc.FetchTaskDetail("task_nested", "")
	if err != nil {
		t.Fatalf("FetchTaskDetail: %v", err)
	}
	if len(detail.ProjectRepos[0].RepoURLs) != 2 {
		t.Fatalf("urls=%v", detail.ProjectRepos[0].RepoURLs)
	}
	foundChild := false
	for _, e := range detail.ProjectRepos[0].RepoEntries {
		if e.URL == child && e.CloneAlias == "task2app" {
			foundChild = true
			if e.ParentRepoURL != parent {
				t.Fatalf("child parent_repo_url=%q want %q", e.ParentRepoURL, parent)
			}
		}
	}
	if !foundChild {
		t.Fatalf("missing child entry: %+v", detail.ProjectRepos[0].RepoEntries)
	}
	if len(detail.RepoGitIdentities) != 2 {
		t.Fatalf("identities=%+v want parent+child", detail.RepoGitIdentities)
	}
}

func TestFetchTaskDetailMergesNestedWhenIdentitiesEmpty(t *testing.T) {
	parent := "https://github.com/task2money/ram-work.git"
	child := "https://github.com/task2money/AiMonitor.git"
	repo := &stubBusinessRepo{
		snap: &domain.TaskSnapshot{
			ID: "task_no_ident", Title: "n", Description: "d",
			CompanyID: "t1", WorkspaceID: "w1",
		},
		repos: []domain.TaskRepoSnapshot{{
			ProjectID: "p1", ProjectName: "meta",
			RepoURLs:             []string{parent},
			RepoEntries:          []domain.RepoCloneEntry{{URL: parent}},
			AutoCloneNestedRepos: true,
		}},
		identities: nil,
	}
	fetcher := &stubNestedFetcher{
		byParent: map[string][]application.NestedRepoItem{
			parent: {{Path: "AiMonitor", URL: child, Source: "gitmodules"}},
		},
	}
	svc := application.NewTaskDetailService(repo).WithNestedFetcher(fetcher)
	detail, err := svc.FetchTaskDetail("task_no_ident", "")
	if err != nil {
		t.Fatalf("FetchTaskDetail: %v", err)
	}
	if fetcher.calls != 1 {
		t.Fatalf("must enrich nested even when identities empty, calls=%d", fetcher.calls)
	}
	if len(detail.ProjectRepos[0].RepoURLs) != 2 {
		t.Fatalf("urls=%v want parent+child", detail.ProjectRepos[0].RepoURLs)
	}
}

func TestFetchTaskDetailSkipsNestedWhenAutoCloneDisabled(t *testing.T) {
	parent := "https://gitlab.example/g/ram-work.git"
	child := "https://gitlab.example/g/task2app.git"
	repo := &stubBusinessRepo{
		snap: &domain.TaskSnapshot{
			ID: "task_nested_off", Title: "n", Description: "d",
			CompanyID: "t1", WorkspaceID: "w1",
		},
		repos: []domain.TaskRepoSnapshot{{
			ProjectID: "p1", ProjectName: "meta",
			RepoURLs:             []string{parent},
			RepoEntries:          []domain.RepoCloneEntry{{URL: parent}},
			AutoCloneNestedRepos: false,
		}},
		identities: []domain.GitIdentitySnapshot{{
			RepoURL: parent, UserID: 42, GitIdentityID: "id1",
			UserName: "Alice", UserEmail: "alice@example.com",
		}},
	}
	fetcher := &stubNestedFetcher{
		byParent: map[string][]application.NestedRepoItem{
			parent: {{Path: "task2app", URL: child, Source: "gitmodules"}},
		},
	}
	svc := application.NewTaskDetailService(repo).WithNestedFetcher(fetcher)
	detail, err := svc.FetchTaskDetail("task_nested_off", "")
	if err != nil {
		t.Fatalf("FetchTaskDetail: %v", err)
	}
	if len(detail.ProjectRepos[0].RepoURLs) != 1 || detail.ProjectRepos[0].RepoURLs[0] != parent {
		t.Fatalf("disabled must keep parent only: %v", detail.ProjectRepos[0].RepoURLs)
	}
	if fetcher.calls != 0 {
		t.Fatalf("nested fetcher must not run when auto_clone_nested_repos=false, calls=%d", fetcher.calls)
	}
	if detail.ProjectRepos[0].AutoCloneNestedRepos {
		t.Fatalf("task-detail must surface auto_clone_nested_repos=false")
	}
}

func TestFetchTaskDetailMergesNestedUsingCommentAuthor(t *testing.T) {
	parent := "https://github.com/task2money/private-meta.git"
	child := "https://github.com/task2money/private-sub.git"
	repo := &stubBusinessRepo{
		snap: &domain.TaskSnapshot{
			ID: "task_cmt", Title: "n", Description: "d",
			CompanyID: "t1", WorkspaceID: "w1",
		},
		repos: []domain.TaskRepoSnapshot{{
			ProjectID: "p1", ProjectName: "meta",
			RepoURLs:             []string{parent},
			RepoEntries:          []domain.RepoCloneEntry{{URL: parent}},
			AutoCloneNestedRepos: true,
		}},
		identities: []domain.GitIdentitySnapshot{{
			RepoURL: parent, UserID: 1, GitIdentityID: "owner-id",
			UserName: "Owner", UserEmail: "owner@example.com",
		}},
		commentAuthorID: 99,
		userIdentities: []domain.GitIdentitySnapshot{{
			UserID: 99, GitIdentityID: "cmt-id",
			UserName: "Ann", UserEmail: "ann@example.com",
		}},
	}
	fetcher := &stubNestedFetcher{
		byParent: map[string][]application.NestedRepoItem{
			parent: {{Path: "private-sub", URL: child, Source: "gitmodules"}},
		},
	}
	svc := application.NewTaskDetailService(repo).WithNestedFetcher(fetcher)
	detail, err := svc.FetchTaskDetail("task_cmt", "cmt_1")
	if err != nil {
		t.Fatalf("FetchTaskDetail: %v", err)
	}
	if fetcher.lastUserID != 99 {
		t.Fatalf("nested fetch user=%d want comment author 99", fetcher.lastUserID)
	}
	if len(detail.ProjectRepos[0].RepoURLs) != 2 {
		t.Fatalf("urls=%v want parent+child", detail.ProjectRepos[0].RepoURLs)
	}
	for _, id := range detail.RepoGitIdentities {
		if id.UserName != "Ann" {
			t.Fatalf("must expose comment author git identity, got %+v", id)
		}
	}
}

func TestFetchTaskDetailPassesCommentIDAndKeepsCommentIdentity(t *testing.T) {
	repoURL := "https://git.example/a.git"
	repo := &stubBusinessRepo{
		snap: &domain.TaskSnapshot{
			ID: "task_1", Title: "demo", Description: "d", CompanyID: "t1", WorkspaceID: "w1",
		},
		repos: []domain.TaskRepoSnapshot{{
			ProjectID: "p1", RepoURLs: []string{repoURL}, AutoCloneNestedRepos: false,
		}},
		identities: []domain.GitIdentitySnapshot{{
			RepoURL: repoURL, GitIdentityID: "gid-comment", UserID: 99,
			UserName: "Ann", UserEmail: "ann@example.com", FromComment: true,
		}},
		commentAuthorID: 99,
		userIdentities: []domain.GitIdentitySnapshot{
			{GitIdentityID: "gid-other", UserID: 99, UserName: "Other", UserEmail: "other@example.com"},
		},
	}
	svc := application.NewTaskDetailService(repo)
	detail, err := svc.FetchTaskDetail("task_1", "cmt_ann")
	if err != nil {
		t.Fatalf("FetchTaskDetail: %v", err)
	}
	if repo.lastIdentTask != "task_1" || repo.lastIdentCmt != "cmt_ann" {
		t.Fatalf("FetchRepoIdentities got task=%s comment=%s", repo.lastIdentTask, repo.lastIdentCmt)
	}
	if len(detail.RepoGitIdentities) != 1 {
		t.Fatalf("identities=%+v", detail.RepoGitIdentities)
	}
	got := detail.RepoGitIdentities[0]
	if got.IdentityID != "gid-comment" || got.UserName != "Ann" {
		t.Fatalf("want comment identity, got %+v", got)
	}
}

type stubPolicyFetcher struct {
	minutes int
	sts     map[string]interface{}
	err     error
	calls   int
}

func (s *stubPolicyFetcher) FetchWorkspaceMachinePolicy(companyID, workspaceID, taskID, commentID string) (ports.WorkspaceMachinePolicy, error) {
	s.calls++
	if s.err != nil {
		return ports.WorkspaceMachinePolicy{}, s.err
	}
	return ports.WorkspaceMachinePolicy{IdleRecycleMinutes: s.minutes, MachineReleaseSTS: s.sts}, nil
}

func TestFetchTaskDetailUsesCloudPolicyMinutes(t *testing.T) {
	repo := &stubBusinessRepo{
		snap: &domain.TaskSnapshot{ID: "task_p", Title: "p", CompanyID: "t1", WorkspaceID: "w1"},
	}
	pol := &stubPolicyFetcher{minutes: 45}
	svc := application.NewTaskDetailService(repo).WithPolicyFetcher(pol)
	detail, err := svc.FetchTaskDetail("task_p", "cmt-1")
	if err != nil {
		t.Fatal(err)
	}
	if pol.calls != 1 {
		t.Fatalf("policy calls=%d", pol.calls)
	}
	if detail.IdleRecycleMinutes != 45 || !detail.InstructionIdle.Enabled {
		t.Fatalf("detail idle=%d policy=%+v", detail.IdleRecycleMinutes, detail.InstructionIdle)
	}
}

func TestFetchTaskDetailMinutesZeroDisablesIdle(t *testing.T) {
	repo := &stubBusinessRepo{
		snap: &domain.TaskSnapshot{ID: "task_z", Title: "z", CompanyID: "t1", WorkspaceID: "w1"},
	}
	svc := application.NewTaskDetailService(repo).WithPolicyFetcher(&stubPolicyFetcher{minutes: 0})
	detail, err := svc.FetchTaskDetail("task_z", "")
	if err != nil {
		t.Fatal(err)
	}
	if detail.IdleRecycleMinutes != 0 || detail.InstructionIdle.Enabled {
		t.Fatalf("want disabled, got minutes=%d enabled=%v", detail.IdleRecycleMinutes, detail.InstructionIdle.Enabled)
	}
}

func TestAttachIdlePolicyOmitsSTSWithoutMap(t *testing.T) {
	repo := &stubBusinessRepo{
		snap: &domain.TaskSnapshot{ID: "task_s", Title: "s", CompanyID: "t1", WorkspaceID: "w1"},
	}
	svc := application.NewTaskDetailService(repo).WithPolicyFetcher(&stubPolicyFetcher{
		minutes: 30,
		sts:     map[string]interface{}{"access_key_id": "STS.x"},
	})
	detail, err := svc.FetchTaskDetail("task_s", "cmt")
	if err != nil {
		t.Fatal(err)
	}
	if detail.MachineReleaseSTS["access_key_id"] != "STS.x" {
		t.Fatalf("sts=%v", detail.MachineReleaseSTS)
	}
	svc.WithPolicyFetcher(&stubPolicyFetcher{minutes: 30})
	svc.AttachIdlePolicy(detail, "cmt")
	if detail.MachineReleaseSTS != nil {
		t.Fatalf("sts should be omitted without role: %#v", detail.MachineReleaseSTS)
	}
}
