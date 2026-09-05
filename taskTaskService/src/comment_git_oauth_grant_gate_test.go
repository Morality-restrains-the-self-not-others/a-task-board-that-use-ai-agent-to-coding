package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestCreateCommentMentionGitOAuthGate_BlocksGithubWithoutCommentGrant(t *testing.T) {
	setupTestDB(t)
	startMockProjectServiceWithAtMode(t, true)
	startMockCloudInstalledImageLookup(t, map[string]string{"img-42": "My Image"})
	prevURL := cfg.GitOAuthURL
	cfg.GitOAuthURL = "http://git-oauth.test"
	t.Cleanup(func() { cfg.GitOAuthURL = prevURL })

	prevConn := gitOAuthUserAppConnection
	gitOAuthUserAppConnection = func(_, _, _ string) (bool, error) { return true, nil }
	t.Cleanup(func() { gitOAuthUserAppConnection = prevConn })

	body := `{"content":"@My Image run","mentions":[{"type":"installed_image","id":"img-42","name":"My Image"}],"repo_identities":[{"repo_url":"https://github.com/acme/demo.git","git_identity_id":"gid-1","github_user_id":"9"}]}`
	rec := commentWithMentionAndRepo(t, body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("json: %v", err)
	}
	if out["code"] != errCodeCommentOAuthGrantMissing {
		t.Fatalf("code=%v body=%s", out["code"], rec.Body.String())
	}
	if out["gitsite"] != "github.com" {
		t.Fatalf("gitsite=%v", out["gitsite"])
	}
	if !strings.Contains(strings.TrimSpace(fmtString(out["error"])), "使用授权") {
		t.Fatalf("error=%v", out["error"])
	}
}

func TestCreateCommentMentionGitOAuthGate_AllowsGithubWithGrantTicket(t *testing.T) {
	setupTestDB(t)
	startMockProjectServiceWithAtMode(t, true)
	startMockCloudInstalledImageLookup(t, map[string]string{"img-42": "My Image"})
	prevURL := cfg.GitOAuthURL
	cfg.GitOAuthURL = "http://git-oauth.test"
	t.Cleanup(func() { cfg.GitOAuthURL = prevURL })

	prevConn := gitOAuthUserAppConnection
	gitOAuthUserAppConnection = func(_, _, _ string) (bool, error) { return true, nil }
	t.Cleanup(func() { gitOAuthUserAppConnection = prevConn })

	prevConsume := consumeGitOAuthGrantTicketFn
	consumeGitOAuthGrantTicketFn = func(userID, gitsite, ticketID string) (string, bool) {
		if userID != "u1" || gitsite != "github.com" || ticketID != "tkt-github" {
			t.Fatalf("consume user=%s site=%s ticket=%s", userID, gitsite, ticketID)
		}
		return "gh-user-1", true
	}
	t.Cleanup(func() { consumeGitOAuthGrantTicketFn = prevConsume })

	body := `{"content":"@My Image run","mentions":[{"type":"installed_image","id":"img-42","name":"My Image"}],"grant_ticket":"tkt-github","repo_identities":[{"repo_url":"https://github.com/acme/demo.git","git_identity_id":"gid-1","github_user_id":"9"}]}`
	rec := commentWithMentionAndRepo(t, body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("json: %v", err)
	}
	idents, _ := out["repo_identities"].([]interface{})
	if len(idents) != 1 {
		t.Fatalf("repo_identities=%v", out["repo_identities"])
	}
	row, _ := idents[0].(map[string]interface{})
	if strings.ToLower(fmtString(row["oauth_gitsite"])) != "github.com" {
		t.Fatalf("oauth_gitsite=%v", row["oauth_gitsite"])
	}
}

func TestLoadCommentOAuthGrantsForTask_IncludesStampedSites(t *testing.T) {
	setupTestDB(t)
	startMockProjectServiceWithAtMode(t, true)
	startMockCloudInstalledImageLookup(t, map[string]string{"img-42": "My Image"})
	taskID := createTestTaskForComments(t)
	_, err := db.Exec(
		`INSERT INTO task_comments(id,task_id,created_by_id,content,mentions_json,repo_identities_json,created_at) VALUES(?,?,?,?,?,?,NOW())`,
		"cmt_grant", taskID, "u1", "run", "[]",
		`[{"repo_url":"https://github.com/acme/demo.git","git_identity_id":"gid-1","oauth_gitsite":"github.com"}]`,
	)
	if err != nil {
		t.Fatalf("insert comment: %v", err)
	}
	grants := loadCommentOAuthGrantsForTask(taskID)
	if len(grants) != 1 {
		t.Fatalf("grants=%v", grants)
	}
	if grants[0]["user_id"] != "u1" || grants[0]["gitsite"] != "github.com" {
		t.Fatalf("grant=%v", grants[0])
	}
}

func TestMissingOAuthCapableGrantSite_UsesLinkedRepoURLs(t *testing.T) {
	if got := missingOAuthCapableGrantSite(nil, "https://github.com/acme/demo.git"); got != "github.com" {
		t.Fatalf("empty selections got=%q", got)
	}
	stamped := []RepoIdentitySelection{{
		RepoURL:      "https://github.com/acme/demo.git",
		OauthGitsite: "github.com",
	}}
	if got := missingOAuthCapableGrantSite(stamped, "https://github.com/acme/demo.git"); got != "" {
		t.Fatalf("stamped L2 got=%q", got)
	}
}

func fmtString(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}
