package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSubstBranchPlaceholders(t *testing.T) {
	got := substBranchPlaceholders(
		"feature/${taskId}_____${taskTitle}",
		"task_123",
		"hello world!",
	)
	want := "feature/task_123_____hello_world_"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestResolveLayerPushGithubPRMetadata_FromTaskService(t *testing.T) {
	prevTask := cfg.TaskServiceURL
	t.Cleanup(func() { cfg.TaskServiceURL = prevTask })

	taskSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":           "task_13762772779307981876",
			"title":        "写一个 hello world",
			"workspace_id": "w1",
			"branch_strategy": map[string]any{
				"merge_target_branch_name": "master",
				"target_branch_name":       "feature/2026-07-20_daydaymoney${taskId}_____hello_world",
				"work_branch_name":         "feature/2026-07-20_daydaymoney${taskId}_____hello_world",
			},
		})
	}))
	defer taskSrv.Close()
	cfg.TaskServiceURL = taskSrv.URL

	meta := resolveLayerPushGithubPRMetadata(
		"t1", "w1", "task_13762772779307981876",
		"feature/2026-07-20_daydaymoney${taskId}_____hello_world",
	)
	if meta == nil {
		t.Fatal("expected PR metadata")
	}
	if meta["pr_base_branch"] != "master" {
		t.Fatalf("pr_base_branch=%q", meta["pr_base_branch"])
	}
	wantHead := "feature/2026-07-20_daydaymoneytask_13762772779307981876_____hello_world"
	if meta["pr_title"] != "PR: 写一个 hello world" {
		t.Fatalf("pr_title=%q", meta["pr_title"])
	}
	// head is not returned in meta; ensure attach uses target_branch substitution via prepare
	_ = wantHead
	if !strings.Contains(meta["pr_body"], "task_13762772779307981876") {
		t.Fatalf("pr_body=%q", meta["pr_body"])
	}
}

func TestLayerGitPushPrepare_AttachesPRBaseBranch(t *testing.T) {
	cfg.InternalSecret = ""
	prevTask := cfg.TaskServiceURL
	prevOauth := cfg.GitOauthBaseURL
	t.Cleanup(func() {
		cfg.TaskServiceURL = prevTask
		cfg.GitOauthBaseURL = prevOauth
	})

	taskSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":           "task_1",
			"title":        "demo",
			"workspace_id": "w1",
			"branch_strategy": map[string]any{
				"merge_target_branch_name": "main",
				"target_branch_name":       "feat/x",
			},
			"projects": []any{
				map[string]any{
					"project_id":          "p1",
					"stored_repo_address": "https://github.com/acme/demo.git",
					"project_repo_url":    "https://github.com/acme/demo.git",
				},
			},
			"comments": []any{
				map[string]any{
					"created_by": map[string]any{"id": "42"},
					"repo_identities": []any{
						map[string]any{"oauth_gitsite": "github.com"},
					},
				},
			},
		})
	}))
	defer taskSrv.Close()
	cfg.TaskServiceURL = taskSrv.URL

	oauth := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "ghu_tok"})
	}))
	defer oauth.Close()
	cfg.GitOauthBaseURL = oauth.URL

	body := `{"tenant_id":"t1","workspace_id":"w1","task_id":"task_1","layer_id":"L1","user_id":"42","prefer_container_remote":true,"target_branch":"feat/x","repo_url":"https://github.com/acme/demo.git"}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/layer-git-push/prepare", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleInternalLayerGitPushPrepare(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	pb, _ := out["push_body"].(map[string]any)
	if pb["pr_base_branch"] != "main" {
		t.Fatalf("push_body missing pr_base_branch: %v", pb)
	}
	if pb["pr_title"] != "PR: demo" {
		t.Fatalf("pr_title=%v", pb["pr_title"])
	}
}
