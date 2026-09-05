package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

const forkSeedGitlabSite = "gitlab-tencent-sh-1.daydaymoney.com"
const forkSeedGitlabURL = "https://gitlab-tencent-sh-1.daydaymoney.com/example-user/ram-work.git"

func insertForkSeedTask(t *testing.T, taskID, forkFrom, owner string) {
	t.Helper()
	now := time.Now().UTC()
	_, err := db.Exec(
		`INSERT INTO task_tasks(id,tenant_id,title,description,completed,priority,order_num,workspace_id,owner_id,deliverable_obj_id,progress_column_id,parent_task_id,fork_from_id,installed_image_id,auto_run,feature_params_source,personal_feature_params_config_id,task_kind,code_lang,created_at,updated_at)
		 VALUES(?,?,?,?,0,'medium',1,?,?,?,?,?,?,?,1,'','','','',?,?)`,
		taskID, "t1", "T", "D", "ws1", owner, "", "", "", forkFrom, "img-1", now, now,
	)
	if err != nil {
		t.Fatal(err)
	}
}

func insertForkSeedComment(t *testing.T, commentID, taskID, userID, identitiesJSON string) {
	t.Helper()
	now := time.Now().UTC()
	if _, err := db.Exec(
		`INSERT INTO task_comments(id,task_id,created_by_id,content,mentions_json,repo_identities_json,created_at) VALUES(?,?,?,?,?,?,?)`,
		commentID, taskID, userID, autoRunAtCommentPrefix+"\nrun", "[]", identitiesJSON, now,
	); err != nil {
		t.Fatal(err)
	}
}

func TestApplyForkSourceCommentL2SeedToIdentities_StampsSameUser(t *testing.T) {
	setupTestDB(t)
	insertForkSeedTask(t, "task_fork_src", "", "u1")
	insertForkSeedComment(t, "cmt_src", "task_fork_src", "u1",
		`[{"repo_url":"`+forkSeedGitlabURL+`","git_identity_id":"gid-src","oauth_gitsite":"`+forkSeedGitlabSite+`","oauth_remote_user_id":"gl-42"}]`)
	insertForkSeedTask(t, "task_fork_dst", "task_fork_src", "u1")

	got := applyForkSourceCommentL2SeedToIdentities("cmt_dst", "u1", "task_fork_dst", []RepoIdentitySelection{{
		RepoURL:       forkSeedGitlabURL,
		GitIdentityID: "gid-dst",
	}})
	if len(got) != 1 {
		t.Fatalf("len=%d", len(got))
	}
	if got[0].OauthGitsite != forkSeedGitlabSite || got[0].OauthRemoteUserID != "gl-42" {
		t.Fatalf("stamped=%+v", got[0])
	}
	if got[0].OauthGrantedAt == "" {
		t.Fatal("oauth_granted_at empty")
	}
}

func TestApplyForkSourceCommentL2SeedToIdentities_SiteLevelSource(t *testing.T) {
	setupTestDB(t)
	insertForkSeedTask(t, "task_fork_src_site", "", "u1")
	insertForkSeedComment(t, "cmt_src_site", "task_fork_src_site", "u1",
		`[{"repo_url":"","oauth_gitsite":"`+forkSeedGitlabSite+`","oauth_remote_user_id":"gl-site"}]`)
	insertForkSeedTask(t, "task_fork_dst_site", "task_fork_src_site", "u1")

	got := applyForkSourceCommentL2SeedToIdentities("cmt_dst_site", "u1", "task_fork_dst_site", []RepoIdentitySelection{{
		RepoURL:       forkSeedGitlabURL,
		GitIdentityID: "gid-dst",
	}})
	if got[0].OauthGitsite != forkSeedGitlabSite || got[0].OauthRemoteUserID != "gl-site" {
		t.Fatalf("site-level seed=%+v", got[0])
	}
}

func TestApplyForkSourceCommentL2SeedToIdentities_SkipsOtherUserAndNonFork(t *testing.T) {
	setupTestDB(t)
	insertForkSeedTask(t, "task_fork_src_o", "", "u-src")
	insertForkSeedComment(t, "cmt_src_o", "task_fork_src_o", "u-src",
		`[{"repo_url":"`+forkSeedGitlabURL+`","oauth_gitsite":"`+forkSeedGitlabSite+`","oauth_remote_user_id":"gl-other"}]`)
	insertForkSeedTask(t, "task_fork_dst_o", "task_fork_src_o", "u1")
	insertForkSeedTask(t, "task_plain", "", "u1")

	other := applyForkSourceCommentL2SeedToIdentities("cmt_o", "u1", "task_fork_dst_o", []RepoIdentitySelection{{
		RepoURL: forkSeedGitlabURL,
	}})
	if strings.TrimSpace(other[0].OauthGitsite) != "" {
		t.Fatalf("other user must not inherit: %+v", other[0])
	}

	plain := applyForkSourceCommentL2SeedToIdentities("cmt_p", "u1", "task_plain", []RepoIdentitySelection{{
		RepoURL: forkSeedGitlabURL,
	}})
	if strings.TrimSpace(plain[0].OauthGitsite) != "" {
		t.Fatalf("non-fork must not seed: %+v", plain[0])
	}
}

func TestApplyForkSourceCommentL2SeedToIdentities_DoesNotOverwrite(t *testing.T) {
	setupTestDB(t)
	insertForkSeedTask(t, "task_fork_src_k", "", "u1")
	insertForkSeedComment(t, "cmt_src_k", "task_fork_src_k", "u1",
		`[{"repo_url":"`+forkSeedGitlabURL+`","oauth_gitsite":"`+forkSeedGitlabSite+`","oauth_remote_user_id":"gl-src"}]`)
	insertForkSeedTask(t, "task_fork_dst_k", "task_fork_src_k", "u1")

	got := applyForkSourceCommentL2SeedToIdentities("cmt_k", "u1", "task_fork_dst_k", []RepoIdentitySelection{{
		RepoURL:           forkSeedGitlabURL,
		GitIdentityID:     "gid-1",
		OauthGitsite:      forkSeedGitlabSite,
		OauthRemoteUserID: "from-ticket",
	}})
	if got[0].OauthRemoteUserID != "from-ticket" {
		t.Fatalf("existing L2 must win: %+v", got[0])
	}
}

func TestApplyForkSourceCommentL2SeedToIdentities_EmitsEvent(t *testing.T) {
	setupTestDB(t)
	insertForkSeedTask(t, "task_fork_src_ev", "", "u1")
	insertForkSeedComment(t, "cmt_src_ev", "task_fork_src_ev", "u1",
		`[{"repo_url":"`+forkSeedGitlabURL+`","oauth_gitsite":"`+forkSeedGitlabSite+`","oauth_remote_user_id":"gl-ev"}]`)
	insertForkSeedTask(t, "task_fork_dst_ev", "task_fork_src_ev", "u1")

	type captured struct {
		typ  string
		data map[string]interface{}
		key  string
	}
	var got []captured
	prevPub := publishDomainEventFn
	publishDomainEventFn = func(_ context.Context, eventType string, data map[string]interface{}, key string) error {
		got = append(got, captured{eventType, data, key})
		return nil
	}
	t.Cleanup(func() { publishDomainEventFn = prevPub })

	_ = applyForkSourceCommentL2SeedToIdentities("cmt_ev", "u1", "task_fork_dst_ev", []RepoIdentitySelection{{
		RepoURL:       forkSeedGitlabURL,
		GitIdentityID: "gid-1",
	}})
	if len(got) != 1 {
		t.Fatalf("events=%+v", got)
	}
	if got[0].typ != "COMMENT_GIT_OAUTH_GRANTED" {
		t.Fatalf("type=%q", got[0].typ)
	}
	wantKey := "grant:comment:cmt_ev:u1:" + forkSeedGitlabSite
	if got[0].key != wantKey {
		t.Fatalf("key=%q want %q", got[0].key, wantKey)
	}
	if got[0].data["via"] != "fork_source_comment_l2_seed" {
		t.Fatalf("via=%v", got[0].data["via"])
	}
	if got[0].data["fork_from_task_id"] != "task_fork_src_ev" {
		t.Fatalf("fork_from=%v", got[0].data["fork_from_task_id"])
	}
}

func TestApplyForkSourceCommentL2SeedToIdentities_WalksGrandparent(t *testing.T) {
	setupTestDB(t)
	insertForkSeedTask(t, "task_fork_root", "", "u1")
	insertForkSeedComment(t, "cmt_root", "task_fork_root", "u1",
		`[{"repo_url":"`+forkSeedGitlabURL+`","oauth_gitsite":"`+forkSeedGitlabSite+`","oauth_remote_user_id":"gl-root"}]`)
	insertForkSeedTask(t, "task_fork_mid", "task_fork_root", "u1")
	insertForkSeedComment(t, "cmt_mid", "task_fork_mid", "u1",
		`[{"repo_url":"`+forkSeedGitlabURL+`","git_identity_id":"gid-mid"}]`)
	insertForkSeedTask(t, "task_fork_leaf", "task_fork_mid", "u1")

	got := applyForkSourceCommentL2SeedToIdentities("cmt_leaf", "u1", "task_fork_leaf", []RepoIdentitySelection{{
		RepoURL:       forkSeedGitlabURL,
		GitIdentityID: "gid-leaf",
	}})
	if got[0].OauthGitsite != forkSeedGitlabSite || got[0].OauthRemoteUserID != "gl-root" {
		t.Fatalf("grandparent seed=%+v", got[0])
	}
}

func TestLoadSnapshotRepoIdentities_ImplicitGitsiteWhenSeedsMiss(t *testing.T) {
	setupTestDB(t)
	insertForkSeedTask(t, "task_imp_src", "", "u1")
	insertForkSeedComment(t, "cmt_imp_src", "task_imp_src", "u1",
		`[{"repo_url":"`+forkSeedGitlabURL+`","git_identity_id":"gid-src"}]`)
	insertForkSeedTask(t, "task_imp_dst", "task_imp_src", "u1")
	insertForkSeedComment(t, "cmt_imp_dst", "task_imp_dst", "u1",
		`[{"repo_url":"`+forkSeedGitlabURL+`","git_identity_id":"gid-dst"}]`)
	if _, err := db.Exec(
		`INSERT INTO task_projects(id,task_id,project_id,base_branch,target_branch,repo_address) VALUES(?,?,?,?,?,?)`,
		"tp_imp", "task_imp_dst", "proj_no_grant", "main", "feat", forkSeedGitlabURL,
	); err != nil {
		t.Fatal(err)
	}
	prev := lookupProjectGitOAuthGrantFn
	lookupProjectGitOAuthGrantFn = func(projectID, userID, gitsite string) (string, bool) {
		return "", false
	}
	t.Cleanup(func() { lookupProjectGitOAuthGrantFn = prev })

	maps, err := loadSnapshotRepoIdentities("task_imp_dst", "cmt_imp_dst")
	if err != nil {
		t.Fatal(err)
	}
	if len(maps) != 1 || maps[0]["oauth_gitsite"] != forkSeedGitlabSite {
		t.Fatalf("implicit snapshot maps=%v", maps)
	}
}

func TestLoadSnapshotRepoIdentities_PersistsProjectL2Seed(t *testing.T) {
	setupTestDB(t)
	insertForkSeedTask(t, "task_proj_l2", "", "u1")
	insertForkSeedComment(t, "cmt_proj_l2", "task_proj_l2", "u1",
		`[{"repo_url":"`+forkSeedGitlabURL+`","git_identity_id":"gid-dst"}]`)
	if _, err := db.Exec(
		`INSERT INTO task_projects(id,task_id,project_id,base_branch,target_branch,repo_address) VALUES(?,?,?,?,?,?)`,
		"tp_proj_l2", "task_proj_l2", "proj_has_grant", "main", "feat", forkSeedGitlabURL,
	); err != nil {
		t.Fatal(err)
	}
	prev := lookupProjectGitOAuthGrantFn
	lookupProjectGitOAuthGrantFn = func(projectID, userID, gitsite string) (string, bool) {
		if projectID == "proj_has_grant" && userID == "u1" && gitsite == forkSeedGitlabSite {
			return "gl-proj", true
		}
		return "", false
	}
	t.Cleanup(func() { lookupProjectGitOAuthGrantFn = prev })

	maps, err := loadSnapshotRepoIdentities("task_proj_l2", "cmt_proj_l2")
	if err != nil {
		t.Fatal(err)
	}
	if len(maps) != 1 || maps[0]["oauth_gitsite"] != forkSeedGitlabSite || maps[0]["oauth_remote_user_id"] != "gl-proj" {
		t.Fatalf("project L2 snapshot maps=%v", maps)
	}
	var stored string
	if err := db.QueryRow(`SELECT COALESCE(repo_identities_json,'') FROM task_comments WHERE id=?`, "cmt_proj_l2").Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stored, `"oauth_gitsite":"`+forkSeedGitlabSite+`"`) || !strings.Contains(stored, `"oauth_remote_user_id":"gl-proj"`) {
		t.Fatalf("db not repaired after in-place project seed: %s", stored)
	}
}

func TestLoadSnapshotRepoIdentities_RepairsForkSourceL2(t *testing.T) {
	setupTestDB(t)
	insertForkSeedTask(t, "task_snap_src", "", "u1")
	insertForkSeedComment(t, "cmt_snap_src", "task_snap_src", "u1",
		`[{"repo_url":"`+forkSeedGitlabURL+`","git_identity_id":"gid-src","oauth_gitsite":"`+forkSeedGitlabSite+`","oauth_remote_user_id":"gl-snap"}]`)
	insertForkSeedTask(t, "task_snap_dst", "task_snap_src", "u1")
	insertForkSeedComment(t, "cmt_snap_dst", "task_snap_dst", "u1",
		`[{"repo_url":"`+forkSeedGitlabURL+`","git_identity_id":"gid-dst"}]`)

	maps, err := loadSnapshotRepoIdentities("task_snap_dst", "cmt_snap_dst")
	if err != nil {
		t.Fatal(err)
	}
	if len(maps) != 1 || maps[0]["oauth_gitsite"] != forkSeedGitlabSite || maps[0]["oauth_remote_user_id"] != "gl-snap" {
		t.Fatalf("snapshot maps=%v", maps)
	}
	var stored string
	if err := db.QueryRow(`SELECT COALESCE(repo_identities_json,'') FROM task_comments WHERE id=?`, "cmt_snap_dst").Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stored, `"oauth_gitsite":"`+forkSeedGitlabSite+`"`) {
		t.Fatalf("db not repaired: %s", stored)
	}
}

func TestEnsureAutoRunAtComment_StampsImplicitGitLabWithoutForkL2(t *testing.T) {
	setupTestDB(t)
	insertForkSeedTask(t, "task_impl_plain", "", "u1")

	prevLookup := lookupInstalledImageFn
	lookupInstalledImageFn = func(tenantID, imageID string, _ ...string) (*InstalledImageLookup, error) {
		return &InstalledImageLookup{ID: imageID, Name: "img", ExternalImageID: "ext"}, nil
	}
	t.Cleanup(func() { lookupInstalledImageFn = prevLookup })

	prevURL := cfg.AICommentServiceURL
	cfg.AICommentServiceURL = ""
	t.Cleanup(func() { cfg.AICommentServiceURL = prevURL })

	id, err := ensureAutoRunAtComment(autoRunTriggerParams{
		TenantID:    "t1",
		WorkspaceID: "ws1",
		TaskID:      "task_impl_plain",
		UserID:      "u1",
		ImageID:     "img-1",
		RepoIdentities: []RepoIdentitySelection{{
			RepoURL:       forkSeedGitlabURL,
			GitIdentityID: "gid-plain",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var identitiesJSON string
	if err := db.QueryRow(`SELECT COALESCE(repo_identities_json,'') FROM task_comments WHERE id=?`, id).Scan(&identitiesJSON); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(identitiesJSON, `"oauth_gitsite":"`+forkSeedGitlabSite+`"`) {
		t.Fatalf("missing implicit oauth_gitsite: %s", identitiesJSON)
	}
}

func TestEnsureAutoRunAtComment_SeedsFromForkSource(t *testing.T) {
	setupTestDB(t)
	insertForkSeedTask(t, "task_ens_src", "", "u1")
	insertForkSeedComment(t, "cmt_ens_src", "task_ens_src", "u1",
		`[{"repo_url":"`+forkSeedGitlabURL+`","git_identity_id":"gid-src","oauth_gitsite":"`+forkSeedGitlabSite+`","oauth_remote_user_id":"gl-ens"}]`)
	insertForkSeedTask(t, "task_ens_dst", "task_ens_src", "u1")

	prevLookup := lookupInstalledImageFn
	lookupInstalledImageFn = func(tenantID, imageID string, _ ...string) (*InstalledImageLookup, error) {
		return &InstalledImageLookup{ID: imageID, Name: "img", ExternalImageID: "ext"}, nil
	}
	t.Cleanup(func() { lookupInstalledImageFn = prevLookup })

	prevURL := cfg.AICommentServiceURL
	cfg.AICommentServiceURL = ""
	t.Cleanup(func() { cfg.AICommentServiceURL = prevURL })

	id, err := ensureAutoRunAtComment(autoRunTriggerParams{
		TenantID:    "t1",
		WorkspaceID: "ws1",
		TaskID:      "task_ens_dst",
		UserID:      "u1",
		ImageID:     "img-1",
		RepoIdentities: []RepoIdentitySelection{{
			RepoURL:       forkSeedGitlabURL,
			GitIdentityID: "gid-dst",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var identitiesJSON string
	if err := db.QueryRow(`SELECT COALESCE(repo_identities_json,'') FROM task_comments WHERE id=?`, id).Scan(&identitiesJSON); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(identitiesJSON, `"oauth_gitsite":"`+forkSeedGitlabSite+`"`) {
		t.Fatalf("missing oauth_gitsite: %s", identitiesJSON)
	}
	if !strings.Contains(identitiesJSON, `"oauth_remote_user_id":"gl-ens"`) {
		t.Fatalf("missing remote: %s", identitiesJSON)
	}
	var decoded []map[string]interface{}
	if err := json.Unmarshal([]byte(identitiesJSON), &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded) == 0 {
		t.Fatal("empty identities")
	}
}
