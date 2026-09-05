package main

import (
	"context"
	"strings"
	"testing"
)

func TestValidateRepoIdentitiesForRunEmptyRequired(t *testing.T) {
	if err := ValidateRepoIdentitiesForRun(nil, nil); err != nil {
		t.Fatalf("empty required: %v", err)
	}
}

func TestValidateRepoIdentitiesForRunMissingGitIdentity(t *testing.T) {
	err := ValidateRepoIdentitiesForRun(
		[]RepoIdentitySelection{{RepoURL: "https://gitlab.example/c.git"}},
		[]string{"https://gitlab.example/c.git"},
	)
	if err == nil || !strings.Contains(err.Error(), errMsgRepoIdentitiesRequired) {
		t.Fatalf("err=%v", err)
	}
}

func TestValidateRepoIdentitiesForRunMissingGithubUser(t *testing.T) {
	err := ValidateRepoIdentitiesForRun(
		[]RepoIdentitySelection{{
			RepoURL:       "https://github.com/a/b.git",
			GitIdentityID: "gid-1",
		}},
		[]string{"https://github.com/a/b.git"},
	)
	if err == nil {
		t.Fatal("expected github user error")
	}
}

func TestValidateGitIdentitiesForCreateAutoRun(t *testing.T) {
	if err := ValidateGitIdentitiesForCreateAutoRun(nil, nil); err != nil {
		t.Fatalf("empty required: %v", err)
	}
	err := ValidateGitIdentitiesForCreateAutoRun(
		[]RepoIdentitySelection{{RepoURL: "https://github.com/a/b.git"}},
		[]string{"https://github.com/a/b.git"},
	)
	if err == nil || !strings.Contains(err.Error(), errMsgRepoIdentitiesRequired) {
		t.Fatalf("missing git identity err=%v", err)
	}
	if err := ValidateGitIdentitiesForCreateAutoRun(
		[]RepoIdentitySelection{{
			RepoURL:       "https://github.com/a/b.git",
			GitIdentityID: "gid-1",
		}},
		[]string{"https://github.com/a/b.git"},
	); err != nil {
		t.Fatalf("github create auto_run should not require github_user_id: %v", err)
	}
}

// OPT-20260821-022: auto_run 的 git_identity_id 必须属于当前用户/租户。
func TestValidateGitIdentitiesOwnership(t *testing.T) {
	setupTestDB(t)
	if _, err := db.Exec(`
		INSERT INTO task_git_identities (id, user_id, company_id, label, git_user_name, git_user_email, is_default, created_at, updated_at)
		VALUES ('gid-owner', 'u-owner', 'c-owner', '', 'Owner', 'owner@example.com', 0, NOW(), NOW()),
		       ('gid-other', 'u-other', 'c-owner', '', 'Other', 'other@example.com', 0, NOW(), NOW())`); err != nil {
		t.Fatalf("seed identities: %v", err)
	}

	// 自己身份 + 同租户 → OK
	if err := validateGitIdentitiesOwnership([]RepoIdentitySelection{{GitIdentityID: "gid-owner"}}, "u-owner", "c-owner"); err != nil {
		t.Fatalf("own identity should pass: %v", err)
	}
	// 他人身份 → 报错
	if err := validateGitIdentitiesOwnership([]RepoIdentitySelection{{GitIdentityID: "gid-other"}}, "u-owner", "c-owner"); err == nil {
		t.Fatal("expected error for other user's identity")
	}
	// 租户不匹配 → 报错
	if err := validateGitIdentitiesOwnership([]RepoIdentitySelection{{GitIdentityID: "gid-owner"}}, "u-owner", "c-other"); err == nil {
		t.Fatal("expected error for wrong tenant")
	}
	// 空 userID 跳过用户归属校验（内部调用），仍校验租户
	if err := validateGitIdentitiesOwnership([]RepoIdentitySelection{{GitIdentityID: "gid-owner"}}, "", "c-owner"); err != nil {
		t.Fatalf("empty userID should skip user check: %v", err)
	}
}

func TestResolveAutoRunRepoIdentitiesSkipsWhenAutoRunOff(t *testing.T) {
	got, err := resolveAutoRunRepoIdentities(context.Background(), false, "", "t1", "", map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatalf("got=%v", got)
	}
}

func TestValidateRepoIdentitiesForRunComplete(t *testing.T) {
	err := ValidateRepoIdentitiesForRun(
		[]RepoIdentitySelection{{
			RepoURL:       "https://github.com/a/b.git",
			GitIdentityID: "gid-1",
			GithubUserID:  "9",
		}},
		[]string{"https://github.com/a/b.git"},
	)
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
}

func TestParseAndJSONRoundTrip(t *testing.T) {
	raw := []interface{}{
		map[string]interface{}{
			"repo_url": "https://github.com/a/b.git", "git_identity_id": "gid-1", "github_user_id": "9",
		},
	}
	got, err := ParseRepoIdentities(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].GitIdentityID != "gid-1" || got[0].GithubUserID != "9" {
		t.Fatalf("got %#v", got)
	}
	js := RepoIdentitiesJSON(got)
	if js == "" || js == "[]" {
		t.Fatalf("json=%s", js)
	}
	maps := repoIdentityMapsFromJSON(js)
	if len(maps) != 1 || maps[0]["github_user_id"] != "9" {
		t.Fatalf("maps=%v", maps)
	}
}

func TestParseRepoIdentitiesInvalid(t *testing.T) {
	if _, err := ParseRepoIdentities("nope"); err == nil {
		t.Fatal("expected invalid")
	}
}

func TestResolveCommentRepoIdentitiesJSONRequiresOnMention(t *testing.T) {
	body := map[string]interface{}{}
	_, err := resolveCommentRepoIdentitiesJSON(body, []string{"https://git.example/a.git"}, true)
	if err == nil {
		t.Fatal("mention without identities")
	}
	js, err := resolveCommentRepoIdentitiesJSON(body, []string{"https://git.example/a.git"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if js != "[]" {
		t.Fatalf("plain comment json=%s", js)
	}
}

func TestRepoIdentityMapsFromJSONKeepsSiteLevelOAuthGrant(t *testing.T) {
	got := repoIdentityMapsFromJSON(`[{"repo_url":"","oauth_gitsite":"gitlab-tencent-sh-1.daydaymoney.com","oauth_remote_user_id":"gl-user"}]`)
	if len(got) != 1 {
		t.Fatalf("site-level L2 must not be dropped, got %+v", got)
	}
	if got[0]["oauth_gitsite"] != "gitlab-tencent-sh-1.daydaymoney.com" || got[0]["repo_url"] != "" {
		t.Fatalf("want site-level grant, got %+v", got[0])
	}
}
