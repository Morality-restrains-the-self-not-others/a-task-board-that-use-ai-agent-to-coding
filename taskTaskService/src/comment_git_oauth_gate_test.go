package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// commentWithMentionAndRepo posts an @镜像 comment with a repo identity.
// Task has no linked repos (required empty) so ValidateRepoIdentitiesForRun passes;
// the OAuth gate iterates the explicit repo identity.
func commentWithMentionAndRepo(t *testing.T, body string) *httptest.ResponseRecorder {
	t.Helper()
	taskID := createTestTaskForComments(t)
	return postComment(t, taskID, body)
}

func TestCreateCommentMentionGitOAuthGate_BlocksUnbound(t *testing.T) {
	setupTestDB(t)
	startMockProjectServiceWithAtMode(t, true)
	startMockCloudInstalledImageLookup(t, map[string]string{"img-42": "My Image"})
	prevURL := cfg.GitOAuthURL
	cfg.GitOAuthURL = "http://git-oauth.test"
	t.Cleanup(func() { cfg.GitOAuthURL = prevURL })

	prev := gitOAuthUserAppConnection
	gitOAuthUserAppConnection = func(tenantID, userID, repoURL string) (bool, error) {
		if tenantID != "t1" || userID != "u1" {
			t.Fatalf("gate called tenant=%s user=%s", tenantID, userID)
		}
		if repoURL != "https://git.example/repo.git" {
			t.Fatalf("repo_url=%s", repoURL)
		}
		return false, nil
	}
	t.Cleanup(func() { gitOAuthUserAppConnection = prev })

	body := `{"content":"@My Image run","mentions":[{"type":"installed_image","id":"img-42","name":"My Image"}],"repo_identities":[{"repo_url":"https://git.example/repo.git","git_identity_id":"gid-1"}]}`
	rec := commentWithMentionAndRepo(t, body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("json: %v", err)
	}
	if out["code"] != "git_oauth_not_connected" {
		t.Fatalf("code=%v", out["code"])
	}
	if strings.TrimSpace(strings.ToLower(out["repo_url"].(string))) != "https://git.example/repo.git" {
		t.Fatalf("repo_url=%v", out["repo_url"])
	}
}

func TestCreateCommentMentionGitOAuthGate_AllowsBound(t *testing.T) {
	setupTestDB(t)
	startMockProjectServiceWithAtMode(t, true)
	startMockCloudInstalledImageLookup(t, map[string]string{"img-42": "My Image"})
	prevURL := cfg.GitOAuthURL
	cfg.GitOAuthURL = "http://git-oauth.test"
	t.Cleanup(func() { cfg.GitOAuthURL = prevURL })

	prev := gitOAuthUserAppConnection
	gitOAuthUserAppConnection = func(_, _, _ string) (bool, error) { return true, nil }
	t.Cleanup(func() { gitOAuthUserAppConnection = prev })

	body := `{"content":"@My Image run","mentions":[{"type":"installed_image","id":"img-42","name":"My Image"}],"repo_identities":[{"repo_url":"https://git.example/repo.git","git_identity_id":"gid-1"}]}`
	rec := commentWithMentionAndRepo(t, body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestCreateCommentMentionGitOAuthGate_FailClosedOnError(t *testing.T) {
	setupTestDB(t)
	startMockProjectServiceWithAtMode(t, true)
	startMockCloudInstalledImageLookup(t, map[string]string{"img-42": "My Image"})
	prevURL := cfg.GitOAuthURL
	cfg.GitOAuthURL = "http://git-oauth.test"
	t.Cleanup(func() { cfg.GitOAuthURL = prevURL })

	prev := gitOAuthUserAppConnection
	gitOAuthUserAppConnection = func(_, _, _ string) (bool, error) {
		return false, errors.New("connection refused")
	}
	t.Cleanup(func() { gitOAuthUserAppConnection = prev })

	body := `{"content":"@My Image run","mentions":[{"type":"installed_image","id":"img-42","name":"My Image"}],"repo_identities":[{"repo_url":"https://git.example/repo.git","git_identity_id":"gid-1"}]}`
	rec := commentWithMentionAndRepo(t, body)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("json: %v", err)
	}
	if out["code"] != "git_oauth_check_unavailable" {
		t.Fatalf("code=%v", out["code"])
	}
}

func TestCreateCommentMentionSSHRepo_SkipsGate(t *testing.T) {
	setupTestDB(t)
	startMockProjectServiceWithAtMode(t, true)
	startMockCloudInstalledImageLookup(t, map[string]string{"img-42": "My Image"})

	called := false
	prev := gitOAuthUserAppConnection
	gitOAuthUserAppConnection = func(_, _, _ string) (bool, error) {
		called = true
		return true, nil
	}
	t.Cleanup(func() { gitOAuthUserAppConnection = prev })

	body := `{"content":"@My Image run","mentions":[{"type":"installed_image","id":"img-42","name":"My Image"}],"repo_identities":[{"repo_url":"git@git.example:org/repo.git","git_identity_id":"gid-1"}]}`
	rec := commentWithMentionAndRepo(t, body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if called {
		t.Fatal("SSH remote must not trigger Git OAuth gate")
	}
}

func TestCreateCommentWithoutMention_SkipsGate(t *testing.T) {
	setupTestDB(t)
	startMockProjectServiceWithAtMode(t, true)
	startMockCloudInstalledImageLookup(t, map[string]string{"img-42": "My Image"})

	called := false
	prev := gitOAuthUserAppConnection
	gitOAuthUserAppConnection = func(_, _, _ string) (bool, error) {
		called = true
		return true, nil
	}
	t.Cleanup(func() { gitOAuthUserAppConnection = prev })

	taskID := createTestTaskForComments(t)
	body := `{"content":"plain comment","repo_identities":[{"repo_url":"https://git.example/repo.git","git_identity_id":"gid-1"}]}`
	rec := postComment(t, taskID, body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if called {
		t.Fatal("non-mention comment must skip Git OAuth gate")
	}
}
