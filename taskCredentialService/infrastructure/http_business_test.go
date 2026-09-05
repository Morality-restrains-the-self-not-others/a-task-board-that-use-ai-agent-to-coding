package infrastructure

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"taskCredentialService/application"
)

func TestHTTPBusinessRepository_FetchAll(t *testing.T) {
	taskSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/created-by") {
			_ = json.NewEncoder(w).Encode(map[string]string{"comment_id": "cmt-1", "user_id": "99"})
			return
		}
		if !strings.HasSuffix(r.URL.Path, "/container-snapshot") {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":                 "task-1",
			"title":              "T",
			"description":        "D",
			"workspace_id":       "ws1",
			"tenant_id":          "ten1",
			"auto_run":           true,
			"installed_image_id": "img-1",
			"owner_id":           "873438061961179136",
			"target_branch":      "work/x",
			"branch_strategy": map[string]string{
				"work_branch_name":         "work/x",
				"merge_target_branch_name": "main",
				"target_branch_name":       "feat",
			},
			"projects": []map[string]interface{}{
				{
					"project_id":   "p1",
					"project_name": "Proj",
					"git_repos":    []string{"https://git.example/a.git"},
					"repo_branches": []map[string]string{
						{"git_repo": "https://git.example/a.git", "base_branch": "main", "target_branch": "feat"},
					},
				},
			},
			"repo_identities": []map[string]string{
				{
					"repo_url":        "https://git.example/a.git",
					"git_identity_id": "gid-comment",
					"user_id":         "99",
					"user_name":       "Ann",
					"user_email":      "ann@example.com",
				},
			},
		})
	}))
	t.Cleanup(taskSrv.Close)

	repo := NewHTTPBusinessRepository(taskSrv.URL, "", 5)

	snap, err := repo.FetchTaskSnapshot("task-1", "")
	if err != nil {
		t.Fatalf("FetchTaskSnapshot: %v", err)
	}
	if snap.Title != "T" || snap.CompanyID != "ten1" || snap.TargetBranch != "work/x" || !snap.AutoRun {
		t.Fatalf("unexpected snapshot: %+v", snap)
	}
	if snap.InstalledImageID != "img-1" {
		t.Fatalf("installed_image_id=%q", snap.InstalledImageID)
	}

	repos, err := repo.FetchTaskRepos("task-1", "")
	if err != nil {
		t.Fatalf("FetchTaskRepos: %v", err)
	}
	if len(repos) != 1 || repos[0].ProjectName != "Proj" || len(repos[0].RepoURLs) != 1 {
		t.Fatalf("unexpected repos: %+v", repos)
	}
	if !repos[0].AutoCloneNestedRepos {
		t.Fatalf("missing auto_clone_nested_repos should default true, got %+v", repos[0])
	}

	idents, err := repo.FetchRepoIdentities("task-1", "cmt-1")
	if err != nil {
		t.Fatalf("FetchRepoIdentities: %v", err)
	}
	if len(idents) != 1 || idents[0].GitIdentityID != "gid-comment" || idents[0].UserName != "Ann" || !idents[0].FromComment {
		t.Fatalf("HTTP snapshot identities=%+v", idents)
	}
	authorID, err := repo.FetchCommentCreatedByUserID("cmt-1")
	if err != nil {
		t.Fatalf("FetchCommentCreatedByUserID: %v", err)
	}
	if authorID != 99 {
		t.Fatalf("comment author=%d want 99 (HTTP fallback resolves via internal API)", authorID)
	}
}

func TestHTTPBusinessRepository_TaskNotFound(t *testing.T) {
	taskSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
	}))
	t.Cleanup(taskSrv.Close)
	repo := NewHTTPBusinessRepository(taskSrv.URL, "", 5)
	_, err := repo.FetchTaskSnapshot("missing", "")
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestHTTPBusinessRepository_FetchTaskReposHonorsAutoCloneNestedReposFalse(t *testing.T) {
	taskSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/container-snapshot") {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":           "task-off",
			"title":        "T",
			"description":  "D",
			"workspace_id": "ws1",
			"tenant_id":    "ten1",
			"projects": []map[string]interface{}{
				{
					"project_id":              "p-off",
					"project_name":            "Off",
					"git_repos":               []string{"https://git.example/parent.git"},
					"auto_clone_nested_repos": false,
					"git_repo_entries":        []map[string]string{{"url": "https://git.example/parent.git"}},
				},
			},
		})
	}))
	t.Cleanup(taskSrv.Close)

	repo := NewHTTPBusinessRepository(taskSrv.URL, "", 5)
	repos, err := repo.FetchTaskRepos("task-off", "")
	if err != nil {
		t.Fatalf("FetchTaskRepos: %v", err)
	}
	if len(repos) != 1 {
		t.Fatalf("repos len=%d want 1", len(repos))
	}
	if repos[0].AutoCloneNestedRepos {
		t.Fatalf("auto_clone_nested_repos=false must pass through snapshot, got %+v", repos[0])
	}
}

func TestHTTPBusinessRepository_FetchRepoIdentitiesKeepsSiteLevelOAuthGrant(t *testing.T) {
	taskSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/container-snapshot") {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":           "task-1",
			"title":        "T",
			"description":  "D",
			"workspace_id": "ws1",
			"tenant_id":    "ten1",
			"repo_identities": []map[string]string{
				{
					"repo_url":             "",
					"oauth_gitsite":        "gitlab-tencent-sh-1.daydaymoney.com",
					"oauth_remote_user_id": "gl-user",
				},
			},
		})
	}))
	t.Cleanup(taskSrv.Close)

	repo := NewHTTPBusinessRepository(taskSrv.URL, "", 5)
	idents, err := repo.FetchRepoIdentities("task-1", "cmt-1")
	if err != nil {
		t.Fatalf("FetchRepoIdentities: %v", err)
	}
	if len(idents) != 1 {
		t.Fatalf("site-level L2 must not be dropped, got %+v", idents)
	}
	if idents[0].OauthGitsite != "gitlab-tencent-sh-1.daydaymoney.com" || idents[0].RepoURL != "" || !idents[0].FromComment {
		t.Fatalf("want site-level comment grant, got %+v", idents[0])
	}
}

func TestHTTPBusinessRepository_LoadSnapshotPassesCommentID(t *testing.T) {
	var gotQuery string
	taskSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":           "task-c",
			"title":        "T",
			"description":  "D",
			"workspace_id": "ws1",
			"tenant_id":    "ten1",
		})
	}))
	t.Cleanup(taskSrv.Close)

	repo := NewHTTPBusinessRepository(taskSrv.URL, "", 5)
	if _, err := repo.FetchTaskRepos("task-c", "cmt_1"); err != nil {
		t.Fatalf("FetchTaskRepos: %v", err)
	}
	if gotQuery != "comment_id=cmt_1" {
		t.Fatalf("expected comment_id query, got %q", gotQuery)
	}
}

func TestHTTPBusinessRepository_LoadSnapshotWithoutCommentIDKeepsTaskLevel(t *testing.T) {
	var gotQuery string
	taskSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":           "task-t",
			"title":        "T",
			"description":  "D",
			"workspace_id": "ws1",
			"tenant_id":    "ten1",
		})
	}))
	t.Cleanup(taskSrv.Close)

	repo := NewHTTPBusinessRepository(taskSrv.URL, "", 5)
	if _, err := repo.FetchTaskSnapshot("task-t", ""); err != nil {
		t.Fatalf("FetchTaskSnapshot: %v", err)
	}
	if gotQuery != "" {
		t.Fatalf("expected no comment_id query for task-level fallback, got %q", gotQuery)
	}
}

// TestHTTPBusinessRepository_EnrichDiscoveryUserIsCommentAuthor is the OPT-021
// regression: on the HTTP fallback path, nested-repo discovery must run as the
// comment author (resolved via the owner-service internal API), never task-level
// or anonymous.
func TestHTTPBusinessRepository_EnrichDiscoveryUserIsCommentAuthor(t *testing.T) {
	taskSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/created-by") {
			_ = json.NewEncoder(w).Encode(map[string]string{"comment_id": "cmt_author", "user_id": "12345"})
			return
		}
		if !strings.HasSuffix(r.URL.Path, "/container-snapshot") {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":            "task-1",
			"title":         "T",
			"description":   "D",
			"workspace_id":  "ws1",
			"tenant_id":     "ten1",
			"auto_run":      true,
			"owner_id":      "11111",
			"target_branch": "work/x",
			"projects": []map[string]interface{}{
				{
					"project_id":   "p1",
					"project_name": "Proj",
					"git_repos":    []string{"https://git.example/a.git"},
				},
			},
			"repo_identities": []map[string]string{},
		})
	}))
	t.Cleanup(taskSrv.Close)

	repo := NewHTTPBusinessRepository(taskSrv.URL, "", 5)

	repos, err := repo.FetchTaskRepos("task-1", "cmt_author")
	if err != nil {
		t.Fatalf("FetchTaskRepos: %v", err)
	}
	idents, err := repo.FetchRepoIdentities("task-1", "cmt_author")
	if err != nil {
		t.Fatalf("FetchRepoIdentities: %v", err)
	}
	authorID, err := repo.FetchCommentCreatedByUserID("cmt_author")
	if err != nil {
		t.Fatalf("FetchCommentCreatedByUserID: %v", err)
	}
	if authorID != 12345 {
		t.Fatalf("author=%d want 12345", authorID)
	}

	_, _, discoveryUserID := application.EnrichReposForCommentAuthor(
		authorID, nil, idents, repos, "", nil,
	)
	if discoveryUserID != authorID {
		t.Fatalf("discovery_user=%d want author=%d (HTTP fallback must not drop to task-level/anon)", discoveryUserID, authorID)
	}
}
