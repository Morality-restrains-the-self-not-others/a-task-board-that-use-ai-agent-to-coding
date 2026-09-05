package main

import (
	"strings"
	"testing"
	"time"
)

func TestProjectIDsForRepoURL_PrefersExactThenGitsite(t *testing.T) {
	t.Parallel()
	links := []taskLinkedProjectRepo{
		{ProjectID: "p-other", RepoAddress: "https://gitlab.example/x.git"},
		{ProjectID: "p-exact", RepoAddress: "https://github.com/acme/demo.git"},
		{ProjectID: "p-site", RepoAddress: "https://github.com/acme/other.git"},
	}
	got := projectIDsForRepoURL(links, "https://github.com/acme/demo.git")
	if len(got) != 1 || got[0] != "p-exact" {
		t.Fatalf("exact match got=%v", got)
	}
	got = projectIDsForRepoURL(links, "https://github.com/acme/missing.git")
	if len(got) != 2 || got[0] != "p-exact" || got[1] != "p-site" {
		t.Fatalf("gitsite fallback got=%v", got)
	}
}

func TestApplyProjectL2SeedToIdentities_StampsMatchingProjectOnly(t *testing.T) {
	setupTestDB(t)
	now := time.Now().UTC()
	_, err := db.Exec(
		`INSERT INTO task_tasks(id,tenant_id,title,description,completed,priority,order_num,workspace_id,owner_id,deliverable_obj_id,progress_column_id,parent_task_id,fork_from_id,installed_image_id,auto_run,feature_params_source,personal_feature_params_config_id,task_kind,code_lang,created_at,updated_at)
		 VALUES(?,?,?,?,0,'medium',1,?,?,?,?,?,?,?,1,'','','','',?,?)`,
		"task_seed_l2", "t1", "T", "D", "ws1", "u1", "", "", "", "", "", now, now,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(
		`INSERT INTO task_projects(id,task_id,project_id,base_branch,target_branch,repo_address) VALUES(?,?,?,?,?,?)`,
		"tp1", "task_seed_l2", "proj_granted", "main", "feat", "https://github.com/acme/demo.git",
	); err != nil {
		t.Fatal(err)
	}

	calls := []string{}
	prev := lookupProjectGitOAuthGrantFn
	lookupProjectGitOAuthGrantFn = func(projectID, userID, gitsite string) (string, bool) {
		calls = append(calls, projectID+"|"+userID+"|"+gitsite)
		if projectID == "proj_granted" && userID == "u1" && gitsite == "github.com" {
			return "gh-from-project", true
		}
		return "", false
	}
	t.Cleanup(func() { lookupProjectGitOAuthGrantFn = prev })

	got := applyProjectL2SeedToIdentities("cmt_seed", "u1", "task_seed_l2", []RepoIdentitySelection{{
		RepoURL:       "https://github.com/acme/demo.git",
		GitIdentityID: "gid-1",
	}})
	if len(got) != 1 {
		t.Fatalf("len=%d", len(got))
	}
	if got[0].OauthGitsite != "github.com" || got[0].OauthRemoteUserID != "gh-from-project" {
		t.Fatalf("seeded=%+v", got[0])
	}
	if got[0].OauthGrantedAt == "" {
		t.Fatal("oauth_granted_at empty")
	}
	if len(calls) != 1 || calls[0] != "proj_granted|u1|github.com" {
		t.Fatalf("lookup calls=%v", calls)
	}
}

func TestApplyProjectL2SeedToIdentities_SkipsOtherUserAndExistingTicket(t *testing.T) {
	setupTestDB(t)
	now := time.Now().UTC()
	_, err := db.Exec(
		`INSERT INTO task_tasks(id,tenant_id,title,description,completed,priority,order_num,workspace_id,owner_id,deliverable_obj_id,progress_column_id,parent_task_id,fork_from_id,installed_image_id,auto_run,feature_params_source,personal_feature_params_config_id,task_kind,code_lang,created_at,updated_at)
		 VALUES(?,?,?,?,0,'medium',1,?,?,?,?,?,?,?,1,'','','','',?,?)`,
		"task_seed_skip", "t1", "T", "D", "ws1", "u1", "", "", "", "", "", now, now,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(
		`INSERT INTO task_projects(id,task_id,project_id,base_branch,target_branch,repo_address) VALUES(?,?,?,?,?,?)`,
		"tp2", "task_seed_skip", "proj_granted", "main", "feat", "https://github.com/acme/demo.git",
	); err != nil {
		t.Fatal(err)
	}

	prev := lookupProjectGitOAuthGrantFn
	lookupProjectGitOAuthGrantFn = func(projectID, userID, gitsite string) (string, bool) {
		if userID == "u1" {
			return "should-not-overwrite", true
		}
		return "", false
	}
	t.Cleanup(func() { lookupProjectGitOAuthGrantFn = prev })

	ticketed := applyProjectL2SeedToIdentities("cmt_t", "u1", "task_seed_skip", []RepoIdentitySelection{{
		RepoURL:           "https://github.com/acme/demo.git",
		GitIdentityID:     "gid-1",
		OauthGitsite:      "github.com",
		OauthRemoteUserID: "from-ticket",
	}})
	if ticketed[0].OauthRemoteUserID != "from-ticket" {
		t.Fatalf("ticket must win: %+v", ticketed[0])
	}

	other := applyProjectL2SeedToIdentities("cmt_o", "u-other", "task_seed_skip", []RepoIdentitySelection{{
		RepoURL:       "https://github.com/acme/demo.git",
		GitIdentityID: "gid-1",
	}})
	if strings.TrimSpace(other[0].OauthGitsite) != "" {
		t.Fatalf("other user must not inherit: %+v", other[0])
	}
}

func TestApplyProjectL2SeedToIdentities_LookupFailureLeavesUnmarked(t *testing.T) {
	setupTestDB(t)
	now := time.Now().UTC()
	_, err := db.Exec(
		`INSERT INTO task_tasks(id,tenant_id,title,description,completed,priority,order_num,workspace_id,owner_id,deliverable_obj_id,progress_column_id,parent_task_id,fork_from_id,installed_image_id,auto_run,feature_params_source,personal_feature_params_config_id,task_kind,code_lang,created_at,updated_at)
		 VALUES(?,?,?,?,0,'medium',1,?,?,?,?,?,?,?,1,'','','','',?,?)`,
		"task_seed_fail", "t1", "T", "D", "ws1", "u1", "", "", "", "", "", now, now,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(
		`INSERT INTO task_projects(id,task_id,project_id,base_branch,target_branch,repo_address) VALUES(?,?,?,?,?,?)`,
		"tp3", "task_seed_fail", "proj_x", "main", "feat", "https://github.com/acme/demo.git",
	); err != nil {
		t.Fatal(err)
	}
	prev := lookupProjectGitOAuthGrantFn
	lookupProjectGitOAuthGrantFn = func(projectID, userID, gitsite string) (string, bool) {
		return "", false
	}
	t.Cleanup(func() { lookupProjectGitOAuthGrantFn = prev })

	got := applyProjectL2SeedToIdentities("cmt_fail", "u1", "task_seed_fail", []RepoIdentitySelection{{
		RepoURL: "https://github.com/acme/demo.git",
	}})
	if strings.TrimSpace(got[0].OauthGitsite) != "" {
		t.Fatalf("want unmarked, got %+v", got[0])
	}
}
